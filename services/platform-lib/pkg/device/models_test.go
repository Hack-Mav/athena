package device

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceStatus(t *testing.T) {
	tests := []struct {
		name   string
		status DeviceStatus
		valid  bool
	}{
		{"Provisioned", DeviceStatusProvisioned, true},
		{"Online", DeviceStatusOnline, true},
		{"Offline", DeviceStatusOffline, true},
		{"Error", DeviceStatusError, true},
		{"Invalid", DeviceStatus("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test string conversion
			statusStr := string(tt.status)
			assert.NotEmpty(t, statusStr)

			// Test conversion back
			converted := DeviceStatus(statusStr)
			assert.Equal(t, tt.status, converted)
		})
	}
}

func TestDevice_ToEntity(t *testing.T) {
	now := time.Now()
	device := &Device{
		DeviceID:        "test-device-001",
		BoardType:       "arduino-uno",
		Status:          DeviceStatusOnline,
		TemplateID:      "sensor-template",
		TemplateVersion: "1.0.0",
		Parameters: map[string]interface{}{
			"sensor_pin": 2,
			"led_pin":    13,
		},
		SecretsRef:   "secret-ref-123",
		FirmwareHash: "abc123def456",
		LastSeen:     now,
		OTAChannel:   "stable",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	entity, err := device.ToEntity()
	require.NoError(t, err)
	require.NotNil(t, entity)

	assert.Equal(t, device.DeviceID, entity.DeviceID)
	assert.Equal(t, device.BoardType, entity.BoardType)
	assert.Equal(t, string(device.Status), entity.Status)
	assert.Equal(t, device.TemplateID, entity.TemplateID)
	assert.Equal(t, device.TemplateVersion, entity.TemplateVersion)
	assert.Equal(t, device.SecretsRef, entity.SecretsRef)
	assert.Equal(t, device.FirmwareHash, entity.FirmwareHash)
	assert.Equal(t, device.LastSeen, entity.LastSeen)
	assert.Equal(t, device.OTAChannel, entity.OTAChannel)
	assert.Equal(t, device.CreatedAt, entity.CreatedAt)
	assert.Equal(t, device.UpdatedAt, entity.UpdatedAt)

	// Verify parameters JSON
	var params map[string]interface{}
	err = json.Unmarshal([]byte(entity.ParametersJSON), &params)
	require.NoError(t, err)
	// JSON unmarshaling converts numbers to float64, so we need to check the values differently
	assert.Equal(t, float64(2), params["sensor_pin"])
	assert.Equal(t, float64(13), params["led_pin"])
}

func TestDeviceEntity_FromEntity(t *testing.T) {
	now := time.Now()
	parameters := map[string]interface{}{
		"sensor_pin": 2,
		"led_pin":    13,
	}
	parametersJSON, _ := json.Marshal(parameters)

	entity := &DeviceEntity{
		DeviceID:        "test-device-001",
		BoardType:       "arduino-uno",
		Status:          string(DeviceStatusOnline),
		TemplateID:      "sensor-template",
		TemplateVersion: "1.0.0",
		ParametersJSON:  string(parametersJSON),
		SecretsRef:      "secret-ref-123",
		FirmwareHash:    "abc123def456",
		LastSeen:        now,
		OTAChannel:      "stable",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	device, err := entity.FromEntity()
	require.NoError(t, err)
	require.NotNil(t, device)

	assert.Equal(t, entity.DeviceID, device.DeviceID)
	assert.Equal(t, entity.BoardType, device.BoardType)
	assert.Equal(t, DeviceStatus(entity.Status), device.Status)
	assert.Equal(t, entity.TemplateID, device.TemplateID)
	assert.Equal(t, entity.TemplateVersion, device.TemplateVersion)
	assert.Equal(t, entity.SecretsRef, device.SecretsRef)
	assert.Equal(t, entity.FirmwareHash, device.FirmwareHash)
	assert.Equal(t, entity.LastSeen, device.LastSeen)
	assert.Equal(t, entity.OTAChannel, device.OTAChannel)
	assert.Equal(t, entity.CreatedAt, device.CreatedAt)
	assert.Equal(t, entity.UpdatedAt, device.UpdatedAt)
	// JSON unmarshaling converts numbers to float64, so we need to check the values differently
	assert.Equal(t, float64(2), device.Parameters["sensor_pin"])
	assert.Equal(t, float64(13), device.Parameters["led_pin"])
}

func TestDevice_IsOnline(t *testing.T) {
	tests := []struct {
		name           string
		lastSeen       time.Time
		status         DeviceStatus
		timeout        time.Duration
		expectedOnline bool
	}{
		{
			name:           "Recently seen online device",
			lastSeen:       time.Now().Add(-1 * time.Minute),
			status:         DeviceStatusOnline,
			timeout:        5 * time.Minute,
			expectedOnline: true,
		},
		{
			name:           "Old last seen online device",
			lastSeen:       time.Now().Add(-10 * time.Minute),
			status:         DeviceStatusOnline,
			timeout:        5 * time.Minute,
			expectedOnline: false,
		},
		{
			name:           "Recently seen error device",
			lastSeen:       time.Now().Add(-1 * time.Minute),
			status:         DeviceStatusError,
			timeout:        5 * time.Minute,
			expectedOnline: false,
		},
		{
			name:           "Recently seen offline device",
			lastSeen:       time.Now().Add(-1 * time.Minute),
			status:         DeviceStatusOffline,
			timeout:        5 * time.Minute,
			expectedOnline: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := &Device{
				LastSeen: tt.lastSeen,
				Status:   tt.status,
			}

			isOnline := device.IsOnline(tt.timeout)
			assert.Equal(t, tt.expectedOnline, isOnline)
		})
	}
}

func TestFromRegistrationRequest(t *testing.T) {
	req := &DeviceRegistrationRequest{
		DeviceID:        "test-device-001",
		BoardType:       "arduino-uno",
		TemplateID:      "sensor-template",
		TemplateVersion: "1.0.0",
		Parameters: map[string]interface{}{
			"sensor_pin": 2,
		},
		SecretsRef:   "secret-ref-123",
		FirmwareHash: "abc123def456",
		OTAChannel:   "beta",
	}

	device := FromRegistrationRequest(req)
	require.NotNil(t, device)

	assert.Equal(t, req.DeviceID, device.DeviceID)
	assert.Equal(t, req.BoardType, device.BoardType)
	assert.Equal(t, DeviceStatusProvisioned, device.Status)
	assert.Equal(t, req.TemplateID, device.TemplateID)
	assert.Equal(t, req.TemplateVersion, device.TemplateVersion)
	assert.Equal(t, req.Parameters, device.Parameters)
	assert.Equal(t, req.SecretsRef, device.SecretsRef)
	assert.Equal(t, req.FirmwareHash, device.FirmwareHash)
	assert.Equal(t, req.OTAChannel, device.OTAChannel)
	assert.False(t, device.LastSeen.IsZero())
	assert.False(t, device.CreatedAt.IsZero())
	assert.False(t, device.UpdatedAt.IsZero())
}

func TestFromRegistrationRequest_DefaultOTAChannel(t *testing.T) {
	req := &DeviceRegistrationRequest{
		DeviceID:        "test-device-001",
		BoardType:       "arduino-uno",
		TemplateID:      "sensor-template",
		TemplateVersion: "1.0.0",
		FirmwareHash:    "abc123def456",
		// OTAChannel not specified
	}

	device := FromRegistrationRequest(req)
	require.NotNil(t, device)

	assert.Equal(t, "stable", device.OTAChannel)
}

func TestDevice_ToRegistrationRequest(t *testing.T) {
	device := &Device{
		DeviceID:        "test-device-001",
		BoardType:       "arduino-uno",
		TemplateID:      "sensor-template",
		TemplateVersion: "1.0.0",
		Parameters: map[string]interface{}{
			"sensor_pin": 2,
		},
		SecretsRef:   "secret-ref-123",
		FirmwareHash: "abc123def456",
		OTAChannel:   "beta",
	}

	req := device.ToRegistrationRequest()
	require.NotNil(t, req)

	assert.Equal(t, device.DeviceID, req.DeviceID)
	assert.Equal(t, device.BoardType, req.BoardType)
	assert.Equal(t, device.TemplateID, req.TemplateID)
	assert.Equal(t, device.TemplateVersion, req.TemplateVersion)
	assert.Equal(t, device.Parameters, req.Parameters)
	assert.Equal(t, device.SecretsRef, req.SecretsRef)
	assert.Equal(t, device.FirmwareHash, req.FirmwareHash)
	assert.Equal(t, device.OTAChannel, req.OTAChannel)
}

func TestDeviceEntity_FromEntity_InvalidJSON(t *testing.T) {
	entity := &DeviceEntity{
		DeviceID:       "test-device-001",
		ParametersJSON: "invalid-json",
	}

	device, err := entity.FromEntity()
	assert.Error(t, err)
	assert.Nil(t, device)
}

func TestDevice_ToEntity_InvalidParameters(t *testing.T) {
	device := &Device{
		DeviceID: "test-device-001",
		Parameters: map[string]interface{}{
			"invalid": make(chan int), // channels can't be marshaled to JSON
		},
	}

	entity, err := device.ToEntity()
	assert.Error(t, err)
	assert.Nil(t, entity)
}
