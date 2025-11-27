package telemetry

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/google/uuid"
)

// DatastoreRepository implements the Repository interface using Google Cloud Datastore
type DatastoreRepository struct {
	client *datastore.Client
}

// NewDatastoreRepository creates a new Datastore-backed telemetry repository
func NewDatastoreRepository(client *datastore.Client) *DatastoreRepository {
	return &DatastoreRepository{
		client: client,
	}
}

// StoreTelemetry stores telemetry data in Datastore
func (r *DatastoreRepository) StoreTelemetry(ctx context.Context, data *TelemetryData) error {
	entities, err := data.ToEntities()
	if err != nil {
		return fmt.Errorf("failed to convert telemetry data to entities: %w", err)
	}

	keys := make([]*datastore.Key, len(entities))
	for i, entity := range entities {
		// Key format: telemetry/{device_id}#{timestamp_nanos}#{metric_name}
		keyName := fmt.Sprintf("%s#%d#%s", entity.DeviceID, entity.Timestamp.UnixNano(), entity.MetricName)
		keys[i] = datastore.NameKey("Telemetry", keyName, nil)
	}

	if _, err := r.client.PutMulti(ctx, keys, entities); err != nil {
		return fmt.Errorf("failed to store telemetry data: %w", err)
	}

	return nil
}

// StoreTelemetryBatch stores multiple telemetry data points in a batch
func (r *DatastoreRepository) StoreTelemetryBatch(ctx context.Context, batch []*TelemetryData) error {
	var allEntities []*TelemetryEntity
	var allKeys []*datastore.Key

	for _, data := range batch {
		entities, err := data.ToEntities()
		if err != nil {
			return fmt.Errorf("failed to convert telemetry data to entities: %w", err)
		}

		for _, entity := range entities {
			keyName := fmt.Sprintf("%s#%d#%s", entity.DeviceID, entity.Timestamp.UnixNano(), entity.MetricName)
			key := datastore.NameKey("Telemetry", keyName, nil)
			
			allKeys = append(allKeys, key)
			allEntities = append(allEntities, entity)
		}
	}

	if len(allKeys) == 0 {
		return nil
	}

	// Datastore has a limit of 500 entities per batch
	batchSize := 500
	for i := 0; i < len(allKeys); i += batchSize {
		end := i + batchSize
		if end > len(allKeys) {
			end = len(allKeys)
		}

		if _, err := r.client.PutMulti(ctx, allKeys[i:end], allEntities[i:end]); err != nil {
			return fmt.Errorf("failed to store telemetry batch: %w", err)
		}
	}

	return nil
}

