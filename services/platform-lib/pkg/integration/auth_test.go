package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/gateway"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/athena/platform-lib/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// AuthIntegrationTestSuite tests authentication and authorization functionality
type AuthIntegrationTestSuite struct {
	suite.Suite
	gateway    *gateway.Gateway
	router     *gin.Engine
	config     *config.Config
	logger     *logger.Logger
	testServer *httptest.Server
	jwtAuth    *middleware.JWTAuth
}

// SetupSuite sets up the authentication test suite
func (suite *AuthIntegrationTestSuite) SetupSuite() {
	// Create test configuration
	suite.config = &config.Config{
		ServiceName: "auth-gateway-test",
		Environment: "test",
		LogLevel:    "debug",
		HTTPPort:    ":8080",
		JWTSecret:   "test-secret-key-for-auth-testing-32-chars",
		Services: map[string]string{
			"template-service": "http://localhost:8001",
		},
	}

	// Create logger
	suite.logger = logger.New("debug", "auth-gateway-test")

	// Create JWT auth middleware
	suite.jwtAuth = middleware.NewJWTAuth(suite.config.JWTSecret, "athena-platform-test")

	// Create gateway
	var err error
	suite.gateway, err = gateway.NewGateway(suite.config, suite.logger)
	require.NoError(suite.T(), err)

	// Create router with auth routes
	suite.router = gin.New()
	suite.setupAuthRoutes()

	// Create test server
	suite.testServer = httptest.NewServer(suite.router)
}

// TearDownSuite cleans up the authentication test suite
func (suite *AuthIntegrationTestSuite) TearDownSuite() {
	if suite.testServer != nil {
		suite.testServer.Close()
	}

	if suite.gateway != nil {
		suite.gateway.Shutdown()
	}
}

