package device

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/athena/platform-lib/pkg/logger"
)

// MonitoringServiceInterface defines the interface for device monitoring operations
type MonitoringServiceInterface interface {
	Start(ctx context.Context) error
	Stop() error
	IsRunning() bool
	ProcessHeartbeat(ctx context.Context, heartbeat *DeviceHeartbeat) error
	GetDeviceUptime(ctx context.Context, deviceID string) (time.Duration, error)
	GetDeviceLastSeenDuration(ctx context.Context, deviceID string) (time.Duration, error)
	CheckDeviceOnlineStatus(ctx context.Context, deviceID string) (bool, error)
	GetConfiguration() *MonitoringConfig
	SetOfflineTimeout(timeout time.Duration)
	SetCheckInterval(interval time.Duration)
}

// MonitoringService handles device status monitoring and health checks
type MonitoringService struct {
	repository     Repository
	logger         *logger.Logger
	offlineTimeout time.Duration
	checkInterval  time.Duration
	stopChan       chan struct{}
	wg             sync.WaitGroup
	mu             sync.RWMutex
	isRunning      bool
}

// MonitoringConfig holds configuration for the monitoring service
type MonitoringConfig struct {
	OfflineTimeout time.Duration `json:"offline_timeout"`
	CheckInterval  time.Duration `json:"check_interval"`
}

// DefaultMonitoringConfig returns default monitoring configuration
func DefaultMonitoringConfig() *MonitoringConfig {
	return &MonitoringConfig{
		OfflineTimeout: 5 * time.Minute,
		CheckInterval:  1 * time.Minute,
	}
}

// NewMonitoringService creates a new device monitoring service
func NewMonitoringService(repository Repository, logger *logger.Logger, config *MonitoringConfig) *MonitoringService {
	if config == nil {
		config = DefaultMonitoringConfig()
	}

	return &MonitoringService{
		repository:     repository,
		logger:         logger,
		offlineTimeout: config.OfflineTimeout,
		checkInterval:  config.CheckInterval,
		stopChan:       make(chan struct{}),
	}
}

// Start begins the monitoring service
func (m *MonitoringService) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return fmt.Errorf("monitoring service is already running")
	}

	m.isRunning = true
	m.logger.Infof("Starting device monitoring service with offline timeout: %v, check interval: %v",
		m.offlineTimeout, m.checkInterval)

	// Start the monitoring goroutine
	m.wg.Add(1)
	go m.monitorDevices(ctx)

	return nil
}

// Stop stops the monitoring service
func (m *MonitoringService) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return fmt.Errorf("monitoring service is not running")
	}

	m.logger.Info("Stopping device monitoring service...")
	close(m.stopChan)
	m.wg.Wait()
	m.isRunning = false
	m.logger.Info("Device monitoring service stopped")

	return nil
}

// IsRunning returns whether the monitoring service is currently running
func (m *MonitoringService) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

// monitorDevices runs the main monitoring loop
func (m *MonitoringService) monitorDevices(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	// Run initial check
	m.checkDeviceStatus(ctx)

	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Context cancelled, stopping device monitoring")
			return
		case <-m.stopChan:
			m.logger.Info("Stop signal received, stopping device monitoring")
			return
		case <-ticker.C:
			m.checkDeviceStatus(ctx)
		}
	}
}

// checkDeviceStatus performs a status check on all devices
func (m *MonitoringService) checkDeviceStatus(ctx context.Context) {
	m.logger.Debug("Performing device status check")

	// Get devices that should be marked as offline
	cutoffTime := time.Now().Add(-m.offlineTimeout)
	staleDevices, err := m.repository.GetDevicesLastSeenBefore(ctx, cutoffTime)
	if err != nil {
		m.logger.Errorf("Failed to get stale devices: %v", err)
		return
	}

	// Update status for devices that haven't been seen recently
	offlineCount := 0
	for _, device := range staleDevices {
		// Only update if device is currently marked as online
		if device.Status == DeviceStatusOnline {
			err := m.repository.UpdateDeviceStatus(ctx, device.DeviceID, DeviceStatusOffline, device.LastSeen)
			if err != nil {
				m.logger.Errorf("Failed to mark device %s as offline: %v", device.DeviceID, err)
				continue
			}
			offlineCount++
			m.logger.Infof("Device %s marked as offline (last seen: %v)", device.DeviceID, device.LastSeen)
		}
	}

	if offlineCount > 0 {
		m.logger.Infof("Marked %d devices as offline", offlineCount)
	}

	// Log health summary
	m.logHealthSummary(ctx)
}

