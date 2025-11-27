package ota

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/athena/platform-lib/internal/device"
	"github.com/google/uuid"
)

// DeployRelease creates and initiates a new deployment for a firmware release
func (s *Service) DeployRelease(ctx context.Context, releaseID string, config *DeploymentConfig) (*OTADeployment, error) {
	// Validate release exists
	release, err := s.repository.GetRelease(ctx, releaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get release: %w", err)
	}

	// Validate deployment configuration
	if err := s.validateDeploymentConfig(config); err != nil {
		return nil, fmt.Errorf("invalid deployment configuration: %w", err)
	}

	// Determine target devices
	targetDevices, err := s.determineTargetDevices(ctx, release, config)
	if err != nil {
		return nil, fmt.Errorf("failed to determine target devices: %w", err)
	}

	if len(targetDevices) == 0 {
		return nil, fmt.Errorf("no target devices found for deployment")
	}

	// Create deployment
	deployment := &OTADeployment{
		DeploymentID:      uuid.New().String(),
		ReleaseID:         releaseID,
		Strategy:          config.Strategy,
		TargetDevices:     targetDevices,
		RolloutPercentage: config.RolloutPercentage,
		Status:            DeploymentStatusPending,
		FailureThreshold:  config.FailureThreshold,
		SuccessCount:      0,
		FailureCount:      0,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	// Store deployment
	err = s.repository.CreateDeployment(ctx, deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	// Initialize device updates based on strategy
	err = s.initializeDeviceUpdates(ctx, deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize device updates: %w", err)
	}

	// Start deployment if immediate strategy
	if config.Strategy == DeploymentStrategyImmediate {
		deployment.Status = DeploymentStatusActive
		err = s.repository.UpdateDeployment(ctx, deployment)
		if err != nil {
			return nil, fmt.Errorf("failed to activate deployment: %w", err)
		}
	}

	s.logger.Info("Created deployment", "deployment_id", deployment.DeploymentID, "release_id", releaseID, "strategy", config.Strategy, "target_devices", len(targetDevices))

	return deployment, nil
}

// validateDeploymentConfig validates the deployment configuration
func (s *Service) validateDeploymentConfig(config *DeploymentConfig) error {
	if config == nil {
		return fmt.Errorf("deployment config cannot be nil")
	}

	// Validate strategy
	if config.Strategy != DeploymentStrategyImmediate &&
		config.Strategy != DeploymentStrategyStaged &&
		config.Strategy != DeploymentStrategyCanary {
		return fmt.Errorf("invalid deployment strategy: %s", config.Strategy)
	}

	// Validate rollout percentage for staged deployments
	if config.Strategy == DeploymentStrategyStaged || config.Strategy == DeploymentStrategyCanary {
		if config.RolloutPercentage < 1 || config.RolloutPercentage > 100 {
			return fmt.Errorf("rollout percentage must be between 1 and 100")
		}
	}

	// Set default failure threshold if not specified
	if config.FailureThreshold == 0 {
		config.FailureThreshold = 10 // Default 10% failure threshold
	}

	return nil
}

// determineTargetDevices determines which devices should receive the update
func (s *Service) determineTargetDevices(ctx context.Context, release *FirmwareRelease, config *DeploymentConfig) ([]string, error) {
	var targetDevices []string

	// If specific devices are provided, use them
	if len(config.TargetDevices) > 0 {
		targetDevices = config.TargetDevices
	} else {
		// Query devices by template and OTA channel
		filters := &device.DeviceFilters{
			TemplateID: release.TemplateID,
			OTAChannel: string(release.Channel),
		}

		devices, err := s.deviceRepository.ListDevices(ctx, filters)
		if err != nil {
			return nil, fmt.Errorf("failed to list devices: %w", err)
		}

		for _, dev := range devices {
			targetDevices = append(targetDevices, dev.DeviceID)
		}
	}

	return targetDevices, nil
}

// initializeDeviceUpdates creates device update records for the deployment
func (s *Service) initializeDeviceUpdates(ctx context.Context, deployment *OTADeployment) error {
	// Determine how many devices to update based on strategy
	devicesToUpdate := s.selectDevicesForUpdate(deployment)

	for _, deviceID := range devicesToUpdate {
		update := &DeviceUpdate{
			DeviceID:     deviceID,
			ReleaseID:    deployment.ReleaseID,
			DeploymentID: deployment.DeploymentID,
			Status:       UpdateStatusPending,
			Progress:     0,
			StartedAt:    time.Now(),
		}

		err := s.repository.CreateDeviceUpdate(ctx, update)
		if err != nil {
			s.logger.Warn("Failed to create device update", "device_id", deviceID, "error", err)
			continue
		}
	}

	return nil
}

// selectDevicesForUpdate selects devices for update based on deployment strategy
func (s *Service) selectDevicesForUpdate(deployment *OTADeployment) []string {
	totalDevices := len(deployment.TargetDevices)

	switch deployment.Strategy {
	case DeploymentStrategyImmediate:
		// Update all devices immediately
		return deployment.TargetDevices

	case DeploymentStrategyStaged:
		// Update a percentage of devices
		numDevices := (totalDevices * deployment.RolloutPercentage) / 100
		if numDevices == 0 {
			numDevices = 1
		}
		return deployment.TargetDevices[:numDevices]

	case DeploymentStrategyCanary:
		// Start with a small canary group (use rollout percentage or default to 5%)
		numDevices := (totalDevices * deployment.RolloutPercentage) / 100
		if numDevices == 0 {
			numDevices = 1
		}
		// Shuffle to get random canary devices
		shuffled := make([]string, len(deployment.TargetDevices))
		copy(shuffled, deployment.TargetDevices)
		rand.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})
		return shuffled[:numDevices]

	default:
		return deployment.TargetDevices
	}
}