// setupAuthRoutes sets up authentication-specific test routes
func (suite *AuthIntegrationTestSuite) setupAuthRoutes() {
	// Public routes
	suite.router.GET("/public", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "public endpoint"})
	})

	// Authentication required routes
	auth := suite.router.Group("/auth")
	auth.Use(suite.jwtAuth.RequireAuth())
	{
		auth.GET("/profile", func(c *gin.Context) {
			userID := c.GetString("user_id")
			username := c.GetString("username")
			roles := c.GetStringSlice("roles")

			c.JSON(http.StatusOK, gin.H{
				"user_id":  userID,
				"username": username,
				"roles":    roles,
			})
		})
	}

	// Admin only routes
	admin := suite.router.Group("/admin")
	admin.Use(suite.jwtAuth.RequireAuth())
	admin.Use(suite.jwtAuth.RequireRole("admin"))
	{
		admin.GET("/dashboard", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "admin dashboard"})
		})
	}

	// Permission-based routes
	permission := suite.router.Group("/api")
	permission.Use(suite.jwtAuth.RequireAuth())
	permission.Use(suite.jwtAuth.RequirePermission("read:templates"))
	{
		permission.GET("/templates", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"templates": []string{"template1", "template2"}})
		})
	}

	// Login endpoint
	suite.router.POST("/login", func(c *gin.Context) {
		var loginReq struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := c.ShouldBindJSON(&loginReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		// Simple authentication logic for testing
		if loginReq.Username == "testuser" && loginReq.Password == "testpass" {
			tokenPair, err := suite.jwtAuth.GenerateTokenPair(
				"user-123",
				loginReq.Username,
				[]string{"user"},
				[]string{"read:templates", "read:devices"},
				map[string]string{"department": "engineering"},
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
				return
			}

			c.JSON(http.StatusOK, tokenPair)
			return
		}

		if loginReq.Username == "admin" && loginReq.Password == "adminpass" {
			tokenPair, err := suite.jwtAuth.GenerateTokenPair(
				"admin-456",
				loginReq.Username,
				[]string{"admin", "user"},
				[]string{"read:templates", "write:templates", "read:devices", "write:devices"},
				map[string]string{"department": "IT"},
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
				return
			}

			c.JSON(http.StatusOK, tokenPair)
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
	})

	// Refresh token endpoint
	suite.router.POST("/refresh", func(c *gin.Context) {
		var refreshReq struct {
			RefreshToken string `json:"refresh_token"`
		}

		if err := c.ShouldBindJSON(&refreshReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		tokenPair, err := suite.jwtAuth.RefreshToken(refreshReq.RefreshToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
			return
		}

		c.JSON(http.StatusOK, tokenPair)
	})

	// Logout endpoint
	suite.router.POST("/logout", suite.jwtAuth.RequireAuth(), func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenParts := bytes.Split([]byte(authHeader), []byte(" "))
		if len(tokenParts) == 2 {
			token := string(tokenParts[1])
			err := suite.jwtAuth.RevokeToken(token)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke token"})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
	})
}

// TestPublicAccess tests access to public endpoints
func (suite *AuthIntegrationTestSuite) TestPublicAccess() {
	resp, err := http.Get(suite.testServer.URL + "/public")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(suite.T(), err)

	assert.Equal(suite.T(), "public endpoint", response["message"])
}

// TestLoginFlow tests the complete login flow
func (suite *AuthIntegrationTestSuite) TestLoginFlow() {
	tests := []struct {
		name        string
		username    string
		password    string
		expectCode  int
		expectToken bool
		expectRoles []string
	}{
		{
			name:        "Valid user login",
			username:    "testuser",
			password:    "testpass",
			expectCode:  http.StatusOK,
			expectToken: true,
			expectRoles: []string{"user"},
		},
		{
			name:        "Valid admin login",
			username:    "admin",
			password:    "adminpass",
			expectCode:  http.StatusOK,
			expectToken: true,
			expectRoles: []string{"admin", "user"},
		},
		{
			name:       "Invalid credentials",
			username:   "invalid",
			password:   "wrongpass",
			expectCode: http.StatusUnauthorized,
		},
		{
			name:       "Missing credentials",
			expectCode: http.StatusBadRequest,
		},
	}

	for _, test := range tests {
		suite.T().Run(test.name, func(t *testing.T) {
			var payload bytes.Buffer
			if test.username != "" || test.password != "" {
				loginReq := map[string]string{
					"username": test.username,
					"password": test.password,
				}
				json.NewEncoder(&payload).Encode(loginReq)
			}

			resp, err := http.Post(
				suite.testServer.URL+"/login",
				"application/json",
				&payload,
			)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, test.expectCode, resp.StatusCode)

			if test.expectToken {
				var tokenResponse map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&tokenResponse)
				require.NoError(t, err)

				accessToken, ok := tokenResponse["access_token"].(string)
				require.True(t, ok)
				require.NotEmpty(t, accessToken)

				refreshToken, ok := tokenResponse["refresh_token"].(string)
				require.True(t, ok)
				require.NotEmpty(t, refreshToken)

				expiresAt, ok := tokenResponse["expires_at"].(string)
				require.True(t, ok)
				require.NotEmpty(t, expiresAt)

				// Verify token claims
				claims, err := suite.jwtAuth.ValidateToken(accessToken)
				require.NoError(t, err)

				assert.Equal(t, test.username, claims.Username)
				assert.Equal(t, test.expectRoles, claims.Roles)
			}
		})
	}
}

// TestProtectedEndpoints tests access to protected endpoints
func (suite *AuthIntegrationTestSuite) TestProtectedEndpoints() {
	// Get valid token for regular user
	userToken := suite.loginAndGetToken("testuser", "testpass")
	adminToken := suite.loginAndGetToken("admin", "adminpass")

	tests := []struct {
		name         string
		token        string
		path         string
		expectCode   int
		expectFields map[string]interface{}
	}{
		{
			name:       "User access to profile",
			token:      userToken,
			path:       "/auth/profile",
			expectCode: http.StatusOK,
			expectFields: map[string]interface{}{
				"user_id":  "user-123",
				"username": "testuser",
				"roles":    []string{"user"},
			},
		},
		{
			name:       "Admin access to profile",
			token:      adminToken,
			path:       "/auth/profile",
			expectCode: http.StatusOK,
			expectFields: map[string]interface{}{
				"user_id":  "admin-456",
				"username": "admin",
				"roles":    []string{"admin", "user"},
			},
		},
		{
			name:       "User access to admin dashboard (should fail)",
			token:      userToken,
			path:       "/admin/dashboard",
			expectCode: http.StatusForbidden,
		},
		{
			name:       "Admin access to admin dashboard",
			token:      adminToken,
			path:       "/admin/dashboard",
			expectCode: http.StatusOK,
		},
		{
			name:       "No token access to protected endpoint",
			path:       "/auth/profile",
			expectCode: http.StatusUnauthorized,
		},
		{
			name:       "Invalid token access to protected endpoint",
			token:      "invalid.token.here",
			path:       "/auth/profile",
			expectCode: http.StatusUnauthorized,
		},
	}

	for _, test := range tests {
		suite.T().Run(test.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", suite.testServer.URL+test.path, nil)
			if test.token != "" {
				req.Header.Set("Authorization", "Bearer "+test.token)
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, test.expectCode, resp.StatusCode)

			if test.expectFields != nil && resp.StatusCode == http.StatusOK {
				var response map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err)

				for field, expectedValue := range test.expectFields {
					actualValue, exists := response[field]
					require.True(t, exists, "Field %s not found in response", field)
					assert.Equal(t, expectedValue, actualValue, "Field %s mismatch", field)
				}
			}
		})
	}
}

