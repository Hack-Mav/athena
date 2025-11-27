package telemetry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/athena/platform-lib/pkg/logger"
	"github.com/google/uuid"
)

// AlertMonitor monitors telemetry data and triggers alerts based on thresholds
type AlertMonitor struct {
	repository Repository
	notifier   *AlertNotifier
	logger     *logger.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
	thresholds map[string][]*ThresholdConfig // deviceID -> thresholds
}

// ThresholdConfig represents a threshold configuration with metadata
type ThresholdConfig struct {
	ThresholdID string
	DeviceID    string
	Threshold   *AlertThreshold
	LastCheck   time.Time
	LastAlert   time.Time
}

// NewAlertMonitor creates a new alert monitor
func NewAlertMonitor(repository Repository, notifier *AlertNotifier, logger *logger.Logger) *AlertMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &AlertMonitor{
		repository: repository,
		notifier:   notifier,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
		thresholds: make(map[string][]*ThresholdConfig),
	}
}

// Start starts the alert monitoring process
func (am *AlertMonitor) Start(checkInterval time.Duration) {
	am.wg.Add(1)
	go am.monitorLoop(checkInterval)
	am.logger.Info("Alert monitor started")
}

// Stop stops the alert monitoring process
func (am *AlertMonitor) Stop() {
	am.cancel()
	am.wg.Wait()
	am.logger.Info("Alert monitor stopped")
}

// LoadThresholds loads all thresholds from the repository
func (am *AlertMonitor) LoadThresholds(deviceIDs []string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	for _, deviceID := range deviceIDs {
		ctx, cancel := context.WithTimeout(am.ctx, 5*time.Second)
		thresholds, err := am.repository.ListThresholds(ctx, deviceID)
		cancel()

		if err != nil {
			am.logger.Error(fmt.Sprintf("Failed to load thresholds for device %s: %v", deviceID, err))
			continue
		}

		configs := make([]*ThresholdConfig, 0, len(thresholds))
		for _, threshold := range thresholds {
			if threshold.Enabled {
				configs = append(configs, &ThresholdConfig{
					ThresholdID: uuid.New().String(),
					DeviceID:    deviceID,
					Threshold:   threshold,
					LastCheck:   time.Now(),
				})
			}
		}

		am.thresholds[deviceID] = configs
	}

	return nil
}

// AddThreshold adds a threshold to monitor
func (am *AlertMonitor) AddThreshold(deviceID string, thresholdID string, threshold *AlertThreshold) {
	am.mu.Lock()
	defer am.mu.Unlock()

	config := &ThresholdConfig{
		ThresholdID: thresholdID,
		DeviceID:    deviceID,
		Threshold:   threshold,
		LastCheck:   time.Now(),
	}

	am.thresholds[deviceID] = append(am.thresholds[deviceID], config)
	am.logger.Info(fmt.Sprintf("Added threshold %s for device %s", thresholdID, deviceID))
}

// RemoveThreshold removes a threshold from monitoring
func (am *AlertMonitor) RemoveThreshold(deviceID string, thresholdID string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	thresholds := am.thresholds[deviceID]
	for i, config := range thresholds {
		if config.ThresholdID == thresholdID {
			am.thresholds[deviceID] = append(thresholds[:i], thresholds[i+1:]...)
			am.logger.Info(fmt.Sprintf("Removed threshold %s for device %s", thresholdID, deviceID))
			break
		}
	}
}

// monitorLoop is the main monitoring loop
func (am *AlertMonitor) monitorLoop(checkInterval time.Duration) {
	defer am.wg.Done()

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-am.ctx.Done():
			return
		case <-ticker.C:
			am.checkAllThresholds()
		}
	}
}

// checkAllThresholds checks all configured thresholds
func (am *AlertMonitor) checkAllThresholds() {
	am.mu.RLock()
	deviceIDs := make([]string, 0, len(am.thresholds))
	for deviceID := range am.thresholds {
		deviceIDs = append(deviceIDs, deviceID)
	}
	am.mu.RUnlock()

	for _, deviceID := range deviceIDs {
		am.checkDeviceThresholds(deviceID)
	}
}

