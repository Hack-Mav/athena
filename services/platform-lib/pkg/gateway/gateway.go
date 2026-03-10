package gateway

import (
	"context"
	"net/http"
	"time"

	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/discovery"
	"github.com/athena/platform-lib/pkg/health"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/athena/platform-lib/pkg/middleware"
	"github.com/athena/platform-lib/pkg/proxy"
	"github.com/athena/platform-lib/pkg/tracing"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

// Gateway represents the API gateway
type Gateway struct {
	config        *config.Config
	logger        *logger.Logger
	authHandler   *AuthHandler
	jwtAuth       *middleware.JWTAuth
	healthChecker *health.HealthChecker
	registry      *discovery.ServiceRegistry
	reverseProxy  *proxy.ReverseProxy
	tracingMgr    *tracing.TracingManager
}

// NewGateway creates a new API gateway instance
func NewGateway(cfg *config.Config, log *logger.Logger) (*Gateway, error) {
	// Initialize components
	authHandler := NewAuthHandler(cfg, *log)
	jwtAuth := middleware.NewJWTAuth(cfg.JWTSecret, "athena-platform")
	healthChecker := health.NewHealthChecker("1.0.0")

	// Initialize service discovery
	registry := discovery.NewServiceRegistry(log, &discovery.RegistryConfig{
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
		MaxRetries:          3,
	})

	// Initialize tracing
	tracingMgr, err := tracing.NewTracingManager(&tracing.TracerConfig{
		ServiceName:   "api-gateway",
		Environment:   cfg.Environment,
		Provider:      "stdout", // Can be changed to jaeger or zipkin
		SampleRate:    1.0,
		EnableTracing: true,
	}, log)
	if err != nil {
		log.Warnf("Failed to initialize tracing: %v", err)
	}

	// Initialize reverse proxy
	loadBalancer := discovery.NewRoundRobinLoadBalancer()
	var tracer trace.Tracer
	if tracingMgr != nil {
		tracer = tracingMgr.GetTracer()
	}
	reverseProxy := proxy.NewReverseProxy(registry, loadBalancer, log, tracer, &proxy.ProxyConfig{
		Timeout:                30 * time.Second,
		RetryAttempts:          3,
		CircuitBreakerFailures: 5,
		EnableTracing:          true,
		StripPrefix:            true,
	})

	// Register services from configuration
	registerServicesFromConfig(registry, cfg)

	return &Gateway{
		config:        cfg,
		logger:        log,
		authHandler:   authHandler,
		jwtAuth:       jwtAuth,
		healthChecker: healthChecker,
		registry:      registry,
		reverseProxy:  reverseProxy,
		tracingMgr:    tracingMgr,
	}, nil
}

// registerServicesFromConfig registers services from configuration
func registerServicesFromConfig(registry *discovery.ServiceRegistry, cfg *config.Config) {
	for serviceName, serviceURL := range cfg.Services {
		// Parse URL to extract address and port
		if serviceURL != "" {
			instance := &discovery.ServiceInstance{
				ID:      serviceName + "-1",
				Name:    serviceName,
				Address: "localhost", // Extract from URL in production
				Port:    8001,        // Extract from URL in production
				Status:  "healthy",
				Metadata: map[string]string{
					"url": serviceURL,
				},
			}
			registry.RegisterService(instance)
		}
	}
}

// Shutdown gracefully shuts down the gateway
func (g *Gateway) Shutdown() error {
	if g.tracingMgr != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return g.tracingMgr.Shutdown(ctx)
	}
	return nil
}

