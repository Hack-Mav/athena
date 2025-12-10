package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/athena/platform-lib/pkg/device"
	"github.com/athena/platform-lib/pkg/logger"
)

// DeviceCache provides caching for device operations
type DeviceCache struct {
	cache  *Cache
	logger logger.Logger
}

// NewDeviceCache creates a new device cache instance
func NewDeviceCache(cache *Cache, logger logger.Logger) *DeviceCache {
	return &DeviceCache{
		cache:  cache,
		logger: logger,
	}
}

// GetDevice retrieves a device from cache or database
func (dc *DeviceCache) GetDevice(ctx context.Context, deviceID string) (*device.Device, error) {
	// Try cache first
	cacheKey := GenerateDeviceKey(deviceID)

	var cachedDevice device.Device
	if err := dc.cache.Get(ctx, cacheKey, &cachedDevice); err == nil {
		dc.logger.Debug("Device retrieved from cache", "device_id", deviceID)
		return &cachedDevice, nil
	}

	// Cache miss
	dc.logger.Debug("Device cache miss", "device_id", deviceID)
	return nil, ErrCacheMiss
}

// SetDevice stores a device in cache
func (dc *DeviceCache) SetDevice(ctx context.Context, dev *device.Device, ttl time.Duration) error {
	if dev == nil {
		return ErrInvalidValue
	}

	cacheKey := GenerateDeviceKey(dev.DeviceID)
	if err := dc.cache.Set(ctx, cacheKey, dev, ttl); err != nil {
		dc.logger.Error("Failed to cache device", "device_id", dev.DeviceID, "error", err)
		return err
	}

	dc.logger.Debug("Device cached", "device_id", dev.DeviceID, "ttl", ttl)
	return nil
}

// InvalidateDevice removes a device from cache
func (dc *DeviceCache) InvalidateDevice(ctx context.Context, deviceID string) error {
	cacheKey := GenerateDeviceKey(deviceID)
	if err := dc.cache.Delete(ctx, cacheKey); err != nil {
		dc.logger.Error("Failed to invalidate device cache", "device_id", deviceID, "error", err)
		return err
	}

	dc.logger.Debug("Device cache invalidated", "device_id", deviceID)
	return nil
}

// GetDeviceList retrieves a list of devices from cache
func (dc *DeviceCache) GetDeviceList(ctx context.Context, filter string) ([]*device.Device, error) {
	cacheKey := fmt.Sprintf("%s:list:%s", DeviceKeyPrefix, filter)

	var cachedDevices []*device.Device
	if err := dc.cache.Get(ctx, cacheKey, &cachedDevices); err == nil {
		dc.logger.Debug("Device list retrieved from cache", "filter", filter)
		return cachedDevices, nil
	}

	dc.logger.Debug("Device list cache miss", "filter", filter)
	return nil, ErrCacheMiss
}

// SetDeviceList stores a list of devices in cache
func (dc *DeviceCache) SetDeviceList(ctx context.Context, devices []*device.Device, filter string, ttl time.Duration) error {
	cacheKey := fmt.Sprintf("%s:list:%s", DeviceKeyPrefix, filter)

	if err := dc.cache.Set(ctx, cacheKey, devices, ttl); err != nil {
		dc.logger.Error("Failed to cache device list", "filter", filter, "error", err)
		return err
	}

	dc.logger.Debug("Device list cached", "filter", filter, "count", len(devices), "ttl", ttl)
	return nil
}

// InvalidateDeviceList removes a device list from cache
func (dc *DeviceCache) InvalidateDeviceList(ctx context.Context, filter string) error {
	cacheKey := fmt.Sprintf("%s:list:%s", DeviceKeyPrefix, filter)
	if err := dc.cache.Delete(ctx, cacheKey); err != nil {
		dc.logger.Error("Failed to invalidate device list cache", "filter", filter, "error", err)
		return err
	}

	dc.logger.Debug("Device list cache invalidated", "filter", filter)
	return nil
}

