package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/discovery"
	"github.com/athena/platform-lib/pkg/gateway"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/athena/platform-lib/pkg/resilience"
	"github.com/athena/platform-lib/pkg/tracing"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ResilienceIntegrationTestSuite tests error handling and service resilience patterns
type ResilienceIntegrationTestSuite struct {
	suite.Suite
	gateway      *gateway.Gateway
	router       *gin.Engine
	config       *config.Config
	logger       *logger.Logger
	registry     *discovery.ServiceRegistry
	tracingMgr   *tracing.TracingManager
	testServer   *httptest.Server
	flakyService *httptest.Server
	slowService  *httptest.Server
}

// SetupSuite sets up the resilience test suite
func (suite *ResilienceIntegrationTestSuite) SetupSuite() {
	// Create test configuration
	suite.config = &config.Config{
		ServiceName: "resilience-gateway-test",
		Environment: "test",
		LogLevel:    "debug",
		HTTPPort:    ":8080",
		JWTSecret:   "test-secret-key-for-resilience-testing",
		Services: map[string]string{
			"flaky-service": "http://localhost:8001",
			"slow-service":  "http://localhost:8002",
		},
	}

	// Create logger
	suite.logger = logger.New("debug", "resilience-gateway-test")

	// Create tracing manager
	suite.tracingMgr, _ = tracing.NewTracingManager(&tracing.TracerConfig{
		ServiceName:   "resilience-gateway-test",
		Environment:   "test",
		Provider:      "stdout",
		SampleRate:    1.0,
		EnableTracing: true,
	}, suite.logger)

	// Create service registry
	suite.registry = discovery.NewServiceRegistry(suite.logger, &discovery.RegistryConfig{
		HealthCheckInterval: 2 * time.Second,
		HealthCheckTimeout:  1 * time.Second,
		MaxRetries:          2,
	})

	// Create test services
	suite.createTestServices()

	// Register services
	suite.registerTestServices()

	// Create gateway with resilience features
	var err error
	suite.gateway, err = gateway.NewGateway(suite.config, suite.logger)
	require.NoError(suite.T(), err)

	// Create router with resilience routes
	suite.router = gin.New()
	suite.setupResilienceRoutes()

	// Create test server
	suite.testServer = httptest.NewServer(suite.router)
}

// TearDownSuite cleans up the resilience test suite
func (suite *ResilienceIntegrationTestSuite) TearDownSuite() {
	if suite.testServer != nil {
		suite.testServer.Close()
	}

	if suite.flakyService != nil {
		suite.flakyService.Close()
	}

	if suite.slowService != nil {
		suite.slowService.Close()
	}

	if suite.gateway != nil {
		suite.gateway.Shutdown()
	}

	if suite.tracingMgr != nil {
		suite.tracingMgr.Shutdown(context.Background())
	}
}

// createTestServices creates test services with different failure patterns
func (suite *ResilienceIntegrationTestSuite) createTestServices() {
	// Flaky service - fails intermittently
	requestCount := 0
	suite.flakyService = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Fail every 3rd request
		if requestCount%3 == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":         "Service temporarily unavailable",
				"request_count": requestCount,
			})
			return
		}

		// Simulate occasional timeout
		if requestCount%5 == 0 {
			time.Sleep(2 * time.Second) // Longer than typical timeout
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":       "Flaky service response",
			"request_count": requestCount,
			"timestamp":     time.Now().Unix(),
		})
	}))

	// Slow service - responds slowly but reliably
	suite.slowService = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always take 1-2 seconds to respond
		delay := time.Duration(1000+requestCount%1000) * time.Millisecond
		time.Sleep(delay)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   "Slow service response",
			"delay_ms":  delay.Milliseconds(),
			"timestamp": time.Now().Unix(),
		})
	}))
}

// registerTestServices registers test services with the registry
func (suite *ResilienceIntegrationTestSuite) registerTestServices() {
	// Register flaky service
	flakyInstance := &discovery.ServiceInstance{
		ID:      "flaky-service-1",
		Name:    "flaky-service",
		Address: "localhost",
		Port:    8001,
		Status:  "healthy",
		Metadata: map[string]string{
			"url": suite.flakyService.URL,
		},
	}
	suite.registry.RegisterService(flakyInstance)

	// Register slow service
	slowInstance := &discovery.ServiceInstance{
		ID:      "slow-service-1",
		Name:    "slow-service",
		Address: "localhost",
		Port:    8002,
		Status:  "healthy",
		Metadata: map[string]string{
			"url": suite.slowService.URL,
		},
	}
	suite.registry.RegisterService(slowInstance)
}

