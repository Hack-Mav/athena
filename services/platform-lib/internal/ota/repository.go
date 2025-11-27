package ota

import (
	"context"
)

// Repository defines the interface for OTA data operations
type Repository interface {
	// Firmware release operations
	CreateRelease(ctx context.Context, release *FirmwareRelease) error
	GetRelease(ctx context.Context, releaseID string) (*FirmwareRelease, error)
	GetReleaseByVersion(ctx context.Context, templateID, version string, channel ReleaseChannel) (*FirmwareRelease, error)
	ListReleases(ctx context.Context, templateID string, channel ReleaseChannel) ([]*FirmwareRelease, error)
	DeleteRelease(ctx context.Context, releaseID string) error
	ReleaseExists(ctx context.Context, releaseID string) (bool, error)

	// Deployment operations
	CreateDeployment(ctx context.Context, deployment *OTADeployment) error
	GetDeployment(ctx context.Context, deploymentID string) (*OTADeployment, error)
	UpdateDeployment(ctx context.Context, deployment *OTADeployment) error
	ListDeployments(ctx context.Context, releaseID string) ([]*OTADeployment, error)
	GetActiveDeployments(ctx context.Context) ([]*OTADeployment, error)

	// Device update operations
	CreateDeviceUpdate(ctx context.Context, update *DeviceUpdate) error
	GetDeviceUpdate(ctx context.Context, deviceID, releaseID string) (*DeviceUpdate, error)
	UpdateDeviceUpdate(ctx context.Context, update *DeviceUpdate) error
	ListDeviceUpdates(ctx context.Context, deploymentID string) ([]*DeviceUpdate, error)
	GetDeviceUpdatesByStatus(ctx context.Context, deploymentID string, status UpdateStatus) ([]*DeviceUpdate, error)
	GetLatestUpdateForDevice(ctx context.Context, deviceID string) (*DeviceUpdate, error)

	// Query operations
	GetDeploymentStats(ctx context.Context, deploymentID string) (successCount, failureCount, pendingCount int, err error)
	GetDevicesPendingUpdate(ctx context.Context, deploymentID string, limit int) ([]*DeviceUpdate, error)
}
