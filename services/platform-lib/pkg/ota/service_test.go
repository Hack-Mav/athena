package ota

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/athena/platform-lib/internal/device"
	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupTestService() (*Service, *MockRepository, *MockDeviceRepository, *MockStorageBackend) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		LogLevel:    "debug",
		ServiceName: "test-ota-service",
	}
	logger := logger.New("debug", "test")
	mockRepo := new(MockRepository)
	mockDeviceRepo := new(MockDeviceRepository)
	mockStorage := new(MockStorageBackend)

	// Generate test keys for signing
	privateKeyPEM, publicKeyPEM, _ := GenerateKeyPair(2048)
	signer, _ := NewSigner(privateKeyPEM, publicKeyPEM)

	service := &Service{
		config:           cfg,
		logger:           logger,
		repository:       mockRepo,
		deviceRepository: mockDeviceRepo,
		signer:           signer,
		storageBackend:   mockStorage,
	}

	return service, mockRepo, mockDeviceRepo, mockStorage
}

func TestService_CreateRelease(t *testing.T) {
	service, mockRepo, _, mockStorage := setupTestService()

	binaryData := []byte("test firmware binary data")
	req := &CreateReleaseRequest{
		TemplateID:   "template-001",
		Version:      "1.0.0",
		Channel:      ReleaseChannelStable,
		BinaryData:   binaryData,
		ReleaseNotes: "Initial release",
		CreatedBy:    "admin",
	}

	mockStorage.On("StoreBinary", mock.Anything, mock.AnythingOfType("string"), binaryData).Return("/binaries/release-123.bin", nil)
	mockRepo.On("CreateRelease", mock.Anything, mock.MatchedBy(func(release *FirmwareRelease) bool {
		return release.TemplateID == req.TemplateID &&
			release.Version == req.Version &&
			release.Channel == req.Channel &&
			release.BinarySize == int64(len(binaryData))
	})).Return(nil)

	release, err := service.CreateRelease(context.Background(), req)

	require.NoError(t, err)
	require.NotNil(t, release)
	assert.Equal(t, req.TemplateID, release.TemplateID)
	assert.Equal(t, req.Version, release.Version)
	assert.Equal(t, req.Channel, release.Channel)
	assert.NotEmpty(t, release.ReleaseID)
	assert.NotEmpty(t, release.BinaryHash)
	assert.NotEmpty(t, release.Signature)
	assert.Equal(t, int64(len(binaryData)), release.BinarySize)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestService_CreateRelease_InvalidChannel(t *testing.T) {
	service, _, _, _ := setupTestService()

	req := &CreateReleaseRequest{
		TemplateID: "template-001",
		Version:    "1.0.0",
		Channel:    "invalid-channel",
		BinaryData: []byte("test data"),
	}

	release, err := service.CreateRelease(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, release)
	assert.Contains(t, err.Error(), "invalid release channel")
}

func TestService_CreateRelease_MissingFields(t *testing.T) {
	service, _, _, _ := setupTestService()

	tests := []struct {
		name string
		req  *CreateReleaseRequest
	}{
		{
			name: "missing template ID",
			req: &CreateReleaseRequest{
				Version:    "1.0.0",
				Channel:    ReleaseChannelStable,
				BinaryData: []byte("test"),
			},
		},
		{
			name: "missing version",
			req: &CreateReleaseRequest{
				TemplateID: "template-001",
				Channel:    ReleaseChannelStable,
				BinaryData: []byte("test"),
			},
		},
		{
			name: "missing binary data",
			req: &CreateReleaseRequest{
				TemplateID: "template-001",
				Version:    "1.0.0",
				Channel:    ReleaseChannelStable,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			release, err := service.CreateRelease(context.Background(), tt.req)
			assert.Error(t, err)
			assert.Nil(t, release)
		})
	}
}

func TestService_GetRelease(t *testing.T) {
	service, mockRepo, _, _ := setupTestService()

	expectedRelease := createTestRelease("release-001")
	mockRepo.On("GetRelease", mock.Anything, "release-001").Return(expectedRelease, nil)

	release, err := service.GetRelease(context.Background(), "release-001")

	require.NoError(t, err)
	require.NotNil(t, release)
	assert.Equal(t, expectedRelease.ReleaseID, release.ReleaseID)
	assert.Equal(t, expectedRelease.Version, release.Version)

	mockRepo.AssertExpectations(t)
}

func TestService_ListReleases(t *testing.T) {
	service, mockRepo, _, _ := setupTestService()

	expectedReleases := []*FirmwareRelease{
		createTestRelease("release-001"),
		createTestRelease("release-002"),
	}

	mockRepo.On("ListReleases", mock.Anything, "template-001", ReleaseChannelStable).Return(expectedReleases, nil)

	releases, err := service.ListReleases(context.Background(), "template-001", ReleaseChannelStable)

	require.NoError(t, err)
	assert.Len(t, releases, 2)

	mockRepo.AssertExpectations(t)
}

func TestService_DeleteRelease(t *testing.T) {
	service, mockRepo, _, mockStorage := setupTestService()

	release := createTestRelease("release-001")
	mockRepo.On("GetRelease", mock.Anything, "release-001").Return(release, nil)
	mockStorage.On("DeleteBinary", mock.Anything, release.BinaryPath).Return(nil)
	mockRepo.On("DeleteRelease", mock.Anything, "release-001").Return(nil)

	err := service.DeleteRelease(context.Background(), "release-001")

	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestService_VerifyRelease(t *testing.T) {
	service, mockRepo, _, mockStorage := setupTestService()

	binaryData := []byte("test firmware binary data")
	release := createTestRelease("release-001")
	release.BinaryHash = ComputeHash(binaryData)

	mockRepo.On("GetRelease", mock.Anything, "release-001").Return(release, nil)
	mockStorage.On("GetBinary", mock.Anything, release.BinaryPath).Return(binaryData, nil)

	err := service.VerifyRelease(context.Background(), "release-001")

	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestService_VerifyRelease_HashMismatch(t *testing.T) {
	service, mockRepo, _, mockStorage := setupTestService()

	binaryData := []byte("test firmware binary data")
	release := createTestRelease("release-001")
	release.BinaryHash = "incorrect-hash"

	mockRepo.On("GetRelease", mock.Anything, "release-001").Return(release, nil)
	mockStorage.On("GetBinary", mock.Anything, release.BinaryPath).Return(binaryData, nil)

	err := service.VerifyRelease(context.Background(), "release-001")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hash mismatch")

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestService_CreateReleaseHandler(t *testing.T) {
	service, mockRepo, _, mockStorage := setupTestService()

	router := gin.New()
	RegisterRoutes(router, service)

	binaryData := []byte("test firmware binary")
	mockStorage.On("StoreBinary", mock.Anything, mock.AnythingOfType("string"), binaryData).Return("/binaries/test.bin", nil)
	mockRepo.On("CreateRelease", mock.Anything, mock.AnythingOfType("*ota.FirmwareRelease")).Return(nil)

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("template_id", "template-001")
	writer.WriteField("version", "1.0.0")
	writer.WriteField("channel", "stable")
	writer.WriteField("release_notes", "Test release")
	writer.WriteField("created_by", "admin")

	part, _ := writer.CreateFormFile("binary", "firmware.bin")
	part.Write(binaryData)
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/v1/ota/releases", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response FirmwareRelease
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "template-001", response.TemplateID)
	assert.Equal(t, "1.0.0", response.Version)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestService_GetReleaseHandler(t *testing.T) {
	service, mockRepo, _, _ := setupTestService()

	router := gin.New()
	RegisterRoutes(router, service)

	expectedRelease := createTestRelease("release-001")
	mockRepo.On("GetRelease", mock.Anything, "release-001").Return(expectedRelease, nil)

	req, _ := http.NewRequest("GET", "/api/v1/ota/releases/release-001", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response FirmwareRelease
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, expectedRelease.ReleaseID, response.ReleaseID)

	mockRepo.AssertExpectations(t)
}

func TestService_GetUpdateForDeviceHandler(t *testing.T) {
	service, mockRepo, _, mockStorage := setupTestService()

	router := gin.New()
	RegisterRoutes(router, service)

	deviceUpdate := &DeviceUpdate{
		DeviceID:     "device-001",
		ReleaseID:    "release-001",
		DeploymentID: "deployment-001",
		Status:       UpdateStatusPending,
		StartedAt:    time.Now(),
	}

	release := createTestRelease("release-001")

	mockRepo.On("GetLatestUpdateForDevice", mock.Anything, "device-001").Return(deviceUpdate, nil)
	mockRepo.On("GetRelease", mock.Anything, "release-001").Return(release, nil)
	mockStorage.On("GetBinaryURL", mock.Anything, release.BinaryPath, mock.AnythingOfType("time.Duration")).Return("https://storage.example.com/firmware.bin", nil)

	req, _ := http.NewRequest("GET", "/api/v1/ota/updates/device-001", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response FirmwareUpdate
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, release.ReleaseID, response.ReleaseID)
	assert.Equal(t, release.Version, response.Version)
	assert.NotEmpty(t, response.BinaryURL)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestService_ReportUpdateStatusHandler(t *testing.T) {
	service, mockRepo, _, _ := setupTestService()

	router := gin.New()
	RegisterRoutes(router, service)

	deviceUpdate := &DeviceUpdate{
		DeviceID:     "device-001",
		ReleaseID:    "release-001",
		DeploymentID: "deployment-001",
		Status:       UpdateStatusPending,
		StartedAt:    time.Now(),
	}

	deployment := &OTADeployment{
		DeploymentID:     "deployment-001",
		ReleaseID:        "release-001",
		Status:           DeploymentStatusActive,
		FailureThreshold: 10,
	}

	mockRepo.On("GetDeviceUpdate", mock.Anything, "device-001", "release-001").Return(deviceUpdate, nil)
	mockRepo.On("UpdateDeviceUpdate", mock.Anything, mock.AnythingOfType("*ota.DeviceUpdate")).Return(nil)
	mockRepo.On("GetDeployment", mock.Anything, "deployment-001").Return(deployment, nil)
	mockRepo.On("GetDeploymentStats", mock.Anything, "deployment-001").Return(1, 0, 0, nil)
	mockRepo.On("UpdateDeployment", mock.Anything, mock.AnythingOfType("*ota.OTADeployment")).Return(nil)

	report := UpdateStatusReport{
		DeviceID:  "device-001",
		ReleaseID: "release-001",
		Status:    UpdateStatusCompleted,
		Progress:  100,
	}

	reqBody, _ := json.Marshal(report)
	req, _ := http.NewRequest("POST", "/api/v1/ota/updates/status", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mockRepo.AssertExpectations(t)
}

// Helper functions

func createTestRelease(releaseID string) *FirmwareRelease {
	return &FirmwareRelease{
		ReleaseID:    releaseID,
		TemplateID:   "template-001",
		Version:      "1.0.0",
		Channel:      ReleaseChannelStable,
		BinaryHash:   "abc123def456",
		BinaryPath:   "/binaries/" + releaseID + ".bin",
		BinarySize:   1024,
		Signature:    "signature-data",
		ReleaseNotes: "Test release",
		CreatedAt:    time.Now(),
		CreatedBy:    "admin",
	}
}

// Mock implementations

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateRelease(ctx context.Context, release *FirmwareRelease) error {
	args := m.Called(ctx, release)
	return args.Error(0)
}

func (m *MockRepository) GetRelease(ctx context.Context, releaseID string) (*FirmwareRelease, error) {
	args := m.Called(ctx, releaseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FirmwareRelease), args.Error(1)
}

func (m *MockRepository) ListReleases(ctx context.Context, templateID string, channel ReleaseChannel) ([]*FirmwareRelease, error) {
	args := m.Called(ctx, templateID, channel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*FirmwareRelease), args.Error(1)
}

func (m *MockRepository) DeleteRelease(ctx context.Context, releaseID string) error {
	args := m.Called(ctx, releaseID)
	return args.Error(0)
}

func (m *MockRepository) CreateDeployment(ctx context.Context, deployment *OTADeployment) error {
	args := m.Called(ctx, deployment)
	return args.Error(0)
}

func (m *MockRepository) GetDeployment(ctx context.Context, deploymentID string) (*OTADeployment, error) {
	args := m.Called(ctx, deploymentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*OTADeployment), args.Error(1)
}

func (m *MockRepository) UpdateDeployment(ctx context.Context, deployment *OTADeployment) error {
	args := m.Called(ctx, deployment)
	return args.Error(0)
}

func (m *MockRepository) CreateDeviceUpdate(ctx context.Context, update *DeviceUpdate) error {
	args := m.Called(ctx, update)
	return args.Error(0)
}

func (m *MockRepository) GetDeviceUpdate(ctx context.Context, deviceID, releaseID string) (*DeviceUpdate, error) {
	args := m.Called(ctx, deviceID, releaseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DeviceUpdate), args.Error(1)
}

func (m *MockRepository) GetLatestUpdateForDevice(ctx context.Context, deviceID string) (*DeviceUpdate, error) {
	args := m.Called(ctx, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DeviceUpdate), args.Error(1)
}

func (m *MockRepository) UpdateDeviceUpdate(ctx context.Context, update *DeviceUpdate) error {
	args := m.Called(ctx, update)
	return args.Error(0)
}

func (m *MockRepository) ListDeviceUpdates(ctx context.Context, deploymentID string) ([]*DeviceUpdate, error) {
	args := m.Called(ctx, deploymentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DeviceUpdate), args.Error(1)
}

func (m *MockRepository) GetDeploymentStats(ctx context.Context, deploymentID string) (int, int, int, error) {
	args := m.Called(ctx, deploymentID)
	return args.Int(0), args.Int(1), args.Int(2), args.Error(3)
}

func (m *MockRepository) GetReleaseByVersion(ctx context.Context, templateID, version string, channel ReleaseChannel) (*FirmwareRelease, error) {
	args := m.Called(ctx, templateID, version, channel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FirmwareRelease), args.Error(1)
}

func (m *MockRepository) ReleaseExists(ctx context.Context, releaseID string) (bool, error) {
	args := m.Called(ctx, releaseID)
	return args.Bool(0), args.Error(1)
}

func (m *MockRepository) ListDeployments(ctx context.Context, releaseID string) ([]*OTADeployment, error) {
	args := m.Called(ctx, releaseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*OTADeployment), args.Error(1)
}

func (m *MockRepository) GetActiveDeployments(ctx context.Context) ([]*OTADeployment, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*OTADeployment), args.Error(1)
}

func (m *MockRepository) GetDeviceUpdatesByStatus(ctx context.Context, deploymentID string, status UpdateStatus) ([]*DeviceUpdate, error) {
	args := m.Called(ctx, deploymentID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DeviceUpdate), args.Error(1)
}

func (m *MockRepository) GetDevicesPendingUpdate(ctx context.Context, deploymentID string, limit int) ([]*DeviceUpdate, error) {
	args := m.Called(ctx, deploymentID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*DeviceUpdate), args.Error(1)
}

type MockDeviceRepository struct {
	mock.Mock
}

func (m *MockDeviceRepository) ListDevices(ctx context.Context, filters *device.DeviceFilters) ([]*device.Device, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*device.Device), args.Error(1)
}

func (m *MockDeviceRepository) DeleteDevice(ctx context.Context, deviceID string) error {
	args := m.Called(ctx, deviceID)
	return args.Error(0)
}

func (m *MockDeviceRepository) DeviceExists(ctx context.Context, deviceID string) (bool, error) {
	args := m.Called(ctx, deviceID)
	return args.Bool(0), args.Error(1)
}

func (m *MockDeviceRepository) GetDevice(ctx context.Context, deviceID string) (*device.Device, error) {
	args := m.Called(ctx, deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*device.Device), args.Error(1)
}

func (m *MockDeviceRepository) RegisterDevice(ctx context.Context, device *device.Device) error {
	args := m.Called(ctx, device)
	return args.Error(0)
}

func (m *MockDeviceRepository) UpdateDevice(ctx context.Context, device *device.Device) error {
	args := m.Called(ctx, device)
	return args.Error(0)
}

func (m *MockDeviceRepository) GetDeviceCount(ctx context.Context, filters *device.DeviceFilters) (int64, error) {
	args := m.Called(ctx, filters)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockDeviceRepository) SearchDevices(ctx context.Context, query string, filters *device.DeviceFilters) ([]*device.Device, error) {
	args := m.Called(ctx, query, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*device.Device), args.Error(1)
}

func (m *MockDeviceRepository) UpdateDeviceStatus(ctx context.Context, deviceID string, status device.DeviceStatus, lastSeen time.Time) error {
	args := m.Called(ctx, deviceID, status, lastSeen)
	return args.Error(0)
}

func (m *MockDeviceRepository) GetDevicesByStatus(ctx context.Context, status device.DeviceStatus) ([]*device.Device, error) {
	args := m.Called(ctx, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*device.Device), args.Error(1)
}

func (m *MockDeviceRepository) GetOfflineDevices(ctx context.Context, timeout time.Duration) ([]*device.Device, error) {
	args := m.Called(ctx, timeout)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*device.Device), args.Error(1)
}

func (m *MockDeviceRepository) GetDeviceHealthStatus(ctx context.Context) (*device.DeviceHealthStatus, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*device.DeviceHealthStatus), args.Error(1)
}

func (m *MockDeviceRepository) GetDevicesByTemplate(ctx context.Context, templateID string) ([]*device.Device, error) {
	args := m.Called(ctx, templateID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*device.Device), args.Error(1)
}

func (m *MockDeviceRepository) GetDevicesByOTAChannel(ctx context.Context, channel string) ([]*device.Device, error) {
	args := m.Called(ctx, channel)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*device.Device), args.Error(1)
}

func (m *MockDeviceRepository) GetDevicesLastSeenBefore(ctx context.Context, before time.Time) ([]*device.Device, error) {
	args := m.Called(ctx, before)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*device.Device), args.Error(1)
}

