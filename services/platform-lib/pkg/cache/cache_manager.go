package cache

import (
	"context"
	"time"

	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/logger"
)

// CacheManager manages all cache services
type CacheManager struct {
	redisCache    *Cache
	templateCache *TemplateCache
	deviceCache   *DeviceCache
	logger        logger.Logger
	config        *config.Config
}

// NewCacheManager creates a new cache manager instance
func NewCacheManager(cfg *config.Config, log logger.Logger) (*CacheManager, error) {
	// Initialize Redis cache
	redisCache, err := NewCache(cfg, log)
	if err != nil {
		return nil, err
	}

	// Initialize specialized caches
	templateCache := NewTemplateCache(redisCache, log)
	deviceCache := NewDeviceCache(redisCache, log)

	log.Info("Cache manager initialized successfully")

	return &CacheManager{
		redisCache:    redisCache,
		templateCache: templateCache,
		deviceCache:   deviceCache,
		logger:        log,
		config:        cfg,
	}, nil
}

// GetTemplateCache returns the template cache service
func (cm *CacheManager) GetTemplateCache() *TemplateCache {
	return cm.templateCache
}

// GetDeviceCache returns the device cache service
func (cm *CacheManager) GetDeviceCache() *DeviceCache {
	return cm.deviceCache
}

// GetRedisCache returns the underlying Redis cache
func (cm *CacheManager) GetRedisCache() *Cache {
	return cm.redisCache
}

// Health checks the health of all cache services
func (cm *CacheManager) Health(ctx context.Context) map[string]error {
	results := make(map[string]error)

	// Check Redis connection
	if err := cm.redisCache.Health(ctx); err != nil {
		results["redis"] = err
	} else {
		results["redis"] = nil
	}

	return results
}

// Stats returns statistics for all cache services
func (cm *CacheManager) Stats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get template cache stats
	templateStats, err := cm.templateCache.CacheStats(ctx)
	if err != nil {
		stats["template_cache"] = map[string]interface{}{"error": err.Error()}
	} else {
		stats["template_cache"] = templateStats
	}

	// Get device cache stats
	deviceStats, err := cm.deviceCache.CacheStats(ctx)
	if err != nil {
		stats["device_cache"] = map[string]interface{}{"error": err.Error()}
	} else {
		stats["device_cache"] = deviceStats
	}

	// Add overall cache manager stats
	stats["cache_manager"] = map[string]interface{}{
		"type":     "cache_manager",
		"backend":  "redis",
		"services": []string{"template", "device"},
		"status":   "active",
	}

	return stats, nil
}

// InvalidateAll invalidates all cache entries
func (cm *CacheManager) InvalidateAll(ctx context.Context) error {
	cm.logger.Info("Invalidating all cache entries")

	// Clear Redis cache
	if err := cm.redisCache.Clear(ctx); err != nil {
		cm.logger.Error("Failed to clear all cache", "error", err)
		return err
	}

	cm.logger.Info("All cache entries invalidated")
	return nil
}

// Warmup preloads commonly accessed data into cache
func (cm *CacheManager) Warmup(ctx context.Context) error {
	cm.logger.Info("Starting cache warmup")

	// This would typically preload frequently accessed templates and devices
	// For now, we'll just log the warmup attempt

	cm.logger.Info("Cache warmup completed")
	return nil
}

// Close closes all cache connections
func (cm *CacheManager) Close() error {
	cm.logger.Info("Closing cache manager")

	if err := cm.redisCache.Close(); err != nil {
		cm.logger.Error("Failed to close Redis connection", "error", err)
		return err
	}

	cm.logger.Info("Cache manager closed")
	return nil
}

// DefaultTTLs defines default cache expiration times
const (
	DefaultTemplateTTL = 30 * time.Minute
	DefaultDeviceTTL   = 15 * time.Minute
	DefaultListTTL     = 5 * time.Minute
	DefaultOnlineTTL   = 2 * time.Minute
	DefaultSessionTTL  = 24 * time.Hour
)

// Cache configuration
type CacheConfig struct {
	TemplateTTL time.Duration
	DeviceTTL   time.Duration
	ListTTL     time.Duration
	OnlineTTL   time.Duration
	SessionTTL  time.Duration
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		TemplateTTL: DefaultTemplateTTL,
		DeviceTTL:   DefaultDeviceTTL,
		ListTTL:     DefaultListTTL,
		OnlineTTL:   DefaultOnlineTTL,
		SessionTTL:  DefaultSessionTTL,
	}
}

// GetCacheConfig returns cache configuration from app config
func (cm *CacheManager) GetCacheConfig() *CacheConfig {
	config := DefaultCacheConfig()

	// Override with values from app config if available
	// This would typically read from the config file

	return config
}
