package middleware

import (
	"net/http"

	"github.com/athena/platform-lib/pkg/validation"
	"github.com/gin-gonic/gin"
)

// ValidationMiddleware provides input validation middleware
type ValidationMiddleware struct {
	validator *validation.Validator
}

// NewValidationMiddleware creates a new validation middleware
func NewValidationMiddleware() *ValidationMiddleware {
	return &ValidationMiddleware{
		validator: validation.NewValidator(),
	}
}

// ValidateBody validates request body against a struct
func (vm *ValidationMiddleware) ValidateBody(obj interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := c.ShouldBindJSON(obj); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request format",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		if err := vm.validator.Validate(obj); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Validation failed",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		c.Set("validated_body", obj)
		c.Next()
	}
}

// ValidateQuery validates query parameters
func (vm *ValidationMiddleware) ValidateQuery(rules map[string]string) gin.HandlerFunc {
	return func(c *gin.Context) {
		errors := make(map[string]string)

		for field, rule := range rules {
			value := c.Query(field)
			if err := vm.validator.ValidateVar(value, rule); err != nil {
				errors[field] = err.Error()
			}
		}

		if len(errors) > 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":  "Query parameter validation failed",
				"fields": errors,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SanitizeInput sanitizes request inputs
func (vm *ValidationMiddleware) SanitizeInput() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Sanitize query parameters
		for key, values := range c.Request.URL.Query() {
			for i, value := range values {
				values[i] = vm.validator.SanitizeString(value)
			}
			c.Request.URL.Query()[key] = values
		}

		// For POST/PUT requests, the body sanitization should be done
		// in the validation middleware after binding

		c.Next()
	}
}

// RateLimitMiddleware provides basic rate limiting
type RateLimitMiddleware struct {
	// In production, this should use a proper rate limiting library
	// like go-redis rate limiter or token bucket algorithm
	requests    map[string]int
	maxRequests int
}

// NewRateLimitMiddleware creates a new rate limiting middleware
func NewRateLimitMiddleware(maxRequests int) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		requests:    make(map[string]int),
		maxRequests: maxRequests,
	}
}

// RateLimit applies rate limiting based on client IP
func (rlm *RateLimitMiddleware) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		// Simple in-memory rate limiting (for demo purposes)
		// In production, use Redis or similar for distributed rate limiting
		if rlm.requests[clientIP] >= rlm.maxRequests {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests, please try again later",
			})
			c.Abort()
			return
		}

		rlm.requests[clientIP]++
		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Content Security Policy
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")

		// HSTS (only in production with HTTPS)
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Referrer Policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}

// CORSMiddleware provides CORS handling
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