// GetDeployment retrieves a deployment by ID
func (s *Service) GetDeployment(ctx context.Context, deploymentID string) (*OTADeployment, error) {
	return s.repository.GetDeployment(ctx, deploymentID)
}

// PauseDeployment pauses an active deployment
func (s *Service) PauseDeployment(ctx context.Context, deploymentID string) error {
	deployment, err := s.repository.GetDeployment(ctx, deploymentID)
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	if deployment.Status != DeploymentStatusActive {
		return fmt.Errorf("can only pause active deployments, current status: %s", deployment.Status)
	}

	deployment.Status = DeploymentStatusPaused
	deployment.UpdatedAt = time.Now()

	err = s.repository.UpdateDeployment(ctx, deployment)
	if err != nil {
		return fmt.Errorf("failed to pause deployment: %w", err)
	}

	s.logger.Info("Paused deployment", "deployment_id", deploymentID)

	return nil
}

// ResumeDeployment resumes a paused deployment
func (s *Service) ResumeDeployment(ctx context.Context, deploymentID string) error {
	deployment, err := s.repository.GetDeployment(ctx, deploymentID)
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	if deployment.Status != DeploymentStatusPaused {
		return fmt.Errorf("can only resume paused deployments, current status: %s", deployment.Status)
	}

	deployment.Status = DeploymentStatusActive
	deployment.UpdatedAt = time.Now()

	err = s.repository.UpdateDeployment(ctx, deployment)
	if err != nil {
		return fmt.Errorf("failed to resume deployment: %w", err)
	}

	s.logger.Info("Resumed deployment", "deployment_id", deploymentID)

	return nil
}

