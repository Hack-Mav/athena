package device

import (
	"context"
	"time"
)

// Repository defines the interface for device data operations
type Repository interface {
	// Device CRUD operations
	RegisterDevice(ctx context.Context, device *Device) error
	GetDevice(ctx context.Context, deviceID string) (*Device, error)
	UpdateDevice(ctx context.Context, device *Device) error
	DeleteDevice(ctx context.Context, deviceID string) error

	// Device listing and filtering
	ListDevices(ctx context.Context, filters *DeviceFilters) ([]*Device, error)
	GetDeviceCount(ctx context.Context, filters *DeviceFilters) (int64, error)
	SearchDevices(ctx context.Context, query string, filters *DeviceFilters) ([]*Device, error)

	// Device status operations
	UpdateDeviceStatus(ctx context.Context, deviceID string, status DeviceStatus, lastSeen time.Time) error
	GetDevicesByStatus(ctx context.Context, status DeviceStatus) ([]*Device, error)
	GetOfflineDevices(ctx context.Context, timeout time.Duration) ([]*Device, error)

	// Device health monitoring
	GetDeviceHealthStatus(ctx context.Context) (*DeviceHealthStatus, error)
	GetDevicesByTemplate(ctx context.Context, templateID string) ([]*Device, error)
	GetDevicesByOTAChannel(ctx context.Context, channel string) ([]*Device, error)

	// Utility methods
	DeviceExists(ctx context.Context, deviceID string) (bool, error)
	GetDevicesLastSeenBefore(ctx context.Context, before time.Time) ([]*Device, error)
}
