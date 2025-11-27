package ota

import (
	"context"
	"testing"
	"time"

	"github.com/athena/platform-lib/internal/device"
	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupDeploymentTestService() (*Service, *MockRepository, *MockDeviceRepository, *MockStorageBackend) {
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

// Test staged deployment with percentage-based rollout
func TestService_DeployRelease_StagedDeployment(t *testing.T) {
	service, mockRepo, mockDeviceRepo, _ := setupDeploymentTestService()

	release := createTestRelease("release-001")

	mockRepo.On("GetRelease", mock.Anything, "release-001").Return(release, nil)
	mockDeviceRepo.On("ListDevices", mock.Anything, mock.MatchedBy(func(filters *device.DeviceFilters) bool {
		return filters.TemplateID == release.TemplateID && filters.OTAChannel == string(release.Channel)
	})).Return([]*device.Device{
		{DeviceID: "device-001"},
		{DeviceID: "device-002"},
		{DeviceID: "device-003"},
		{DeviceID: "device-004"},
		{DeviceID: "device-005"},
	}, nil)

	mockRepo.On("CreateDeployment", mock.Anything, mock.MatchedBy(func(deployment *OTADeployment) bool {
		return deployment.ReleaseID == "release-001" &&
			deployment.Strategy == DeploymentStrategyStaged &&
			deployment.RolloutPercentage == 40 &&
			len(deployment.TargetDevices) == 5
	})).Return(nil)

	// Expect device updates for 40% of devices (2 out of 5)
	mockRepo.On("CreateDeviceUpdate", mock.Anything, mock.AnythingOfType("*ota.DeviceUpdate")).Return(nil).Times(2)

	config := &DeploymentConfig{
		Strategy:          DeploymentStrategyStaged,
		RolloutPercentage: 40,
		FailureThreshold:  10,
	}

	deployment, err := service.DeployRelease(context.Background(), "release-001", config)

	require.NoError(t, err)
	require.NotNil(t, deployment)
	assert.Equal(t, DeploymentStrategyStaged, deployment.Strategy)
	assert.Equal(t, 40, deployment.RolloutPercentage)
	assert.Equal(t, DeploymentStatusPending, deployment.Status)
	assert.Len(t, deployment.TargetDevices, 5)

	mockRepo.AssertExpectations(t)
	mockDeviceRepo.AssertExpectations(t)
}

// Test immediate deployment strategy
func TestService_DeployRelease_ImmediateDeployment(t *testing.T) {
	service, mockRepo, mockDeviceRepo, _ := setupDeploymentTestService()

	release := createTestRelease("release-001")

	mockRepo.On("GetRelease", mock.Anything, "release-001").Return(release, nil)
	mockDeviceRepo.On("ListDevices", mock.Anything, mock.AnythingOfType("*device.DeviceFilters")).Return([]*device.Device{
		{DeviceID: "device-001"},
		{DeviceID: "device-002"},
	}, nil)

	mockRepo.On("CreateDeployment", mock.Anything, mock.AnythingOfType("*ota.OTADeployment")).Return(nil)
	mockRepo.On("CreateDeviceUpdate", mock.Anything, mock.AnythingOfType("*ota.DeviceUpdate")).Return(nil).Times(2)
	mockRepo.On("UpdateDeployment", mock.Anything, mock.MatchedBy(func(deployment *OTADeployment) bool {
		return deployment.Status == DeploymentStatusActive
	})).Return(nil)

	config := &DeploymentConfig{
		Strategy:         DeploymentStrategyImmediate,
		FailureThreshold: 10,
	}

	deployment, err := service.DeployRelease(context.Background(), "release-001", config)

	require.NoError(t, err)
	require.NotNil(t, deployment)
	assert.Equal(t, DeploymentStrategyImmediate, deployment.Strategy)
	assert.Equal(t, DeploymentStatusActive, deployment.Status)

	mockRepo.AssertExpectations(t)
	mockDeviceRepo.AssertExpectations(t)
}

// Test rollback deployment
func TestService_RollbackDeployment(t *testing.T) {
	service, mockRepo, _, _ := setupDeploymentTestService()

	currentRelease := createTestRelease("release-002")
	currentRelease.Version = "2.0.0"
	currentRelease.CreatedAt = time.Now()

	previousRelease := createTestRelease("release-001")
	previousRelease.Version = "1.0.0"
	previousRelease.CreatedAt = time.Now().Add(-24 * time.Hour)

	deployment := &OTADeployment{
		DeploymentID:  "deployment-001",
		ReleaseID:     "release-002",
		Status:        DeploymentStatusActive,
		TargetDevices: []string{"device-001", "device-002"},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	mockRepo.On("GetDeployment", mock.Anything, "deployment-001").Return(deployment, nil)
	mockRepo.On("GetRelease", mock.Anything, "release-002").Return(currentRelease, nil)
	mockRepo.On("ListReleases", mock.Anything, currentRelease.TemplateID, ReleaseChannelStable).Return([]*FirmwareRelease{
		currentRelease,
		previousRelease,
	}, nil)
	mockRepo.On("UpdateDeployment", mock.Anything, mock.MatchedBy(func(d *OTADeployment) bool {
		return d.Status == DeploymentStatusFailed
	})).Return(nil)

	// Expect new deployment for rollback
	mockRepo.On("CreateDeployment", mock.Anything, mock.MatchedBy(func(d *OTADeployment) bool {
		return d.ReleaseID == "release-001" && d.Strategy == DeploymentStrategyImmediate
	})).Return(nil)
	mockRepo.On("CreateDeviceUpdate", mock.Anything, mock.AnythingOfType("*ota.DeviceUpdate")).Return(nil).Times(2)
	mockRepo.On("UpdateDeployment", mock.Anything, mock.MatchedBy(func(d *OTADeployment) bool {
		return d.Status == DeploymentStatusActive
	})).Return(nil)

	err := service.RollbackDeployment(context.Background(), "deployment-001")

	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test deployment status reporting
func TestService_GetDeploymentStatus(t *testing.T) {
	service, mockRepo, _, _ := setupDeploymentTestService()

	deployment := &OTADeployment{
		DeploymentID:  "deployment-001",
		ReleaseID:     "release-001",
		Status:        DeploymentStatusActive,
		Strategy:      DeploymentStrategyStaged,
		TargetDevices: []string{"device-001", "device-002", "device-003", "device-004", "device-005"},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	deviceUpdates := []*DeviceUpdate{
		{DeviceID: "device-001", Status: UpdateStatusCompleted},
		{DeviceID: "device-002", Status: UpdateStatusCompleted},
		{DeviceID: "device-003", Status: UpdateStatusInstalling},
		{DeviceID: "device-004", Status: UpdateStatusPending},
		{DeviceID: "device-005", Status: UpdateStatusFailed},
	}

	mockRepo.On("GetDeployment", mock.Anything, "deployment-001").Return(deployment, nil)
	mockRepo.On("ListDeviceUpdates", mock.Anything, "deployment-001").Return(deviceUpdates, nil)

	statusReport, err := service.GetDeploymentStatus(context.Background(), "deployment-001")

	require.NoError(t, err)
	require.NotNil(t, statusReport)
	assert.Equal(t, "deployment-001", statusReport.DeploymentID)
	assert.Equal(t, 5, statusReport.TotalDevices)
	assert.Equal(t, 2, statusReport.CompletedCount)
	assert.Equal(t, 1, statusReport.InstallingCount)
	assert.Equal(t, 1, statusReport.PendingCount)
	assert.Equal(t, 1, statusReport.FailedCount)
	assert.Equal(t, 40, statusReport.ProgressPercentage) // 2/5 = 40%

	mockRepo.AssertExpectations(t)
}
