package device

import (
	"context"
	"testing"
	"time"

	"github.com/athena/platform-lib/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDefaultMonitoringConfig(t *testing.T) {
	config := DefaultMonitoringConfig()
	require.NotNil(t, config)

	assert.Equal(t, 5*time.Minute, config.OfflineTimeout)
	assert.Equal(t, 1*time.Minute, config.CheckInterval)
}

func TestNewMonitoringService(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := logger.New("debug", "test")
	config := DefaultMonitoringConfig()

	service := NewMonitoringService(mockRepo, logger, config)
	require.NotNil(t, service)

	assert.Equal(t, mockRepo, service.repository)
	assert.Equal(t, logger, service.logger)
	assert.Equal(t, config.OfflineTimeout, service.offlineTimeout)
	assert.Equal(t, config.CheckInterval, service.checkInterval)
	assert.False(t, service.IsRunning())
}

func TestNewMonitoringService_NilConfig(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := logger.New("debug", "test")

	service := NewMonitoringService(mockRepo, logger, nil)
	require.NotNil(t, service)

	// Should use default config
	assert.Equal(t, 5*time.Minute, service.offlineTimeout)
	assert.Equal(t, 1*time.Minute, service.checkInterval)
}

func TestMonitoringService_StartStop(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := logger.New("debug", "test")
	config := &MonitoringConfig{
		OfflineTimeout: 1 * time.Second,
		CheckInterval:  100 * time.Millisecond,
	}

	service := NewMonitoringService(mockRepo, logger, config)
	ctx := context.Background()

	// Mock repository calls for the monitoring loop
	mockRepo.On("GetDevicesLastSeenBefore", mock.Anything, mock.Anything).Return([]*Device{}, nil).Maybe()
	mockRepo.On("GetDeviceHealthStatus", mock.Anything).Return(&DeviceHealthStatus{}, nil).Maybe()

	// Test start
	err := service.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, service.IsRunning())

	// Test start when already running
	err = service.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Test stop
	err = service.Stop()
	assert.NoError(t, err)
	assert.False(t, service.IsRunning())

	// Test stop when not running
	err = service.Stop()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")

	mockRepo.AssertExpectations(t)
}

