package provisioning

import (
	"fmt"
	"net/http"

	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Service represents the provisioning service
type Service struct {
	config          *config.Config
	logger          *logger.Logger
	cli             *ArduinoCLI
	boardManager    *BoardManager
	libraryManager  *LibraryManager
	compiler        *Compiler
	artifactManager *ArtifactManager
	flasher         *Flasher
}

// NewService creates a new provisioning service instance
func NewService(cfg *config.Config, logger *logger.Logger) (*Service, error) {
	// Initialize Arduino CLI wrapper
	cli := NewArduinoCLI(cfg.ArduinoCLIPath)

	// Initialize managers
	boardManager := NewBoardManager(cli)
	libraryManager := NewLibraryManager(cli)

	// Initialize compiler and artifact manager
	workspaceDir := "/tmp/athena/workspace"
	cacheDir := "/tmp/athena/cache"
	artifactDir := "/tmp/athena/artifacts"

	compiler := NewCompiler(cli, workspaceDir, cacheDir)
	artifactManager := NewArtifactManager(artifactDir)
	flasher := NewFlasher(cli)

	return &Service{
		config:          cfg,
		logger:          logger,
		cli:             cli,
		boardManager:    boardManager,
		libraryManager:  libraryManager,
		compiler:        compiler,
		artifactManager: artifactManager,
		flasher:         flasher,
	}, nil
}

// RegisterRoutes registers HTTP routes for the provisioning service
func RegisterRoutes(router *gin.Engine, service *Service) {
	v1 := router.Group("/api/v1/provisioning")
	{
		v1.GET("/health", service.healthCheck)

		// Board management endpoints
		v1.GET("/boards", service.listBoards)
		v1.GET("/boards/:fqbn", service.getBoard)
		v1.GET("/boards/detect", service.detectBoards)
		v1.POST("/boards/validate", service.validateBoardCompatibility)
		v1.POST("/boards/validate-pins", service.validatePinAssignments)

		// Library management endpoints
		v1.GET("/libraries", service.getInstalledLibraries)
		v1.GET("/libraries/search", service.searchLibraries)
		v1.POST("/libraries/resolve", service.resolveDependencies)
		v1.POST("/libraries/install", service.installLibraries)

		// Compilation endpoints
		v1.POST("/compile", service.compileTemplate)
		v1.GET("/artifacts/:id", service.GetArtifact)
		v1.GET("/artifacts/:id/binary", service.getArtifactBinary)
		v1.DELETE("/artifacts/:id", service.deleteArtifact)
		v1.POST("/artifacts/search", service.searchArtifacts)

		// Flashing endpoints
		v1.POST("/flash", service.flashDevice)
		v1.GET("/ports", service.getAvailablePorts)
	}
}

func (s *Service) healthCheck(c *gin.Context) {
	ctx := c.Request.Context()

	// Check Arduino CLI availability
	version, err := s.cli.Version(ctx)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"service": "provisioning-service",
			"error":   "Arduino CLI not available: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":              "healthy",
		"service":             "provisioning-service",
		"arduino_cli_version": version,
	})
}

func (s *Service) listBoards(c *gin.Context) {
	ctx := c.Request.Context()

	boards, err := s.boardManager.ListBoards(ctx)
	if err != nil {
		s.logger.Error("Failed to list boards", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list boards: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"boards": boards,
	})
}

func (s *Service) getBoard(c *gin.Context) {
	ctx := c.Request.Context()
	fqbn := c.Param("fqbn")

	board, err := s.boardManager.GetBoard(ctx, fqbn)
	if err != nil {
		s.logger.Error("Failed to get board", "fqbn", fqbn, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Board not found: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, board)
}

func (s *Service) detectBoards(c *gin.Context) {
	ctx := c.Request.Context()

	ports, err := s.boardManager.DetectConnectedBoards(ctx)
	if err != nil {
		s.logger.Error("Failed to detect boards", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to detect boards: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ports": ports,
	})
}

type ValidateBoardRequest struct {
	FQBN         string            `json:"fqbn" binding:"required"`
	Requirements BoardRequirements `json:"requirements" binding:"required"`
}

func (s *Service) validateBoardCompatibility(c *gin.Context) {
	ctx := c.Request.Context()

	var req ValidateBoardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	compatibility, err := s.boardManager.ValidateBoardCompatibility(ctx, req.FQBN, req.Requirements)
	if err != nil {
		s.logger.Error("Failed to validate board compatibility", "fqbn", req.FQBN, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to validate compatibility: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, compatibility)
}

type ValidatePinsRequest struct {
	FQBN        string          `json:"fqbn" binding:"required"`
	Assignments []PinAssignment `json:"assignments" binding:"required"`
}

func (s *Service) validatePinAssignments(c *gin.Context) {
	ctx := c.Request.Context()

	var req ValidatePinsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	check, err := s.boardManager.ValidatePinAssignments(ctx, req.FQBN, req.Assignments)
	if err != nil {
		s.logger.Error("Failed to validate pin assignments", "fqbn", req.FQBN, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to validate pin assignments: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, check)
}

func (s *Service) getInstalledLibraries(c *gin.Context) {
	ctx := c.Request.Context()

	libraries, err := s.libraryManager.GetInstalledLibraries(ctx)
	if err != nil {
		s.logger.Error("Failed to get installed libraries", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get installed libraries: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"libraries": libraries,
	})
}

func (s *Service) searchLibraries(c *gin.Context) {
	ctx := c.Request.Context()
	query := c.Query("q")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Query parameter 'q' is required",
		})
		return
	}

	libraries, err := s.libraryManager.SearchLibrary(ctx, query)
	if err != nil {
		s.logger.Error("Failed to search libraries", "query", query, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to search libraries: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"libraries": libraries,
	})
}

