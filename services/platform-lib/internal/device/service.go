package device

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Service represents the device service
type Service struct {
	config     *config.Config
	logger     *logger.Logger
	repository Repository
	monitoring MonitoringServiceInterface
}

// NewService creates a new device service instance
func NewService(cfg *config.Config, logger *logger.Logger, repository Repository) (*Service, error) {
	// Initialize monitoring service
	monitoringConfig := DefaultMonitoringConfig()
	monitoring := NewMonitoringService(repository, logger, monitoringConfig)

	service := &Service{
		config:     cfg,
		logger:     logger,
		repository: repository,
		monitoring: monitoring,
	}

	// Start monitoring service
	ctx := context.Background()
	if err := monitoring.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start monitoring service: %w", err)
	}

	return service, nil
}

// RegisterRoutes registers HTTP routes for the device service
func RegisterRoutes(router *gin.Engine, service *Service) {
	v1 := router.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", service.healthCheck)

		// Device CRUD operations
		v1.POST("/devices", service.registerDevice)
		v1.GET("/devices", service.listDevices)
		v1.GET("/devices/:id", service.getDevice)
		v1.PUT("/devices/:id", service.updateDevice)
		v1.DELETE("/devices/:id", service.deleteDevice)

		// Device status operations
		v1.PUT("/devices/:id/status", service.updateDeviceStatus)
		v1.POST("/devices/:id/heartbeat", service.deviceHeartbeat)

		// Device monitoring and health
		v1.GET("/devices/health", service.getDeviceHealth)
		v1.GET("/devices/status/:status", service.getDevicesByStatus)
		v1.GET("/devices/offline", service.getOfflineDevices)
		v1.GET("/devices/:id/uptime", service.getDeviceUptime)
		v1.GET("/devices/:id/last-seen", service.getDeviceLastSeen)
		v1.GET("/monitoring/config", service.getMonitoringConfig)
		v1.PUT("/monitoring/config", service.updateMonitoringConfig)

		// Device search and filtering
		v1.GET("/devices/search", service.searchDevices)
		v1.GET("/devices/template/:templateId", service.getDevicesByTemplate)
		v1.GET("/devices/ota-channel/:channel", service.getDevicesByOTAChannel)
	}
}

func (s *Service) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "device-service",
	})
}

func (s *Service) registerDevice(c *gin.Context) {
	var req DeviceRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.logger.Errorf("Invalid device registration request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Create device from request
	device := FromRegistrationRequest(&req)

	// Register device
	ctx := context.Background()
	if err := s.repository.RegisterDevice(ctx, device); err != nil {
		s.logger.Errorf("Failed to register device %s: %v", req.DeviceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to register device",
			"details": err.Error(),
		})
		return
	}

	s.logger.Infof("Device %s registered successfully", device.DeviceID)
	c.JSON(http.StatusCreated, device)
}

func (s *Service) listDevices(c *gin.Context) {
	// Parse query parameters
	filters := &DeviceFilters{}

	if status := c.Query("status"); status != "" {
		filters.Status = DeviceStatus(status)
	}
	if boardType := c.Query("board_type"); boardType != "" {
		filters.BoardType = boardType
	}
	if templateID := c.Query("template_id"); templateID != "" {
		filters.TemplateID = templateID
	}
	if otaChannel := c.Query("ota_channel"); otaChannel != "" {
		filters.OTAChannel = otaChannel
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filters.Limit = limit
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filters.Offset = offset
		}
	}

	// Set default limit if not specified
	if filters.Limit == 0 {
		filters.Limit = 50
	}

	ctx := context.Background()

	// Get devices
	devices, err := s.repository.ListDevices(ctx, filters)
	if err != nil {
		s.logger.Errorf("Failed to list devices: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list devices",
			"details": err.Error(),
		})
		return
	}

	// Get total count
	total, err := s.repository.GetDeviceCount(ctx, filters)
	if err != nil {
		s.logger.Errorf("Failed to get device count: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get device count",
			"details": err.Error(),
		})
		return
	}

	// Convert to response format
	deviceList := make([]Device, len(devices))
	for i, device := range devices {
		deviceList[i] = *device
	}

	response := DeviceListResponse{
		Devices: deviceList,
		Total:   total,
		Limit:   filters.Limit,
		Offset:  filters.Offset,
	}

	c.JSON(http.StatusOK, response)
}

func (s *Service) getDevice(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device ID is required",
		})
		return
	}

	ctx := context.Background()
	device, err := s.repository.GetDevice(ctx, deviceID)
	if err != nil {
		s.logger.Errorf("Failed to get device %s: %v", deviceID, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Device not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, device)
}

func (s *Service) updateDevice(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device ID is required",
		})
		return
	}

	var device Device
	if err := c.ShouldBindJSON(&device); err != nil {
		s.logger.Errorf("Invalid device update request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Ensure device ID matches URL parameter
	device.DeviceID = deviceID

	ctx := context.Background()
	if err := s.repository.UpdateDevice(ctx, &device); err != nil {
		s.logger.Errorf("Failed to update device %s: %v", deviceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update device",
			"details": err.Error(),
		})
		return
	}

	s.logger.Infof("Device %s updated successfully", deviceID)
	c.JSON(http.StatusOK, device)
}