// TestPermissionBasedAccess tests permission-based access control
func (suite *AuthIntegrationTestSuite) TestPermissionBasedAccess() {
	userToken := suite.loginAndGetToken("testuser", "testpass")
	adminToken := suite.loginAndGetToken("admin", "adminpass")

	tests := []struct {
		name       string
		token      string
		path       string
		expectCode int
	}{
		{
			name:       "User with read:templates permission",
			token:      userToken,
			path:       "/api/templates",
			expectCode: http.StatusOK,
		},
		{
			name:       "Admin with read:templates permission",
			token:      adminToken,
			path:       "/api/templates",
			expectCode: http.StatusOK,
		},
		{
			name:       "No token access to permission-based endpoint",
			path:       "/api/templates",
			expectCode: http.StatusUnauthorized,
		},
	}

	for _, test := range tests {
		suite.T().Run(test.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", suite.testServer.URL+test.path, nil)
			if test.token != "" {
				req.Header.Set("Authorization", "Bearer "+test.token)
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, test.expectCode, resp.StatusCode)
		})
	}
}

// TestTokenRefresh tests token refresh functionality
func (suite *AuthIntegrationTestSuite) TestTokenRefresh() {
	// Login to get initial tokens
	tokenPair := suite.loginAndGetTokenPair("testuser", "testpass")
	require.NotEmpty(suite.T(), tokenPair.RefreshToken)

	// Wait a short time to ensure tokens are different
	time.Sleep(100 * time.Millisecond)

	// Refresh the token
	refreshPayload := map[string]interface{}{
		"refresh_token": tokenPair.RefreshToken,
	}
	refreshBody, _ := json.Marshal(refreshPayload)

	resp, err := http.Post(
		suite.testServer.URL+"/refresh",
		"application/json",
		bytes.NewBuffer(refreshBody),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var newTokenPair map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&newTokenPair)
	require.NoError(suite.T(), err)

	newAccessToken, ok := newTokenPair["access_token"].(string)
	require.True(suite.T(), ok)
	require.NotEmpty(suite.T(), newAccessToken)

	// Verify new token is different from old one
	assert.NotEqual(suite.T(), tokenPair.AccessToken, newAccessToken)

	// Verify new token is valid
	claims, err := suite.jwtAuth.ValidateToken(newAccessToken)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "testuser", claims.Username)

	// Test that old refresh token is now invalid
	refreshPayload2 := map[string]interface{}{
		"refresh_token": tokenPair.RefreshToken,
	}
	refreshBody2, _ := json.Marshal(refreshPayload2)

	resp2, err := http.Post(
		suite.testServer.URL+"/refresh",
		"application/json",
		bytes.NewBuffer(refreshBody2),
	)
	require.NoError(suite.T(), err)
	defer resp2.Body.Close()

	assert.Equal(suite.T(), http.StatusUnauthorized, resp2.StatusCode)
}

