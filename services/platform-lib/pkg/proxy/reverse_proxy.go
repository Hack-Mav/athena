package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/athena/platform-lib/pkg/discovery"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/athena/platform-lib/pkg/resilience"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// ReverseProxy handles HTTP reverse proxying with service discovery
type ReverseProxy struct {
	registry     *discovery.ServiceRegistry
	loadBalancer discovery.LoadBalancer
	resilience   map[string]*resilience.ResilienceClient
	logger       *logger.Logger
	tracer       trace.Tracer
	config       *ProxyConfig
}

// ProxyConfig holds proxy configuration
type ProxyConfig struct {
	Timeout                time.Duration `json:"timeout"`
	RetryAttempts          int           `json:"retry_attempts"`
	CircuitBreakerFailures int           `json:"circuit_breaker_failures"`
	EnableTracing          bool          `json:"enable_tracing"`
	StripPrefix            bool          `json:"strip_prefix"`
}

// ServiceProxy represents a proxy for a specific service
type ServiceProxy struct {
	serviceName string
	proxy       *ReverseProxy
}

// NewReverseProxy creates a new reverse proxy
func NewReverseProxy(
	registry *discovery.ServiceRegistry,
	loadBalancer discovery.LoadBalancer,
	logger *logger.Logger,
	tracer trace.Tracer,
	config *ProxyConfig,
) *ReverseProxy {
	if config == nil {
		config = &ProxyConfig{
			Timeout:                30 * time.Second,
			RetryAttempts:          3,
			CircuitBreakerFailures: 5,
			EnableTracing:          true,
			StripPrefix:            true,
		}
	}

	proxy := &ReverseProxy{
		registry:     registry,
		loadBalancer: loadBalancer,
		resilience:   make(map[string]*resilience.ResilienceClient),
		logger:       logger,
		tracer:       tracer,
		config:       config,
	}

	return proxy
}

// ProxyHandler returns a gin handler for proxying requests
func (rp *ReverseProxy) ProxyHandler(serviceName string) gin.HandlerFunc {
	// Initialize resilience client for this service if not exists
	if _, exists := rp.resilience[serviceName]; !exists {
		cbConfig := &resilience.CircuitBreakerConfig{
			Name:           serviceName,
			MaxFailures:    rp.config.CircuitBreakerFailures,
			ResetTimeout:   60 * time.Second,
			HalfOpenMaxReq: 3,
		}
		retryConfig := &resilience.RetryConfig{
			MaxRetries:    rp.config.RetryAttempts,
			InitialDelay:  100 * time.Millisecond,
			MaxDelay:      5 * time.Second,
			BackoffFactor: 2.0,
		}
		rp.resilience[serviceName] = resilience.NewResilienceClient(cbConfig, retryConfig, rp.logger)
	}

	return func(c *gin.Context) {
		// Get service URL
		serviceURL, err := rp.registry.GetServiceURL(serviceName, rp.loadBalancer)
		if err != nil {
			rp.logger.Errorf("Failed to get service URL for %s: %v", serviceName, err)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": fmt.Sprintf("Service %s is unavailable", serviceName),
			})
			return
		}

		// Create target URL
		target, err := url.Parse(serviceURL)
		if err != nil {
			rp.logger.Errorf("Failed to parse service URL %s: %v", serviceURL, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
			})
			return
		}

		// Create proxy
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.ErrorHandler = rp.errorHandler
		proxy.ModifyResponse = rp.modifyResponse

		// Set timeout
		proxy.Transport = &http.Transport{
			ResponseHeaderTimeout: rp.config.Timeout,
			IdleConnTimeout:       rp.config.Timeout,
		}

		// Handle request with resilience
		err = rp.resilience[serviceName].Execute(c.Request.Context(), func() error {
			// Add tracing headers
			if rp.config.EnableTracing {
				rp.addTracingHeaders(c, proxy)
			}

			// Modify request path if needed
			if rp.config.StripPrefix {
				rp.stripServicePrefix(c, serviceName)
			}

			proxy.ServeHTTP(c.Writer, c.Request)
			return nil
		})

		if err != nil {
			rp.logger.Errorf("Proxy request failed for %s: %v", serviceName, err)
			if !c.Writer.Written() {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error": fmt.Sprintf("Service %s is temporarily unavailable", serviceName),
				})
			}
		}
	}
}