// checkDeviceThresholds checks thresholds for a specific device
func (am *AlertMonitor) checkDeviceThresholds(deviceID string) {
	am.mu.RLock()
	configs := am.thresholds[deviceID]
	am.mu.RUnlock()

	if len(configs) == 0 {
		return
	}

	for _, config := range configs {
		if !config.Threshold.Enabled {
			continue
		}

		if err := am.checkThreshold(config); err != nil {
			am.logger.Error(fmt.Sprintf("Error checking threshold %s: %v", config.ThresholdID, err))
		}
	}
}

// checkThreshold checks a single threshold
func (am *AlertMonitor) checkThreshold(config *ThresholdConfig) error {
	ctx, cancel := context.WithTimeout(am.ctx, 5*time.Second)
	defer cancel()

	// Get recent metrics for the threshold duration
	timeRange := TimeRange{
		Start: time.Now().Add(-config.Threshold.Duration),
		End:   time.Now(),
	}

	metrics, err := am.repository.GetDeviceMetricsByName(ctx, config.DeviceID, config.Threshold.MetricName, timeRange)
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}

	if len(metrics) == 0 {
		return nil
	}

	// Check if threshold is violated
	violated, currentValue := am.evaluateThreshold(metrics, config.Threshold)

	if violated {
		// Check if we should trigger an alert (avoid alert spam)
		if time.Since(config.LastAlert) < config.Threshold.Duration {
			return nil
		}

		// Create and store alert
		alert := &Alert{
			AlertID:        uuid.New().String(),
			DeviceID:       config.DeviceID,
			ThresholdID:    config.ThresholdID,
			MetricName:     config.Threshold.MetricName,
			CurrentValue:   currentValue,
			ThresholdValue: config.Threshold.Value,
			Severity:       config.Threshold.Severity,
			Message:        am.generateAlertMessage(config.Threshold, currentValue),
			TriggeredAt:    time.Now(),
			Status:         "active",
		}

		if err := am.repository.CreateAlert(ctx, alert); err != nil {
			return fmt.Errorf("failed to create alert: %w", err)
		}

		// Send notification
		if am.notifier != nil {
			am.notifier.SendAlert(alert)
		}

		// Update last alert time
		am.mu.Lock()
		config.LastAlert = time.Now()
		am.mu.Unlock()

		am.logger.Warn(fmt.Sprintf("Alert triggered for device %s: %s", config.DeviceID, alert.Message))
	}

	// Update last check time
	am.mu.Lock()
	config.LastCheck = time.Now()
	am.mu.Unlock()

	return nil
}

// evaluateThreshold evaluates if a threshold is violated
func (am *AlertMonitor) evaluateThreshold(metrics []*MetricPoint, threshold *AlertThreshold) (bool, float64) {
	if len(metrics) == 0 {
		return false, 0
	}

	// Get the latest value
	latestMetric := metrics[len(metrics)-1]

	var currentValue float64
	switch v := latestMetric.MetricValue.(type) {
	case float64:
		currentValue = v
	case float32:
		currentValue = float64(v)
	case int:
		currentValue = float64(v)
	case int64:
		currentValue = float64(v)
	default:
		return false, 0
	}

	// Evaluate based on operator
	switch threshold.Operator {
	case "gt":
		return currentValue > threshold.Value, currentValue
	case "gte":
		return currentValue >= threshold.Value, currentValue
	case "lt":
		return currentValue < threshold.Value, currentValue
	case "lte":
		return currentValue <= threshold.Value, currentValue
	case "eq":
		return currentValue == threshold.Value, currentValue
	default:
		return false, currentValue
	}
}

// generateAlertMessage generates a human-readable alert message
func (am *AlertMonitor) generateAlertMessage(threshold *AlertThreshold, currentValue float64) string {
	operatorText := map[string]string{
		"gt":  "greater than",
		"gte": "greater than or equal to",
		"lt":  "less than",
		"lte": "less than or equal to",
		"eq":  "equal to",
	}

	op := operatorText[threshold.Operator]
	if op == "" {
		op = threshold.Operator
	}

	return fmt.Sprintf("Metric '%s' is %s threshold: current value %.2f is %s %.2f",
		threshold.MetricName, threshold.Severity, currentValue, op, threshold.Value)
}
