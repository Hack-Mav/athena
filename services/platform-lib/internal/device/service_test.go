package device

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupTestService() (*Service, *MockRepository) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		LogLevel:    "debug",
		ServiceName: "test-device-service",
	}
	logger := logger.New("debug", "test")
	mockRepo := new(MockRepository)

	// Create service without starting monitoring (to avoid goroutines in tests)
	service := &Service{
		config:     cfg,
		logger:     logger,
		repository: mockRepo,
	}

	return service, mockRepo
}

func TestService_HealthCheck(t *testing.T) {
	service, _ := setupTestService()

	router := gin.New()
	RegisterRoutes(router, service)

	req, _ := http.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "device-service", response["service"])
}

func TestService_RegisterDevice(t *testing.T) {
	service, mockRepo := setupTestService()

	router := gin.New()
	RegisterRoutes(router, service)

	req := &DeviceRegistrationRequest{
		DeviceID:        "test-device-001",
		BoardType:       "arduino-uno",
		TemplateID:      "sensor-template",
		TemplateVersion: "1.0.0",
		FirmwareHash:    "abc123def456",
	}

	mockRepo.On("RegisterDevice", mock.Anything, mock.AnythingOfType("*device.Device")).Return(nil)

	reqBody, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/api/v1/devices", bytes.NewBuffer(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response Device
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, req.DeviceID, response.DeviceID)
	assert.Equal(t, req.BoardType, response.BoardType)
	assert.Equal(t, DeviceStatusProvisioned, response.Status)

	mockRepo.AssertExpectations(t)
}

