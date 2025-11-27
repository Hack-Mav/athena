package ota

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/athena/platform-lib/internal/device"
	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Service represents the OTA service
type Service struct {
	config           *config.Config
	logger           *logger.Logger
	repository       Repository
	deviceRepository device.Repository
	signer           *Signer
	storageBackend   StorageBackend
}

// StorageBackend defines the interface for binary storage
type StorageBackend interface {
	StoreBinary(ctx context.Context, releaseID string, data []byte) (string, error)
	GetBinary(ctx context.Context, path string) ([]byte, error)
	GetBinaryURL(ctx context.Context, path string, expiry time.Duration) (string, error)
	DeleteBinary(ctx context.Context, path string) error
}

// NewService creates a new OTA service instance
func NewService(cfg *config.Config, logger *logger.Logger, repo Repository, deviceRepo device.Repository, signer *Signer, storage StorageBackend) (*Service, error) {
	return &Service{
		config:           cfg,
		logger:           logger,
		repository:       repo,
		deviceRepository: deviceRepo,
		signer:           signer,
		storageBackend:   storage,
	}, nil
}

// CreateRelease creates a new firmware release with signing
func (s *Service) CreateRelease(ctx context.Context, req *CreateReleaseRequest) (*FirmwareRelease, error) {
	// Validate request
	if req.TemplateID == "" || req.Version == "" || len(req.BinaryData) == 0 {
		return nil, fmt.Errorf("template ID, version, and binary data are required")
	}

	// Validate channel
	if req.Channel != ReleaseChannelStable && req.Channel != ReleaseChannelBeta && req.Channel != ReleaseChannelAlpha {
		return nil, fmt.Errorf("invalid release channel: %s", req.Channel)
	}

	// Generate release ID
	releaseID := uuid.New().String()

	// Compute binary hash
	binaryHash := ComputeHash(req.BinaryData)

	// Sign the binary
	signature, err := s.signer.SignBinary(req.BinaryData)
	if err != nil {
		return nil, fmt.Errorf("failed to sign binary: %w", err)
	}

	// Store binary in storage backend
	binaryPath, err := s.storageBackend.StoreBinary(ctx, releaseID, req.BinaryData)
	if err != nil {
		return nil, fmt.Errorf("failed to store binary: %w", err)
	}

	// Create release object
	release := &FirmwareRelease{
		ReleaseID:    releaseID,
		TemplateID:   req.TemplateID,
		Version:      req.Version,
		Channel:      req.Channel,
		BinaryHash:   binaryHash,
		BinaryPath:   binaryPath,
		BinarySize:   int64(len(req.BinaryData)),
		Signature:    signature,
		ReleaseNotes: req.ReleaseNotes,
		CreatedAt:    time.Now(),
		CreatedBy:    req.CreatedBy,
	}

	// Store release metadata in repository
	err = s.repository.CreateRelease(ctx, release)
	if err != nil {
		// Clean up binary if metadata storage fails
		_ = s.storageBackend.DeleteBinary(ctx, binaryPath)
		return nil, fmt.Errorf("failed to create release: %w", err)
	}

	s.logger.Info("Created firmware release", "release_id", releaseID, "template_id", req.TemplateID, "version", req.Version)

	return release, nil
}

// GetRelease retrieves a firmware release by ID
func (s *Service) GetRelease(ctx context.Context, releaseID string) (*FirmwareRelease, error) {
	return s.repository.GetRelease(ctx, releaseID)
}

// ListReleases lists firmware releases for a template and channel
func (s *Service) ListReleases(ctx context.Context, templateID string, channel ReleaseChannel) ([]*FirmwareRelease, error) {
	return s.repository.ListReleases(ctx, templateID, channel)
}

// DeleteRelease deletes a firmware release
func (s *Service) DeleteRelease(ctx context.Context, releaseID string) error {
	// Get release to find binary path
	release, err := s.repository.GetRelease(ctx, releaseID)
	if err != nil {
		return fmt.Errorf("failed to get release: %w", err)
	}

	// Delete binary from storage
	err = s.storageBackend.DeleteBinary(ctx, release.BinaryPath)
	if err != nil {
		s.logger.Warn("Failed to delete binary from storage", "error", err)
	}

	// Delete release metadata
	err = s.repository.DeleteRelease(ctx, releaseID)
	if err != nil {
		return fmt.Errorf("failed to delete release: %w", err)
	}

	s.logger.Info("Deleted firmware release", "release_id", releaseID)

	return nil
}

