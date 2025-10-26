package ota

import (
	"github.com/athena/platform/pkg/config"
	"github.com/athena/platform/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Service represents the OTA service
type Service struct {
	config *config.Config
	logger *logger.Logger
}

// NewService creates a new OTA service instance
func NewService(cfg *config.Config, logger *logger.Logger) (*Service, error) {
	return &Service{
		config: cfg,
		logger: logger,
	}, nil
}

// RegisterRoutes registers HTTP routes for the OTA service
func RegisterRoutes(router *gin.Engine, service *Service) {
	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", service.healthCheck)
		v1.POST("/releases", service.createRelease)
		v1.GET("/updates/:deviceId", service.getUpdate)
	}
}

func (s *Service) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "healthy",
		"service": "ota-service",
	})
}

func (s *Service) createRelease(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Release creation placeholder",
	})
}

func (s *Service) getUpdate(c *gin.Context) {
	deviceID := c.Param("deviceId")
	c.JSON(200, gin.H{
		"device_id": deviceID,
		"message":   "Update check placeholder",
	})
}