func TestService_RegisterDevice_InvalidRequest(t *testing.T) {
	service, _ := setupTestService()

	router := gin.New()
	RegisterRoutes(router, service)

	// Missing required fields
	req := map[string]interface{}{
		"device_id": "test-device-001",
		// Missing board_type, template_id, etc.
	}

	reqBody, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", "/api/v1/devices", bytes.NewBuffer(reqBody))
	httpReq.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestService_GetDevice(t *testing.T) {
	service, mockRepo := setupTestService()

	router := gin.New()
	RegisterRoutes(router, service)

	deviceID := "test-device-001"
	expectedDevice := createTestDevice(deviceID)

	mockRepo.On("GetDevice", mock.Anything, deviceID).Return(expectedDevice, nil)

	req, _ := http.NewRequest("GET", "/api/v1/devices/"+deviceID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response Device
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, expectedDevice.DeviceID, response.DeviceID)
	assert.Equal(t, expectedDevice.BoardType, response.BoardType)

	mockRepo.AssertExpectations(t)
}

func TestService_GetDevice_NotFound(t *testing.T) {
	service, mockRepo := setupTestService()

	router := gin.New()
	RegisterRoutes(router, service)

	deviceID := "non-existent-device"
	mockRepo.On("GetDevice", mock.Anything, deviceID).Return(nil, assert.AnError)

	req, _ := http.NewRequest("GET", "/api/v1/devices/"+deviceID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	mockRepo.AssertExpectations(t)
}

func TestService_ListDevices(t *testing.T) {
	service, mockRepo := setupTestService()

	router := gin.New()
	RegisterRoutes(router, service)

	expectedDevices := []*Device{
		createTestDevice("device-001"),
		createTestDevice("device-002"),
	}
	expectedCount := int64(2)

	mockRepo.On("ListDevices", mock.Anything, mock.AnythingOfType("*device.DeviceFilters")).Return(expectedDevices, nil)
	mockRepo.On("GetDeviceCount", mock.Anything, mock.AnythingOfType("*device.DeviceFilters")).Return(expectedCount, nil)

	req, _ := http.NewRequest("GET", "/api/v1/devices", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response DeviceListResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response.Devices, 2)
	assert.Equal(t, expectedCount, response.Total)
	assert.Equal(t, 50, response.Limit) // Default limit

	mockRepo.AssertExpectations(t)
}

func TestService_ListDevices_WithFilters(t *testing.T) {
	service, mockRepo := setupTestService()

	router := gin.New()
	RegisterRoutes(router, service)

	expectedDevices := []*Device{createTestDevice("online-device")}
	expectedCount := int64(1)

	mockRepo.On("ListDevices", mock.Anything, mock.MatchedBy(func(filters *DeviceFilters) bool {
		return filters.Status == DeviceStatusOnline && filters.Limit == 10
	})).Return(expectedDevices, nil)
	mockRepo.On("GetDeviceCount", mock.Anything, mock.AnythingOfType("*device.DeviceFilters")).Return(expectedCount, nil)

	req, _ := http.NewRequest("GET", "/api/v1/devices?status=online&limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response DeviceListResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Len(t, response.Devices, 1)
	assert.Equal(t, expectedCount, response.Total)
	assert.Equal(t, 10, response.Limit)

	mockRepo.AssertExpectations(t)
}

func TestService_UpdateDevice(t *testing.T) {
	service, mockRepo := setupTestService()

	router := gin.New()
	RegisterRoutes(router, service)

	deviceID := "test-device-001"
	updatedDevice := createTestDevice(deviceID)
	updatedDevice.Status = DeviceStatusOffline

	mockRepo.On("UpdateDevice", mock.Anything, mock.MatchedBy(func(device *Device) bool {
		return device.DeviceID == deviceID && device.Status == DeviceStatusOffline
	})).Return(nil)

	reqBody, _ := json.Marshal(updatedDevice)
	req, _ := http.NewRequest("PUT", "/api/v1/devices/"+deviceID, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mockRepo.AssertExpectations(t)
}

func TestService_DeleteDevice(t *testing.T) {
	service, mockRepo := setupTestService()

	router := gin.New()
	RegisterRoutes(router, service)

	deviceID := "test-device-001"
	mockRepo.On("DeleteDevice", mock.Anything, deviceID).Return(nil)

	req, _ := http.NewRequest("DELETE", "/api/v1/devices/"+deviceID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["message"], "deleted successfully")

	mockRepo.AssertExpectations(t)
}

func TestService_UpdateDeviceStatus(t *testing.T) {
	service, mockRepo := setupTestService()

	router := gin.New()
	RegisterRoutes(router, service)

	deviceID := "test-device-001"
	statusUpdate := DeviceStatusUpdate{
		Status: DeviceStatusOffline,
	}

	mockRepo.On("UpdateDeviceStatus", mock.Anything, deviceID, DeviceStatusOffline, mock.AnythingOfType("time.Time")).Return(nil)

	reqBody, _ := json.Marshal(statusUpdate)
	req, _ := http.NewRequest("PUT", "/api/v1/devices/"+deviceID+"/status", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mockRepo.AssertExpectations(t)
}

func TestService_DeviceHeartbeat(t *testing.T) {
	service, _ := setupTestService()

	// Mock monitoring service
	mockMonitoring := &MockMonitoringService{}
	service.monitoring = mockMonitoring

	router := gin.New()
	RegisterRoutes(router, service)

	deviceID := "test-device-001"
	heartbeat := DeviceHeartbeat{
		DeviceID:  deviceID,
		Timestamp: time.Now(),
		Status:    DeviceStatusOnline,
	}

	mockMonitoring.On("ProcessHeartbeat", mock.Anything, mock.MatchedBy(func(hb *DeviceHeartbeat) bool {
		return hb.DeviceID == deviceID
	})).Return(nil)

	reqBody, _ := json.Marshal(heartbeat)
	req, _ := http.NewRequest("POST", "/api/v1/devices/"+deviceID+"/heartbeat", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mockMonitoring.AssertExpectations(t)
}

func TestService_GetDeviceHealth(t *testing.T) {
	service, mockRepo := setupTestService()

	router := gin.New()
	RegisterRoutes(router, service)

	expectedHealth := &DeviceHealthStatus{
		TotalDevices:   10,
		OnlineDevices:  7,
		OfflineDevices: 2,
		ErrorDevices:   1,
	}

	mockRepo.On("GetDeviceHealthStatus", mock.Anything).Return(expectedHealth, nil)

	req, _ := http.NewRequest("GET", "/api/v1/devices/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response DeviceHealthStatus
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, expectedHealth.TotalDevices, response.TotalDevices)
	assert.Equal(t, expectedHealth.OnlineDevices, response.OnlineDevices)
	assert.Equal(t, expectedHealth.OfflineDevices, response.OfflineDevices)
	assert.Equal(t, expectedHealth.ErrorDevices, response.ErrorDevices)

	mockRepo.AssertExpectations(t)
}

func TestService_GetDevicesByStatus(t *testing.T) {
	service, mockRepo := setupTestService()

	router := gin.New()
	RegisterRoutes(router, service)

	status := DeviceStatusOnline
	expectedDevices := []*Device{createTestDevice("online-device")}

	mockRepo.On("GetDevicesByStatus", mock.Anything, status).Return(expectedDevices, nil)

	req, _ := http.NewRequest("GET", "/api/v1/devices/status/online", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "online", response["status"])
	assert.Equal(t, float64(1), response["count"])

	mockRepo.AssertExpectations(t)
}

func TestService_GetOfflineDevices(t *testing.T) {
	service, mockRepo := setupTestService()

	router := gin.New()
	RegisterRoutes(router, service)

	expectedDevices := []*Device{createTestDevice("offline-device")}
	mockRepo.On("GetOfflineDevices", mock.Anything, mock.AnythingOfType("time.Duration")).Return(expectedDevices, nil)

	req, _ := http.NewRequest("GET", "/api/v1/devices/offline?timeout=10m", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "10m", response["timeout"])
	assert.Equal(t, float64(1), response["count"])

	mockRepo.AssertExpectations(t)
}

func TestService_SearchDevices(t *testing.T) {
	service, mockRepo := setupTestService()

	router := gin.New()
	RegisterRoutes(router, service)

	query := "arduino"
	expectedDevices := []*Device{createTestDevice("arduino-device")}

	mockRepo.On("SearchDevices", mock.Anything, query, mock.AnythingOfType("*device.DeviceFilters")).Return(expectedDevices, nil)

	req, _ := http.NewRequest("GET", "/api/v1/devices/search?q="+query, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, query, response["query"])
	assert.Equal(t, float64(1), response["count"])

	mockRepo.AssertExpectations(t)
}

// MockMonitoringService for testing
type MockMonitoringService struct {
	mock.Mock
}

func (m *MockMonitoringService) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockMonitoringService) Stop() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockMonitoringService) ProcessHeartbeat(ctx context.Context, heartbeat *DeviceHeartbeat) error {
	args := m.Called(ctx, heartbeat)
	return args.Error(0)
}

func (m *MockMonitoringService) GetDeviceUptime(ctx context.Context, deviceID string) (time.Duration, error) {
	args := m.Called(ctx, deviceID)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *MockMonitoringService) GetDeviceLastSeenDuration(ctx context.Context, deviceID string) (time.Duration, error) {
	args := m.Called(ctx, deviceID)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *MockMonitoringService) CheckDeviceOnlineStatus(ctx context.Context, deviceID string) (bool, error) {
	args := m.Called(ctx, deviceID)
	return args.Bool(0), args.Error(1)
}

func (m *MockMonitoringService) GetConfiguration() *MonitoringConfig {
	args := m.Called()
	return args.Get(0).(*MonitoringConfig)
}

func (m *MockMonitoringService) IsRunning() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockMonitoringService) SetOfflineTimeout(timeout time.Duration) {
	m.Called(timeout)
}

func (m *MockMonitoringService) SetCheckInterval(interval time.Duration) {
	m.Called(interval)
}
