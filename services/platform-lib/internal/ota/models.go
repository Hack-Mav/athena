package ota

import (
	"encoding/json"
	"time"
)

// ReleaseChannel represents the release channel type
type ReleaseChannel string

const (
	ReleaseChannelStable ReleaseChannel = "stable"
	ReleaseChannelBeta   ReleaseChannel = "beta"
	ReleaseChannelAlpha  ReleaseChannel = "alpha"
)

// DeploymentStatus represents the status of a deployment
type DeploymentStatus string

const (
	DeploymentStatusPending   DeploymentStatus = "pending"
	DeploymentStatusActive    DeploymentStatus = "active"
	DeploymentStatusPaused    DeploymentStatus = "paused"
	DeploymentStatusCompleted DeploymentStatus = "completed"
	DeploymentStatusFailed    DeploymentStatus = "failed"
)

// DeploymentStrategy represents the deployment strategy type
type DeploymentStrategy string

const (
	DeploymentStrategyImmediate DeploymentStrategy = "immediate"
	DeploymentStrategyStaged    DeploymentStrategy = "staged"
	DeploymentStrategyCanary    DeploymentStrategy = "canary"
)

// UpdateStatus represents the status of a device update
type UpdateStatus string

const (
	UpdateStatusPending     UpdateStatus = "pending"
	UpdateStatusDownloading UpdateStatus = "downloading"
	UpdateStatusInstalling  UpdateStatus = "installing"
	UpdateStatusCompleted   UpdateStatus = "completed"
	UpdateStatusFailed      UpdateStatus = "failed"
)

// FirmwareRelease represents a firmware release
type FirmwareRelease struct {
	ReleaseID    string         `json:"release_id"`
	TemplateID   string         `json:"template_id"`
	Version      string         `json:"version"`
	Channel      ReleaseChannel `json:"channel"`
	BinaryHash   string         `json:"binary_hash"`
	BinaryPath   string         `json:"binary_path"`
	BinarySize   int64          `json:"binary_size"`
	Signature    string         `json:"signature"`
	ReleaseNotes string         `json:"release_notes"`
	CreatedAt    time.Time      `json:"created_at"`
	CreatedBy    string         `json:"created_by"`
}

// FirmwareReleaseEntity represents the Datastore entity for firmware releases
type FirmwareReleaseEntity struct {
	ReleaseID    string    `datastore:"release_id"`
	TemplateID   string    `datastore:"template_id"`
	Version      string    `datastore:"version"`
	Channel      string    `datastore:"channel"`
	BinaryHash   string    `datastore:"binary_hash"`
	BinaryPath   string    `datastore:"binary_path"`
	BinarySize   int64     `datastore:"binary_size"`
	Signature    string    `datastore:"signature,noindex"`
	ReleaseNotes string    `datastore:"release_notes,noindex"`
	CreatedAt    time.Time `datastore:"created_at"`
	CreatedBy    string    `datastore:"created_by"`
}

