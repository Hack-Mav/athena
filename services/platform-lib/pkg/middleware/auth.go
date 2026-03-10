package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/athena/platform-lib/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

// Claims represents JWT claims
type Claims struct {
	UserID      string            `json:"user_id"`
	Username    string            `json:"username"`
	Roles       []string          `json:"roles"`
	Permissions []string          `json:"permissions"`
	SessionID   string            `json:"session_id"`
	JTI         string            `json:"jti"`        // JWT ID for token identification
	TokenType   string            `json:"token_type"` // access or refresh
	Metadata    map[string]string `json:"metadata"`
	jwt.RegisteredClaims
}

// TokenPair represents access and refresh token pair
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// JWTAuth provides enhanced JWT authentication middleware
type JWTAuth struct {
	secretKey       []byte
	issuer          string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	logger          *logger.Logger
	redisClient     *redis.Client
	blacklist       *TokenBlacklist
	sessionManager  *SessionManager
}

// TokenBlacklist manages blacklisted tokens
type TokenBlacklist struct {
	tokens map[string]time.Time
	mu     sync.RWMutex
	redis  *redis.Client
}

// SessionManager manages user sessions
type SessionManager struct {
	sessions map[string]*SessionInfo
	mu       sync.RWMutex
	redis    *redis.Client
}

