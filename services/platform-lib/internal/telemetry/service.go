package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Service represents the telemetry service
type Service struct {
	config        *config.Config
	logger        *logger.Logger
	repository    Repository
	mqttClient    *MQTTClient
	streamManager *StreamManager
	exporter      *Exporter
	alertMonitor  *AlertMonitor
	alertNotifier *AlertNotifier
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewService creates a new telemetry service instance
func NewService(cfg *config.Config, logger *logger.Logger, repository Repository) (*Service, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize alert notifier with default log channel
	notificationChannels := []NotificationConfig{
		{
			Channel: ChannelLog,
			Enabled: true,
		},
	}
	alertNotifier := NewAlertNotifier(logger, notificationChannels)

	// Initialize alert monitor
	alertMonitor := NewAlertMonitor(repository, alertNotifier, logger)

	service := &Service{
		config:        cfg,
		logger:        logger,
		repository:    repository,
		streamManager: NewStreamManager(logger, repository),
		exporter:      NewExporter(repository),
		alertMonitor:  alertMonitor,
		alertNotifier: alertNotifier,
		ctx:           ctx,
		cancel:        cancel,
	}

	// Initialize MQTT client if configured
	if cfg.MQTT.Enabled {
		mqttConfig := &MQTTConfig{
			BrokerURL:      cfg.MQTT.BrokerURL,
			ClientID:       "telemetry-service",
			Username:       cfg.MQTT.Username,
			Password:       cfg.MQTT.Password,
			QoS:            1,
			CleanSession:   true,
			ConnectTimeout: 10 * time.Second,
			KeepAlive:      60 * time.Second,
		}

		mqttClient, err := NewMQTTClient(mqttConfig, repository, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create MQTT client: %w", err)
		}

		service.mqttClient = mqttClient
	}

	return service, nil
}

// Start starts the telemetry service
func (s *Service) Start() error {
	if s.mqttClient != nil {
		if err := s.mqttClient.Connect(); err != nil {
			return fmt.Errorf("failed to connect MQTT client: %w", err)
		}

		// Subscribe to telemetry topics
		if err := s.mqttClient.SubscribeToDeviceTelemetry(); err != nil {
			return fmt.Errorf("failed to subscribe to device telemetry: %w", err)
		}

		if err := s.mqttClient.SubscribeToDeviceHeartbeats(); err != nil {
			return fmt.Errorf("failed to subscribe to device heartbeats: %w", err)
		}

		s.logger.Info("Telemetry service started with MQTT support")
	} else {
		s.logger.Info("Telemetry service started (HTTP only)")
	}

	// Start alert monitoring
	if s.alertMonitor != nil {
		s.alertMonitor.Start(30 * time.Second) // Check every 30 seconds
		s.logger.Info("Alert monitoring started")
	}

	return nil
}

// Stop stops the telemetry service
func (s *Service) Stop() {
	s.cancel()
	if s.alertMonitor != nil {
		s.alertMonitor.Stop()
	}
	if s.mqttClient != nil {
		s.mqttClient.Disconnect()
	}
	if s.streamManager != nil {
		s.streamManager.CloseAllConnections()
	}
	s.logger.Info("Telemetry service stopped")
}

// IngestTelemetry ingests telemetry data via HTTP
func (s *Service) IngestTelemetry(deviceID string, data *TelemetryData) error {
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	// Set device ID and timestamp if not provided
	if data.DeviceID == "" {
		data.DeviceID = deviceID
	}
	if data.Timestamp.IsZero() {
		data.Timestamp = time.Now()
	}

	// Store telemetry
	if err := s.repository.StoreTelemetry(ctx, data); err != nil {
		return err
	}

	// Broadcast to WebSocket clients
	if s.streamManager != nil {
		s.streamManager.BroadcastTelemetry(deviceID, data)
	}

	return nil
}

// GetDeviceMetrics retrieves metrics for a device
func (s *Service) GetDeviceMetrics(deviceID string, timeRange TimeRange) ([]*MetricPoint, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	return s.repository.GetDeviceMetrics(ctx, deviceID, timeRange)
}

// StreamDeviceData creates a channel for streaming device data
func (s *Service) StreamDeviceData(deviceID string) (<-chan *TelemetryData, error) {
	// This would be implemented with WebSocket or SSE support
	// For now, return a placeholder channel
	ch := make(chan *TelemetryData)
	return ch, nil
}

// SetAlertThreshold sets an alert threshold for a device metric
func (s *Service) SetAlertThreshold(deviceID string, metric string, threshold *AlertThreshold) error {
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	_, err := s.repository.CreateThreshold(ctx, deviceID, threshold)
	return err
}

// RegisterRoutes registers HTTP routes for the telemetry service
func RegisterRoutes(router *gin.Engine, service *Service) {
	v1 := router.Group("/api/v1/telemetry")
	{
		v1.GET("/health", service.healthCheck)
		v1.POST("/ingest/:deviceId", service.ingestTelemetryHandler)
		v1.GET("/metrics/:deviceId", service.getMetricsHandler)
		v1.GET("/metrics/:deviceId/:metricName", service.getMetricByNameHandler)
		v1.POST("/aggregate", service.aggregateMetricsHandler)
		v1.POST("/thresholds/:deviceId", service.createThresholdHandler)
		v1.GET("/thresholds/:deviceId", service.listThresholdsHandler)
		v1.GET("/alerts/:deviceId", service.listAlertsHandler)
		v1.POST("/alerts/:alertId/acknowledge", service.acknowledgeAlertHandler)
		v1.POST("/alerts/:alertId/resolve", service.resolveAlertHandler)

		// Streaming endpoints
		v1.GET("/stream/:deviceId", service.streamDeviceDataHandler)

		// Export endpoints
		v1.POST("/export", service.exportDataHandler)
		v1.POST("/export/aggregated", service.exportAggregatedHandler)

		// Notification channel management
		v1.GET("/notifications/channels", service.listNotificationChannelsHandler)
		v1.POST("/notifications/channels", service.addNotificationChannelHandler)
		v1.PUT("/notifications/channels/:channel", service.updateNotificationChannelHandler)
		v1.DELETE("/notifications/channels/:channel", service.deleteNotificationChannelHandler)
	}
}

func (s *Service) healthCheck(c *gin.Context) {
	status := "healthy"
	mqttStatus := "disabled"

	if s.mqttClient != nil {
		if s.mqttClient.IsConnected() {
			mqttStatus = "connected"
		} else {
			mqttStatus = "disconnected"
			status = "degraded"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      status,
		"service":     "telemetry-service",
		"mqtt_status": mqttStatus,
	})
}

func (s *Service) ingestTelemetryHandler(c *gin.Context) {
	deviceID := c.Param("deviceId")

	var data TelemetryData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid telemetry data", "details": err.Error()})
		return
	}

	if err := s.IngestTelemetry(deviceID, &data); err != nil {
		s.logger.Error(fmt.Sprintf("Failed to ingest telemetry: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store telemetry data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Telemetry data ingested successfully"})
}

func (s *Service) getMetricsHandler(c *gin.Context) {
	deviceID := c.Param("deviceId")

	// Parse time range from query parameters
	startStr := c.DefaultQuery("start", time.Now().Add(-24*time.Hour).Format(time.RFC3339))
	endStr := c.DefaultQuery("end", time.Now().Format(time.RFC3339))

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start time format"})
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end time format"})
		return
	}

	timeRange := TimeRange{Start: start, End: end}
	metrics, err := s.GetDeviceMetrics(deviceID, timeRange)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to get metrics: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve metrics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id": deviceID,
		"metrics":   metrics,
		"count":     len(metrics),
	})
}