func TestMonitoringService_ProcessHeartbeat(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := logger.New("debug", "test")
	service := NewMonitoringService(mockRepo, logger, nil)
	ctx := context.Background()

	tests := []struct {
		name      string
		heartbeat *DeviceHeartbeat
		expectErr bool
		setupMock func()
	}{
		{
			name: "Valid heartbeat",
			heartbeat: &DeviceHeartbeat{
				DeviceID:  "test-device-001",
				Timestamp: time.Now(),
				Status:    DeviceStatusOnline,
			},
			expectErr: false,
			setupMock: func() {
				mockRepo.On("UpdateDeviceStatus", ctx, "test-device-001", DeviceStatusOnline, mock.AnythingOfType("time.Time")).Return(nil).Once()
			},
		},
		{
			name:      "Nil heartbeat",
			heartbeat: nil,
			expectErr: true,
			setupMock: func() {},
		},
		{
			name: "Empty device ID",
			heartbeat: &DeviceHeartbeat{
				DeviceID: "",
			},
			expectErr: true,
			setupMock: func() {},
		},
		{
			name: "Zero timestamp gets current time",
			heartbeat: &DeviceHeartbeat{
				DeviceID: "test-device-002",
				// Timestamp is zero value
			},
			expectErr: false,
			setupMock: func() {
				mockRepo.On("UpdateDeviceStatus", ctx, "test-device-002", DeviceStatusOnline, mock.AnythingOfType("time.Time")).Return(nil).Once()
			},
		},
		{
			name: "Empty status defaults to online",
			heartbeat: &DeviceHeartbeat{
				DeviceID:  "test-device-003",
				Timestamp: time.Now(),
				// Status is empty
			},
			expectErr: false,
			setupMock: func() {
				mockRepo.On("UpdateDeviceStatus", ctx, "test-device-003", DeviceStatusOnline, mock.AnythingOfType("time.Time")).Return(nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			err := service.ProcessHeartbeat(ctx, tt.heartbeat)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	mockRepo.AssertExpectations(t)
}

func TestMonitoringService_GetOfflineDevices(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := logger.New("debug", "test")
	service := NewMonitoringService(mockRepo, logger, nil)
	ctx := context.Background()

	expectedDevices := []*Device{createTestDevice("offline-device")}
	mockRepo.On("GetOfflineDevices", ctx, service.offlineTimeout).Return(expectedDevices, nil)

	devices, err := service.GetOfflineDevices(ctx)
	require.NoError(t, err)
	assert.Equal(t, expectedDevices, devices)
	mockRepo.AssertExpectations(t)
}

func TestMonitoringService_GetDeviceHealthStatus(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := logger.New("debug", "test")
	service := NewMonitoringService(mockRepo, logger, nil)
	ctx := context.Background()

	expectedHealth := &DeviceHealthStatus{
		TotalDevices:   10,
		OnlineDevices:  7,
		OfflineDevices: 2,
		ErrorDevices:   1,
	}
	mockRepo.On("GetDeviceHealthStatus", ctx).Return(expectedHealth, nil)

	health, err := service.GetDeviceHealthStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, expectedHealth, health)
	mockRepo.AssertExpectations(t)
}

func TestMonitoringService_CheckDeviceOnlineStatus(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := logger.New("debug", "test")
	config := &MonitoringConfig{
		OfflineTimeout: 5 * time.Minute,
		CheckInterval:  1 * time.Minute,
	}
	service := NewMonitoringService(mockRepo, logger, config)
	ctx := context.Background()

	tests := []struct {
		name           string
		device         *Device
		expectedOnline bool
	}{
		{
			name: "Online device",
			device: &Device{
				DeviceID: "online-device",
				Status:   DeviceStatusOnline,
				LastSeen: time.Now().Add(-2 * time.Minute), // Within timeout
			},
			expectedOnline: true,
		},
		{
			name: "Offline device - old last seen",
			device: &Device{
				DeviceID: "offline-device",
				Status:   DeviceStatusOnline,
				LastSeen: time.Now().Add(-10 * time.Minute), // Beyond timeout
			},
			expectedOnline: false,
		},
		{
			name: "Error device",
			device: &Device{
				DeviceID: "error-device",
				Status:   DeviceStatusError,
				LastSeen: time.Now().Add(-1 * time.Minute), // Recent but error status
			},
			expectedOnline: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.On("GetDevice", ctx, tt.device.DeviceID).Return(tt.device, nil).Once()

			isOnline, err := service.CheckDeviceOnlineStatus(ctx, tt.device.DeviceID)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedOnline, isOnline)
		})
	}

	mockRepo.AssertExpectations(t)
}

func TestMonitoringService_GetDeviceUptime(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := logger.New("debug", "test")
	service := NewMonitoringService(mockRepo, logger, nil)
	ctx := context.Background()

	createdAt := time.Now().Add(-2 * time.Hour)
	device := &Device{
		DeviceID:  "test-device",
		CreatedAt: createdAt,
	}

	mockRepo.On("GetDevice", ctx, "test-device").Return(device, nil)

	uptime, err := service.GetDeviceUptime(ctx, "test-device")
	require.NoError(t, err)

	// Uptime should be approximately 2 hours (allowing for small timing differences)
	assert.True(t, uptime > 1*time.Hour+50*time.Minute)
	assert.True(t, uptime < 2*time.Hour+10*time.Minute)

	mockRepo.AssertExpectations(t)
}

func TestMonitoringService_GetDeviceLastSeenDuration(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := logger.New("debug", "test")
	service := NewMonitoringService(mockRepo, logger, nil)
	ctx := context.Background()

	lastSeen := time.Now().Add(-30 * time.Minute)
	device := &Device{
		DeviceID: "test-device",
		LastSeen: lastSeen,
	}

	mockRepo.On("GetDevice", ctx, "test-device").Return(device, nil)

	duration, err := service.GetDeviceLastSeenDuration(ctx, "test-device")
	require.NoError(t, err)

	// Duration should be approximately 30 minutes (allowing for small timing differences)
	assert.True(t, duration > 29*time.Minute)
	assert.True(t, duration < 31*time.Minute)

	mockRepo.AssertExpectations(t)
}

func TestMonitoringService_SetOfflineTimeout(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := logger.New("debug", "test")
	service := NewMonitoringService(mockRepo, logger, nil)

	newTimeout := 10 * time.Minute
	service.SetOfflineTimeout(newTimeout)

	config := service.GetConfiguration()
	assert.Equal(t, newTimeout, config.OfflineTimeout)
}

func TestMonitoringService_SetCheckInterval(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := logger.New("debug", "test")
	service := NewMonitoringService(mockRepo, logger, nil)

	newInterval := 2 * time.Minute
	service.SetCheckInterval(newInterval)

	config := service.GetConfiguration()
	assert.Equal(t, newInterval, config.CheckInterval)
}

func TestMonitoringService_GetConfiguration(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := logger.New("debug", "test")
	originalConfig := &MonitoringConfig{
		OfflineTimeout: 7 * time.Minute,
		CheckInterval:  30 * time.Second,
	}
	service := NewMonitoringService(mockRepo, logger, originalConfig)

	config := service.GetConfiguration()
	require.NotNil(t, config)
	assert.Equal(t, originalConfig.OfflineTimeout, config.OfflineTimeout)
	assert.Equal(t, originalConfig.CheckInterval, config.CheckInterval)
}

func TestMonitoringService_DeviceNotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	logger := logger.New("debug", "test")
	service := NewMonitoringService(mockRepo, logger, nil)
	ctx := context.Background()

	deviceID := "non-existent-device"
	mockRepo.On("GetDevice", ctx, deviceID).Return(nil, assert.AnError)

	// Test CheckDeviceOnlineStatus
	isOnline, err := service.CheckDeviceOnlineStatus(ctx, deviceID)
	assert.Error(t, err)
	assert.False(t, isOnline)

	// Test GetDeviceUptime
	uptime, err := service.GetDeviceUptime(ctx, deviceID)
	assert.Error(t, err)
	assert.Equal(t, time.Duration(0), uptime)

	// Test GetDeviceLastSeenDuration
	duration, err := service.GetDeviceLastSeenDuration(ctx, deviceID)
	assert.Error(t, err)
	assert.Equal(t, time.Duration(0), duration)

	mockRepo.AssertExpectations(t)
}
