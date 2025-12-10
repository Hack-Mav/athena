package telemetry

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRepository for testing
type MockRepository struct {
	metrics           []*MetricPoint
	aggregationResult []*AggregationResult
}

func (m *MockRepository) StoreTelemetry(ctx context.Context, data *TelemetryData) error {
	return nil
}

func (m *MockRepository) StoreTelemetryBatch(ctx context.Context, batch []*TelemetryData) error {
	return nil
}

func (m *MockRepository) GetDeviceMetrics(ctx context.Context, deviceID string, timeRange TimeRange) ([]*MetricPoint, error) {
	return m.metrics, nil
}

func (m *MockRepository) GetDeviceMetricsByName(ctx context.Context, deviceID string, metricName string, timeRange TimeRange) ([]*MetricPoint, error) {
	return m.metrics, nil
}

func (m *MockRepository) GetLatestMetrics(ctx context.Context, deviceID string, limit int) ([]*MetricPoint, error) {
	return m.metrics, nil
}

func (m *MockRepository) AggregateMetrics(ctx context.Context, query *AggregationQuery) ([]*AggregationResult, error) {
	return m.aggregationResult, nil
}

func (m *MockRepository) CreateThreshold(ctx context.Context, deviceID string, threshold *AlertThreshold) (string, error) {
	return "threshold-001", nil
}

func (m *MockRepository) GetThreshold(ctx context.Context, thresholdID string) (*AlertThreshold, error) {
	return nil, nil
}

func (m *MockRepository) ListThresholds(ctx context.Context, deviceID string) ([]*AlertThreshold, error) {
	return nil, nil
}

func (m *MockRepository) UpdateThreshold(ctx context.Context, thresholdID string, threshold *AlertThreshold) error {
	return nil
}

func (m *MockRepository) DeleteThreshold(ctx context.Context, thresholdID string) error {
	return nil
}

func (m *MockRepository) CreateAlert(ctx context.Context, alert *Alert) error {
	return nil
}

func (m *MockRepository) GetAlert(ctx context.Context, alertID string) (*Alert, error) {
	return nil, nil
}

func (m *MockRepository) ListAlerts(ctx context.Context, deviceID string, status string) ([]*Alert, error) {
	return nil, nil
}

func (m *MockRepository) AcknowledgeAlert(ctx context.Context, alertID string) error {
	return nil
}

func (m *MockRepository) ResolveAlert(ctx context.Context, alertID string) error {
	return nil
}