type MockStorageBackend struct {
	mock.Mock
}

func (m *MockStorageBackend) StoreBinary(ctx context.Context, releaseID string, data []byte) (string, error) {
	args := m.Called(ctx, releaseID, data)
	return args.String(0), args.Error(1)
}

func (m *MockStorageBackend) GetBinary(ctx context.Context, path string) ([]byte, error) {
	args := m.Called(ctx, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockStorageBackend) GetBinaryURL(ctx context.Context, path string, expiry time.Duration) (string, error) {
	args := m.Called(ctx, path, expiry)
	return args.String(0), args.Error(1)
}

func (m *MockStorageBackend) DeleteBinary(ctx context.Context, path string) error {
	args := m.Called(ctx, path)
	return args.Error(0)
}

// Test device authentication for OTA updates
func TestService_GetUpdateForDevice_Authentication(t *testing.T) {
	service, mockRepo, _, mockStorage := setupTestService()

	deviceUpdate := &DeviceUpdate{
		DeviceID:     "device-001",
		ReleaseID:    "release-001",
		DeploymentID: "deployment-001",
		Status:       UpdateStatusPending,
		StartedAt:    time.Now(),
	}

	release := createTestRelease("release-001")

	mockRepo.On("GetLatestUpdateForDevice", mock.Anything, "device-001").Return(deviceUpdate, nil)
	mockRepo.On("GetRelease", mock.Anything, "release-001").Return(release, nil)
	mockStorage.On("GetBinaryURL", mock.Anything, release.BinaryPath, mock.AnythingOfType("time.Duration")).Return("https://storage.example.com/firmware.bin", nil)

	update, err := service.GetUpdateForDevice(context.Background(), "device-001")

	require.NoError(t, err)
	require.NotNil(t, update)
	assert.Equal(t, release.ReleaseID, update.ReleaseID)
	assert.Equal(t, release.Version, update.Version)
	assert.Equal(t, release.BinaryHash, update.BinaryHash)
	assert.Equal(t, release.Signature, update.Signature)
	assert.NotEmpty(t, update.BinaryURL)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

// Test device authentication - no pending update
func TestService_GetUpdateForDevice_NoPendingUpdate(t *testing.T) {
	service, mockRepo, _, _ := setupTestService()

	deviceUpdate := &DeviceUpdate{
		DeviceID:     "device-001",
		ReleaseID:    "release-001",
		DeploymentID: "deployment-001",
		Status:       UpdateStatusCompleted, // Not pending
		StartedAt:    time.Now(),
	}

	mockRepo.On("GetLatestUpdateForDevice", mock.Anything, "device-001").Return(deviceUpdate, nil)

	update, err := service.GetUpdateForDevice(context.Background(), "device-001")

	assert.Error(t, err)
	assert.Nil(t, update)
	assert.Contains(t, err.Error(), "no pending update")

	mockRepo.AssertExpectations(t)
}

// Test update status reporting with progress tracking
func TestService_ReportUpdateStatus_ProgressTracking(t *testing.T) {
	service, mockRepo, _, _ := setupTestService()

	deviceUpdate := &DeviceUpdate{
		DeviceID:     "device-001",
		ReleaseID:    "release-001",
		DeploymentID: "deployment-001",
		Status:       UpdateStatusPending,
		Progress:     0,
		StartedAt:    time.Now(),
	}

	deployment := &OTADeployment{
		DeploymentID:     "deployment-001",
		ReleaseID:        "release-001",
		Status:           DeploymentStatusActive,
		FailureThreshold: 10,
	}

	mockRepo.On("GetDeviceUpdate", mock.Anything, "device-001", "release-001").Return(deviceUpdate, nil)
	mockRepo.On("UpdateDeviceUpdate", mock.Anything, mock.MatchedBy(func(update *DeviceUpdate) bool {
		return update.Status == UpdateStatusDownloading && update.Progress == 50
	})).Return(nil)
	mockRepo.On("GetDeployment", mock.Anything, "deployment-001").Return(deployment, nil)
	mockRepo.On("GetDeploymentStats", mock.Anything, "deployment-001").Return(0, 0, 1, nil)
	mockRepo.On("UpdateDeployment", mock.Anything, mock.AnythingOfType("*ota.OTADeployment")).Return(nil)

	report := &UpdateStatusReport{
		DeviceID:  "device-001",
		ReleaseID: "release-001",
		Status:    UpdateStatusDownloading,
		Progress:  50,
	}

	err := service.ReportUpdateStatus(context.Background(), report)

	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test update status reporting - completion
func TestService_ReportUpdateStatus_Completion(t *testing.T) {
	service, mockRepo, _, _ := setupTestService()

	deviceUpdate := &DeviceUpdate{
		DeviceID:     "device-001",
		ReleaseID:    "release-001",
		DeploymentID: "deployment-001",
		Status:       UpdateStatusInstalling,
		Progress:     90,
		StartedAt:    time.Now(),
	}

	deployment := &OTADeployment{
		DeploymentID:     "deployment-001",
		ReleaseID:        "release-001",
		Status:           DeploymentStatusActive,
		FailureThreshold: 10,
	}

	mockRepo.On("GetDeviceUpdate", mock.Anything, "device-001", "release-001").Return(deviceUpdate, nil)
	mockRepo.On("UpdateDeviceUpdate", mock.Anything, mock.MatchedBy(func(update *DeviceUpdate) bool {
		return update.Status == UpdateStatusCompleted && update.Progress == 100 && update.CompletedAt != nil
	})).Return(nil)
	mockRepo.On("GetDeployment", mock.Anything, "deployment-001").Return(deployment, nil)
	mockRepo.On("GetDeploymentStats", mock.Anything, "deployment-001").Return(1, 0, 0, nil)
	mockRepo.On("UpdateDeployment", mock.Anything, mock.MatchedBy(func(d *OTADeployment) bool {
		return d.SuccessCount == 1 && d.Status == DeploymentStatusCompleted
	})).Return(nil)

	report := &UpdateStatusReport{
		DeviceID:  "device-001",
		ReleaseID: "release-001",
		Status:    UpdateStatusCompleted,
		Progress:  100,
	}

	err := service.ReportUpdateStatus(context.Background(), report)

	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test update status reporting - failure
func TestService_ReportUpdateStatus_Failure(t *testing.T) {
	service, mockRepo, _, _ := setupTestService()

	deviceUpdate := &DeviceUpdate{
		DeviceID:     "device-001",
		ReleaseID:    "release-001",
		DeploymentID: "deployment-001",
		Status:       UpdateStatusDownloading,
		Progress:     30,
		StartedAt:    time.Now(),
	}

	deployment := &OTADeployment{
		DeploymentID:     "deployment-001",
		ReleaseID:        "release-001",
		Status:           DeploymentStatusActive,
		FailureThreshold: 50,
		SuccessCount:     0,
		FailureCount:     0,
	}

	mockRepo.On("GetDeviceUpdate", mock.Anything, "device-001", "release-001").Return(deviceUpdate, nil)
	mockRepo.On("UpdateDeviceUpdate", mock.Anything, mock.MatchedBy(func(update *DeviceUpdate) bool {
		return update.Status == UpdateStatusFailed && update.ErrorMessage == "Download failed" && update.CompletedAt != nil
	})).Return(nil)
	mockRepo.On("GetDeployment", mock.Anything, "deployment-001").Return(deployment, nil).Times(2)
	mockRepo.On("GetDeploymentStats", mock.Anything, "deployment-001").Return(0, 1, 0, nil)
	mockRepo.On("UpdateDeployment", mock.Anything, mock.MatchedBy(func(d *OTADeployment) bool {
		return d.FailureCount == 1 && d.Status == DeploymentStatusFailed
	})).Return(nil)

	report := &UpdateStatusReport{
		DeviceID:     "device-001",
		ReleaseID:    "release-001",
		Status:       UpdateStatusFailed,
		Progress:     30,
		ErrorMessage: "Download failed",
	}

	err := service.ReportUpdateStatus(context.Background(), report)

	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test binary hash verification during update
func TestService_VerifyRelease_Integration(t *testing.T) {
	service, mockRepo, _, mockStorage := setupTestService()

	binaryData := []byte("test firmware binary data for verification")
	expectedHash := ComputeHash(binaryData)

	release := createTestRelease("release-001")
	release.BinaryHash = expectedHash

	mockRepo.On("GetRelease", mock.Anything, "release-001").Return(release, nil)
	mockStorage.On("GetBinary", mock.Anything, release.BinaryPath).Return(binaryData, nil)

	err := service.VerifyRelease(context.Background(), "release-001")

	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

// Test release channels
func TestService_CreateRelease_DifferentChannels(t *testing.T) {
	service, mockRepo, _, mockStorage := setupTestService()

	channels := []ReleaseChannel{
		ReleaseChannelStable,
		ReleaseChannelBeta,
		ReleaseChannelAlpha,
	}

	for _, channel := range channels {
		t.Run(string(channel), func(t *testing.T) {
			binaryData := []byte("test firmware for " + string(channel))
			req := &CreateReleaseRequest{
				TemplateID: "template-001",
				Version:    "1.0.0",
				Channel:    channel,
				BinaryData: binaryData,
			}

			mockStorage.On("StoreBinary", mock.Anything, mock.AnythingOfType("string"), binaryData).Return("/binaries/test.bin", nil).Once()
			mockRepo.On("CreateRelease", mock.Anything, mock.MatchedBy(func(release *FirmwareRelease) bool {
				return release.Channel == channel
			})).Return(nil).Once()

			release, err := service.CreateRelease(context.Background(), req)

			require.NoError(t, err)
			assert.Equal(t, channel, release.Channel)
		})
	}

	mockRepo.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

// Test deployment with no target devices
func TestService_DeployRelease_NoTargetDevices(t *testing.T) {
	service, mockRepo, mockDeviceRepo, _ := setupTestService()

	release := createTestRelease("release-001")

	mockRepo.On("GetRelease", mock.Anything, "release-001").Return(release, nil)
	mockDeviceRepo.On("ListDevices", mock.Anything, mock.AnythingOfType("*device.DeviceFilters")).Return([]*device.Device{}, nil)

	config := &DeploymentConfig{
		Strategy:         DeploymentStrategyImmediate,
		FailureThreshold: 10,
	}

	deployment, err := service.DeployRelease(context.Background(), "release-001", config)

	assert.Error(t, err)
	assert.Nil(t, deployment)
	assert.Contains(t, err.Error(), "no target devices found")

	mockRepo.AssertExpectations(t)
	mockDeviceRepo.AssertExpectations(t)
}

// Test concurrent update status reports
func TestService_ReportUpdateStatus_Concurrent(t *testing.T) {
	service, mockRepo, _, _ := setupTestService()

	deviceUpdate1 := &DeviceUpdate{
		DeviceID:     "device-001",
		ReleaseID:    "release-001",
		DeploymentID: "deployment-001",
		Status:       UpdateStatusPending,
		StartedAt:    time.Now(),
	}

	deviceUpdate2 := &DeviceUpdate{
		DeviceID:     "device-002",
		ReleaseID:    "release-001",
		DeploymentID: "deployment-001",
		Status:       UpdateStatusPending,
		StartedAt:    time.Now(),
	}

	deployment := &OTADeployment{
		DeploymentID:     "deployment-001",
		ReleaseID:        "release-001",
		Status:           DeploymentStatusActive,
		FailureThreshold: 10,
	}

	mockRepo.On("GetDeviceUpdate", mock.Anything, "device-001", "release-001").Return(deviceUpdate1, nil)
	mockRepo.On("GetDeviceUpdate", mock.Anything, "device-002", "release-001").Return(deviceUpdate2, nil)
	mockRepo.On("UpdateDeviceUpdate", mock.Anything, mock.AnythingOfType("*ota.DeviceUpdate")).Return(nil).Times(2)
	mockRepo.On("GetDeployment", mock.Anything, "deployment-001").Return(deployment, nil).Times(2)
	mockRepo.On("GetDeploymentStats", mock.Anything, "deployment-001").Return(1, 0, 1, nil).Times(2)
	mockRepo.On("UpdateDeployment", mock.Anything, mock.AnythingOfType("*ota.OTADeployment")).Return(nil).Times(2)

	report1 := &UpdateStatusReport{
		DeviceID:  "device-001",
		ReleaseID: "release-001",
		Status:    UpdateStatusDownloading,
		Progress:  25,
	}

	report2 := &UpdateStatusReport{
		DeviceID:  "device-002",
		ReleaseID: "release-001",
		Status:    UpdateStatusDownloading,
		Progress:  50,
	}

	err1 := service.ReportUpdateStatus(context.Background(), report1)
	err2 := service.ReportUpdateStatus(context.Background(), report2)

	require.NoError(t, err1)
	require.NoError(t, err2)

	mockRepo.AssertExpectations(t)
}