// errorHandler handles proxy errors
func (rp *ReverseProxy) errorHandler(rw http.ResponseWriter, r *http.Request, err error) {
	rp.logger.Errorf("Proxy error: %v", err)

	if r.Context().Err() == context.DeadlineExceeded {
		http.Error(rw, "Gateway timeout", http.StatusGatewayTimeout)
		return
	}

	http.Error(rw, "Bad gateway", http.StatusBadGateway)
}

// modifyResponse modifies the response from the target service
func (rp *ReverseProxy) modifyResponse(resp *http.Response) error {
	// Add custom headers
	resp.Header.Set("X-Gateway-Service", "athena-api-gateway")
	resp.Header.Set("X-Gateway-Version", "1.0.0")

	// Remove hop-by-hop headers
	hopHeaders := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"Te",
		"Trailers",
		"Transfer-Encoding",
		"Upgrade",
	}

	for _, header := range hopHeaders {
		resp.Header.Del(header)
	}

	return nil
}

// addTracingHeaders adds tracing headers to the request
func (rp *ReverseProxy) addTracingHeaders(c *gin.Context, proxy *httputil.ReverseProxy) {
	ctx := c.Request.Context()
	span := trace.SpanFromContext(ctx)

	if span.SpanContext().IsValid() {
		spanCtx := span.SpanContext()
		c.Request.Header.Set("X-Trace-Id", spanCtx.TraceID().String())
		c.Request.Header.Set("X-Span-Id", spanCtx.SpanID().String())
	}
}

// stripServicePrefix removes service prefix from request path
func (rp *ReverseProxy) stripServicePrefix(c *gin.Context, serviceName string) {
	path := c.Request.URL.Path
	prefix := fmt.Sprintf("/api/v1/%s", serviceName)

	if strings.HasPrefix(path, prefix) {
		c.Request.URL.Path = strings.TrimPrefix(path, prefix)
		if c.Request.URL.Path == "" {
			c.Request.URL.Path = "/"
		}
	}
}

// ProxyRequest handles direct proxy requests
func (rp *ReverseProxy) ProxyRequest(serviceName, method, path string, headers map[string]string, body []byte) (*http.Response, error) {
	// Get service URL
	serviceURL, err := rp.registry.GetServiceURL(serviceName, rp.loadBalancer)
	if err != nil {
		return nil, fmt.Errorf("service %s is unavailable: %w", serviceName, err)
	}

	// Create target URL
	targetURL := serviceURL + path

	// Create request
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, targetURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute with resilience
	var resp *http.Response
	err = rp.resilience[serviceName].Execute(context.Background(), func() error {
		client := &http.Client{
			Timeout: rp.config.Timeout,
		}

		var err error
		resp, err = client.Do(req)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetServiceHealth returns health information for a service
func (rp *ReverseProxy) GetServiceHealth(serviceName string) map[string]interface{} {
	stats := rp.registry.GetServiceStats(serviceName)

	health := map[string]interface{}{
		"service": serviceName,
		"stats":   stats,
		"status":  "unknown",
	}

	// Determine overall status
	if stats["total"] == 0 {
		health["status"] = "no_instances"
	} else if stats["healthy"] == stats["total"] {
		health["status"] = "healthy"
	} else if stats["healthy"] > 0 {
		health["status"] = "degraded"
	} else {
		health["status"] = "unhealthy"
	}

	// Add circuit breaker status if available
	if resilienceClient, exists := rp.resilience[serviceName]; exists {
		health["circuit_breaker"] = resilienceClient.GetCircuitBreaker().GetStats()
	}

	return health
}

// GetAllServicesHealth returns health information for all services
func (rp *ReverseProxy) GetAllServicesHealth() map[string]interface{} {
	services := []string{
		"template-service",
		"nlp-service",
		"provisioning-service",
		"device-service",
		"telemetry-service",
		"ota-service",
		"secrets-service",
	}

	health := make(map[string]interface{})
	for _, serviceName := range services {
		health[serviceName] = rp.GetServiceHealth(serviceName)
	}

	return health
}

// NewServiceProxy creates a new service proxy
func NewServiceProxy(serviceName string, proxy *ReverseProxy) *ServiceProxy {
	return &ServiceProxy{
		serviceName: serviceName,
		proxy:       proxy,
	}
}

// Handle returns the gin handler for this service proxy
func (sp *ServiceProxy) Handle() gin.HandlerFunc {
	return sp.proxy.ProxyHandler(sp.serviceName)
}