// RollbackDeployment rolls back a deployment to the previous firmware version
func (s *Service) RollbackDeployment(ctx context.Context, deploymentID string) error {
	deployment, err := s.repository.GetDeployment(ctx, deploymentID)
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Get the release being deployed
	release, err := s.repository.GetRelease(ctx, deployment.ReleaseID)
	if err != nil {
		return fmt.Errorf("failed to get release: %w", err)
	}

	// Find previous stable release for the same template
	releases, err := s.repository.ListReleases(ctx, release.TemplateID, ReleaseChannelStable)
	if err != nil {
		return fmt.Errorf("failed to list releases: %w", err)
	}

	var previousRelease *FirmwareRelease
	for _, r := range releases {
		if r.ReleaseID != release.ReleaseID && r.CreatedAt.Before(release.CreatedAt) {
			previousRelease = r
			break
		}
	}

	if previousRelease == nil {
		return fmt.Errorf("no previous release found for rollback")
	}

	// Mark current deployment as failed
	deployment.Status = DeploymentStatusFailed
	deployment.UpdatedAt = time.Now()
	err = s.repository.UpdateDeployment(ctx, deployment)
	if err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}

	// Create a new deployment for the previous release
	rollbackConfig := &DeploymentConfig{
		Strategy:          DeploymentStrategyImmediate,
		TargetDevices:     deployment.TargetDevices,
		RolloutPercentage: 100,
		FailureThreshold:  deployment.FailureThreshold,
	}

	rollbackDeployment, err := s.DeployRelease(ctx, previousRelease.ReleaseID, rollbackConfig)
	if err != nil {
		return fmt.Errorf("failed to create rollback deployment: %w", err)
	}

	s.logger.Info("Rolled back deployment", "original_deployment_id", deploymentID, "rollback_deployment_id", rollbackDeployment.DeploymentID, "previous_release_id", previousRelease.ReleaseID)

	return nil
}

// GetUpdateForDevice retrieves the pending update for a device
func (s *Service) GetUpdateForDevice(ctx context.Context, deviceID string) (*FirmwareUpdate, error) {
	// Get the latest update for the device
	update, err := s.repository.GetLatestUpdateForDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("no pending update for device: %w", err)
	}

	// Only return if status is pending
	if update.Status != UpdateStatusPending {
		return nil, fmt.Errorf("no pending update for device")
	}

	// Get the release details
	release, err := s.repository.GetRelease(ctx, update.ReleaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get release: %w", err)
	}

	// Generate signed URL for binary download
	binaryURL, err := s.storageBackend.GetBinaryURL(ctx, release.BinaryPath, 1*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate binary URL: %w", err)
	}

	firmwareUpdate := &FirmwareUpdate{
		ReleaseID:    release.ReleaseID,
		Version:      release.Version,
		BinaryURL:    binaryURL,
		BinaryHash:   release.BinaryHash,
		BinarySize:   release.BinarySize,
		Signature:    release.Signature,
		ReleaseNotes: release.ReleaseNotes,
		CreatedAt:    release.CreatedAt,
	}

	return firmwareUpdate, nil
}

// ReportUpdateStatus updates the status of a device update
func (s *Service) ReportUpdateStatus(ctx context.Context, report *UpdateStatusReport) error {
	// Get the device update
	update, err := s.repository.GetDeviceUpdate(ctx, report.DeviceID, report.ReleaseID)
	if err != nil {
		return fmt.Errorf("failed to get device update: %w", err)
	}

	// Update status
	update.Status = report.Status
	update.Progress = report.Progress
	update.ErrorMessage = report.ErrorMessage

	// Set completion time if completed or failed
	if report.Status == UpdateStatusCompleted || report.Status == UpdateStatusFailed {
		now := time.Now()
		update.CompletedAt = &now
	}

	err = s.repository.UpdateDeviceUpdate(ctx, update)
	if err != nil {
		return fmt.Errorf("failed to update device update: %w", err)
	}

	// Update deployment statistics
	err = s.updateDeploymentStats(ctx, update.DeploymentID)
	if err != nil {
		s.logger.Warn("Failed to update deployment stats", "deployment_id", update.DeploymentID, "error", err)
	}

	// Check for automatic failure detection and rollback
	if report.Status == UpdateStatusFailed {
		err = s.checkAndHandleFailures(ctx, update.DeploymentID)
		if err != nil {
			s.logger.Warn("Failed to handle deployment failures", "deployment_id", update.DeploymentID, "error", err)
		}
	}

	s.logger.Info("Updated device update status", "device_id", report.DeviceID, "release_id", report.ReleaseID, "status", report.Status)

	return nil
}