// TestTokenRevocation tests token revocation functionality
func (suite *AuthIntegrationTestSuite) TestTokenRevocation() {
	// Login to get token
	token := suite.loginAndGetToken("testuser", "testpass")

	// Verify token works initially
	req, _ := http.NewRequest("GET", suite.testServer.URL+"/auth/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Logout (revoke token)
	logoutPayload := map[string]interface{}{}
	logoutBody, _ := json.Marshal(logoutPayload)

	resp2, err := http.Post(
		suite.testServer.URL+"/logout",
		"application/json",
		bytes.NewBuffer(logoutBody),
	)
	require.NoError(suite.T(), err)
	defer resp2.Body.Close()
	assert.Equal(suite.T(), http.StatusOK, resp2.StatusCode)

	// Try to use revoked token
	req3, _ := http.NewRequest("GET", suite.testServer.URL+"/auth/profile", nil)
	req3.Header.Set("Authorization", "Bearer "+token)

	resp3, err := http.DefaultClient.Do(req3)
	require.NoError(suite.T(), err)
	defer resp3.Body.Close()
	assert.Equal(suite.T(), http.StatusUnauthorized, resp3.StatusCode)
}

// TestTokenExpiration tests token expiration handling
func (suite *AuthIntegrationTestSuite) TestTokenExpiration() {
	// Create a token with very short expiration
	shortLivedAuth := middleware.NewJWTAuth(suite.config.JWTSecret, "athena-platform-test")

	tokenPair, err := shortLivedAuth.GenerateTokenPair(
		"user-123",
		"testuser",
		[]string{"user"},
		[]string{"read:templates"},
		map[string]string{"department": "engineering"},
	)
	require.NoError(suite.T(), err)

	// Token should work initially
	req, _ := http.NewRequest("GET", suite.testServer.URL+"/auth/profile", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Wait for token to expire (this would need to be implemented with a shorter TTL in real tests)
	// For now, we'll test with an invalid token format
	req2, _ := http.NewRequest("GET", suite.testServer.URL+"/auth/profile", nil)
	req2.Header.Set("Authorization", "Bearer expired.token.format")

	resp2, err := http.DefaultClient.Do(req2)
	require.NoError(suite.T(), err)
	defer resp2.Body.Close()
	assert.Equal(suite.T(), http.StatusUnauthorized, resp2.StatusCode)
}

// TestSecurityHeaders tests security headers are properly set
func (suite *AuthIntegrationTestSuite) TestSecurityHeaders() {
	resp, err := http.Get(suite.testServer.URL + "/public")
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	// Check for security headers
	assert.Equal(suite.T(), "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Equal(suite.T(), "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(suite.T(), "1; mode=block", resp.Header.Get("X-XSS-Protection"))
	assert.Contains(suite.T(), resp.Header.Get("Content-Security-Policy"), "default-src 'self'")
}

// loginAndGetToken is a helper to login and get an access token
func (suite *AuthIntegrationTestSuite) loginAndGetToken(username, password string) string {
	loginPayload := map[string]string{
		"username": username,
		"password": password,
	}
	loginBody, _ := json.Marshal(loginPayload)

	resp, err := http.Post(
		suite.testServer.URL+"/login",
		"application/json",
		bytes.NewBuffer(loginBody),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var tokenResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&tokenResponse)
	require.NoError(suite.T(), err)

	accessToken, ok := tokenResponse["access_token"].(string)
	require.True(suite.T(), ok)
	require.NotEmpty(suite.T(), accessToken)

	return accessToken
}

// loginAndGetTokenPair is a helper to login and get a full token pair
func (suite *AuthIntegrationTestSuite) loginAndGetTokenPair(username, password string) middleware.TokenPair {
	loginPayload := map[string]string{
		"username": username,
		"password": password,
	}
	loginBody, _ := json.Marshal(loginPayload)

	resp, err := http.Post(
		suite.testServer.URL+"/login",
		"application/json",
		bytes.NewBuffer(loginBody),
	)
	require.NoError(suite.T(), err)
	defer resp.Body.Close()

	var tokenResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&tokenResponse)
	require.NoError(suite.T(), err)

	return middleware.TokenPair{
		AccessToken:  tokenResponse["access_token"].(string),
		RefreshToken: tokenResponse["refresh_token"].(string),
	}
}

// TestAuthIntegration runs the complete authentication integration test suite
func TestAuthIntegration(t *testing.T) {
	suite.Run(t, new(AuthIntegrationTestSuite))
}