func (s *Service) deleteDevice(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device ID is required",
		})
		return
	}

	ctx := context.Background()
	if err := s.repository.DeleteDevice(ctx, deviceID); err != nil {
		s.logger.Errorf("Failed to delete device %s: %v", deviceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete device",
			"details": err.Error(),
		})
		return
	}

	s.logger.Infof("Device %s deleted successfully", deviceID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Device deleted successfully",
	})
}

func (s *Service) updateDeviceStatus(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device ID is required",
		})
		return
	}

	var statusUpdate DeviceStatusUpdate
	if err := c.ShouldBindJSON(&statusUpdate); err != nil {
		s.logger.Errorf("Invalid status update request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Use provided timestamp or current time
	lastSeen := time.Now()
	if statusUpdate.LastSeen != nil {
		lastSeen = *statusUpdate.LastSeen
	}

	ctx := context.Background()
	if err := s.repository.UpdateDeviceStatus(ctx, deviceID, statusUpdate.Status, lastSeen); err != nil {
		s.logger.Errorf("Failed to update device status for %s: %v", deviceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update device status",
			"details": err.Error(),
		})
		return
	}

	s.logger.Infof("Device %s status updated to %s", deviceID, statusUpdate.Status)
	c.JSON(http.StatusOK, gin.H{
		"message": "Device status updated successfully",
	})
}

func (s *Service) deviceHeartbeat(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device ID is required",
		})
		return
	}

	var heartbeat DeviceHeartbeat
	if err := c.ShouldBindJSON(&heartbeat); err != nil {
		s.logger.Errorf("Invalid heartbeat request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Ensure device ID matches URL parameter
	heartbeat.DeviceID = deviceID

	// Use monitoring service to process heartbeat
	ctx := context.Background()
	if err := s.monitoring.ProcessHeartbeat(ctx, &heartbeat); err != nil {
		s.logger.Errorf("Failed to process heartbeat for device %s: %v", deviceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process heartbeat",
			"details": err.Error(),
		})
		return
	}

	s.logger.Debugf("Heartbeat received from device %s", deviceID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Heartbeat processed successfully",
	})
}

func (s *Service) getDeviceHealth(c *gin.Context) {
	ctx := context.Background()
	health, err := s.repository.GetDeviceHealthStatus(ctx)
	if err != nil {
		s.logger.Errorf("Failed to get device health status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get device health status",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, health)
}

func (s *Service) getDevicesByStatus(c *gin.Context) {
	status := DeviceStatus(c.Param("status"))
	if status == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Status is required",
		})
		return
	}

	ctx := context.Background()
	devices, err := s.repository.GetDevicesByStatus(ctx, status)
	if err != nil {
		s.logger.Errorf("Failed to get devices by status %s: %v", status, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get devices by status",
			"details": err.Error(),
		})
		return
	}

	// Convert to response format
	deviceList := make([]Device, len(devices))
	for i, device := range devices {
		deviceList[i] = *device
	}

	c.JSON(http.StatusOK, gin.H{
		"devices": deviceList,
		"status":  status,
		"count":   len(deviceList),
	})
}

func (s *Service) getOfflineDevices(c *gin.Context) {
	// Parse timeout parameter (default to 5 minutes)
	timeoutStr := c.DefaultQuery("timeout", "5m")
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid timeout format",
			"details": "Use duration format like '5m', '1h', '30s'",
		})
		return
	}

	ctx := context.Background()
	devices, err := s.repository.GetOfflineDevices(ctx, timeout)
	if err != nil {
		s.logger.Errorf("Failed to get offline devices: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get offline devices",
			"details": err.Error(),
		})
		return
	}

	// Convert to response format
	deviceList := make([]Device, len(devices))
	for i, device := range devices {
		deviceList[i] = *device
	}

	c.JSON(http.StatusOK, gin.H{
		"devices": deviceList,
		"timeout": timeoutStr,
		"count":   len(deviceList),
	})
}

func (s *Service) searchDevices(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Search query is required",
		})
		return
	}

	// Parse additional filters
	filters := &DeviceFilters{}
	if status := c.Query("status"); status != "" {
		filters.Status = DeviceStatus(status)
	}
	if boardType := c.Query("board_type"); boardType != "" {
		filters.BoardType = boardType
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filters.Limit = limit
		}
	}

	// Set default limit
	if filters.Limit == 0 {
		filters.Limit = 50
	}

	ctx := context.Background()
	devices, err := s.repository.SearchDevices(ctx, query, filters)
	if err != nil {
		s.logger.Errorf("Failed to search devices: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to search devices",
			"details": err.Error(),
		})
		return
	}

	// Convert to response format
	deviceList := make([]Device, len(devices))
	for i, device := range devices {
		deviceList[i] = *device
	}

	c.JSON(http.StatusOK, gin.H{
		"devices": deviceList,
		"query":   query,
		"count":   len(deviceList),
	})
}