// VerifyRelease verifies the signature of a firmware release
func (s *Service) VerifyRelease(ctx context.Context, releaseID string) error {
	// Get release metadata
	release, err := s.repository.GetRelease(ctx, releaseID)
	if err != nil {
		return fmt.Errorf("failed to get release: %w", err)
	}

	// Get binary data
	binaryData, err := s.storageBackend.GetBinary(ctx, release.BinaryPath)
	if err != nil {
		return fmt.Errorf("failed to get binary: %w", err)
	}

	// Verify hash
	computedHash := ComputeHash(binaryData)
	if computedHash != release.BinaryHash {
		return fmt.Errorf("binary hash mismatch: expected %s, got %s", release.BinaryHash, computedHash)
	}

	// Verify signature
	err = s.signer.VerifySignature(binaryData, release.Signature)
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// RegisterRoutes registers HTTP routes for the OTA service
func RegisterRoutes(router *gin.Engine, service *Service) {
	v1 := router.Group("/api/v1/ota")
	{
		v1.GET("/health", service.healthCheck)

		// Release management
		v1.POST("/releases", service.createReleaseHandler)
		v1.GET("/releases/:releaseId", service.getReleaseHandler)
		v1.GET("/releases", service.listReleasesHandler)
		v1.DELETE("/releases/:releaseId", service.deleteReleaseHandler)
		v1.POST("/releases/:releaseId/verify", service.verifyReleaseHandler)

		// Deployment management
		v1.POST("/deployments", service.createDeploymentHandler)
		v1.GET("/deployments/:deploymentId", service.getDeploymentHandler)
		v1.PUT("/deployments/:deploymentId/pause", service.pauseDeploymentHandler)
		v1.PUT("/deployments/:deploymentId/resume", service.resumeDeploymentHandler)
		v1.POST("/deployments/:deploymentId/rollback", service.rollbackDeploymentHandler)

		// Device update endpoints
		v1.GET("/updates/:deviceId", service.getUpdateForDeviceHandler)
		v1.POST("/updates/status", service.reportUpdateStatusHandler)
	}
}

func (s *Service) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "ota-service",
	})
}

func (s *Service) createReleaseHandler(c *gin.Context) {
	var req CreateReleaseRequest

	// Parse multipart form
	err := c.Request.ParseMultipartForm(100 << 20) // 100 MB max
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}

	// Get form fields
	req.TemplateID = c.PostForm("template_id")
	req.Version = c.PostForm("version")
	req.Channel = ReleaseChannel(c.PostForm("channel"))
	req.ReleaseNotes = c.PostForm("release_notes")
	req.CreatedBy = c.PostForm("created_by")

	// Get binary file
	file, _, err := c.Request.FormFile("binary")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "binary file is required"})
		return
	}
	defer file.Close()

	// Read binary data
	binaryData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read binary file"})
		return
	}

	req.BinaryData = binaryData

	// Create release
	release, err := s.CreateRelease(c.Request.Context(), &req)
	if err != nil {
		s.logger.Error("Failed to create release", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, release)
}

func (s *Service) getReleaseHandler(c *gin.Context) {
	releaseID := c.Param("releaseId")

	release, err := s.GetRelease(c.Request.Context(), releaseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, release)
}

func (s *Service) listReleasesHandler(c *gin.Context) {
	templateID := c.Query("template_id")
	channel := ReleaseChannel(c.Query("channel"))

	releases, err := s.ListReleases(c.Request.Context(), templateID, channel)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"releases": releases})
}

func (s *Service) deleteReleaseHandler(c *gin.Context) {
	releaseID := c.Param("releaseId")

	err := s.DeleteRelease(c.Request.Context(), releaseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "release deleted successfully"})
}

func (s *Service) verifyReleaseHandler(c *gin.Context) {
	releaseID := c.Param("releaseId")

	err := s.VerifyRelease(c.Request.Context(), releaseID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "verified": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"verified": true, "message": "release signature verified successfully"})
}

// Deployment handlers
func (s *Service) createDeploymentHandler(c *gin.Context) {
	var req struct {
		ReleaseID string            `json:"release_id" binding:"required"`
		Config    *DeploymentConfig `json:"config" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deployment, err := s.DeployRelease(c.Request.Context(), req.ReleaseID, req.Config)
	if err != nil {
		s.logger.Error("Failed to create deployment", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, deployment)
}

func (s *Service) getDeploymentHandler(c *gin.Context) {
	deploymentID := c.Param("deploymentId")

	// Check if status query parameter is present
	if c.Query("status") == "true" {
		// Return detailed status report
		statusReport, err := s.GetDeploymentStatus(c.Request.Context(), deploymentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, statusReport)
		return
	}

	// Return basic deployment info
	deployment, err := s.GetDeployment(c.Request.Context(), deploymentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

func (s *Service) pauseDeploymentHandler(c *gin.Context) {
	deploymentID := c.Param("deploymentId")

	err := s.PauseDeployment(c.Request.Context(), deploymentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deployment paused successfully", "deployment_id": deploymentID})
}

func (s *Service) resumeDeploymentHandler(c *gin.Context) {
	deploymentID := c.Param("deploymentId")

	err := s.ResumeDeployment(c.Request.Context(), deploymentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deployment resumed successfully", "deployment_id": deploymentID})
}

func (s *Service) rollbackDeploymentHandler(c *gin.Context) {
	deploymentID := c.Param("deploymentId")

	err := s.RollbackDeployment(c.Request.Context(), deploymentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deployment rolled back successfully", "deployment_id": deploymentID})
}

func (s *Service) getUpdateForDeviceHandler(c *gin.Context) {
	deviceID := c.Param("deviceId")

	update, err := s.GetUpdateForDevice(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, update)
}

func (s *Service) reportUpdateStatusHandler(c *gin.Context) {
	var report UpdateStatusReport

	if err := c.ShouldBindJSON(&report); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := s.ReportUpdateStatus(c.Request.Context(), &report)
	if err != nil {
		s.logger.Error("Failed to report update status", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "update status reported successfully"})
}
