package device

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRepository is a mock implementation of the Repository interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) RegisterDevice(ctx context.Context, device *Device) error {
	args := m.Called(ctx, device)
	return args.Error(0)
}

func (m *MockRepository) GetDevice(ctx context.Context, deviceID string) (*Device, error) {
	args := m.Called(ctx, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Device), args.Error(1)
}

func (m *MockRepository) UpdateDevice(ctx context.Context, device *Device) error {
	args := m.Called(ctx, device)
	return args.Error(0)
}

func (m *MockRepository) DeleteDevice(ctx context.Context, deviceID string) error {
	args := m.Called(ctx, deviceID)
	return args.Error(0)
}

func (m *MockRepository) ListDevices(ctx context.Context, filters *DeviceFilters) ([]*Device, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Device), args.Error(1)
}

func (m *MockRepository) GetDeviceCount(ctx context.Context, filters *DeviceFilters) (int64, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepository) SearchDevices(ctx context.Context, query string, filters *DeviceFilters) ([]*Device, error) {
	args := m.Called(ctx, query, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Device), args.Error(1)
}

func (m *MockRepository) UpdateDeviceStatus(ctx context.Context, deviceID string, status DeviceStatus, lastSeen time.Time) error {
	args := m.Called(ctx, deviceID, status, lastSeen)
	return args.Error(0)
}

func (m *MockRepository) GetDevicesByStatus(ctx context.Context, status DeviceStatus) ([]*Device, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Device), args.Error(1)
}

func (m *MockRepository) GetOfflineDevices(ctx context.Context, timeout time.Duration) ([]*Device, error) {
	args := m.Called(ctx, timeout)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Device), args.Error(1)
}

func (m *MockRepository) GetDeviceHealthStatus(ctx context.Context) (*DeviceHealthStatus, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DeviceHealthStatus), args.Error(1)
}

func (m *MockRepository) GetDevicesByTemplate(ctx context.Context, templateID string) ([]*Device, error) {
	args := m.Called(ctx, templateID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Device), args.Error(1)
}

func (m *MockRepository) GetDevicesByOTAChannel(ctx context.Context, channel string) ([]*Device, error) {
	args := m.Called(ctx, channel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Device), args.Error(1)
}

func (m *MockRepository) DeviceExists(ctx context.Context, deviceID string) (bool, error) {
	args := m.Called(ctx, deviceID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) GetDevicesLastSeenBefore(ctx context.Context, before time.Time) ([]*Device, error) {
	args := m.Called(ctx, before)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Device), args.Error(1)
}

