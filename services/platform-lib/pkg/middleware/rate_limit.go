package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/athena/platform-lib/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiterConfig holds rate limiting configuration
type RateLimiterConfig struct {
	RequestsPerSecond int           `json:"requests_per_second"`
	BurstSize         int           `json:"burst_size"`
	WindowDuration    time.Duration `json:"window_duration"`
	KeyGenerator      func(*gin.Context) string
	RedisClient       *redis.Client
	Logger            *logger.Logger
}

// RateLimiter implements rate limiting using token bucket algorithm
type RateLimiter struct {
	config      *RateLimiterConfig
	buckets     map[string]*TokenBucket
	mu          sync.RWMutex
	redisClient *redis.Client
	logger      *logger.Logger
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	capacity       int
	tokens         int
	refillRate     int
	lastRefillTime time.Time
	mu             sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(capacity, refillRate int) *TokenBucket {
	return &TokenBucket{
		capacity:       capacity,
		tokens:         capacity,
		refillRate:     refillRate,
		lastRefillTime: time.Now(),
	}
}

// TakeToken attempts to take a token from the bucket
func (tb *TokenBucket) TakeToken() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(tb.lastRefillTime)
	tokensToAdd := int(elapsed.Seconds()) * tb.refillRate

	if tokensToAdd > 0 {
		tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
		tb.lastRefillTime = now
	}

	// Check if token is available
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// GetAvailableTokens returns the number of available tokens
func (tb *TokenBucket) GetAvailableTokens() int {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(tb.lastRefillTime)
	tokensToAdd := int(elapsed.Seconds()) * tb.refillRate

	if tokensToAdd > 0 {
		tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
		tb.lastRefillTime = now
	}

	return tb.tokens
}

// NewRateLimitMiddleware creates a new rate limiting middleware
func NewRateLimitMiddleware(requestsPerSecond int) *RateLimiter {
	config := &RateLimiterConfig{
		RequestsPerSecond: requestsPerSecond,
		BurstSize:         requestsPerSecond * 2, // Allow bursts
		WindowDuration:    time.Second,
		KeyGenerator:      defaultKeyGenerator,
	}

	return NewRateLimiterWithConfig(config)
}

// NewRateLimiterWithConfig creates a rate limiter with custom configuration
func NewRateLimiterWithConfig(config *RateLimiterConfig) *RateLimiter {
	return &RateLimiter{
		config:      config,
		buckets:     make(map[string]*TokenBucket),
		redisClient: config.RedisClient,
		logger:      config.Logger,
	}
}

// RateLimit returns the Gin middleware function
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := rl.config.KeyGenerator(c)

		// Check Redis first if available
		if rl.redisClient != nil {
			allowed, err := rl.checkRedisRateLimit(key)
			if err != nil {
				rl.logger.Errorf("Redis rate limit check failed: %v", err)
				// Fallback to in-memory rate limiting
			} else {
				if !allowed {
					rl.handleRateLimitExceeded(c, key)
					return
				}
				c.Next()
				return
			}
		}

		// In-memory rate limiting
		if !rl.checkInMemoryRateLimit(key) {
			rl.handleRateLimitExceeded(c, key)
			return
		}

		c.Next()
	}
}

// checkInMemoryRateLimit checks rate limit using in-memory buckets
func (rl *RateLimiter) checkInMemoryRateLimit(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[key]
	if !exists {
		bucket = NewTokenBucket(rl.config.BurstSize, rl.config.RequestsPerSecond)
		rl.buckets[key] = bucket
	}

	return bucket.TakeToken()
}

// checkRedisRateLimit checks rate limit using Redis
func (rl *RateLimiter) checkRedisRateLimit(key string) (bool, error) {
	if rl.redisClient == nil {
		return false, fmt.Errorf("Redis client not configured")
	}

	ctx := context.Background()
	now := time.Now().Unix()

	// Use Redis sliding window algorithm
	script := `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local max_requests = tonumber(ARGV[3])
		
		-- Remove old entries
		redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
		
		-- Count current requests
		local current = redis.call('ZCARD', key)
		
		if current < max_requests then
			-- Add current request
			redis.call('ZADD', key, now, now)
			redis.call('EXPIRE', key, window)
			return 1
		else
			return 0
		end
	`

	result, err := rl.redisClient.Eval(ctx, script, []string{key}, now, int(rl.config.WindowDuration.Seconds()), rl.config.RequestsPerSecond).Result()
	if err != nil {
		return false, err
	}

	allowed, ok := result.(int64)
	if !ok {
		return false, fmt.Errorf("unexpected result type from Redis script")
	}

	return allowed == 1, nil
}

