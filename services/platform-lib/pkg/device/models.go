package device

import (
	"encoding/json"
	"time"
)

// DeviceStatus represents the current status of a device
type DeviceStatus string

const (
	DeviceStatusProvisioned DeviceStatus = "provisioned"
	DeviceStatusOnline      DeviceStatus = "online"
	DeviceStatusOffline     DeviceStatus = "offline"
	DeviceStatusError       DeviceStatus = "error"
)

// Device represents an Arduino device in the registry
type Device struct {
	DeviceID        string                 `json:"device_id"`
	BoardType       string                 `json:"board_type"`
	Status          DeviceStatus           `json:"status"`
	TemplateID      string                 `json:"template_id"`
	TemplateVersion string                 `json:"template_version"`
	Parameters      map[string]interface{} `json:"parameters"`
	SecretsRef      string                 `json:"secrets_ref,omitempty"`
	FirmwareHash    string                 `json:"firmware_hash"`
	LastSeen        time.Time              `json:"last_seen"`
	OTAChannel      string                 `json:"ota_channel"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// DeviceEntity represents the Datastore entity for devices
type DeviceEntity struct {
	DeviceID        string    `datastore:"device_id"`
	BoardType       string    `datastore:"board_type"`
	Status          string    `datastore:"status"`
	TemplateID      string    `datastore:"template_id"`
	TemplateVersion string    `datastore:"template_version"`
	ParametersJSON  string    `datastore:"parameters_json,noindex"`
	SecretsRef      string    `datastore:"secrets_ref"`
	FirmwareHash    string    `datastore:"firmware_hash"`
	LastSeen        time.Time `datastore:"last_seen"`
	OTAChannel      string    `datastore:"ota_channel"`
	CreatedAt       time.Time `datastore:"created_at"`
	UpdatedAt       time.Time `datastore:"updated_at"`
}

// DeviceFilters represents filters for device queries
type DeviceFilters struct {
	Status          DeviceStatus `json:"status,omitempty"`
	BoardType       string       `json:"board_type,omitempty"`
	TemplateID      string       `json:"template_id,omitempty"`
	OTAChannel      string       `json:"ota_channel,omitempty"`
	LastSeenBefore  *time.Time   `json:"last_seen_before,omitempty"`
	LastSeenAfter   *time.Time   `json:"last_seen_after,omitempty"`
	Limit           int          `json:"limit,omitempty"`
	Offset          int          `json:"offset,omitempty"`
}

// DeviceRegistrationRequest represents a request to register a new device
type DeviceRegistrationRequest struct {
	DeviceID        string                 `json:"device_id" binding:"required"`
	BoardType       string                 `json:"board_type" binding:"required"`
	TemplateID      string                 `json:"template_id" binding:"required"`
	TemplateVersion string                 `json:"template_version" binding:"required"`
	Parameters      map[string]interface{} `json:"parameters"`
	SecretsRef      string                 `json:"secrets_ref,omitempty"`
	FirmwareHash    string                 `json:"firmware_hash" binding:"required"`
	OTAChannel      string                 `json:"ota_channel,omitempty"`
}

// DeviceStatusUpdate represents a device status update
type DeviceStatusUpdate struct {
	Status   DeviceStatus `json:"status" binding:"required"`
	LastSeen *time.Time   `json:"last_seen,omitempty"`
}

// DeviceHeartbeat represents a device heartbeat message
type DeviceHeartbeat struct {
	DeviceID  string                 `json:"device_id" binding:"required"`
	Timestamp time.Time              `json:"timestamp"`
	Status    DeviceStatus           `json:"status"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
}

// DeviceListResponse represents the response for device listing
type DeviceListResponse struct {
	Devices []Device `json:"devices"`
	Total   int64    `json:"total"`
	Limit   int      `json:"limit"`
	Offset  int      `json:"offset"`
}

// DeviceHealthStatus represents aggregated health status
type DeviceHealthStatus struct {
	TotalDevices   int64 `json:"total_devices"`
	OnlineDevices  int64 `json:"online_devices"`
	OfflineDevices int64 `json:"offline_devices"`
	ErrorDevices   int64 `json:"error_devices"`
}

// ToEntity converts a Device to a DeviceEntity for Datastore storage
func (d *Device) ToEntity() (*DeviceEntity, error) {
	parametersJSON, err := json.Marshal(d.Parameters)
	if err != nil {
		return nil, err
	}

	return &DeviceEntity{
		DeviceID:        d.DeviceID,
		BoardType:       d.BoardType,
		Status:          string(d.Status),
		TemplateID:      d.TemplateID,
		TemplateVersion: d.TemplateVersion,
		ParametersJSON:  string(parametersJSON),
		SecretsRef:      d.SecretsRef,
		FirmwareHash:    d.FirmwareHash,
		LastSeen:        d.LastSeen,
		OTAChannel:      d.OTAChannel,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
	}, nil
}

// FromEntity converts a DeviceEntity to a Device
func (de *DeviceEntity) FromEntity() (*Device, error) {
	var parameters map[string]interface{}
	if de.ParametersJSON != "" {
		if err := json.Unmarshal([]byte(de.ParametersJSON), &parameters); err != nil {
			return nil, err
		}
	}

	return &Device{
		DeviceID:        de.DeviceID,
		BoardType:       de.BoardType,
		Status:          DeviceStatus(de.Status),
		TemplateID:      de.TemplateID,
		TemplateVersion: de.TemplateVersion,
		Parameters:      parameters,
		SecretsRef:      de.SecretsRef,
		FirmwareHash:    de.FirmwareHash,
		LastSeen:        de.LastSeen,
		OTAChannel:      de.OTAChannel,
		CreatedAt:       de.CreatedAt,
		UpdatedAt:       de.UpdatedAt,
	}, nil
}

// IsOnline checks if the device is considered online based on last seen time
func (d *Device) IsOnline(timeout time.Duration) bool {
	return time.Since(d.LastSeen) <= timeout && d.Status == DeviceStatusOnline
}

// ToRegistrationRequest converts a Device to a DeviceRegistrationRequest
func (d *Device) ToRegistrationRequest() *DeviceRegistrationRequest {
	return &DeviceRegistrationRequest{
		DeviceID:        d.DeviceID,
		BoardType:       d.BoardType,
		TemplateID:      d.TemplateID,
		TemplateVersion: d.TemplateVersion,
		Parameters:      d.Parameters,
		SecretsRef:      d.SecretsRef,
		FirmwareHash:    d.FirmwareHash,
		OTAChannel:      d.OTAChannel,
	}
}

// FromRegistrationRequest creates a Device from a DeviceRegistrationRequest
func FromRegistrationRequest(req *DeviceRegistrationRequest) *Device {
	now := time.Now()
	
	otaChannel := req.OTAChannel
	if otaChannel == "" {
		otaChannel = "stable"
	}

	return &Device{
		DeviceID:        req.DeviceID,
		BoardType:       req.BoardType,
		Status:          DeviceStatusProvisioned,
		TemplateID:      req.TemplateID,
		TemplateVersion: req.TemplateVersion,
		Parameters:      req.Parameters,
		SecretsRef:      req.SecretsRef,
		FirmwareHash:    req.FirmwareHash,
		LastSeen:        now,
		OTAChannel:      otaChannel,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}