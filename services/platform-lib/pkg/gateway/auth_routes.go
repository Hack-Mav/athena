package gateway

import (
	"net/http"
	"strings"
	"time"

	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/athena/platform-lib/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	jwtAuth *middleware.JWTAuth
	logger  logger.Logger
	config  *config.Config
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(cfg *config.Config, logger logger.Logger) *AuthHandler {
	return &AuthHandler{
		jwtAuth: middleware.NewJWTAuth(cfg.JWTSecret, "athena-platform"),
		logger:  logger,
		config:  cfg,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      UserInfo  `json:"user"`
}

// UserInfo represents user information
type UserInfo struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

// RegisterAuthRoutes registers authentication routes
func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.RefreshToken)
		auth.POST("/logout", h.jwtAuth.RequireAuth(), h.Logout)

		// Protected routes for testing
		protected := auth.Group("/protected")
		protected.Use(h.jwtAuth.RequireAuth())
		{
			protected.GET("/profile", h.GetProfile)
			protected.GET("/admin", h.jwtAuth.RequireRole("admin"), h.AdminOnly)
		}
	}
}

// Login handles user authentication
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// TODO: Implement actual user authentication against database
	// For now, using a simple mock authentication
	if !h.authenticateUser(req.Username, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid credentials",
		})
		return
	}

	// Generate JWT token
	token, err := h.jwtAuth.GenerateToken(
		"user-123", // Mock user ID
		req.Username,
		[]string{"user"}, // Mock roles
		24*time.Hour,     // 24 hour expiry
	)
	if err != nil {
		h.logger.Error("Failed to generate token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate authentication token",
		})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		User: UserInfo{
			ID:       "user-123",
			Username: req.Username,
			Roles:    []string{"user"},
		},
	})
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authorization header required",
		})
		return
	}

	// Extract and validate current token
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid authorization header format",
		})
		return
	}

	claims, err := h.jwtAuth.ValidateToken(tokenParts[1])
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid or expired token",
		})
		return
	}

	// Generate new token with same claims
	newToken, err := h.jwtAuth.GenerateToken(
		claims.UserID,
		claims.Username,
		claims.Roles,
		24*time.Hour,
	)
	if err != nil {
		h.logger.Error("Failed to refresh token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      newToken,
		"expires_at": time.Now().Add(24 * time.Hour),
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// TODO: Implement token blacklisting if needed
	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully logged out",
	})
}

// GetProfile returns user profile information
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	roles, _ := c.Get("roles")

	c.JSON(http.StatusOK, gin.H{
		"user": UserInfo{
			ID:       userID.(string),
			Username: username.(string),
			Roles:    roles.([]string),
		},
	})
}

// AdminOnly is an example of a role-protected endpoint
func (h *AuthHandler) AdminOnly(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Welcome admin!",
		"data": map[string]interface{}{
			"system_info": "This is admin-only data",
			"timestamp":   time.Now(),
		},
	})
}

// authenticateUser is a mock authentication function
// TODO: Replace with actual database authentication
func (h *AuthHandler) authenticateUser(username, password string) bool {
	// Mock authentication for demo purposes
	// In production, this should verify against a secure password hash
	if username == "admin" && password == "admin123" {
		return true
	}
	if username == "user" && password == "user123" {
		return true
	}
	return false
}