// SessionInfo represents user session information
type SessionInfo struct {
	SessionID   string            `json:"session_id"`
	UserID      string            `json:"user_id"`
	Username    string            `json:"username"`
	Roles       []string          `json:"roles"`
	Permissions []string          `json:"permissions"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	LastActive  time.Time         `json:"last_active"`
	ExpiresAt   time.Time         `json:"expires_at"`
	UserAgent   string            `json:"user_agent"`
	IPAddress   string            `json:"ip_address"`
}

// NewJWTAuth creates a new enhanced JWT authentication middleware
func NewJWTAuth(secretKey, issuer string) *JWTAuth {
	return &JWTAuth{
		secretKey:       []byte(secretKey),
		issuer:          issuer,
		accessTokenTTL:  15 * time.Minute,
		refreshTokenTTL: 7 * 24 * time.Hour, // 7 days
		logger:          &logger.Logger{},   // Will be injected
		blacklist:       NewTokenBlacklist(nil),
		sessionManager:  NewSessionManager(nil),
	}
}

// NewJWTAuthWithRedis creates a JWT auth middleware with Redis support
func NewJWTAuthWithRedis(secretKey, issuer string, redisClient *redis.Client, logger *logger.Logger) *JWTAuth {
	return &JWTAuth{
		secretKey:       []byte(secretKey),
		issuer:          issuer,
		accessTokenTTL:  15 * time.Minute,
		refreshTokenTTL: 7 * 24 * time.Hour,
		logger:          logger,
		redisClient:     redisClient,
		blacklist:       NewTokenBlacklist(redisClient),
		sessionManager:  NewSessionManager(redisClient),
	}
}

// NewTokenBlacklist creates a new token blacklist
func NewTokenBlacklist(redisClient *redis.Client) *TokenBlacklist {
	return &TokenBlacklist{
		tokens: make(map[string]time.Time),
		redis:  redisClient,
	}
}

// NewSessionManager creates a new session manager
func NewSessionManager(redisClient *redis.Client) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*SessionInfo),
		redis:    redisClient,
	}
}

// GenerateSessionID generates a secure session ID
func GenerateSessionID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

// GenerateTokenPair generates access and refresh tokens
func (j *JWTAuth) GenerateTokenPair(userID, username string, roles, permissions []string, metadata map[string]string) (*TokenPair, error) {
	sessionID := GenerateSessionID()
	now := time.Now()

	// Create session info
	sessionInfo := &SessionInfo{
		SessionID:   sessionID,
		UserID:      userID,
		Username:    username,
		Roles:       roles,
		Permissions: permissions,
		CreatedAt:   now,
		LastActive:  now,
		Metadata:    metadata,
	}

	// Store session
	j.sessionManager.StoreSession(sessionID, sessionInfo)

	// Generate access token
	accessJti := sessionID
	accessClaims := Claims{
		UserID:      userID,
		Username:    username,
		Roles:       roles,
		Permissions: permissions,
		TokenType:   "access",
		SessionID:   accessJti,
		Metadata:    metadata,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.accessTokenTTL)),
			ID:        accessJti,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(j.secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshJti := fmt.Sprintf("%d-%s-refresh", now.UnixNano(), userID)
	refreshClaims := Claims{
		UserID:      userID,
		Username:    username,
		Roles:       roles,
		Permissions: permissions,
		TokenType:   "refresh",
		SessionID:   sessionID,
		Metadata:    metadata,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.refreshTokenTTL)),
			ID:        refreshJti,
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(j.secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    now.Add(j.accessTokenTTL),
		TokenType:    "Bearer",
	}, nil
}

// ValidateToken validates a JWT token and returns claims
func (j *JWTAuth) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return j.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Check if token is blacklisted
		if j.blacklist.IsBlacklisted(claims.JTI) {
			return nil, fmt.Errorf("token is blacklisted")
		}

		// Check if session is valid
		if !j.sessionManager.IsSessionValid(claims.SessionID) {
			return nil, fmt.Errorf("session is invalid or expired")
		}

		// Update session last active time
		j.sessionManager.UpdateSessionActivity(claims.SessionID)

		return claims, nil
	}

	return nil, jwt.ErrInvalidKey
}

// RefreshToken generates a new token pair from a valid refresh token
func (j *JWTAuth) RefreshToken(refreshTokenString string) (*TokenPair, error) {
	claims, err := j.ValidateToken(refreshTokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("token is not a refresh token")
	}

	// Blacklist the old refresh token
	j.blacklist.BlacklistToken(claims.JTI, time.Until(claims.ExpiresAt.Time))

	// Generate new token pair
	return j.GenerateTokenPair(claims.UserID, claims.Username, claims.Roles, claims.Permissions, claims.Metadata)
}

// RevokeToken revokes a token by blacklisting it
func (j *JWTAuth) RevokeToken(tokenString string) error {
	claims, err := j.ValidateToken(tokenString)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	// Blacklist the token
	j.blacklist.BlacklistToken(claims.JTI, time.Until(claims.ExpiresAt.Time))

	// Invalidate the session
	j.sessionManager.InvalidateSession(claims.SessionID)

	return nil
}

// RevokeAllUserTokens revokes all tokens for a user
func (j *JWTAuth) RevokeAllUserTokens(userID string) error {
	sessions := j.sessionManager.GetUserSessions(userID)
	for _, sessionID := range sessions {
		j.sessionManager.InvalidateSession(sessionID)
	}
	return nil
}

// RequireAuth creates a middleware that requires authentication
func (j *JWTAuth) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
				"code":  "AUTH_HEADER_REQUIRED",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>" format
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
				"code":  "INVALID_AUTH_FORMAT",
			})
			c.Abort()
			return
		}

		claims, err := j.ValidateToken(tokenParts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid or expired token",
				"code":    "INVALID_TOKEN",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		// Store claims in context for downstream handlers
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("roles", claims.Roles)
		c.Set("permissions", claims.Permissions)
		c.Set("session_id", claims.SessionID)
		c.Set("token_jti", claims.JTI)

		c.Next()
	}
}

// RequireRole creates a middleware that requires specific roles
func (j *JWTAuth) RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoles, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "User roles not found",
				"code":  "ROLES_NOT_FOUND",
			})
			c.Abort()
			return
		}

		userRoleList, ok := userRoles.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Invalid user roles format",
				"code":  "INVALID_ROLES_FORMAT",
			})
			c.Abort()
			return
		}

		// Check if user has any of the required roles
		hasRole := false
		for _, requiredRole := range roles {
			for _, userRole := range userRoleList {
				if userRole == requiredRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error":          "Insufficient permissions",
				"code":           "INSUFFICIENT_PERMISSIONS",
				"required_roles": roles,
				"user_roles":     userRoleList,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequirePermission creates a middleware that requires specific permissions
func (j *JWTAuth) RequirePermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userPermissions, exists := c.Get("permissions")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "User permissions not found",
				"code":  "PERMISSIONS_NOT_FOUND",
			})
			c.Abort()
			return
		}

		userPermissionList, ok := userPermissions.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Invalid user permissions format",
				"code":  "INVALID_PERMISSIONS_FORMAT",
			})
			c.Abort()
			return
		}

		// Check if user has any of the required permissions
		hasPermission := false
		for _, requiredPermission := range permissions {
			for _, userPermission := range userPermissionList {
				if userPermission == requiredPermission {
					hasPermission = true
					break
				}
			}
			if hasPermission {
				break
			}
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error":                "Insufficient permissions",
				"code":                 "INSUFFICIENT_PERMISSIONS",
				"required_permissions": permissions,
				"user_permissions":     userPermissionList,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// OptionalAuth creates a middleware that optionally extracts authentication
func (j *JWTAuth) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.Next()
			return
		}

		claims, err := j.ValidateToken(tokenParts[1])
		if err != nil {
			c.Next()
			return
		}

		// Store claims in context for downstream handlers
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("roles", claims.Roles)
		c.Set("permissions", claims.Permissions)
		c.Set("session_id", claims.SessionID)
		c.Set("token_jti", claims.JTI)

		c.Next()
	}
}

// BlacklistToken adds a token to the blacklist
func (tb *TokenBlacklist) BlacklistToken(jti string, expiration time.Duration) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	expirationTime := time.Now().Add(expiration)
	tb.tokens[jti] = expirationTime

	// Store in Redis if available
	if tb.redis != nil {
		ctx := context.Background()
		tb.redis.Set(ctx, "blacklist:"+jti, "1", expiration)
	}
}

// IsBlacklisted checks if a token is blacklisted
func (tb *TokenBlacklist) IsBlacklisted(jti string) bool {
	tb.mu.RLock()
	defer tb.mu.RUnlock()

	// Check memory cache
	if expirationTime, exists := tb.tokens[jti]; exists {
		if time.Now().Before(expirationTime) {
			return true
		}
		// Remove expired token
		delete(tb.tokens, jti)
	}

	// Check Redis if available
	if tb.redis != nil {
		ctx := context.Background()
		exists, _ := tb.redis.Exists(ctx, "blacklist:"+jti).Result()
		return exists > 0
	}

	return false
}

// StoreSession stores session information
func (sm *SessionManager) StoreSession(sessionID string, sessionInfo *SessionInfo) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.sessions[sessionID] = sessionInfo

	// Store in Redis if available
	if sm.redis != nil {
		ctx := context.Background()
		sessionData, _ := json.Marshal(sessionInfo)
		sm.redis.Set(ctx, "session:"+sessionID, sessionData, 7*24*time.Hour)
	}
}

// IsSessionValid checks if a session is valid
func (sm *SessionManager) IsSessionValid(sessionID string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessionInfo, exists := sm.sessions[sessionID]
	if !exists {
		// Check Redis if available
		if sm.redis != nil {
			ctx := context.Background()
			exists, _ := sm.redis.Exists(ctx, "session:"+sessionID).Result()
			return exists > 0
		}
		return false
	}

	// Check if session is too old (7 days)
	if time.Since(sessionInfo.CreatedAt) > 7*24*time.Hour {
		delete(sm.sessions, sessionID)
		return false
	}

	return true
}

// UpdateSessionActivity updates the last active time for a session
func (sm *SessionManager) UpdateSessionActivity(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sessionInfo, exists := sm.sessions[sessionID]; exists {
		sessionInfo.LastActive = time.Now()
	}
}

// InvalidateSession invalidates a session
func (sm *SessionManager) InvalidateSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, sessionID)

	// Remove from Redis if available
	if sm.redis != nil {
		ctx := context.Background()
		sm.redis.Del(ctx, "session:"+sessionID)
	}
}

// GetUserSessions returns all session IDs for a user
func (sm *SessionManager) GetUserSessions(userID string) []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var sessions []string
	for sessionID, sessionInfo := range sm.sessions {
		if sessionInfo.UserID == userID {
			sessions = append(sessions, sessionID)
		}
	}

	return sessions
}