func (s *Service) getMetricByNameHandler(c *gin.Context) {
	deviceID := c.Param("deviceId")
	metricName := c.Param("metricName")

	startStr := c.DefaultQuery("start", time.Now().Add(-24*time.Hour).Format(time.RFC3339))
	endStr := c.DefaultQuery("end", time.Now().Format(time.RFC3339))

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start time format"})
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end time format"})
		return
	}

	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	timeRange := TimeRange{Start: start, End: end}
	metrics, err := s.repository.GetDeviceMetricsByName(ctx, deviceID, metricName, timeRange)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to get metric: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve metric"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id":   deviceID,
		"metric_name": metricName,
		"metrics":     metrics,
		"count":       len(metrics),
	})
}

func (s *Service) createThresholdHandler(c *gin.Context) {
	deviceID := c.Param("deviceId")

	var threshold AlertThreshold
	if err := c.ShouldBindJSON(&threshold); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid threshold data", "details": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	thresholdID, err := s.repository.CreateThreshold(ctx, deviceID, &threshold)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to create threshold: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create threshold"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"threshold_id": thresholdID,
		"message":      "Threshold created successfully",
	})
}

func (s *Service) listThresholdsHandler(c *gin.Context) {
	deviceID := c.Param("deviceId")

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	thresholds, err := s.repository.ListThresholds(ctx, deviceID)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to list thresholds: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve thresholds"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id":  deviceID,
		"thresholds": thresholds,
		"count":      len(thresholds),
	})
}