type ResolveDependenciesRequest struct {
	Dependencies []LibraryDependency `json:"dependencies" binding:"required"`
}

func (s *Service) resolveDependencies(c *gin.Context) {
	ctx := c.Request.Context()

	var req ResolveDependenciesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	resolution, err := s.libraryManager.ResolveDependencies(ctx, req.Dependencies)
	if err != nil {
		s.logger.Error("Failed to resolve dependencies", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to resolve dependencies: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resolution)
}

type InstallLibrariesRequest struct {
	Libraries []Library `json:"libraries" binding:"required"`
}

func (s *Service) installLibraries(c *gin.Context) {
	ctx := c.Request.Context()

	var req InstallLibrariesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	result, err := s.libraryManager.InstallLibraries(ctx, req.Libraries)
	if err != nil {
		s.logger.Error("Failed to install libraries", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to install libraries: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (s *Service) compileTemplate(c *gin.Context) {
	ctx := c.Request.Context()

	var req CompilationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	// Validate board exists
	_, err := s.boardManager.GetBoard(ctx, req.Board)
	if err != nil {
		s.logger.Error("Invalid board specified", "board", req.Board, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid board: " + err.Error(),
		})
		return
	}

	// Install required libraries if specified
	if len(req.Libraries) > 0 {
		s.logger.Info("Installing required libraries", "count", len(req.Libraries))

		// Resolve dependencies
		resolution, err := s.libraryManager.ResolveDependencies(ctx, req.Libraries)
		if err != nil {
			s.logger.Error("Failed to resolve library dependencies", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to resolve dependencies: " + err.Error(),
			})
			return
		}

		if resolution.HasConflicts {
			s.logger.Warn("Library dependency conflicts detected", "conflicts", resolution.Conflicts)
			c.JSON(http.StatusBadRequest, gin.H{
				"error":     "Library dependency conflicts",
				"conflicts": resolution.Conflicts,
			})
			return
		}

		// Install libraries
		if len(resolution.ToInstall) > 0 {
			libraries := make([]Library, len(resolution.ToInstall))
			for i, install := range resolution.ToInstall {
				libraries[i] = install.Library
			}

			installResult, err := s.libraryManager.InstallLibraries(ctx, libraries)
			if err != nil {
				s.logger.Error("Failed to install libraries", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to install libraries: " + err.Error(),
				})
				return
			}

			if len(installResult.Failed) > 0 {
				s.logger.Warn("Some libraries failed to install", "failed", installResult.Failed)
				c.JSON(http.StatusPartialContent, gin.H{
					"warning": "Some libraries failed to install",
					"failed":  installResult.Failed,
				})
				return
			}

			s.logger.Info("Libraries installed successfully", "installed", len(installResult.Installed))
		}
	}

	// Compile the template
	s.logger.Info("Starting compilation", "template", req.TemplateID, "board", req.Board)

	result, err := s.compiler.CompileTemplate(ctx, &req)
	if err != nil {
		s.logger.Error("Compilation failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Compilation failed: " + err.Error(),
		})
		return
	}

	if !result.Success {
		s.logger.Warn("Compilation completed with errors", "errors", len(result.Errors))
		c.JSON(http.StatusBadRequest, gin.H{
			"success":  false,
			"errors":   result.Errors,
			"warnings": result.Warnings,
			"duration": result.Duration.String(),
		})
		return
	}

	// Store the artifact
	artifact, err := s.artifactManager.StoreArtifact(ctx, result)
	if err != nil {
		s.logger.Error("Failed to store artifact", "error", err)
		// Don't fail the request, just log the error
	}

	s.logger.Info("Compilation completed successfully",
		"template", req.TemplateID,
		"duration", result.Duration,
		"cache_hit", result.CacheHit,
		"artifact_id", func() string {
			if artifact != nil {
				return artifact.ID
			}
			return "none"
		}())

	response := gin.H{
		"success":     result.Success,
		"duration":    result.Duration.String(),
		"cache_hit":   result.CacheHit,
		"binary_hash": result.BinaryHash,
		"size":        result.Size,
		"warnings":    result.Warnings,
	}

	if artifact != nil {
		response["artifact_id"] = artifact.ID
	}

	c.JSON(http.StatusOK, response)
}