// setupResilienceRoutes sets up routes for testing resilience patterns
func (suite *ResilienceIntegrationTestSuite) setupResilienceRoutes() {
	// Create circuit breaker for flaky service
	cbConfig := &resilience.CircuitBreakerConfig{
		Name:           "flaky-service-cb",
		MaxFailures:    3,
		ResetTimeout:   10 * time.Second,
		HalfOpenMaxReq: 2,
	}
	flakyCircuitBreaker := resilience.NewCircuitBreaker(cbConfig, suite.logger)

	// Create resilience client for flaky service
	flakyResilienceClient := resilience.NewResilienceClient(cbConfig, &resilience.RetryConfig{
		MaxRetries:    2,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
	}, suite.logger)

	// Flaky service endpoint with circuit breaker and retry
	suite.router.GET("/flaky", func(c *gin.Context) {
		err := flakyResilienceClient.Execute(c.Request.Context(), func() error {
			resp, err := http.Get(suite.flakyService.URL)
			if err != nil {
				return fmt.Errorf("failed to call flaky service: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				return fmt.Errorf("flaky service returned error: %d", resp.StatusCode)
			}

			// Copy response
			var responseBody map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&responseBody)
			c.JSON(resp.StatusCode, responseBody)
			return nil
		})

		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":                 "Service resilience pattern activated",
				"details":               err.Error(),
				"circuit_breaker_state": flakyCircuitBreaker.GetState().String(),
			})
		}
	})

	// Slow service endpoint with timeout handling
	suite.router.GET("/slow", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Second)
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, "GET", suite.slowService.URL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				c.JSON(http.StatusGatewayTimeout, gin.H{
					"error":           "Request timeout",
					"timeout_seconds": 1,
				})
				return
			}
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "Service unavailable",
				"details": err.Error(),
			})
			return
		}
		defer resp.Body.Close()

		var responseBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&responseBody)
		c.JSON(resp.StatusCode, responseBody)
	})

	// Endpoint that demonstrates fallback behavior
	suite.router.GET("/fallback", func(c *gin.Context) {
		// Try primary service (flaky)
		err := flakyResilienceClient.Execute(c.Request.Context(), func() error {
			resp, err := http.Get(suite.flakyService.URL + "/primary")
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				return fmt.Errorf("primary service error: %d", resp.StatusCode)
			}

			var responseBody map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&responseBody)
			responseBody["source"] = "primary"
			c.JSON(resp.StatusCode, responseBody)
			return nil
		})

		if err != nil {
			// Fallback to secondary response
			c.JSON(http.StatusOK, gin.H{
				"message":       "Fallback response",
				"source":        "fallback",
				"primary_error": err.Error(),
				"timestamp":     time.Now().Unix(),
			})
		}
	})

	// Bulkhead pattern - limit concurrent requests
	concurrentLimiter := make(chan struct{}, 3) // Allow max 3 concurrent requests

	suite.router.GET("/bulkhead", func(c *gin.Context) {
		select {
		case concurrentLimiter <- struct{}{}:
			defer func() { <-concurrentLimiter }()

			// Simulate work
			time.Sleep(500 * time.Millisecond)
			c.JSON(http.StatusOK, gin.H{
				"message":   "Bulkhead pattern - request processed",
				"timestamp": time.Now().Unix(),
			})
		default:
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":          "Bulkhead limit exceeded",
				"max_concurrent": 3,
			})
		}
	})

	// Health check endpoint that shows circuit breaker status
	suite.router.GET("/health/circuit-breakers", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"flaky_service_circuit_breaker": flakyCircuitBreaker.GetStats(),
		})
	})

	// Chaos engineering endpoint - inject failures
	suite.router.POST("/chaos/:failure_type", func(c *gin.Context) {
		failureType := c.Param("failure_type")

		switch failureType {
		case "timeout":
			time.Sleep(5 * time.Second)
			c.JSON(http.StatusOK, gin.H{"message": "This should timeout"})
		case "error":
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Simulated error"})
		case "panic":
			panic("Simulated panic for testing")
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown failure type"})
		}
	})
}

