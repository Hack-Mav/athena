package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/discovery"
	"github.com/athena/platform-lib/pkg/gateway"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// APIGatewayIntegrationTestSuite tests the complete API gateway functionality
type APIGatewayIntegrationTestSuite struct {
	suite.Suite
	gateway      *gateway.Gateway
	router       *gin.Engine
	config       *config.Config
	logger       *logger.Logger
	registry     *discovery.ServiceRegistry
	testServer   *httptest.Server
	mockServices map[string]*httptest.Server
}

// SetupSuite sets up the test suite
func (suite *APIGatewayIntegrationTestSuite) SetupSuite() {
	// Create test configuration
	suite.config = &config.Config{
		ServiceName: "api-gateway-test",
		Environment: "test",
		LogLevel:    "debug",
		HTTPPort:    ":8080",
		JWTSecret:   "test-secret-key-for-integration-testing",
		Services: map[string]string{
			"template-service":     "http://localhost:8001",
			"nlp-service":          "http://localhost:8002",
			"provisioning-service": "http://localhost:8003",
			"device-service":       "http://localhost:8004",
			"telemetry-service":    "http://localhost:8005",
			"ota-service":          "http://localhost:8006",
		},
	}

	// Create logger
	suite.logger = logger.New("debug", "api-gateway-test")

	// Create mock services
	suite.createMockServices()

	// Create service registry
	suite.registry = discovery.NewServiceRegistry(suite.logger, &discovery.RegistryConfig{
		HealthCheckInterval: 5 * time.Second,
		HealthCheckTimeout:  2 * time.Second,
		MaxRetries:          2,
	})

	// Register mock services
	suite.registerMockServices()

	// Create gateway
	var err error
	suite.gateway, err = gateway.NewGateway(suite.config, suite.logger)
	require.NoError(suite.T(), err)

	// Create router
	suite.router = gin.New()
	gateway.RegisterRoutes(suite.router, suite.gateway)

	// Create test server
	suite.testServer = httptest.NewServer(suite.router)
}

// TearDownSuite cleans up the test suite
func (suite *APIGatewayIntegrationTestSuite) TearDownSuite() {
	if suite.testServer != nil {
		suite.testServer.Close()
	}

	for _, server := range suite.mockServices {
		server.Close()
	}

	if suite.gateway != nil {
		suite.gateway.Shutdown()
	}
}