func (s *Service) flashDevice(c *gin.Context) {
	ctx := c.Request.Context()

	var req FlashRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	// Validate board exists
	_, err := s.boardManager.GetBoard(ctx, req.Board)
	if err != nil {
		s.logger.Error("Invalid board specified for flashing", "board", req.Board, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid board: " + err.Error(),
		})
		return
	}

	// If artifact ID is provided, get the binary path
	if req.ArtifactID != "" && req.BinaryPath == "" {
		artifact, err := s.artifactManager.GetArtifact(ctx, req.ArtifactID)
		if err != nil {
			s.logger.Error("Failed to get artifact for flashing", "artifact_id", req.ArtifactID, "error", err)
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Artifact not found: " + err.Error(),
			})
			return
		}
		req.BinaryPath = artifact.BinaryPath
	}

	s.logger.Info("Starting device flash",
		"port", req.Port,
		"board", req.Board,
		"binary", req.BinaryPath,
		"artifact_id", req.ArtifactID)

	// Flash the device
	result, err := s.flasher.FlashDevice(ctx, &req, nil) // No progress callback for HTTP API
	if err != nil {
		s.logger.Error("Flash operation failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Flash operation failed: " + err.Error(),
		})
		return
	}

	if !result.Success {
		s.logger.Warn("Flash completed with errors", "errors", result.Errors)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":  false,
			"errors":   result.Errors,
			"duration": result.Duration.String(),
		})
		return
	}

	s.logger.Info("Device flashed successfully",
		"port", req.Port,
		"duration", result.Duration,
		"health_check", result.HealthCheck != nil && result.HealthCheck.Success)

	response := gin.H{
		"success":  result.Success,
		"port":     result.Port,
		"board":    result.Board,
		"duration": result.Duration.String(),
	}

	if result.VerifyResult != nil {
		response["verify_result"] = result.VerifyResult
	}

	if result.HealthCheck != nil {
		response["health_check"] = result.HealthCheck
	}

	c.JSON(http.StatusOK, response)
}
func (s *Service) GetArtifact(c *gin.Context) {
	ctx := c.Request.Context()
	artifactID := c.Param("id")

	artifact, err := s.artifactManager.GetArtifact(ctx, artifactID)
	if err != nil {
		s.logger.Error("Failed to get artifact", "id", artifactID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Artifact not found: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, artifact)
}

func (s *Service) getArtifactBinary(c *gin.Context) {
	ctx := c.Request.Context()
	artifactID := c.Param("id")

	binary, err := s.artifactManager.GetArtifactBinary(ctx, artifactID)
	if err != nil {
		s.logger.Error("Failed to get artifact binary", "id", artifactID, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Artifact binary not found: " + err.Error(),
		})
		return
	}

	// Set appropriate headers for binary download
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.hex\"", artifactID))
	c.Data(http.StatusOK, "application/octet-stream", binary)
}

func (s *Service) deleteArtifact(c *gin.Context) {
	ctx := c.Request.Context()
	artifactID := c.Param("id")

	if err := s.artifactManager.DeleteArtifact(ctx, artifactID); err != nil {
		s.logger.Error("Failed to delete artifact", "id", artifactID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete artifact: " + err.Error(),
		})
		return
	}

	s.logger.Info("Artifact deleted successfully", "id", artifactID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Artifact deleted successfully",
	})
}

func (s *Service) searchArtifacts(c *gin.Context) {
	ctx := c.Request.Context()

	var query ArtifactQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid query: " + err.Error(),
		})
		return
	}

	artifacts, err := s.artifactManager.FindArtifacts(ctx, query)
	if err != nil {
		s.logger.Error("Failed to search artifacts", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to search artifacts: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"artifacts": artifacts,
		"count":     len(artifacts),
	})
}

func (s *Service) getAvailablePorts(c *gin.Context) {
	ports, err := s.flasher.GetAvailablePorts()
	if err != nil {
		s.logger.Error("Failed to get available ports", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get available ports: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ports": ports,
		"count": len(ports),
	})
}