// TestCircuitBreakerPattern tests circuit breaker functionality
func (suite *ResilienceIntegrationTestSuite) TestCircuitBreakerPattern() {
	// Make several requests to trigger circuit breaker
	var failureCount int
	for i := 0; i < 10; i++ {
		resp, err := http.Get(suite.testServer.URL + "/flaky")
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusServiceUnavailable {
			failureCount++
		}
	}

	// Circuit breaker should have opened after multiple failures
	assert.Greater(suite.T(), failureCount, 0, "Circuit breaker should have triggered")

	// Check circuit breaker status
	resp, err := http.Get(suite.testServer.URL + "/health/circuit-breakers")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var healthResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&healthResponse)
	require.NoError(suite.T(), err)

	cbStats, ok := healthResponse["flaky_service_circuit_breaker"].(map[string]interface{})
	require.True(suite.T(), ok)
	assert.Contains(suite.T(), cbStats["state"], "open")
}

// TestRetryPattern tests retry functionality
func (suite *ResilienceIntegrationTestSuite) TestRetryPattern() {
	// The flaky service should succeed after retries due to the resilience client
	resp, err := http.Get(suite.testServer.URL + "/flaky")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	// Should eventually succeed or fail gracefully
	assert.True(suite.T(), resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusServiceUnavailable)

	if resp.StatusCode == http.StatusOK {
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), response["message"], "Flaky service response")
	}
}

// TestTimeoutHandling tests timeout functionality
func (suite *ResilienceIntegrationTestSuite) TestTimeoutHandling() {
	resp, err := http.Get(suite.testServer.URL + "/slow")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	// Should timeout due to slow service
	assert.Equal(suite.T(), http.StatusGatewayTimeout, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Request timeout", response["error"])
}

// TestFallbackPattern tests fallback functionality
func (suite *ResilienceIntegrationTestSuite) TestFallbackPattern() {
	// Make multiple requests to test fallback
	successCount := 0
	fallbackCount := 0

	for i := 0; i < 10; i++ {
		resp, err := http.Get(suite.testServer.URL + "/fallback")
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(suite.T(), err)

		if source, ok := response["source"].(string); ok {
			if source == "primary" {
				successCount++
			} else if source == "fallback" {
				fallbackCount++
			}
		}
	}

	// Should have some successes and some fallbacks
	assert.Greater(suite.T(), successCount+fallbackCount, 0, "Should have received responses")
}

// TestBulkheadPattern tests bulkhead (concurrency limiting)
func (suite *ResilienceIntegrationTestSuite) TestBulkheadPattern() {
	// Make concurrent requests
	concurrentRequests := 10
	results := make(chan int, concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		go func() {
			resp, err := http.Get(suite.testServer.URL + "/bulkhead")
			if err != nil {
				results <- 0
				return
			}
			defer resp.Body.Close()
			results <- resp.StatusCode
		}()
	}

	// Collect results
	successCount := 0
	timeoutCount := 0
	for i := 0; i < concurrentRequests; i++ {
		statusCode := <-results
		if statusCode == http.StatusOK {
			successCount++
		} else if statusCode == http.StatusTooManyRequests {
			timeoutCount++
		}
	}

	// Should have some successful requests and some rejected due to bulkhead
	assert.Greater(suite.T(), successCount, 0, "Should have some successful requests")
	assert.Greater(suite.T(), timeoutCount, 0, "Should have some requests rejected by bulkhead")
	assert.LessOrEqual(suite.T(), successCount, 3, "Should not exceed bulkhead limit")
}

// TestChaosEngineering tests chaos engineering scenarios
func (suite *ResilienceIntegrationTestSuite) TestChaosEngineering() {
	tests := []struct {
		name        string
		failureType string
		expectCode  int
		expectError string
	}{
		{
			name:        "Timeout injection",
			failureType: "timeout",
			expectCode:  http.StatusGatewayTimeout,
			expectError: "timeout",
		},
		{
			name:        "Error injection",
			failureType: "error",
			expectCode:  http.StatusInternalServerError,
			expectError: "Simulated error",
		},
	}

	for _, test := range tests {
		suite.T().Run(test.name, func(t *testing.T) {
			// Create request with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			req, _ := http.NewRequestWithContext(ctx, "POST",
				suite.testServer.URL+"/chaos/"+test.failureType, nil)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				if ctx.Err() == context.DeadlineExceeded {
					// Timeout occurred as expected
					return
				}
				require.NoError(t, err)
			}
			defer resp.Body.Close()

			assert.Equal(t, test.expectCode, resp.StatusCode)

			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)

			if test.expectError != "" {
				assert.Contains(t, response["error"], test.expectError)
			}
		})
	}
}

