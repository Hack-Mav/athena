package telemetry

import (
	"encoding/json"
	"time"
)

// TelemetryData represents device telemetry data
type TelemetryData struct {
	DeviceID  string                 `json:"device_id"`
	Timestamp time.Time              `json:"timestamp"`
	Metrics   map[string]interface{} `json:"metrics"`
	Tags      map[string]string      `json:"tags"`
}

// TelemetryEntity represents the Datastore entity for telemetry data
type TelemetryEntity struct {
	DeviceID     string    `datastore:"device_id"`
	Timestamp    time.Time `datastore:"timestamp"`
	MetricName   string    `datastore:"metric_name"`
	MetricValue  float64   `datastore:"metric_value"`
	MetricString string    `datastore:"metric_string"`
	TagsJSON     string    `datastore:"tags_json,noindex"`
}

// MetricPoint represents a single metric data point
type MetricPoint struct {
	Timestamp   time.Time         `json:"timestamp"`
	MetricName  string            `json:"metric_name"`
	MetricValue interface{}       `json:"metric_value"`
	Tags        map[string]string `json:"tags"`
}

// TimeRange represents a time range for queries
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// AlertThreshold represents a threshold configuration for alerts
type AlertThreshold struct {
	MetricName string                 `json:"metric_name"`
	Operator   string                 `json:"operator"` // "gt", "lt", "eq", "gte", "lte"
	Value      float64                `json:"value"`
	Duration   time.Duration          `json:"duration"`
	Severity   string                 `json:"severity"` // "info", "warning", "critical"
	Enabled    bool                   `json:"enabled"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Alert represents a triggered alert
type Alert struct {
	AlertID        string                 `json:"alert_id"`
	DeviceID       string                 `json:"device_id"`
	ThresholdID    string                 `json:"threshold_id"`
	MetricName     string                 `json:"metric_name"`
	CurrentValue   float64                `json:"current_value"`
	ThresholdValue float64                `json:"threshold_value"`
	Severity       string                 `json:"severity"`
	Message        string                 `json:"message"`
	TriggeredAt    time.Time              `json:"triggered_at"`
	AcknowledgedAt *time.Time             `json:"acknowledged_at,omitempty"`
	ResolvedAt     *time.Time             `json:"resolved_at,omitempty"`
	Status         string                 `json:"status"` // "active", "acknowledged", "resolved"
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// AlertEntity represents the Datastore entity for alerts
type AlertEntity struct {
	AlertID        string    `datastore:"alert_id"`
	DeviceID       string    `datastore:"device_id"`
	ThresholdID    string    `datastore:"threshold_id"`
	MetricName     string    `datastore:"metric_name"`
	CurrentValue   float64   `datastore:"current_value"`
	ThresholdValue float64   `datastore:"threshold_value"`
	Severity       string    `datastore:"severity"`
	Message        string    `datastore:"message,noindex"`
	TriggeredAt    time.Time `datastore:"triggered_at"`
	AcknowledgedAt time.Time `datastore:"acknowledged_at"`
	ResolvedAt     time.Time `datastore:"resolved_at"`
	Status         string    `datastore:"status"`
	MetadataJSON   string    `datastore:"metadata_json,noindex"`
}

// ThresholdEntity represents the Datastore entity for alert thresholds
type ThresholdEntity struct {
	ThresholdID   string    `datastore:"threshold_id"`
	DeviceID      string    `datastore:"device_id"`
	MetricName    string    `datastore:"metric_name"`
	Operator      string    `datastore:"operator"`
	Value         float64   `datastore:"value"`
	DurationNanos int64     `datastore:"duration_nanos"`
	Severity      string    `datastore:"severity"`
	Enabled       bool      `datastore:"enabled"`
	MetadataJSON  string    `datastore:"metadata_json,noindex"`
	CreatedAt     time.Time `datastore:"created_at"`
	UpdatedAt     time.Time `datastore:"updated_at"`
}

// AggregationType represents the type of aggregation
type AggregationType string

const (
	AggregationAvg   AggregationType = "avg"
	AggregationSum   AggregationType = "sum"
	AggregationMin   AggregationType = "min"
	AggregationMax   AggregationType = "max"
	AggregationCount AggregationType = "count"
)

// AggregationQuery represents a query with aggregation
type AggregationQuery struct {
	DeviceID    string          `json:"device_id"`
	MetricName  string          `json:"metric_name"`
	TimeRange   TimeRange       `json:"time_range"`
	Aggregation AggregationType `json:"aggregation"`
	Interval    time.Duration   `json:"interval,omitempty"`
}

// AggregationResult represents the result of an aggregation query
type AggregationResult struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// ToEntities converts TelemetryData to multiple TelemetryEntity records
func (td *TelemetryData) ToEntities() ([]*TelemetryEntity, error) {
	tagsJSON, err := json.Marshal(td.Tags)
	if err != nil {
		return nil, err
	}

	entities := make([]*TelemetryEntity, 0, len(td.Metrics))

	for metricName, metricValue := range td.Metrics {
		entity := &TelemetryEntity{
			DeviceID:   td.DeviceID,
			Timestamp:  td.Timestamp,
			MetricName: metricName,
			TagsJSON:   string(tagsJSON),
		}

		// Handle different metric value types
		switch v := metricValue.(type) {
		case float64:
			entity.MetricValue = v
		case float32:
			entity.MetricValue = float64(v)
		case int:
			entity.MetricValue = float64(v)
		case int64:
			entity.MetricValue = float64(v)
		case string:
			entity.MetricString = v
		case bool:
			if v {
				entity.MetricValue = 1.0
			} else {
				entity.MetricValue = 0.0
			}
		default:
			// For complex types, store as string
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				entity.MetricString = ""
			} else {
				entity.MetricString = string(jsonBytes)
			}
		}

		entities = append(entities, entity)
	}

	return entities, nil
}

// FromEntity converts a TelemetryEntity to a MetricPoint
func (te *TelemetryEntity) FromEntity() (*MetricPoint, error) {
	var tags map[string]string
	if te.TagsJSON != "" {
		if err := json.Unmarshal([]byte(te.TagsJSON), &tags); err != nil {
			return nil, err
		}
	}

	var metricValue interface{}
	if te.MetricString != "" {
		metricValue = te.MetricString
	} else {
		metricValue = te.MetricValue
	}

	return &MetricPoint{
		Timestamp:   te.Timestamp,
		MetricName:  te.MetricName,
		MetricValue: metricValue,
		Tags:        tags,
	}, nil
}

// ToEntity converts an Alert to an AlertEntity
func (a *Alert) ToEntity() (*AlertEntity, error) {
	metadataJSON, err := json.Marshal(a.Metadata)
	if err != nil {
		return nil, err
	}

	entity := &AlertEntity{
		AlertID:        a.AlertID,
		DeviceID:       a.DeviceID,
		ThresholdID:    a.ThresholdID,
		MetricName:     a.MetricName,
		CurrentValue:   a.CurrentValue,
		ThresholdValue: a.ThresholdValue,
		Severity:       a.Severity,
		Message:        a.Message,
		TriggeredAt:    a.TriggeredAt,
		Status:         a.Status,
		MetadataJSON:   string(metadataJSON),
	}

	if a.AcknowledgedAt != nil {
		entity.AcknowledgedAt = *a.AcknowledgedAt
	}
	if a.ResolvedAt != nil {
		entity.ResolvedAt = *a.ResolvedAt
	}

	return entity, nil
}

// FromEntity converts an AlertEntity to an Alert
func (ae *AlertEntity) FromEntity() (*Alert, error) {
	var metadata map[string]interface{}
	if ae.MetadataJSON != "" {
		if err := json.Unmarshal([]byte(ae.MetadataJSON), &metadata); err != nil {
			return nil, err
		}
	}

	alert := &Alert{
		AlertID:        ae.AlertID,
		DeviceID:       ae.DeviceID,
		ThresholdID:    ae.ThresholdID,
		MetricName:     ae.MetricName,
		CurrentValue:   ae.CurrentValue,
		ThresholdValue: ae.ThresholdValue,
		Severity:       ae.Severity,
		Message:        ae.Message,
		TriggeredAt:    ae.TriggeredAt,
		Status:         ae.Status,
		Metadata:       metadata,
	}

	if !ae.AcknowledgedAt.IsZero() {
		alert.AcknowledgedAt = &ae.AcknowledgedAt
	}
	if !ae.ResolvedAt.IsZero() {
		alert.ResolvedAt = &ae.ResolvedAt
	}

	return alert, nil
}
