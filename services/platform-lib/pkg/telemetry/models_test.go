package telemetry

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelemetryData_ToEntities(t *testing.T) {
	now := time.Now()
	data := &TelemetryData{
		DeviceID:  "test-device-001",
		Timestamp: now,
		Metrics: map[string]interface{}{
			"temperature": 25.5,
			"humidity":    60,
			"status":      "online",
			"enabled":     true,
		},
		Tags: map[string]string{
			"location": "room1",
			"sensor":   "dht22",
		},
	}

	entities, err := data.ToEntities()
	require.NoError(t, err)
	require.Len(t, entities, 4)

	// Verify each entity
	for _, entity := range entities {
		assert.Equal(t, data.DeviceID, entity.DeviceID)
		assert.Equal(t, data.Timestamp, entity.Timestamp)
		assert.NotEmpty(t, entity.MetricName)
		assert.NotEmpty(t, entity.TagsJSON)

		// Verify tags JSON
		var tags map[string]string
		err := json.Unmarshal([]byte(entity.TagsJSON), &tags)
		require.NoError(t, err)
		assert.Equal(t, data.Tags, tags)
	}
}

func TestTelemetryData_ToEntities_DifferentTypes(t *testing.T) {
	data := &TelemetryData{
		DeviceID:  "test-device-001",
		Timestamp: time.Now(),
		Metrics: map[string]interface{}{
			"float64_val": float64(25.5),
			"float32_val": float32(30.2),
			"int_val":     int(100),
			"int64_val":   int64(200),
			"string_val":  "test",
			"bool_true":   true,
			"bool_false":  false,
		},
		Tags: map[string]string{},
	}

	entities, err := data.ToEntities()
	require.NoError(t, err)
	require.Len(t, entities, 7)

	// Find specific entities and verify conversions
	entityMap := make(map[string]*TelemetryEntity)
	for _, entity := range entities {
		entityMap[entity.MetricName] = entity
	}

	assert.Equal(t, 25.5, entityMap["float64_val"].MetricValue)
	assert.InDelta(t, 30.2, entityMap["float32_val"].MetricValue, 0.1)
	assert.Equal(t, float64(100), entityMap["int_val"].MetricValue)
	assert.Equal(t, float64(200), entityMap["int64_val"].MetricValue)
	assert.Equal(t, "test", entityMap["string_val"].MetricString)
	assert.Equal(t, 1.0, entityMap["bool_true"].MetricValue)
	assert.Equal(t, 0.0, entityMap["bool_false"].MetricValue)
}

func TestTelemetryEntity_FromEntity(t *testing.T) {
	now := time.Now()
	tags := map[string]string{"location": "room1"}
	tagsJSON, _ := json.Marshal(tags)

	entity := &TelemetryEntity{
		DeviceID:     "test-device-001",
		Timestamp:    now,
		MetricName:   "temperature",
		MetricValue:  25.5,
		MetricString: "",
		TagsJSON:     string(tagsJSON),
	}

	metric, err := entity.FromEntity()
	require.NoError(t, err)
	require.NotNil(t, metric)

	assert.Equal(t, entity.Timestamp, metric.Timestamp)
	assert.Equal(t, entity.MetricName, metric.MetricName)
	assert.Equal(t, entity.MetricValue, metric.MetricValue)
	assert.Equal(t, tags, metric.Tags)
}

func TestTelemetryEntity_FromEntity_StringValue(t *testing.T) {
	entity := &TelemetryEntity{
		DeviceID:     "test-device-001",
		Timestamp:    time.Now(),
		MetricName:   "status",
		MetricValue:  0,
		MetricString: "online",
		TagsJSON:     "{}",
	}

	metric, err := entity.FromEntity()
	require.NoError(t, err)
	assert.Equal(t, "online", metric.MetricValue)
}