// Test helper functions
func createTestDevice(deviceID string) *Device {
	now := time.Now()
	return &Device{
		DeviceID:        deviceID,
		BoardType:       "arduino-uno",
		Status:          DeviceStatusOnline,
		TemplateID:      "sensor-template",
		TemplateVersion: "1.0.0",
		Parameters: map[string]interface{}{
			"sensor_pin": 2,
		},
		FirmwareHash: "abc123def456",
		LastSeen:     now,
		OTAChannel:   "stable",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func TestMockRepository_RegisterDevice(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	device := createTestDevice("test-device-001")

	mockRepo.On("RegisterDevice", ctx, device).Return(nil)

	err := mockRepo.RegisterDevice(ctx, device)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetDevice(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	deviceID := "test-device-001"
	expectedDevice := createTestDevice(deviceID)

	mockRepo.On("GetDevice", ctx, deviceID).Return(expectedDevice, nil)

	device, err := mockRepo.GetDevice(ctx, deviceID)
	require.NoError(t, err)
	assert.Equal(t, expectedDevice, device)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetDevice_NotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	deviceID := "non-existent-device"

	mockRepo.On("GetDevice", ctx, deviceID).Return(nil, assert.AnError)

	device, err := mockRepo.GetDevice(ctx, deviceID)
	assert.Error(t, err)
	assert.Nil(t, device)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_UpdateDevice(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	device := createTestDevice("test-device-001")
	device.Status = DeviceStatusOffline

	mockRepo.On("UpdateDevice", ctx, device).Return(nil)

	err := mockRepo.UpdateDevice(ctx, device)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_DeleteDevice(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	deviceID := "test-device-001"

	mockRepo.On("DeleteDevice", ctx, deviceID).Return(nil)

	err := mockRepo.DeleteDevice(ctx, deviceID)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_ListDevices(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	filters := &DeviceFilters{
		Status: DeviceStatusOnline,
		Limit:  10,
	}
	expectedDevices := []*Device{
		createTestDevice("device-001"),
		createTestDevice("device-002"),
	}

	mockRepo.On("ListDevices", ctx, filters).Return(expectedDevices, nil)

	devices, err := mockRepo.ListDevices(ctx, filters)
	require.NoError(t, err)
	assert.Equal(t, expectedDevices, devices)
	assert.Len(t, devices, 2)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetDeviceCount(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	filters := &DeviceFilters{Status: DeviceStatusOnline}
	expectedCount := int64(5)

	mockRepo.On("GetDeviceCount", ctx, filters).Return(expectedCount, nil)

	count, err := mockRepo.GetDeviceCount(ctx, filters)
	require.NoError(t, err)
	assert.Equal(t, expectedCount, count)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_SearchDevices(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	query := "arduino"
	filters := &DeviceFilters{Limit: 10}
	expectedDevices := []*Device{createTestDevice("arduino-device-001")}

	mockRepo.On("SearchDevices", ctx, query, filters).Return(expectedDevices, nil)

	devices, err := mockRepo.SearchDevices(ctx, query, filters)
	require.NoError(t, err)
	assert.Equal(t, expectedDevices, devices)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_UpdateDeviceStatus(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	deviceID := "test-device-001"
	status := DeviceStatusOffline
	lastSeen := time.Now()

	mockRepo.On("UpdateDeviceStatus", ctx, deviceID, status, lastSeen).Return(nil)

	err := mockRepo.UpdateDeviceStatus(ctx, deviceID, status, lastSeen)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetDevicesByStatus(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	status := DeviceStatusOnline
	expectedDevices := []*Device{createTestDevice("online-device-001")}

	mockRepo.On("GetDevicesByStatus", ctx, status).Return(expectedDevices, nil)

	devices, err := mockRepo.GetDevicesByStatus(ctx, status)
	require.NoError(t, err)
	assert.Equal(t, expectedDevices, devices)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetOfflineDevices(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	timeout := 5 * time.Minute
	expectedDevices := []*Device{createTestDevice("offline-device-001")}

	mockRepo.On("GetOfflineDevices", ctx, timeout).Return(expectedDevices, nil)

	devices, err := mockRepo.GetOfflineDevices(ctx, timeout)
	require.NoError(t, err)
	assert.Equal(t, expectedDevices, devices)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetDeviceHealthStatus(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	expectedHealth := &DeviceHealthStatus{
		TotalDevices:   10,
		OnlineDevices:  7,
		OfflineDevices: 2,
		ErrorDevices:   1,
	}

	mockRepo.On("GetDeviceHealthStatus", ctx).Return(expectedHealth, nil)

	health, err := mockRepo.GetDeviceHealthStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, expectedHealth, health)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetDevicesByTemplate(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	templateID := "sensor-template"
	expectedDevices := []*Device{createTestDevice("template-device-001")}

	mockRepo.On("GetDevicesByTemplate", ctx, templateID).Return(expectedDevices, nil)

	devices, err := mockRepo.GetDevicesByTemplate(ctx, templateID)
	require.NoError(t, err)
	assert.Equal(t, expectedDevices, devices)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetDevicesByOTAChannel(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	channel := "beta"
	expectedDevices := []*Device{createTestDevice("beta-device-001")}

	mockRepo.On("GetDevicesByOTAChannel", ctx, channel).Return(expectedDevices, nil)

	devices, err := mockRepo.GetDevicesByOTAChannel(ctx, channel)
	require.NoError(t, err)
	assert.Equal(t, expectedDevices, devices)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_DeviceExists(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	deviceID := "test-device-001"

	mockRepo.On("DeviceExists", ctx, deviceID).Return(true, nil)

	exists, err := mockRepo.DeviceExists(ctx, deviceID)
	require.NoError(t, err)
	assert.True(t, exists)
	mockRepo.AssertExpectations(t)
}

func TestMockRepository_GetDevicesLastSeenBefore(t *testing.T) {
	mockRepo := new(MockRepository)
	ctx := context.Background()
	before := time.Now().Add(-1 * time.Hour)
	expectedDevices := []*Device{createTestDevice("stale-device-001")}

	mockRepo.On("GetDevicesLastSeenBefore", ctx, before).Return(expectedDevices, nil)

	devices, err := mockRepo.GetDevicesLastSeenBefore(ctx, before)
	require.NoError(t, err)
	assert.Equal(t, expectedDevices, devices)
	mockRepo.AssertExpectations(t)
}