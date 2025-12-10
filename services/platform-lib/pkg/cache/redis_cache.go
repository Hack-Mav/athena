package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// Cache provides Redis-based caching functionality
type Cache struct {
	client *redis.Client
	logger logger.Logger
	config *config.Config
}

// NewCache creates a new Redis cache instance
func NewCache(cfg *config.Config, log logger.Logger) (*Cache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Info("Redis cache connected successfully")

	return &Cache{
		client: rdb,
		logger: log,
		config: cfg,
	}, nil
}

// Set stores a value in the cache with an expiration
func (c *Cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}

	if err := c.client.Set(ctx, key, data, expiration).Err(); err != nil {
		c.logger.Error("Failed to set cache value", "key", key, "error", err)
		return fmt.Errorf("failed to set cache value: %w", err)
	}

	c.logger.Debug("Cache value set", "key", key, "expiration", expiration)
	return nil
}

// Get retrieves a value from the cache
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheMiss
		}
		c.logger.Error("Failed to get cache value", "key", key, "error", err)
		return fmt.Errorf("failed to get cache value: %w", err)
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("failed to unmarshal cache value: %w", err)
	}

	c.logger.Debug("Cache value retrieved", "key", key)
	return nil
}

// Delete removes a value from the cache
func (c *Cache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		c.logger.Error("Failed to delete cache value", "key", key, "error", err)
		return fmt.Errorf("failed to delete cache value: %w", err)
	}

	c.logger.Debug("Cache value deleted", "key", key)
	return nil
}

// Clear removes all values from the cache
func (c *Cache) Clear(ctx context.Context) error {
	if err := c.client.FlushDB(ctx).Err(); err != nil {
		c.logger.Error("Failed to clear cache", "error", err)
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	c.logger.Info("Cache cleared")
	return nil
}

// Exists checks if a key exists in the cache
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		c.logger.Error("Failed to check cache existence", "key", key, "error", err)
		return false, fmt.Errorf("failed to check cache existence: %w", err)
	}

	return count > 0, nil
}

// SetMultiple stores multiple values in the cache
func (c *Cache) SetMultiple(ctx context.Context, items map[string]interface{}, expiration time.Duration) error {
	pipe := c.client.Pipeline()

	for key, value := range items {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal cache value for key %s: %w", key, err)
		}
		pipe.Set(ctx, key, data, expiration)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		c.logger.Error("Failed to set multiple cache values", "error", err)
		return fmt.Errorf("failed to set multiple cache values: %w", err)
	}

	c.logger.Debug("Multiple cache values set", "count", len(items))
	return nil
}

// GetMultiple retrieves multiple values from the cache
func (c *Cache) GetMultiple(ctx context.Context, keys []string) (map[string]interface{}, error) {
	pipe := c.client.Pipeline()
	cmds := make(map[string]*redis.StringCmd)

	for _, key := range keys {
		cmds[key] = pipe.Get(ctx, key)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		c.logger.Error("Failed to get multiple cache values", "error", err)
		return nil, fmt.Errorf("failed to get multiple cache values: %w", err)
	}

	result := make(map[string]interface{})
	for key, cmd := range cmds {
		data, err := cmd.Result()
		if err != nil {
			if err == redis.Nil {
				continue // Skip missing keys
			}
			return nil, fmt.Errorf("failed to get cache value for key %s: %w", key, err)
		}

		var value interface{}
		if err := json.Unmarshal([]byte(data), &value); err != nil {
			return nil, fmt.Errorf("failed to unmarshal cache value for key %s: %w", key, err)
		}

		result[key] = value
	}

	c.logger.Debug("Multiple cache values retrieved", "count", len(result))
	return result, nil
}

// Increment increments a numeric value in the cache
func (c *Cache) Increment(ctx context.Context, key string) (int64, error) {
	result, err := c.client.Incr(ctx, key).Result()
	if err != nil {
		c.logger.Error("Failed to increment cache value", "key", key, "error", err)
		return 0, fmt.Errorf("failed to increment cache value: %w", err)
	}

	c.logger.Debug("Cache value incremented", "key", key, "value", result)
	return result, nil
}

// SetWithTTL stores a value with a specific TTL
func (c *Cache) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.Set(ctx, key, value, ttl)
}

// GetTTL returns the remaining time-to-live for a key
func (c *Cache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := c.client.TTL(ctx, key).Result()
	if err != nil {
		c.logger.Error("Failed to get cache TTL", "key", key, "error", err)
		return 0, fmt.Errorf("failed to get cache TTL: %w", err)
	}

	return ttl, nil
}

// Close closes the Redis connection
func (c *Cache) Close() error {
	if err := c.client.Close(); err != nil {
		c.logger.Error("Failed to close cache connection", "error", err)
		return fmt.Errorf("failed to close cache connection: %w", err)
	}

	c.logger.Info("Cache connection closed")
	return nil
}

// Health checks the health of the Redis connection
func (c *Cache) Health(ctx context.Context) error {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("cache health check failed: %w", err)
	}
	return nil
}

// Cache errors
var (
	ErrCacheMiss    = fmt.Errorf("cache miss")
	ErrInvalidKey   = fmt.Errorf("invalid cache key")
	ErrInvalidValue = fmt.Errorf("invalid cache value")
)

// Cache key generators
const (
	TemplateKeyPrefix = "template:"
	DeviceKeyPrefix   = "device:"
	UserKeyPrefix     = "user:"
	SessionKeyPrefix  = "session:"
)

// GenerateTemplateKey generates a cache key for templates
func GenerateTemplateKey(templateID string) string {
	return fmt.Sprintf("%s%s", TemplateKeyPrefix, templateID)
}

// GenerateDeviceKey generates a cache key for devices
func GenerateDeviceKey(deviceID string) string {
	return fmt.Sprintf("%s%s", DeviceKeyPrefix, deviceID)
}

// GenerateUserKey generates a cache key for users
func GenerateUserKey(userID string) string {
	return fmt.Sprintf("%s%s", UserKeyPrefix, userID)
}

// GenerateSessionKey generates a cache key for sessions
func GenerateSessionKey(sessionID string) string {
	return fmt.Sprintf("%s%s", SessionKeyPrefix, sessionID)
}