func (m *MockRepository) DeleteOldTelemetry(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

func TestExporter_ExportJSON(t *testing.T) {
	now := time.Now()
	mockRepo := &MockRepository{
		metrics: []*MetricPoint{
			{
				Timestamp:   now,
				MetricName:  "temperature",
				MetricValue: 25.5,
				Tags:        map[string]string{"location": "room1"},
			},
			{
				Timestamp:   now.Add(1 * time.Minute),
				MetricName:  "humidity",
				MetricValue: 60.0,
				Tags:        map[string]string{"location": "room1"},
			},
		},
	}

	exporter := NewExporter(mockRepo)

	request := &ExportRequest{
		DeviceID: "device-001",
		TimeRange: TimeRange{
			Start: now.Add(-1 * time.Hour),
			End:   now,
		},
		Format: ExportFormatJSON,
	}

	var buf bytes.Buffer
	err := exporter.Export(context.Background(), request, &buf)
	require.NoError(t, err)

	// Parse JSON output
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Contains(t, result, "exported_at")
	assert.Contains(t, result, "count")
	assert.Contains(t, result, "metrics")
	assert.Equal(t, float64(2), result["count"])
}

func TestExporter_ExportCSV(t *testing.T) {
	now := time.Now()
	mockRepo := &MockRepository{
		metrics: []*MetricPoint{
			{
				Timestamp:   now,
				MetricName:  "temperature",
				MetricValue: 25.5,
				Tags:        map[string]string{"location": "room1"},
			},
		},
	}

	exporter := NewExporter(mockRepo)

	request := &ExportRequest{
		DeviceID: "device-001",
		TimeRange: TimeRange{
			Start: now.Add(-1 * time.Hour),
			End:   now,
		},
		Format: ExportFormatCSV,
	}

	var buf bytes.Buffer
	err := exporter.Export(context.Background(), request, &buf)
	require.NoError(t, err)

	// Parse CSV output
	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Should have header + 1 data row
	assert.Len(t, records, 2)

	// Check header
	assert.Equal(t, []string{"timestamp", "metric_name", "metric_value", "tags"}, records[0])

	// Check data row
	assert.Equal(t, "temperature", records[1][1])
	assert.Equal(t, "25.5", records[1][2])
}

func TestExporter_ExportAggregatedJSON(t *testing.T) {
	now := time.Now()
	mockRepo := &MockRepository{
		aggregationResult: []*AggregationResult{
			{Timestamp: now, Value: 25.5},
			{Timestamp: now.Add(1 * time.Hour), Value: 26.0},
		},
	}

	exporter := NewExporter(mockRepo)

	query := &AggregationQuery{
		DeviceID:   "device-001",
		MetricName: "temperature",
		TimeRange: TimeRange{
			Start: now.Add(-24 * time.Hour),
			End:   now,
		},
		Aggregation: AggregationAvg,
		Interval:    1 * time.Hour,
	}

	var buf bytes.Buffer
	err := exporter.ExportAggregated(context.Background(), query, ExportFormatJSON, &buf)
	require.NoError(t, err)

	// Parse JSON output
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Contains(t, result, "exported_at")
	assert.Contains(t, result, "device_id")
	assert.Contains(t, result, "metric_name")
	assert.Contains(t, result, "aggregation")
	assert.Contains(t, result, "results")
	assert.Equal(t, "device-001", result["device_id"])
	assert.Equal(t, "temperature", result["metric_name"])
}

func TestExporter_ExportAggregatedCSV(t *testing.T) {
	now := time.Now()
	mockRepo := &MockRepository{
		aggregationResult: []*AggregationResult{
			{Timestamp: now, Value: 25.5},
			{Timestamp: now.Add(1 * time.Hour), Value: 26.0},
		},
	}

	exporter := NewExporter(mockRepo)

	query := &AggregationQuery{
		DeviceID:   "device-001",
		MetricName: "temperature",
		TimeRange: TimeRange{
			Start: now.Add(-24 * time.Hour),
			End:   now,
		},
		Aggregation: AggregationAvg,
		Interval:    1 * time.Hour,
	}

	var buf bytes.Buffer
	err := exporter.ExportAggregated(context.Background(), query, ExportFormatCSV, &buf)
	require.NoError(t, err)

	// Parse CSV output
	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Should have header + 2 data rows
	assert.Len(t, records, 3)

	// Check header
	assert.Equal(t, []string{"timestamp", "value"}, records[0])

	// Check data rows
	assert.Equal(t, "25.500000", records[1][1])
	assert.Equal(t, "26.000000", records[2][1])
}

func TestExporter_UnsupportedFormat(t *testing.T) {
	mockRepo := &MockRepository{}
	exporter := NewExporter(mockRepo)

	request := &ExportRequest{
		DeviceID: "device-001",
		TimeRange: TimeRange{
			Start: time.Now().Add(-1 * time.Hour),
			End:   time.Now(),
		},
		Format: ExportFormat("xml"),
	}

	var buf bytes.Buffer
	err := exporter.Export(context.Background(), request, &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported export format")
}

func TestExportRequest(t *testing.T) {
	now := time.Now()
	request := &ExportRequest{
		DeviceID:   "device-001",
		MetricName: "temperature",
		TimeRange: TimeRange{
			Start: now.Add(-1 * time.Hour),
			End:   now,
		},
		Format: ExportFormatJSON,
	}

	assert.Equal(t, "device-001", request.DeviceID)
	assert.Equal(t, "temperature", request.MetricName)
	assert.Equal(t, ExportFormatJSON, request.Format)
	assert.False(t, request.TimeRange.Start.IsZero())
	assert.False(t, request.TimeRange.End.IsZero())
}

func TestExporter_ExportCSVWithMultipleMetrics(t *testing.T) {
	now := time.Now()
	mockRepo := &MockRepository{
		metrics: []*MetricPoint{
			{
				Timestamp:   now,
				MetricName:  "temperature",
				MetricValue: 25.5,
				Tags:        map[string]string{"location": "room1"},
			},
			{
				Timestamp:   now.Add(1 * time.Minute),
				MetricName:  "humidity",
				MetricValue: 60.0,
				Tags:        map[string]string{"location": "room1"},
			},
			{
				Timestamp:   now.Add(2 * time.Minute),
				MetricName:  "temperature",
				MetricValue: 26.0,
				Tags:        map[string]string{"location": "room2"},
			},
		},
	}

	exporter := NewExporter(mockRepo)

	request := &ExportRequest{
		DeviceID: "device-001",
		TimeRange: TimeRange{
			Start: now.Add(-1 * time.Hour),
			End:   now.Add(1 * time.Hour),
		},
		Format: ExportFormatCSV,
	}

	var buf bytes.Buffer
	err := exporter.Export(context.Background(), request, &buf)
	require.NoError(t, err)

	// Parse CSV output
	csvContent := buf.String()
	lines := strings.Split(strings.TrimSpace(csvContent), "\n")

	// Should have header + 3 data rows
	assert.Len(t, lines, 4)

	// Verify header
	assert.Contains(t, lines[0], "timestamp")
	assert.Contains(t, lines[0], "metric_name")
	assert.Contains(t, lines[0], "metric_value")
}