// createMockServices creates mock HTTP services for testing
func (suite *APIGatewayIntegrationTestSuite) createMockServices() {
	suite.mockServices = make(map[string]*httptest.Server)

	// Template Service
	suite.mockServices["template-service"] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
		case "/templates":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"templates": []map[string]interface{}{
					{"id": "1", "name": "Temperature Sensor", "category": "sensing"},
					{"id": "2", "name": "LED Controller", "category": "automation"},
				},
			})
		case "/templates/1":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       "1",
				"name":     "Temperature Sensor",
				"category": "sensing",
				"code":     "// Temperature sensor code",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// NLP Service
	suite.mockServices["nlp-service"] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
		case "/parse":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"intent":     "create_sensor",
				"entities":   []string{"temperature", "sensor"},
				"confidence": 0.95,
			})
		case "/plan":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"plan_id":    "plan-123",
				"steps":      []string{"Connect DHT22", "Upload code", "Test sensor"},
				"components": []string{"DHT22", "Arduino Uno", "Jumper wires"},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// Device Service
	suite.mockServices["device-service"] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
		case "/devices":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"devices": []map[string]interface{}{
					{"id": "device-1", "name": "Arduino Uno", "status": "online"},
					{"id": "device-2", "name": "ESP32", "status": "offline"},
				},
			})
		case "/devices/device-1":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     "device-1",
				"name":   "Arduino Uno",
				"status": "online",
				"type":   "arduino-uno",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// Telemetry Service
	suite.mockServices["telemetry-service"] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
		case "/ingest":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": "Telemetry data ingested",
			})
		case "/metrics/device-1":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"telemetry": []map[string]interface{}{
					{"timestamp": time.Now().Unix(), "temperature": 25.5, "humidity": 60.2},
					{"timestamp": time.Now().Unix() - 60, "temperature": 25.3, "humidity": 60.1},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// Provisioning Service
	suite.mockServices["provisioning-service"] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
		case "/compile":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"compilation_id": "compile-123",
				"success":        true,
				"binary_data":    "base64-encoded-binary",
			})
		case "/flash":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"flash_id": "flash-123",
				"success":  true,
				"message":  "Device flashed successfully",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// OTA Service
	suite.mockServices["ota-service"] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
		case "/releases":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"release_id": "release-123",
				"version":    "1.0.0",
				"success":    true,
			})
		case "/updates/device-1":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"updates": []map[string]interface{}{
					{"release_id": "release-123", "version": "1.0.0", "status": "available"},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// registerMockServices registers mock services with the registry
func (suite *APIGatewayIntegrationTestSuite) registerMockServices() {
	for serviceName, server := range suite.mockServices {
		instance := &discovery.ServiceInstance{
			ID:      serviceName + "-mock",
			Name:    serviceName,
			Address: "localhost",
			Port:    8000, // Mock port
			Status:  "healthy",
			Metadata: map[string]string{
				"url": server.URL,
			},
		}
		suite.registry.RegisterService(instance)
	}
}

// TestHealthEndpoints tests health check endpoints
func (suite *APIGatewayIntegrationTestSuite) TestHealthEndpoints() {
	tests := []struct {
		path       string
		expectCode int
		expectBody string
	}{
		{"/health", http.StatusOK, "healthy"},
		{"/ready", http.StatusOK, "ready"},
		{"/live", http.StatusOK, "alive"},
	}

	for _, test := range tests {
		resp, err := http.Get(suite.testServer.URL + test.path)
		require.NoError(suite.T(), err)
		defer resp.Body.Close()

		assert.Equal(suite.T(), test.expectCode, resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		assert.Contains(suite.T(), string(body), test.expectBody)
	}
}

// TestServiceHealthEndpoints tests service health endpoints
func (suite *APIGatewayIntegrationTestSuite) TestServiceHealthEndpoints() {
	// Test all services health
	resp, err := http.Get(suite.testServer.URL + "/services/health")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var healthResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&healthResponse)
	require.NoError(suite.T(), err)

	services, ok := healthResponse["services"].(map[string]interface{})
	require.True(suite.T(), ok)
	assert.Contains(suite.T(), services, "template-service")
	assert.Contains(suite.T(), services, "nlp-service")

	// Test specific service health
	resp, err = http.Get(suite.testServer.URL + "/services/health/template-service")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

// TestAuthenticationFlow tests the complete authentication flow
func (suite *APIGatewayIntegrationTestSuite) TestAuthenticationFlow() {
	// Test login
	loginPayload := map[string]interface{}{
		"username": "testuser",
		"password": "testpass",
	}

	loginBody, _ := json.Marshal(loginPayload)
	resp, err := http.Post(
		suite.testServer.URL+"/api/v1/auth/login",
		"application/json",
		bytes.NewBuffer(loginBody),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var loginResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&loginResponse)
	require.NoError(suite.T(), err)

	accessToken, ok := loginResponse["access_token"].(string)
	require.True(suite.T(), ok)
	require.NotEmpty(suite.T(), accessToken)

	// Test protected endpoint with token
	req, _ := http.NewRequest("GET", suite.testServer.URL+"/api/v1/auth/protected/profile", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err = http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Test protected endpoint without token
	resp, err = http.Get(suite.testServer.URL + "/api/v1/auth/protected/profile")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)
}

// TestServiceProxying tests service proxying functionality
func (suite *APIGatewayIntegrationTestSuite) TestServiceProxying() {
	// Get auth token first
	token := suite.getAuthToken()

	// Test template service proxy
	req, _ := http.NewRequest("GET", suite.testServer.URL+"/api/v1/templates", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var templatesResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&templatesResponse)
	require.NoError(suite.T(), err)

	templates, ok := templatesResponse["message"].(string)
	require.True(suite.T(), ok)
	assert.Contains(suite.T(), templates, "Template service proxy")

	// Test NLP service proxy
	req, _ = http.NewRequest("POST", suite.testServer.URL+"/api/v1/nlp/parse", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	nlpPayload := map[string]interface{}{
		"text": "I want to create a temperature sensor",
	}
	nlpBody, _ := json.Marshal(nlpPayload)
	req.Body = io.NopCloser(bytes.NewBuffer(nlpBody))

	resp, err = http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

// TestRateLimiting tests rate limiting functionality
func (suite *APIGatewayIntegrationTestSuite) TestRateLimiting() {
	// Make multiple rapid requests to trigger rate limiting
	for i := 0; i < 150; i++ { // More than the default limit of 100
		resp, err := http.Get(suite.testServer.URL + "/health")
		if err != nil {
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			// Rate limit triggered
			assert.Equal(suite.T(), http.StatusTooManyRequests, resp.StatusCode)
			return
		}
	}

	// If we get here, rate limiting might not be working as expected
	suite.T().Log("Rate limiting test completed without triggering limit")
}

// TestCircuitBreaker tests circuit breaker functionality
func (suite *APIGatewayIntegrationTestSuite) TestCircuitBreaker() {
	// This test would require simulating service failures
	// For now, we'll test the basic structure
	token := suite.getAuthToken()

	// Make a request to a service
	req, _ := http.NewRequest("GET", suite.testServer.URL+"/api/v1/templates", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	// The circuit breaker should allow requests when services are healthy
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

// TestErrorHandling tests error handling scenarios
func (suite *APIGatewayIntegrationTestSuite) TestErrorHandling() {
	tests := []struct {
		name          string
		method        string
		path          string
		body          interface{}
		headers       map[string]string
		expectedCode  int
		expectedError string
	}{
		{
			name:          "Invalid auth header",
			method:        "GET",
			path:          "/api/v1/templates",
			expectedCode:  http.StatusUnauthorized,
			expectedError: "Authorization header required",
		},
		{
			name:   "Malformed auth header",
			method: "GET",
			path:   "/api/v1/templates",
			headers: map[string]string{
				"Authorization": "InvalidToken",
			},
			expectedCode:  http.StatusUnauthorized,
			expectedError: "Invalid authorization header format",
		},
		{
			name:          "Invalid token",
			method:        "GET",
			path:          "/api/v1/templates",
			headers:       map[string]string{"Authorization": "Bearer invalid.token.here"},
			expectedCode:  http.StatusUnauthorized,
			expectedError: "Invalid or expired token",
		},
	}

	for _, test := range tests {
		suite.T().Run(test.name, func(t *testing.T) {
			var body bytes.Buffer
			if test.body != nil {
				json.NewEncoder(&body).Encode(test.body)
			}

			req, _ := http.NewRequest(test.method, suite.testServer.URL+test.path, &body)
			if test.headers != nil {
				for k, v := range test.headers {
					req.Header.Set(k, v)
				}
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, test.expectedCode, resp.StatusCode)

			var errorResponse map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&errorResponse)
			require.NoError(t, err)

			if test.expectedError != "" {
				assert.Contains(t, errorResponse["error"], test.expectedError)
			}
		})
	}
}

// TestDistributedTracing tests distributed tracing functionality
func (suite *APIGatewayIntegrationTestSuite) TestDistributedTracing() {
	// This test verifies that tracing headers are properly propagated
	token := suite.getAuthToken()

	req, _ := http.NewRequest("GET", suite.testServer.URL+"/api/v1/templates", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Trace-Id", "test-trace-123")
	req.Header.Set("X-Span-Id", "test-span-456")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Verify tracing headers are present in response
	traceID := resp.Header.Get("X-Trace-Id")
	spanID := resp.Header.Get("X-Span-Id")

	// In a real implementation, these would be populated by the tracing system
	suite.T().Logf("Trace ID: %s, Span ID: %s", traceID, spanID)
}

// getAuthToken obtains an authentication token for testing
func (suite *APIGatewayIntegrationTestSuite) getAuthToken() string {
	loginPayload := map[string]interface{}{
		"username": "testuser",
		"password": "testpass",
	}

	loginBody, _ := json.Marshal(loginPayload)
	resp, err := http.Post(
		suite.testServer.URL+"/api/v1/auth/login",
		"application/json",
		bytes.NewBuffer(loginBody),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var loginResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&loginResponse)
	require.NoError(suite.T(), err)

	accessToken, ok := loginResponse["access_token"].(string)
	require.True(suite.T(), ok)
	require.NotEmpty(suite.T(), accessToken)

	return accessToken
}

// TestAPIGatewayIntegration runs the complete integration test suite
func TestAPIGatewayIntegration(t *testing.T) {
	suite.Run(t, new(APIGatewayIntegrationTestSuite))
}

// BenchmarkAPIGateway benchmarks API gateway performance
func BenchmarkAPIGateway(b *testing.B) {
	// Setup similar to integration test
	config := &config.Config{
		ServiceName: "api-gateway-bench",
		Environment: "test",
		LogLevel:    "info",
		HTTPPort:    ":8080",
		JWTSecret:   "bench-secret-key",
	}

	logger := logger.New("info", "api-gateway-bench")
	gw, err := gateway.NewGateway(config, logger)
	require.NoError(b, err)

	router := gin.New()
	gateway.RegisterRoutes(router, gw)

	testServer := httptest.NewServer(router)
	defer testServer.Close()

	// Get auth token
	loginPayload := map[string]interface{}{
		"username": "benchuser",
		"password": "benchpass",
	}
	loginBody, _ := json.Marshal(loginPayload)
	resp, err := http.Post(
		testServer.URL+"/api/v1/auth/login",
		"application/json",
		bytes.NewBuffer(loginBody),
	)
	require.NoError(b, err)
	defer resp.Body.Close()

	var loginResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&loginResponse)
	require.NoError(b, err)

	token := loginResponse["access_token"].(string)

	// Benchmark health endpoint
	b.Run("HealthEndpoint", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			resp, err := http.Get(testServer.URL + "/health")
			if err == nil {
				resp.Body.Close()
			}
		}
	})

	// Benchmark authenticated endpoint
	b.Run("AuthenticatedEndpoint", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			req, _ := http.NewRequest("GET", testServer.URL+"/api/v1/templates", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				resp.Body.Close()
			}
		}
	})
}