// GetDeviceByMAC retrieves a device by device ID from cache
func (dc *DeviceCache) GetDeviceByMAC(ctx context.Context, deviceID string) (*device.Device, error) {
	// Note: Device model doesn't have MAC address field, using device_id instead
	cacheKey := fmt.Sprintf("%s:id:%s", DeviceKeyPrefix, deviceID)

	var cachedDevice device.Device
	if err := dc.cache.Get(ctx, cacheKey, &cachedDevice); err == nil {
		dc.logger.Debug("Device retrieved from cache by ID", "device_id", deviceID)
		return &cachedDevice, nil
	}

	dc.logger.Debug("Device ID cache miss", "device_id", deviceID)
	return nil, ErrCacheMiss
}

// SetDeviceByMAC stores a device indexed by device ID in cache
func (dc *DeviceCache) SetDeviceByMAC(ctx context.Context, dev *device.Device, ttl time.Duration) error {
	if dev == nil || dev.DeviceID == "" {
		return ErrInvalidValue
	}

	// Note: Device model doesn't have MAC address field, using device_id instead
	cacheKey := fmt.Sprintf("%s:id:%s", DeviceKeyPrefix, dev.DeviceID)
	if err := dc.cache.Set(ctx, cacheKey, dev, ttl); err != nil {
		dc.logger.Error("Failed to cache device by ID", "device_id", dev.DeviceID, "error", err)
		return err
	}

	dc.logger.Debug("Device cached by ID", "device_id", dev.DeviceID, "ttl", ttl)
	return nil
}

// InvalidateDeviceByMAC removes a device by MAC address from cache
func (dc *DeviceCache) InvalidateDeviceByMAC(ctx context.Context, deviceID string) error {
	// Note: Device model doesn't have MAC address field, using device_id instead
	cacheKey := fmt.Sprintf("%s:id:%s", DeviceKeyPrefix, deviceID)
	if err := dc.cache.Delete(ctx, cacheKey); err != nil {
		dc.logger.Error("Failed to invalidate device ID cache", "device_id", deviceID, "error", err)
		return err
	}

	dc.logger.Debug("Device ID cache invalidated", "device_id", deviceID)
	return nil
}

// GetOnlineDevices retrieves a list of online devices from cache
func (dc *DeviceCache) GetOnlineDevices(ctx context.Context) ([]*device.Device, error) {
	cacheKey := fmt.Sprintf("%s:online", DeviceKeyPrefix)

	var cachedDevices []*device.Device
	if err := dc.cache.Get(ctx, cacheKey, &cachedDevices); err == nil {
		dc.logger.Debug("Online devices retrieved from cache")
		return cachedDevices, nil
	}

	dc.logger.Debug("Online devices cache miss")
	return nil, ErrCacheMiss
}

// SetOnlineDevices stores a list of online devices in cache
func (dc *DeviceCache) SetOnlineDevices(ctx context.Context, devices []*device.Device, ttl time.Duration) error {
	cacheKey := fmt.Sprintf("%s:online", DeviceKeyPrefix)

	if err := dc.cache.Set(ctx, cacheKey, devices, ttl); err != nil {
		dc.logger.Error("Failed to cache online devices", "error", err)
		return err
	}

	dc.logger.Debug("Online devices cached", "count", len(devices), "ttl", ttl)
	return nil
}

// InvalidateAllDevices removes all device-related cache entries
func (dc *DeviceCache) InvalidateAllDevices(ctx context.Context) error {
	// Simplified approach - clear all cache
	dc.logger.Warn("Clearing all cache (simplified approach)")
	if err := dc.cache.Clear(ctx); err != nil {
		dc.logger.Error("Failed to clear all device cache", "error", err)
		return err
	}

	dc.logger.Info("All device cache invalidated")
	return nil
}

// CacheStats provides cache statistics
func (dc *DeviceCache) CacheStats(ctx context.Context) (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"type":    "device_cache",
		"backend": "redis",
		"status":  "active",
	}

	// Check cache health
	if err := dc.cache.Health(ctx); err != nil {
		stats["health"] = "unhealthy"
		stats["error"] = err.Error()
	} else {
		stats["health"] = "healthy"
	}

	return stats, nil
}