// RegisterRoutes registers HTTP routes for the API gateway
func RegisterRoutes(router *gin.Engine, gateway *Gateway) {
	// Global middleware
	router.Use(middleware.SecurityHeadersMiddleware())
	router.Use(middleware.NewRateLimitMiddleware(100).RateLimit()) // 100 requests per minute
	router.Use(middleware.NewValidationMiddleware().SanitizeInput())

	// Health endpoints (public, no auth required)
	router.GET("/health", gin.WrapH(http.HandlerFunc(gateway.healthChecker.HealthHandlerFunc())))
	router.GET("/ready", gin.WrapH(http.HandlerFunc(gateway.healthChecker.ReadinessHandlerFunc())))
	router.GET("/live", gin.WrapH(http.HandlerFunc(gateway.healthChecker.LivenessHandlerFunc())))

	// Service health endpoints (public, no auth required)
	router.GET("/services/health", gateway.getServiceHealth)
	router.GET("/services/health/:serviceName", gateway.getServiceHealthByName)

	// Authentication routes (public, but validated)
	auth := router.Group("/api/v1")
	{
		auth.POST("/auth/login", gateway.authHandler.Login)
		auth.POST("/auth/refresh", gateway.authHandler.RefreshToken)
		auth.POST("/auth/logout", gateway.jwtAuth.RequireAuth(), gateway.authHandler.Logout)

		// Protected routes for testing
		protected := auth.Group("/auth/protected")
		protected.Use(gateway.jwtAuth.RequireAuth())
		{
			protected.GET("/profile", gateway.authHandler.GetProfile)
			protected.GET("/admin", gateway.jwtAuth.RequireRole("admin"), gateway.authHandler.AdminOnly)
		}
	}

	// API routes with service proxying
	v1 := router.Group("/api/v1")
	v1.Use(gateway.jwtAuth.RequireAuth())
	{
		// Template service routes (with validation)
		templates := v1.Group("/templates")
		templates.Use(middleware.NewValidationMiddleware().ValidateQuery(map[string]string{
			"limit":  "max=100",
			"offset": "min=0",
			"search": "safe_string",
		}))
		{
			templates.GET("", gateway.proxyToTemplateService)
			templates.GET("/:id", gateway.proxyToTemplateService)
		}

		// NLP service routes (with validation)
		nlp := v1.Group("/nlp")
		{
			nlp.POST("/parse", middleware.NewValidationMiddleware().ValidateBody(&struct {
				Text     string `json:"text" binding:"required"`
				Language string `json:"language,omitempty"`
			}{}), gateway.proxyToNLPService)
			nlp.POST("/plan", middleware.NewValidationMiddleware().ValidateBody(&struct {
				Request string `json:"request" binding:"required"`
				Context string `json:"context,omitempty"`
			}{}), gateway.proxyToNLPService)
		}

		// Provisioning service routes (with validation)
		provisioning := v1.Group("/provisioning")
		{
			provisioning.POST("/compile", middleware.NewValidationMiddleware().ValidateBody(&struct {
				Code    string `json:"code" binding:"required"`
				Board   string `json:"board" binding:"required,oneof=uno nano esp32 mega"`
				Options string `json:"options,omitempty"`
			}{}), gateway.proxyToProvisioningService)
			provisioning.POST("/flash", middleware.NewValidationMiddleware().ValidateBody(&struct {
				DeviceID string `json:"device_id" binding:"required"`
				Firmware string `json:"firmware" binding:"required"`
				Port     string `json:"port,omitempty"`
			}{}), gateway.proxyToProvisioningService)
		}

		// Device service routes (with validation)
		devices := v1.Group("/devices")
		devices.Use(middleware.NewValidationMiddleware().ValidateQuery(map[string]string{
			"status": "oneof=online offline error",
			"type":   "safe_string",
		}))
		{
			devices.GET("", gateway.proxyToDeviceService)
			devices.GET("/:id", gateway.proxyToDeviceService)
		}

		// Telemetry service routes (with validation)
		telemetry := v1.Group("/telemetry")
		{
			telemetry.POST("/ingest", middleware.NewValidationMiddleware().ValidateBody(&struct {
				DeviceID  string                 `json:"device_id" binding:"required"`
				Data      map[string]interface{} `json:"data" binding:"required"`
				Timestamp int64                  `json:"timestamp,omitempty"`
			}{}), gateway.proxyToTelemetryService)
			telemetry.GET("/metrics/:deviceId", middleware.NewValidationMiddleware().ValidateQuery(map[string]string{
				"start":    "min=0",
				"end":      "min=0",
				"interval": "oneof=1m 5m 15m 1h",
			}), gateway.proxyToTelemetryService)
		}

		// OTA service routes (with validation)
		ota := v1.Group("/ota")
		{
			ota.POST("/releases", middleware.NewValidationMiddleware().ValidateBody(&struct {
				Version     string `json:"version" binding:"required,semver"`
				Description string `json:"description" binding:"required"`
				Firmware    string `json:"firmware" binding:"required"`
				DeviceType  string `json:"device_type" binding:"required"`
			}{}), gateway.proxyToOTAService)
			ota.GET("/updates/:deviceId", gateway.proxyToOTAService)
		}
	}
}

func (g *Gateway) proxyToTemplateService(c *gin.Context) {
	g.reverseProxy.ProxyHandler("template-service")(c)
}

func (g *Gateway) proxyToNLPService(c *gin.Context) {
	g.reverseProxy.ProxyHandler("nlp-service")(c)
}

func (g *Gateway) proxyToProvisioningService(c *gin.Context) {
	g.reverseProxy.ProxyHandler("provisioning-service")(c)
}

func (g *Gateway) proxyToDeviceService(c *gin.Context) {
	g.reverseProxy.ProxyHandler("device-service")(c)
}

func (g *Gateway) proxyToTelemetryService(c *gin.Context) {
	g.reverseProxy.ProxyHandler("telemetry-service")(c)
}

func (g *Gateway) proxyToOTAService(c *gin.Context) {
	g.reverseProxy.ProxyHandler("ota-service")(c)
}

// getServiceHealth returns health information for all services
func (g *Gateway) getServiceHealth(c *gin.Context) {
	health := g.reverseProxy.GetAllServicesHealth()
	c.JSON(http.StatusOK, gin.H{
		"services":  health,
		"timestamp": time.Now().Unix(),
	})
}

// getServiceHealthByName returns health information for a specific service
func (g *Gateway) getServiceHealthByName(c *gin.Context) {
	serviceName := c.Param("serviceName")
	health := g.reverseProxy.GetServiceHealth(serviceName)
	c.JSON(http.StatusOK, gin.H{
		"service":   serviceName,
		"health":    health,
		"timestamp": time.Now().Unix(),
	})
}
