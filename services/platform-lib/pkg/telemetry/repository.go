package telemetry

import (
	"context"
	"time"
)

// Repository defines the interface for telemetry data operations
type Repository interface {
	// Telemetry ingestion
	StoreTelemetry(ctx context.Context, data *TelemetryData) error
	StoreTelemetryBatch(ctx context.Context, batch []*TelemetryData) error

	// Telemetry queries
	GetDeviceMetrics(ctx context.Context, deviceID string, timeRange TimeRange) ([]*MetricPoint, error)
	GetDeviceMetricsByName(ctx context.Context, deviceID string, metricName string, timeRange TimeRange) ([]*MetricPoint, error)
	GetLatestMetrics(ctx context.Context, deviceID string, limit int) ([]*MetricPoint, error)

	// Aggregation queries
	AggregateMetrics(ctx context.Context, query *AggregationQuery) ([]*AggregationResult, error)

	// Alert threshold management
	CreateThreshold(ctx context.Context, deviceID string, threshold *AlertThreshold) (string, error)
	GetThreshold(ctx context.Context, thresholdID string) (*AlertThreshold, error)
	ListThresholds(ctx context.Context, deviceID string) ([]*AlertThreshold, error)
	UpdateThreshold(ctx context.Context, thresholdID string, threshold *AlertThreshold) error
	DeleteThreshold(ctx context.Context, thresholdID string) error

	// Alert management
	CreateAlert(ctx context.Context, alert *Alert) error
	GetAlert(ctx context.Context, alertID string) (*Alert, error)
	ListAlerts(ctx context.Context, deviceID string, status string) ([]*Alert, error)
	AcknowledgeAlert(ctx context.Context, alertID string) error
	ResolveAlert(ctx context.Context, alertID string) error

	// Cleanup operations
	DeleteOldTelemetry(ctx context.Context, before time.Time) (int64, error)
}
