package telemetry

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// ExportFormat represents the format for data export
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatCSV  ExportFormat = "csv"
)

// ExportRequest represents a request to export telemetry data
type ExportRequest struct {
	DeviceID   string       `json:"device_id"`
	MetricName string       `json:"metric_name,omitempty"`
	TimeRange  TimeRange    `json:"time_range"`
	Format     ExportFormat `json:"format"`
}

// Exporter handles telemetry data export
type Exporter struct {
	repository Repository
}

// NewExporter creates a new exporter
func NewExporter(repository Repository) *Exporter {
	return &Exporter{
		repository: repository,
	}
}

// Export exports telemetry data in the specified format
func (e *Exporter) Export(ctx context.Context, request *ExportRequest, writer io.Writer) error {
	var metrics []*MetricPoint
	var err error

	// Fetch data
	if request.MetricName != "" {
		metrics, err = e.repository.GetDeviceMetricsByName(ctx, request.DeviceID, request.MetricName, request.TimeRange)
	} else {
		metrics, err = e.repository.GetDeviceMetrics(ctx, request.DeviceID, request.TimeRange)
	}

	if err != nil {
		return fmt.Errorf("failed to fetch metrics: %w", err)
	}

	// Export based on format
	switch request.Format {
	case ExportFormatJSON:
		return e.exportJSON(metrics, writer)
	case ExportFormatCSV:
		return e.exportCSV(metrics, writer)
	default:
		return fmt.Errorf("unsupported export format: %s", request.Format)
	}
}

// exportJSON exports metrics as JSON
func (e *Exporter) exportJSON(metrics []*MetricPoint, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	
	data := map[string]interface{}{
		"exported_at": time.Now().Format(time.RFC3339),
		"count":       len(metrics),
		"metrics":     metrics,
	}

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// exportCSV exports metrics as CSV
func (e *Exporter) exportCSV(metrics []*MetricPoint, writer io.Writer) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Write header
	header := []string{"timestamp", "metric_name", "metric_value", "tags"}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, metric := range metrics {
		tagsJSON, _ := json.Marshal(metric.Tags)
		
		row := []string{
			metric.Timestamp.Format(time.RFC3339),
			metric.MetricName,
			fmt.Sprintf("%v", metric.MetricValue),
			string(tagsJSON),
		}

		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// ExportAggregated exports aggregated metrics
func (e *Exporter) ExportAggregated(ctx context.Context, query *AggregationQuery, format ExportFormat, writer io.Writer) error {
	results, err := e.repository.AggregateMetrics(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to aggregate metrics: %w", err)
	}

	switch format {
	case ExportFormatJSON:
		return e.exportAggregatedJSON(results, query, writer)
	case ExportFormatCSV:
		return e.exportAggregatedCSV(results, query, writer)
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}

// exportAggregatedJSON exports aggregated results as JSON
func (e *Exporter) exportAggregatedJSON(results []*AggregationResult, query *AggregationQuery, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	
	data := map[string]interface{}{
		"exported_at": time.Now().Format(time.RFC3339),
		"device_id":   query.DeviceID,
		"metric_name": query.MetricName,
		"aggregation": query.Aggregation,
		"time_range":  query.TimeRange,
		"interval":    query.Interval.String(),
		"count":       len(results),
		"results":     results,
	}

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

// exportAggregatedCSV exports aggregated results as CSV
func (e *Exporter) exportAggregatedCSV(results []*AggregationResult, query *AggregationQuery, writer io.Writer) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Write header
	header := []string{"timestamp", "value"}
	if err := csvWriter.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, result := range results {
		row := []string{
			result.Timestamp.Format(time.RFC3339),
			fmt.Sprintf("%f", result.Value),
		}

		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}
