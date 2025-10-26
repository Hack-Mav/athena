package provisioning

import (
	"github.com/athena/platform/pkg/config"
	"github.com/athena/platform/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Service represents the provisioning service
type Service struct {
	config *config.Config
	logger *logger.Logger
}

// NewService creates a new provisioning service instance
func NewService(cfg *config.Config, logger *logger.Logger) (*Service, error) {
	return &Service{
		config: cfg,
		logger: logger,
	}, nil
}

// RegisterRoutes registers HTTP routes for the provisioning service
func RegisterRoutes(router *gin.Engine, service *Service) {
	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", service.healthCheck)
		v1.POST("/compile", service.compileTemplate)
		v1.POST("/flash", service.flashDevice)
	}
}

func (s *Service) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "healthy",
		"service": "provisioning-service",
	})
}

func (s *Service) compileTemplate(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Compilation placeholder",
	})
}

func (s *Service) flashDevice(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "Flashing placeholder",
	})
}