// OTADeployment represents an OTA deployment configuration
type OTADeployment struct {
	DeploymentID       string             `json:"deployment_id"`
	ReleaseID          string             `json:"release_id"`
	Strategy           DeploymentStrategy `json:"strategy"`
	TargetDevices      []string           `json:"target_devices"`
	RolloutPercentage  int                `json:"rollout_percentage"`
	Status             DeploymentStatus   `json:"status"`
	FailureThreshold   int                `json:"failure_threshold"`
	SuccessCount       int                `json:"success_count"`
	FailureCount       int                `json:"failure_count"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

// OTADeploymentEntity represents the Datastore entity for OTA deployments
type OTADeploymentEntity struct {
	DeploymentID      string    `datastore:"deployment_id"`
	ReleaseID         string    `datastore:"release_id"`
	Strategy          string    `datastore:"strategy"`
	TargetDevicesJSON string    `datastore:"target_devices_json,noindex"`
	RolloutPercentage int       `datastore:"rollout_percentage"`
	Status            string    `datastore:"status"`
	FailureThreshold  int       `datastore:"failure_threshold"`
	SuccessCount      int       `datastore:"success_count"`
	FailureCount      int       `datastore:"failure_count"`
	CreatedAt         time.Time `datastore:"created_at"`
	UpdatedAt         time.Time `datastore:"updated_at"`
}

// DeviceUpdate represents the update status for a specific device
type DeviceUpdate struct {
	DeviceID     string       `json:"device_id"`
	ReleaseID    string       `json:"release_id"`
	DeploymentID string       `json:"deployment_id"`
	Status       UpdateStatus `json:"status"`
	Progress     int          `json:"progress"`
	ErrorMessage string       `json:"error_message,omitempty"`
	StartedAt    time.Time    `json:"started_at"`
	CompletedAt  *time.Time   `json:"completed_at,omitempty"`
}

// DeviceUpdateEntity represents the Datastore entity for device updates
type DeviceUpdateEntity struct {
	DeviceID     string    `datastore:"device_id"`
	ReleaseID    string    `datastore:"release_id"`
	DeploymentID string    `datastore:"deployment_id"`
	Status       string    `datastore:"status"`
	Progress     int       `datastore:"progress"`
	ErrorMessage string    `datastore:"error_message,noindex"`
	StartedAt    time.Time `datastore:"started_at"`
	CompletedAt  time.Time `datastore:"completed_at"`
}

// CreateReleaseRequest represents a request to create a new firmware release
type CreateReleaseRequest struct {
	TemplateID   string         `json:"template_id" binding:"required"`
	Version      string         `json:"version" binding:"required"`
	Channel      ReleaseChannel `json:"channel" binding:"required"`
	BinaryData   []byte         `json:"binary_data" binding:"required"`
	ReleaseNotes string         `json:"release_notes"`
	CreatedBy    string         `json:"created_by"`
}

// DeploymentConfig represents the configuration for a deployment
type DeploymentConfig struct {
	Strategy          DeploymentStrategy `json:"strategy" binding:"required"`
	TargetDevices     []string           `json:"target_devices"`
	RolloutPercentage int                `json:"rollout_percentage"`
	FailureThreshold  int                `json:"failure_threshold"`
}

// UpdateStatusReport represents a status report from a device
type UpdateStatusReport struct {
	DeviceID     string       `json:"device_id" binding:"required"`
	ReleaseID    string       `json:"release_id" binding:"required"`
	Status       UpdateStatus `json:"status" binding:"required"`
	Progress     int          `json:"progress"`
	ErrorMessage string       `json:"error_message,omitempty"`
}

// FirmwareUpdate represents the update information for a device
type FirmwareUpdate struct {
	ReleaseID    string    `json:"release_id"`
	Version      string    `json:"version"`
	BinaryURL    string    `json:"binary_url"`
	BinaryHash   string    `json:"binary_hash"`
	BinarySize   int64     `json:"binary_size"`
	Signature    string    `json:"signature"`
	ReleaseNotes string    `json:"release_notes"`
	CreatedAt    time.Time `json:"created_at"`
}

// ToEntity converts a FirmwareRelease to a FirmwareReleaseEntity
func (r *FirmwareRelease) ToEntity() (*FirmwareReleaseEntity, error) {
	return &FirmwareReleaseEntity{
		ReleaseID:    r.ReleaseID,
		TemplateID:   r.TemplateID,
		Version:      r.Version,
		Channel:      string(r.Channel),
		BinaryHash:   r.BinaryHash,
		BinaryPath:   r.BinaryPath,
		BinarySize:   r.BinarySize,
		Signature:    r.Signature,
		ReleaseNotes: r.ReleaseNotes,
		CreatedAt:    r.CreatedAt,
		CreatedBy:    r.CreatedBy,
	}, nil
}

// FromEntity converts a FirmwareReleaseEntity to a FirmwareRelease
func (e *FirmwareReleaseEntity) FromEntity() (*FirmwareRelease, error) {
	return &FirmwareRelease{
		ReleaseID:    e.ReleaseID,
		TemplateID:   e.TemplateID,
		Version:      e.Version,
		Channel:      ReleaseChannel(e.Channel),
		BinaryHash:   e.BinaryHash,
		BinaryPath:   e.BinaryPath,
		BinarySize:   e.BinarySize,
		Signature:    e.Signature,
		ReleaseNotes: e.ReleaseNotes,
		CreatedAt:    e.CreatedAt,
		CreatedBy:    e.CreatedBy,
	}, nil
}

// ToEntity converts an OTADeployment to an OTADeploymentEntity
func (d *OTADeployment) ToEntity() (*OTADeploymentEntity, error) {
	targetDevicesJSON, err := json.Marshal(d.TargetDevices)
	if err != nil {
		return nil, err
	}

	return &OTADeploymentEntity{
		DeploymentID:      d.DeploymentID,
		ReleaseID:         d.ReleaseID,
		Strategy:          string(d.Strategy),
		TargetDevicesJSON: string(targetDevicesJSON),
		RolloutPercentage: d.RolloutPercentage,
		Status:            string(d.Status),
		FailureThreshold:  d.FailureThreshold,
		SuccessCount:      d.SuccessCount,
		FailureCount:      d.FailureCount,
		CreatedAt:         d.CreatedAt,
		UpdatedAt:         d.UpdatedAt,
	}, nil
}

// FromEntity converts an OTADeploymentEntity to an OTADeployment
func (e *OTADeploymentEntity) FromEntity() (*OTADeployment, error) {
	var targetDevices []string
	if e.TargetDevicesJSON != "" {
		if err := json.Unmarshal([]byte(e.TargetDevicesJSON), &targetDevices); err != nil {
			return nil, err
		}
	}

	return &OTADeployment{
		DeploymentID:      e.DeploymentID,
		ReleaseID:         e.ReleaseID,
		Strategy:          DeploymentStrategy(e.Strategy),
		TargetDevices:     targetDevices,
		RolloutPercentage: e.RolloutPercentage,
		Status:            DeploymentStatus(e.Status),
		FailureThreshold:  e.FailureThreshold,
		SuccessCount:      e.SuccessCount,
		FailureCount:      e.FailureCount,
		CreatedAt:         e.CreatedAt,
		UpdatedAt:         e.UpdatedAt,
	}, nil
}

// ToEntity converts a DeviceUpdate to a DeviceUpdateEntity
func (u *DeviceUpdate) ToEntity() (*DeviceUpdateEntity, error) {
	entity := &DeviceUpdateEntity{
		DeviceID:     u.DeviceID,
		ReleaseID:    u.ReleaseID,
		DeploymentID: u.DeploymentID,
		Status:       string(u.Status),
		Progress:     u.Progress,
		ErrorMessage: u.ErrorMessage,
		StartedAt:    u.StartedAt,
	}

	if u.CompletedAt != nil {
		entity.CompletedAt = *u.CompletedAt
	}

	return entity, nil
}

// FromEntity converts a DeviceUpdateEntity to a DeviceUpdate
func (e *DeviceUpdateEntity) FromEntity() (*DeviceUpdate, error) {
	update := &DeviceUpdate{
		DeviceID:     e.DeviceID,
		ReleaseID:    e.ReleaseID,
		DeploymentID: e.DeploymentID,
		Status:       UpdateStatus(e.Status),
		Progress:     e.Progress,
		ErrorMessage: e.ErrorMessage,
		StartedAt:    e.StartedAt,
	}

	if !e.CompletedAt.IsZero() {
		update.CompletedAt = &e.CompletedAt
	}

	return update, nil
}