// updateDeploymentStats updates the success and failure counts for a deployment
func (s *Service) updateDeploymentStats(ctx context.Context, deploymentID string) error {
	deployment, err := s.repository.GetDeployment(ctx, deploymentID)
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Get deployment statistics
	successCount, failureCount, pendingCount, err := s.repository.GetDeploymentStats(ctx, deploymentID)
	if err != nil {
		return fmt.Errorf("failed to get deployment stats: %w", err)
	}

	deployment.SuccessCount = successCount
	deployment.FailureCount = failureCount
	deployment.UpdatedAt = time.Now()

	// Check if deployment is complete
	if pendingCount == 0 {
		if failureCount == 0 {
			deployment.Status = DeploymentStatusCompleted
		} else if successCount == 0 {
			deployment.Status = DeploymentStatusFailed
		} else {
			deployment.Status = DeploymentStatusCompleted
		}
	}

	err = s.repository.UpdateDeployment(ctx, deployment)
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	return nil
}

// checkAndHandleFailures checks if failure threshold is exceeded and triggers rollback
func (s *Service) checkAndHandleFailures(ctx context.Context, deploymentID string) error {
	deployment, err := s.repository.GetDeployment(ctx, deploymentID)
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Skip if deployment is not active
	if deployment.Status != DeploymentStatusActive {
		return nil
	}

	// Calculate failure rate
	totalAttempts := deployment.SuccessCount + deployment.FailureCount
	if totalAttempts == 0 {
		return nil
	}

	failureRate := (deployment.FailureCount * 100) / totalAttempts

	// Check if failure threshold is exceeded
	if failureRate >= deployment.FailureThreshold {
		s.logger.Warn("Failure threshold exceeded, triggering automatic rollback", "deployment_id", deploymentID, "failure_rate", failureRate, "threshold", deployment.FailureThreshold)

		// Trigger automatic rollback
		err = s.RollbackDeployment(ctx, deploymentID)
		if err != nil {
			return fmt.Errorf("failed to rollback deployment: %w", err)
		}
	}

	return nil
}

// GetDeploymentStatus retrieves the current status and statistics of a deployment
func (s *Service) GetDeploymentStatus(ctx context.Context, deploymentID string) (*DeploymentStatusReport, error) {
	deployment, err := s.repository.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	// Get device updates
	updates, err := s.repository.ListDeviceUpdates(ctx, deploymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list device updates: %w", err)
	}

	// Calculate statistics
	var pendingCount, downloadingCount, installingCount, completedCount, failedCount int
	for _, update := range updates {
		switch update.Status {
		case UpdateStatusPending:
			pendingCount++
		case UpdateStatusDownloading:
			downloadingCount++
		case UpdateStatusInstalling:
			installingCount++
		case UpdateStatusCompleted:
			completedCount++
		case UpdateStatusFailed:
			failedCount++
		}
	}

	totalDevices := len(deployment.TargetDevices)
	progressPercentage := 0
	if totalDevices > 0 {
		progressPercentage = (completedCount * 100) / totalDevices
	}

	report := &DeploymentStatusReport{
		DeploymentID:       deployment.DeploymentID,
		ReleaseID:          deployment.ReleaseID,
		Status:             deployment.Status,
		Strategy:           deployment.Strategy,
		TotalDevices:       totalDevices,
		PendingCount:       pendingCount,
		DownloadingCount:   downloadingCount,
		InstallingCount:    installingCount,
		CompletedCount:     completedCount,
		FailedCount:        failedCount,
		ProgressPercentage: progressPercentage,
		CreatedAt:          deployment.CreatedAt,
		UpdatedAt:          deployment.UpdatedAt,
	}

	return report, nil
}

// DeploymentStatusReport represents the status report for a deployment
type DeploymentStatusReport struct {
	DeploymentID       string             `json:"deployment_id"`
	ReleaseID          string             `json:"release_id"`
	Status             DeploymentStatus   `json:"status"`
	Strategy           DeploymentStrategy `json:"strategy"`
	TotalDevices       int                `json:"total_devices"`
	PendingCount       int                `json:"pending_count"`
	DownloadingCount   int                `json:"downloading_count"`
	InstallingCount    int                `json:"installing_count"`
	CompletedCount     int                `json:"completed_count"`
	FailedCount        int                `json:"failed_count"`
	ProgressPercentage int                `json:"progress_percentage"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}
