package telemetry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAlertMonitor_EvaluateThreshold(t *testing.T) {
	monitor := &AlertMonitor{}

	tests := []struct {
		name           string
		metrics        []*MetricPoint
		threshold      *AlertThreshold
		expectViolated bool
		expectedValue  float64
	}{
		{
			name: "Greater than threshold violated",
			metrics: []*MetricPoint{
				{MetricValue: 85.0},
			},
			threshold: &AlertThreshold{
				Operator: "gt",
				Value:    80.0,
			},
			expectViolated: true,
			expectedValue:  85.0,
		},
		{
			name: "Greater than threshold not violated",
			metrics: []*MetricPoint{
				{MetricValue: 75.0},
			},
			threshold: &AlertThreshold{
				Operator: "gt",
				Value:    80.0,
			},
			expectViolated: false,
			expectedValue:  75.0,
		},
		{
			name: "Less than threshold violated",
			metrics: []*MetricPoint{
				{MetricValue: 15.0},
			},
			threshold: &AlertThreshold{
				Operator: "lt",
				Value:    20.0,
			},
			expectViolated: true,
			expectedValue:  15.0,
		},
		{
			name: "Equal threshold violated",
			metrics: []*MetricPoint{
				{MetricValue: 100.0},
			},
			threshold: &AlertThreshold{
				Operator: "eq",
				Value:    100.0,
			},
			expectViolated: true,
			expectedValue:  100.0,
		},
		{
			name: "Greater than or equal threshold violated",
			metrics: []*MetricPoint{
				{MetricValue: 80.0},
			},
			threshold: &AlertThreshold{
				Operator: "gte",
				Value:    80.0,
			},
			expectViolated: true,
			expectedValue:  80.0,
		},
		{
			name: "Less than or equal threshold violated",
			metrics: []*MetricPoint{
				{MetricValue: 20.0},
			},
			threshold: &AlertThreshold{
				Operator: "lte",
				Value:    20.0,
			},
			expectViolated: true,
			expectedValue:  20.0,
		},
		{
			name:    "Empty metrics",
			metrics: []*MetricPoint{},
			threshold: &AlertThreshold{
				Operator: "gt",
				Value:    80.0,
			},
			expectViolated: false,
			expectedValue:  0,
		},
		{
			name: "Invalid operator",
			metrics: []*MetricPoint{
				{MetricValue: 85.0},
			},
			threshold: &AlertThreshold{
				Operator: "invalid",
				Value:    80.0,
			},
			expectViolated: false,
			expectedValue:  85.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violated, value := monitor.evaluateThreshold(tt.metrics, tt.threshold)
			assert.Equal(t, tt.expectViolated, violated)
			assert.Equal(t, tt.expectedValue, value)
		})
	}
}

func TestAlertMonitor_GenerateAlertMessage(t *testing.T) {
	monitor := &AlertMonitor{}

	tests := []struct {
		name          string
		threshold     *AlertThreshold
		currentValue  float64
		expectedMatch string
	}{
		{
			name: "Greater than critical",
			threshold: &AlertThreshold{
				MetricName: "temperature",
				Operator:   "gt",
				Value:      80.0,
				Severity:   "critical",
			},
			currentValue:  85.5,
			expectedMatch: "temperature",
		},
		{
			name: "Less than warning",
			threshold: &AlertThreshold{
				MetricName: "battery",
				Operator:   "lt",
				Value:      20.0,
				Severity:   "warning",
			},
			currentValue:  15.0,
			expectedMatch: "battery",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := monitor.generateAlertMessage(tt.threshold, tt.currentValue)
			assert.NotEmpty(t, message)
			assert.Contains(t, message, tt.expectedMatch)
			assert.Contains(t, message, tt.threshold.Severity)
		})
	}
}

func TestAlertMonitor_AddRemoveThreshold(t *testing.T) {
	monitor := &AlertMonitor{
		thresholds: make(map[string][]*ThresholdConfig),
	}

	deviceID := "device-001"
	thresholdID := "threshold-001"
	threshold := &AlertThreshold{
		MetricName: "temperature",
		Operator:   "gt",
		Value:      80.0,
		Enabled:    true,
	}

	// Add threshold
	monitor.AddThreshold(deviceID, thresholdID, threshold)
	assert.Len(t, monitor.thresholds[deviceID], 1)
	assert.Equal(t, thresholdID, monitor.thresholds[deviceID][0].ThresholdID)

	// Remove threshold
	monitor.RemoveThreshold(deviceID, thresholdID)
	assert.Len(t, monitor.thresholds[deviceID], 0)
}

func TestThresholdConfig(t *testing.T) {
	config := &ThresholdConfig{
		ThresholdID: "threshold-001",
		DeviceID:    "device-001",
		Threshold: &AlertThreshold{
			MetricName: "temperature",
			Operator:   "gt",
			Value:      80.0,
			Duration:   5 * time.Minute,
			Severity:   "critical",
			Enabled:    true,
		},
		LastCheck: time.Now(),
		LastAlert: time.Now().Add(-10 * time.Minute),
	}

	assert.NotEmpty(t, config.ThresholdID)
	assert.NotEmpty(t, config.DeviceID)
	assert.NotNil(t, config.Threshold)
	assert.False(t, config.LastCheck.IsZero())
	assert.False(t, config.LastAlert.IsZero())
}