func (s *Service) getDevicesByTemplate(c *gin.Context) {
	templateID := c.Param("templateId")
	if templateID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Template ID is required",
		})
		return
	}

	ctx := context.Background()
	devices, err := s.repository.GetDevicesByTemplate(ctx, templateID)
	if err != nil {
		s.logger.Errorf("Failed to get devices by template %s: %v", templateID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get devices by template",
			"details": err.Error(),
		})
		return
	}

	// Convert to response format
	deviceList := make([]Device, len(devices))
	for i, device := range devices {
		deviceList[i] = *device
	}

	c.JSON(http.StatusOK, gin.H{
		"devices":     deviceList,
		"template_id": templateID,
		"count":       len(deviceList),
	})
}

func (s *Service) getDevicesByOTAChannel(c *gin.Context) {
	channel := c.Param("channel")
	if channel == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "OTA channel is required",
		})
		return
	}

	ctx := context.Background()
	devices, err := s.repository.GetDevicesByOTAChannel(ctx, channel)
	if err != nil {
		s.logger.Errorf("Failed to get devices by OTA channel %s: %v", channel, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get devices by OTA channel",
			"details": err.Error(),
		})
		return
	}

	// Convert to response format
	deviceList := make([]Device, len(devices))
	for i, device := range devices {
		deviceList[i] = *device
	}

	c.JSON(http.StatusOK, gin.H{
		"devices":     deviceList,
		"ota_channel": channel,
		"count":       len(deviceList),
	})
}

func (s *Service) getDeviceUptime(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device ID is required",
		})
		return
	}

	ctx := context.Background()
	uptime, err := s.monitoring.GetDeviceUptime(ctx, deviceID)
	if err != nil {
		s.logger.Errorf("Failed to get device uptime for %s: %v", deviceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get device uptime",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id":      deviceID,
		"uptime":         uptime.String(),
		"uptime_seconds": uptime.Seconds(),
	})
}

func (s *Service) getDeviceLastSeen(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Device ID is required",
		})
		return
	}

	ctx := context.Background()
	lastSeenDuration, err := s.monitoring.GetDeviceLastSeenDuration(ctx, deviceID)
	if err != nil {
		s.logger.Errorf("Failed to get device last seen duration for %s: %v", deviceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get device last seen duration",
			"details": err.Error(),
		})
		return
	}

	isOnline, err := s.monitoring.CheckDeviceOnlineStatus(ctx, deviceID)
	if err != nil {
		s.logger.Errorf("Failed to check device online status for %s: %v", deviceID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to check device online status",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id":          deviceID,
		"last_seen_duration": lastSeenDuration.String(),
		"last_seen_seconds":  lastSeenDuration.Seconds(),
		"is_online":          isOnline,
	})
}

func (s *Service) getMonitoringConfig(c *gin.Context) {
	config := s.monitoring.GetConfiguration()

	c.JSON(http.StatusOK, gin.H{
		"offline_timeout": config.OfflineTimeout.String(),
		"check_interval":  config.CheckInterval.String(),
		"is_running":      s.monitoring.IsRunning(),
	})
}

func (s *Service) updateMonitoringConfig(c *gin.Context) {
	var configUpdate struct {
		OfflineTimeout string `json:"offline_timeout,omitempty"`
		CheckInterval  string `json:"check_interval,omitempty"`
	}

	if err := c.ShouldBindJSON(&configUpdate); err != nil {
		s.logger.Errorf("Invalid monitoring config update request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Update offline timeout if provided
	if configUpdate.OfflineTimeout != "" {
		timeout, err := time.ParseDuration(configUpdate.OfflineTimeout)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid offline timeout format",
				"details": "Use duration format like '5m', '1h', '30s'",
			})
			return
		}
		s.monitoring.SetOfflineTimeout(timeout)
	}

	// Update check interval if provided
	if configUpdate.CheckInterval != "" {
		interval, err := time.ParseDuration(configUpdate.CheckInterval)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid check interval format",
				"details": "Use duration format like '1m', '30s', '2h'",
			})
			return
		}
		s.monitoring.SetCheckInterval(interval)
	}

	// Return updated configuration
	config := s.monitoring.GetConfiguration()
	s.logger.Infof("Monitoring configuration updated")

	c.JSON(http.StatusOK, gin.H{
		"message":         "Monitoring configuration updated successfully",
		"offline_timeout": config.OfflineTimeout.String(),
		"check_interval":  config.CheckInterval.String(),
		"is_running":      s.monitoring.IsRunning(),
	})
}

// Shutdown gracefully shuts down the device service
func (s *Service) Shutdown() error {
	s.logger.Info("Shutting down device service...")

	if err := s.monitoring.Stop(); err != nil {
		s.logger.Errorf("Failed to stop monitoring service: %v", err)
		return err
	}

	s.logger.Info("Device service shutdown complete")
	return nil
}