// GetDeviceMetrics retrieves all metrics for a device within a time range
func (r *DatastoreRepository) GetDeviceMetrics(ctx context.Context, deviceID string, timeRange TimeRange) ([]*MetricPoint, error) {
	query := datastore.NewQuery("Telemetry").
		Filter("device_id =", deviceID).
		Filter("timestamp >=", timeRange.Start).
		Filter("timestamp <=", timeRange.End).
		Order("timestamp")

	var entities []*TelemetryEntity
	if _, err := r.client.GetAll(ctx, query, &entities); err != nil {
		return nil, fmt.Errorf("failed to query telemetry data: %w", err)
	}

	metrics := make([]*MetricPoint, 0, len(entities))
	for _, entity := range entities {
		metric, err := entity.FromEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert entity to metric: %w", err)
		}
		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// GetDeviceMetricsByName retrieves specific metric for a device within a time range
func (r *DatastoreRepository) GetDeviceMetricsByName(ctx context.Context, deviceID string, metricName string, timeRange TimeRange) ([]*MetricPoint, error) {
	query := datastore.NewQuery("Telemetry").
		Filter("device_id =", deviceID).
		Filter("metric_name =", metricName).
		Filter("timestamp >=", timeRange.Start).
		Filter("timestamp <=", timeRange.End).
		Order("timestamp")

	var entities []*TelemetryEntity
	if _, err := r.client.GetAll(ctx, query, &entities); err != nil {
		return nil, fmt.Errorf("failed to query telemetry data: %w", err)
	}

	metrics := make([]*MetricPoint, 0, len(entities))
	for _, entity := range entities {
		metric, err := entity.FromEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert entity to metric: %w", err)
		}
		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// GetLatestMetrics retrieves the latest N metrics for a device
func (r *DatastoreRepository) GetLatestMetrics(ctx context.Context, deviceID string, limit int) ([]*MetricPoint, error) {
	query := datastore.NewQuery("Telemetry").
		Filter("device_id =", deviceID).
		Order("-timestamp").
		Limit(limit)

	var entities []*TelemetryEntity
	if _, err := r.client.GetAll(ctx, query, &entities); err != nil {
		return nil, fmt.Errorf("failed to query latest telemetry: %w", err)
	}

	metrics := make([]*MetricPoint, 0, len(entities))
	for _, entity := range entities {
		metric, err := entity.FromEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert entity to metric: %w", err)
		}
		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// AggregateMetrics performs aggregation on metrics
func (r *DatastoreRepository) AggregateMetrics(ctx context.Context, query *AggregationQuery) ([]*AggregationResult, error) {
	// Fetch raw data
	metrics, err := r.GetDeviceMetricsByName(ctx, query.DeviceID, query.MetricName, query.TimeRange)
	if err != nil {
		return nil, err
	}

	if len(metrics) == 0 {
		return []*AggregationResult{}, nil
	}

	// If no interval specified, aggregate all data into one result
	if query.Interval == 0 {
		value := r.aggregateValues(metrics, query.Aggregation)
		return []*AggregationResult{
			{
				Timestamp: query.TimeRange.Start,
				Value:     value,
			},
		}, nil
	}

	// Group by interval and aggregate
	buckets := make(map[time.Time][]*MetricPoint)
	for _, metric := range metrics {
		bucketTime := metric.Timestamp.Truncate(query.Interval)
		buckets[bucketTime] = append(buckets[bucketTime], metric)
	}

	results := make([]*AggregationResult, 0, len(buckets))
	for timestamp, points := range buckets {
		value := r.aggregateValues(points, query.Aggregation)
		results = append(results, &AggregationResult{
			Timestamp: timestamp,
			Value:     value,
		})
	}

	return results, nil
}

// aggregateValues performs the actual aggregation calculation
func (r *DatastoreRepository) aggregateValues(metrics []*MetricPoint, aggType AggregationType) float64 {
	if len(metrics) == 0 {
		return 0
	}

	values := make([]float64, 0, len(metrics))
	for _, m := range metrics {
		if v, ok := m.MetricValue.(float64); ok {
			values = append(values, v)
		}
	}

	if len(values) == 0 {
		return 0
	}

	switch aggType {
	case AggregationAvg:
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum / float64(len(values))
	case AggregationSum:
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum
	case AggregationMin:
		min := values[0]
		for _, v := range values {
			if v < min {
				min = v
			}
		}
		return min
	case AggregationMax:
		max := values[0]
		for _, v := range values {
			if v > max {
				max = v
			}
		}
		return max
	case AggregationCount:
		return float64(len(values))
	default:
		return 0
	}
}

// CreateThreshold creates a new alert threshold
func (r *DatastoreRepository) CreateThreshold(ctx context.Context, deviceID string, threshold *AlertThreshold) (string, error) {
	thresholdID := uuid.New().String()
	
	metadataJSON := "{}"
	if threshold.Metadata != nil {
		// Marshal metadata if present
		// For simplicity, we'll skip error handling here
	}

	entity := &ThresholdEntity{
		ThresholdID:   thresholdID,
		DeviceID:      deviceID,
		MetricName:    threshold.MetricName,
		Operator:      threshold.Operator,
		Value:         threshold.Value,
		DurationNanos: int64(threshold.Duration),
		Severity:      threshold.Severity,
		Enabled:       threshold.Enabled,
		MetadataJSON:  metadataJSON,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	key := datastore.NameKey("Threshold", thresholdID, nil)
	if _, err := r.client.Put(ctx, key, entity); err != nil {
		return "", fmt.Errorf("failed to create threshold: %w", err)
	}

	return thresholdID, nil
}

// GetThreshold retrieves a threshold by ID
func (r *DatastoreRepository) GetThreshold(ctx context.Context, thresholdID string) (*AlertThreshold, error) {
	key := datastore.NameKey("Threshold", thresholdID, nil)
	var entity ThresholdEntity
	
	if err := r.client.Get(ctx, key, &entity); err != nil {
		if err == datastore.ErrNoSuchEntity {
			return nil, fmt.Errorf("threshold not found")
		}
		return nil, fmt.Errorf("failed to get threshold: %w", err)
	}

	return &AlertThreshold{
		MetricName: entity.MetricName,
		Operator:   entity.Operator,
		Value:      entity.Value,
		Duration:   time.Duration(entity.DurationNanos),
		Severity:   entity.Severity,
		Enabled:    entity.Enabled,
	}, nil
}

// ListThresholds lists all thresholds for a device
func (r *DatastoreRepository) ListThresholds(ctx context.Context, deviceID string) ([]*AlertThreshold, error) {
	query := datastore.NewQuery("Threshold").
		Filter("device_id =", deviceID)

	var entities []*ThresholdEntity
	if _, err := r.client.GetAll(ctx, query, &entities); err != nil {
		return nil, fmt.Errorf("failed to list thresholds: %w", err)
	}

	thresholds := make([]*AlertThreshold, 0, len(entities))
	for _, entity := range entities {
		thresholds = append(thresholds, &AlertThreshold{
			MetricName: entity.MetricName,
			Operator:   entity.Operator,
			Value:      entity.Value,
			Duration:   time.Duration(entity.DurationNanos),
			Severity:   entity.Severity,
			Enabled:    entity.Enabled,
		})
	}

	return thresholds, nil
}

// UpdateThreshold updates an existing threshold
func (r *DatastoreRepository) UpdateThreshold(ctx context.Context, thresholdID string, threshold *AlertThreshold) error {
	key := datastore.NameKey("Threshold", thresholdID, nil)
	var entity ThresholdEntity
	
	if err := r.client.Get(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to get threshold: %w", err)
	}

	entity.MetricName = threshold.MetricName
	entity.Operator = threshold.Operator
	entity.Value = threshold.Value
	entity.DurationNanos = int64(threshold.Duration)
	entity.Severity = threshold.Severity
	entity.Enabled = threshold.Enabled
	entity.UpdatedAt = time.Now()

	if _, err := r.client.Put(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to update threshold: %w", err)
	}

	return nil
}

// DeleteThreshold deletes a threshold
func (r *DatastoreRepository) DeleteThreshold(ctx context.Context, thresholdID string) error {
	key := datastore.NameKey("Threshold", thresholdID, nil)
	if err := r.client.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete threshold: %w", err)
	}
	return nil
}

// CreateAlert creates a new alert
func (r *DatastoreRepository) CreateAlert(ctx context.Context, alert *Alert) error {
	entity, err := alert.ToEntity()
	if err != nil {
		return fmt.Errorf("failed to convert alert to entity: %w", err)
	}

	key := datastore.NameKey("Alert", alert.AlertID, nil)
	if _, err := r.client.Put(ctx, key, entity); err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}

	return nil
}

// GetAlert retrieves an alert by ID
func (r *DatastoreRepository) GetAlert(ctx context.Context, alertID string) (*Alert, error) {
	key := datastore.NameKey("Alert", alertID, nil)
	var entity AlertEntity
	
	if err := r.client.Get(ctx, key, &entity); err != nil {
		if err == datastore.ErrNoSuchEntity {
			return nil, fmt.Errorf("alert not found")
		}
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	return entity.FromEntity()
}

// ListAlerts lists alerts for a device, optionally filtered by status
func (r *DatastoreRepository) ListAlerts(ctx context.Context, deviceID string, status string) ([]*Alert, error) {
	query := datastore.NewQuery("Alert").
		Filter("device_id =", deviceID).
		Order("-triggered_at")

	if status != "" {
		query = query.Filter("status =", status)
	}

	var entities []*AlertEntity
	if _, err := r.client.GetAll(ctx, query, &entities); err != nil {
		return nil, fmt.Errorf("failed to list alerts: %w", err)
	}

	alerts := make([]*Alert, 0, len(entities))
	for _, entity := range entities {
		alert, err := entity.FromEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert entity to alert: %w", err)
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// AcknowledgeAlert marks an alert as acknowledged
func (r *DatastoreRepository) AcknowledgeAlert(ctx context.Context, alertID string) error {
	key := datastore.NameKey("Alert", alertID, nil)
	var entity AlertEntity
	
	if err := r.client.Get(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to get alert: %w", err)
	}

	now := time.Now()
	entity.AcknowledgedAt = now
	entity.Status = "acknowledged"

	if _, err := r.client.Put(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to acknowledge alert: %w", err)
	}

	return nil
}

// ResolveAlert marks an alert as resolved
func (r *DatastoreRepository) ResolveAlert(ctx context.Context, alertID string) error {
	key := datastore.NameKey("Alert", alertID, nil)
	var entity AlertEntity
	
	if err := r.client.Get(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to get alert: %w", err)
	}

	now := time.Now()
	entity.ResolvedAt = now
	entity.Status = "resolved"

	if _, err := r.client.Put(ctx, key, &entity); err != nil {
		return fmt.Errorf("failed to resolve alert: %w", err)
	}

	return nil
}

// DeleteOldTelemetry deletes telemetry data older than the specified time
func (r *DatastoreRepository) DeleteOldTelemetry(ctx context.Context, before time.Time) (int64, error) {
	query := datastore.NewQuery("Telemetry").
		Filter("timestamp <", before).
		KeysOnly()

	keys, err := r.client.GetAll(ctx, query, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to query old telemetry: %w", err)
	}

	if len(keys) == 0 {
		return 0, nil
	}

	// Delete in batches of 500
	batchSize := 500
	deleted := int64(0)
	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}

		if err := r.client.DeleteMulti(ctx, keys[i:end]); err != nil {
			return deleted, fmt.Errorf("failed to delete telemetry batch: %w", err)
		}
		deleted += int64(end - i)
	}

	return deleted, nil
}