func (s *Service) listAlertsHandler(c *gin.Context) {
	deviceID := c.Param("deviceId")
	status := c.DefaultQuery("status", "")

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	alerts, err := s.repository.ListAlerts(ctx, deviceID, status)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to list alerts: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve alerts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id": deviceID,
		"alerts":    alerts,
		"count":     len(alerts),
	})
}

func (s *Service) acknowledgeAlertHandler(c *gin.Context) {
	alertID := c.Param("alertId")

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	if err := s.repository.AcknowledgeAlert(ctx, alertID); err != nil {
		s.logger.Error(fmt.Sprintf("Failed to acknowledge alert: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to acknowledge alert"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Alert acknowledged successfully"})
}

func (s *Service) resolveAlertHandler(c *gin.Context) {
	alertID := c.Param("alertId")

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	if err := s.repository.ResolveAlert(ctx, alertID); err != nil {
		s.logger.Error(fmt.Sprintf("Failed to resolve alert: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve alert"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Alert resolved successfully"})
}

func (s *Service) aggregateMetricsHandler(c *gin.Context) {
	var query AggregationQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid aggregation query", "details": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	results, err := s.repository.AggregateMetrics(ctx, &query)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Failed to aggregate metrics: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to aggregate metrics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id":   query.DeviceID,
		"metric_name": query.MetricName,
		"aggregation": query.Aggregation,
		"results":     results,
		"count":       len(results),
	})
}

func (s *Service) streamDeviceDataHandler(c *gin.Context) {
	if s.streamManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Streaming not available"})
		return
	}

	s.streamManager.HandleWebSocket(c)
}

func (s *Service) exportDataHandler(c *gin.Context) {
	var request ExportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid export request", "details": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	// Set content type based on format
	switch request.Format {
	case ExportFormatJSON:
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=telemetry_%s.json", request.DeviceID))
	case ExportFormatCSV:
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=telemetry_%s.csv", request.DeviceID))
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported export format"})
		return
	}

	if err := s.exporter.Export(ctx, &request, c.Writer); err != nil {
		s.logger.Error(fmt.Sprintf("Failed to export data: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export data"})
		return
	}
}

func (s *Service) exportAggregatedHandler(c *gin.Context) {
	var request struct {
		Query  AggregationQuery `json:"query"`
		Format ExportFormat     `json:"format"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid export request", "details": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	// Set content type based on format
	switch request.Format {
	case ExportFormatJSON:
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=aggregated_%s_%s.json", request.Query.DeviceID, request.Query.MetricName))
	case ExportFormatCSV:
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=aggregated_%s_%s.csv", request.Query.DeviceID, request.Query.MetricName))
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported export format"})
		return
	}

	if err := s.exporter.ExportAggregated(ctx, &request.Query, request.Format, c.Writer); err != nil {
		s.logger.Error(fmt.Sprintf("Failed to export aggregated data: %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export aggregated data"})
		return
	}
}

func (s *Service) listNotificationChannelsHandler(c *gin.Context) {
	if s.alertNotifier == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Alert notifications not available"})
		return
	}

	channels := s.alertNotifier.GetChannels()
	c.JSON(http.StatusOK, gin.H{
		"channels": channels,
		"count":    len(channels),
	})
}

func (s *Service) addNotificationChannelHandler(c *gin.Context) {
	if s.alertNotifier == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Alert notifications not available"})
		return
	}

	var config NotificationConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification channel config", "details": err.Error()})
		return
	}

	s.alertNotifier.AddChannel(config)
	c.JSON(http.StatusCreated, gin.H{"message": "Notification channel added successfully"})
}

func (s *Service) updateNotificationChannelHandler(c *gin.Context) {
	if s.alertNotifier == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Alert notifications not available"})
		return
	}

	channel := NotificationChannel(c.Param("channel"))

	var settings map[string]interface{}
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid settings", "details": err.Error()})
		return
	}

	s.alertNotifier.UpdateChannel(channel, settings)
	c.JSON(http.StatusOK, gin.H{"message": "Notification channel updated successfully"})
}

func (s *Service) deleteNotificationChannelHandler(c *gin.Context) {
	if s.alertNotifier == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Alert notifications not available"})
		return
	}

	channel := NotificationChannel(c.Param("channel"))
	s.alertNotifier.RemoveChannel(channel)
	c.JSON(http.StatusOK, gin.H{"message": "Notification channel deleted successfully"})
}