// handleRateLimitExceeded handles rate limit exceeded scenarios
func (rl *RateLimiter) handleRateLimitExceeded(c *gin.Context, key string) {
	// Get bucket info for headers
	var availableTokens int
	var resetTime time.Time

	if rl.redisClient != nil {
		resetTime = time.Now().Add(rl.config.WindowDuration)
		availableTokens = 0
	} else {
		rl.mu.RLock()
		if bucket, exists := rl.buckets[key]; exists {
			availableTokens = bucket.GetAvailableTokens()
			// Estimate reset time based on refill rate
			if availableTokens == 0 {
				resetTime = time.Now().Add(time.Duration(rl.config.RequestsPerSecond) * time.Second)
			} else {
				resetTime = time.Now()
			}
		}
		rl.mu.RUnlock()
	}

	// Set rate limit headers
	c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", rl.config.RequestsPerSecond))
	c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", availableTokens))
	c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

	// Log rate limit event
	if rl.logger != nil {
		rl.logger.Warnf("Rate limit exceeded for key: %s, IP: %s, Path: %s",
			key, c.ClientIP(), c.Request.URL.Path)
	}

	c.JSON(http.StatusTooManyRequests, gin.H{
		"error": "Rate limit exceeded",
		"code":  "RATE_LIMIT_EXCEEDED",
		"details": gin.H{
			"limit":     rl.config.RequestsPerSecond,
			"remaining": availableTokens,
			"reset_at":  resetTime.Unix(),
		},
	})
	c.Abort()
}

// defaultKeyGenerator generates a key based on client IP
func defaultKeyGenerator(c *gin.Context) string {
	return fmt.Sprintf("rate_limit:%s", c.ClientIP())
}

// IPKeyGenerator generates a key based on client IP only
func IPKeyGenerator(c *gin.Context) string {
	return fmt.Sprintf("rate_limit:ip:%s", c.ClientIP())
}

// UserKeyGenerator generates a key based on authenticated user ID
func UserKeyGenerator(c *gin.Context) string {
	userID, exists := c.Get("user_id")
	if !exists {
		return IPKeyGenerator(c)
	}
	return fmt.Sprintf("rate_limit:user:%v", userID)
}

// EndpointKeyGenerator generates a key based on endpoint and user
func EndpointKeyGenerator(c *gin.Context) string {
	userID, exists := c.Get("user_id")
	if !exists {
		return fmt.Sprintf("rate_limit:endpoint:%s:%s", c.ClientIP(), c.Request.URL.Path)
	}
	return fmt.Sprintf("rate_limit:endpoint:%v:%s", userID, c.Request.URL.Path)
}

// CustomKeyGenerator allows custom key generation logic
func CustomKeyGenerator(generator func(*gin.Context) string) func(*RateLimiterConfig) {
	return func(config *RateLimiterConfig) {
		config.KeyGenerator = generator
	}
}

// AdvancedRateLimiter implements multiple rate limiting strategies
type AdvancedRateLimiter struct {
	limiters map[string]*RateLimiter
	mu       sync.RWMutex
	logger   *logger.Logger
}

// NewAdvancedRateLimiter creates an advanced rate limiter with multiple strategies
func NewAdvancedRateLimiter(logger *logger.Logger) *AdvancedRateLimiter {
	return &AdvancedRateLimiter{
		limiters: make(map[string]*RateLimiter),
		logger:   logger,
	}
}

// AddLimiter adds a new rate limiter with a specific name
func (arl *AdvancedRateLimiter) AddLimiter(name string, limiter *RateLimiter) {
	arl.mu.Lock()
	defer arl.mu.Unlock()
	arl.limiters[name] = limiter
}

// GetLimiter returns a specific rate limiter by name
func (arl *AdvancedRateLimiter) GetLimiter(name string) *RateLimiter {
	arl.mu.RLock()
	defer arl.mu.RUnlock()
	return arl.limiters[name]
}

// MultiRateLimit creates middleware that applies multiple rate limiting strategies
func (arl *AdvancedRateLimiter) MultiRateLimit(limiterNames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, name := range limiterNames {
			if limiter := arl.GetLimiter(name); limiter != nil {
				key := limiter.config.KeyGenerator(c)

				// Check Redis first if available
				if limiter.redisClient != nil {
					allowed, err := limiter.checkRedisRateLimit(key)
					if err != nil {
						arl.logger.Errorf("Redis rate limit check failed for %s: %v", name, err)
						// Fallback to in-memory rate limiting
					} else {
						if !allowed {
							limiter.handleRateLimitExceeded(c, key)
							return
						}
						continue
					}
				}

				// In-memory rate limiting
				if !limiter.checkInMemoryRateLimit(key) {
					limiter.handleRateLimitExceeded(c, key)
					return
				}
			}
		}
		c.Next()
	}
}

// CleanupExpiredBuckets cleans up expired buckets periodically
func (rl *RateLimiter) CleanupExpiredBuckets() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, bucket := range rl.buckets {
			bucket.mu.Lock()
			// Remove buckets that haven't been used for 10 minutes
			if now.Sub(bucket.lastRefillTime) > 10*time.Minute {
				delete(rl.buckets, key)
			}
			bucket.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// GetRateLimitStats returns statistics about rate limiting
func (rl *RateLimiter) GetRateLimitStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	stats := map[string]interface{}{
		"total_buckets": len(rl.buckets),
		"config": map[string]interface{}{
			"requests_per_second": rl.config.RequestsPerSecond,
			"burst_size":          rl.config.BurstSize,
			"window_duration":     rl.config.WindowDuration.String(),
		},
	}

	bucketStats := make(map[string]interface{})
	for key, bucket := range rl.buckets {
		bucket.mu.Lock()
		bucketStats[key] = map[string]interface{}{
			"available_tokens": bucket.GetAvailableTokens(),
			"capacity":         bucket.capacity,
			"last_refill":      bucket.lastRefillTime.Format(time.RFC3339),
		}
		bucket.mu.Unlock()
	}
	stats["buckets"] = bucketStats

	return stats
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