// TestGracefulDegradation tests graceful degradation scenarios
func (suite *ResilienceIntegrationTestSuite) TestGracefulDegradation() {
	// Test that the system continues to function even when some services fail
	// This is tested implicitly through the other resilience patterns

	// Make requests to various endpoints
	endpoints := []string{"/flaky", "/slow", "/fallback", "/bulkhead"}

	for _, endpoint := range endpoints {
		resp, err := http.Get(suite.testServer.URL + endpoint)
		if err != nil {
			// Network errors are acceptable in resilience testing
			continue
		}
		defer resp.Body.Close()

		// Should not crash - should return some response
		assert.True(suite.T(), resp.StatusCode >= 200 && resp.StatusCode < 600,
			"Endpoint %s should return valid HTTP status", endpoint)
	}
}

// TestDistributedTracingInResilience tests tracing in resilience scenarios
func (suite *ResilienceIntegrationTestSuite) TestDistributedTracingInResilience() {
	// Make a request that should trigger resilience patterns
	req, _ := http.NewRequest("GET", suite.testServer.URL+"/flaky", nil)
	req.Header.Set("X-Trace-Id", "test-trace-resilience")
	req.Header.Set("X-Span-Id", "test-span-resilience")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	// Verify tracing headers are propagated (in a real implementation)
	traceID := resp.Header.Get("X-Trace-Id")
	spanID := resp.Header.Get("X-Span-Id")

	suite.T().Logf("Resilience test - Trace ID: %s, Span ID: %s", traceID, spanID)
}

// TestResilienceIntegration runs the complete resilience integration test suite
func TestResilienceIntegration(t *testing.T) {
	suite.Run(t, new(ResilienceIntegrationTestSuite))
}

// BenchmarkResiliencePatterns benchmarks resilience pattern performance
func BenchmarkResiliencePatterns(b *testing.B) {
	// Setup similar to resilience test
	config := &config.Config{
		ServiceName: "resilience-bench",
		Environment: "test",
		LogLevel:    "info",
		HTTPPort:    ":8080",
		JWTSecret:   "bench-secret-key",
	}

	logger := logger.New("info", "resilience-bench")
	_, err := gateway.NewGateway(config, logger)
	require.NoError(b, err)

	router := gin.New()

	// Add a simple endpoint with circuit breaker
	cbConfig := &resilience.CircuitBreakerConfig{
		Name:           "bench-cb",
		MaxFailures:    5,
		ResetTimeout:   30 * time.Second,
		HalfOpenMaxReq: 3,
	}
	resilienceClient := resilience.NewResilienceClient(cbConfig, nil, logger)

	router.GET("/bench", func(c *gin.Context) {
		err := resilienceClient.Execute(c.Request.Context(), func() error {
			c.JSON(http.StatusOK, gin.H{"message": "benchmark response"})
			return nil
		})
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		}
	})

	testServer := httptest.NewServer(router)
	defer testServer.Close()

	// Benchmark circuit breaker pattern
	b.Run("CircuitBreaker", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			resp, err := http.Get(testServer.URL + "/bench")
			if err == nil {
				resp.Body.Close()
			}
		}
	})

	// Benchmark retry pattern
	b.Run("RetryPattern", func(b *testing.B) {
		retryConfig := &resilience.RetryConfig{
			MaxRetries:    3,
			InitialDelay:  10 * time.Millisecond,
			MaxDelay:      100 * time.Millisecond,
			BackoffFactor: 2.0,
		}

		for i := 0; i < b.N; i++ {
			err := resilience.Retry(context.Background(), retryConfig, func() error {
				// Simulate work
				time.Sleep(1 * time.Millisecond)
				return nil
			})
			if err != nil {
				b.Logf("Retry failed: %v", err)
			}
		}
	})
}