func TestAlert_ToEntity(t *testing.T) {
	now := time.Now()
	ackTime := now.Add(5 * time.Minute)

	alert := &Alert{
		AlertID:        "alert-001",
		DeviceID:       "device-001",
		ThresholdID:    "threshold-001",
		MetricName:     "temperature",
		CurrentValue:   85.5,
		ThresholdValue: 80.0,
		Severity:       "critical",
		Message:        "Temperature too high",
		TriggeredAt:    now,
		AcknowledgedAt: &ackTime,
		Status:         "acknowledged",
		Metadata: map[string]interface{}{
			"location": "server-room",
		},
	}

	entity, err := alert.ToEntity()
	require.NoError(t, err)
	require.NotNil(t, entity)

	assert.Equal(t, alert.AlertID, entity.AlertID)
	assert.Equal(t, alert.DeviceID, entity.DeviceID)
	assert.Equal(t, alert.ThresholdID, entity.ThresholdID)
	assert.Equal(t, alert.MetricName, entity.MetricName)
	assert.Equal(t, alert.CurrentValue, entity.CurrentValue)
	assert.Equal(t, alert.ThresholdValue, entity.ThresholdValue)
	assert.Equal(t, alert.Severity, entity.Severity)
	assert.Equal(t, alert.Message, entity.Message)
	assert.Equal(t, alert.TriggeredAt, entity.TriggeredAt)
	assert.Equal(t, *alert.AcknowledgedAt, entity.AcknowledgedAt)
	assert.Equal(t, alert.Status, entity.Status)
}

func TestAlertEntity_FromEntity(t *testing.T) {
	now := time.Now()
	metadata := map[string]interface{}{"location": "server-room"}
	metadataJSON, _ := json.Marshal(metadata)

	entity := &AlertEntity{
		AlertID:        "alert-001",
		DeviceID:       "device-001",
		ThresholdID:    "threshold-001",
		MetricName:     "temperature",
		CurrentValue:   85.5,
		ThresholdValue: 80.0,
		Severity:       "critical",
		Message:        "Temperature too high",
		TriggeredAt:    now,
		AcknowledgedAt: now.Add(5 * time.Minute),
		Status:         "acknowledged",
		MetadataJSON:   string(metadataJSON),
	}

	alert, err := entity.FromEntity()
	require.NoError(t, err)
	require.NotNil(t, alert)

	assert.Equal(t, entity.AlertID, alert.AlertID)
	assert.Equal(t, entity.DeviceID, alert.DeviceID)
	assert.Equal(t, entity.ThresholdID, alert.ThresholdID)
	assert.Equal(t, entity.MetricName, alert.MetricName)
	assert.Equal(t, entity.CurrentValue, alert.CurrentValue)
	assert.Equal(t, entity.ThresholdValue, alert.ThresholdValue)
	assert.Equal(t, entity.Severity, alert.Severity)
	assert.Equal(t, entity.Message, alert.Message)
	assert.Equal(t, entity.TriggeredAt, alert.TriggeredAt)
	assert.NotNil(t, alert.AcknowledgedAt)
	assert.Equal(t, entity.AcknowledgedAt, *alert.AcknowledgedAt)
	assert.Equal(t, entity.Status, alert.Status)
}

func TestAggregationType(t *testing.T) {
	tests := []struct {
		name string
		agg  AggregationType
	}{
		{"Average", AggregationAvg},
		{"Sum", AggregationSum},
		{"Min", AggregationMin},
		{"Max", AggregationMax},
		{"Count", AggregationCount},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, string(tt.agg))
		})
	}
}

func TestExportFormat(t *testing.T) {
	tests := []struct {
		name   string
		format ExportFormat
	}{
		{"JSON", ExportFormatJSON},
		{"CSV", ExportFormatCSV},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, string(tt.format))
		})
	}
}

func TestNotificationChannel(t *testing.T) {
	tests := []struct {
		name    string
		channel NotificationChannel
	}{
		{"Webhook", ChannelWebhook},
		{"Email", ChannelEmail},
		{"Log", ChannelLog},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEmpty(t, string(tt.channel))
		})
	}
}