// logHealthSummary logs a summary of device health status
func (m *MonitoringService) logHealthSummary(ctx context.Context) {
	health, err := m.repository.GetDeviceHealthStatus(ctx)
	if err != nil {
		m.logger.Errorf("Failed to get device health status: %v", err)
		return
	}

	m.logger.Debugf("Device health summary - Total: %d, Online: %d, Offline: %d, Error: %d",
		health.TotalDevices, health.OnlineDevices, health.OfflineDevices, health.ErrorDevices)
}

// ProcessHeartbeat processes a device heartbeat and updates status
func (m *MonitoringService) ProcessHeartbeat(ctx context.Context, heartbeat *DeviceHeartbeat) error {
	if heartbeat == nil {
		return fmt.Errorf("heartbeat cannot be nil")
	}

	// Validate heartbeat
	if heartbeat.DeviceID == "" {
		return fmt.Errorf("device ID is required in heartbeat")
	}

	// Use current time if timestamp is not provided
	if heartbeat.Timestamp.IsZero() {
		heartbeat.Timestamp = time.Now()
	}

	// Default to online status if not specified
	if heartbeat.Status == "" {
		heartbeat.Status = DeviceStatusOnline
	}

	// Update device status
	err := m.repository.UpdateDeviceStatus(ctx, heartbeat.DeviceID, heartbeat.Status, heartbeat.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to update device status from heartbeat: %w", err)
	}

	m.logger.Debugf("Processed heartbeat from device %s with status %s", heartbeat.DeviceID, heartbeat.Status)
	return nil
}

// GetOfflineDevices returns devices that are considered offline
func (m *MonitoringService) GetOfflineDevices(ctx context.Context) ([]*Device, error) {
	return m.repository.GetOfflineDevices(ctx, m.offlineTimeout)
}

// GetDeviceHealthStatus returns the current health status of all devices
func (m *MonitoringService) GetDeviceHealthStatus(ctx context.Context) (*DeviceHealthStatus, error) {
	return m.repository.GetDeviceHealthStatus(ctx)
}

// CheckDeviceOnlineStatus checks if a specific device is online
func (m *MonitoringService) CheckDeviceOnlineStatus(ctx context.Context, deviceID string) (bool, error) {
	device, err := m.repository.GetDevice(ctx, deviceID)
	if err != nil {
		return false, fmt.Errorf("failed to get device: %w", err)
	}

	return device.IsOnline(m.offlineTimeout), nil
}

// GetDeviceUptime calculates the uptime for a device based on its creation and last seen times
func (m *MonitoringService) GetDeviceUptime(ctx context.Context, deviceID string) (time.Duration, error) {
	device, err := m.repository.GetDevice(ctx, deviceID)
	if err != nil {
		return 0, fmt.Errorf("failed to get device: %w", err)
	}

	// Calculate uptime as the time since device was created
	// In a real implementation, you might want to track actual uptime differently
	uptime := time.Since(device.CreatedAt)
	return uptime, nil
}

// GetDeviceLastSeenDuration returns how long ago the device was last seen
func (m *MonitoringService) GetDeviceLastSeenDuration(ctx context.Context, deviceID string) (time.Duration, error) {
	device, err := m.repository.GetDevice(ctx, deviceID)
	if err != nil {
		return 0, fmt.Errorf("failed to get device: %w", err)
	}

	lastSeenDuration := time.Since(device.LastSeen)
	return lastSeenDuration, nil
}

// SetOfflineTimeout updates the offline timeout configuration
func (m *MonitoringService) SetOfflineTimeout(timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.offlineTimeout = timeout
	m.logger.Infof("Updated offline timeout to %v", timeout)
}

// SetCheckInterval updates the check interval configuration
func (m *MonitoringService) SetCheckInterval(interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.checkInterval = interval
	m.logger.Infof("Updated check interval to %v", interval)
}

// GetConfiguration returns the current monitoring configuration
func (m *MonitoringService) GetConfiguration() *MonitoringConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &MonitoringConfig{
		OfflineTimeout: m.offlineTimeout,
		CheckInterval:  m.checkInterval,
	}
}
