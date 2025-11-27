package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/athena/platform-lib/pkg/telemetry"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.Load("telemetry-service")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := logger.New(cfg.LogLevel, cfg.ServiceName)

	// Initialize Datastore client
	ctx := context.Background()
	datastoreClient, err := datastore.NewClient(ctx, cfg.DatastoreProject)
	if err != nil {
		logger.Fatalf("Failed to create Datastore client: %v", err)
	}
	defer datastoreClient.Close()

	// Set Datastore emulator host if configured
	if cfg.DatastoreHost != "" {
		os.Setenv("DATASTORE_EMULATOR_HOST", cfg.DatastoreHost)
	}

	// Initialize repository
	repository := telemetry.NewDatastoreRepository(datastoreClient)

	// Initialize service
	service, err := telemetry.NewService(cfg, logger, repository)
	if err != nil {
		logger.Fatalf("Failed to initialize telemetry service: %v", err)
	}

	// Start the service (MQTT connections, etc.)
	if err := service.Start(); err != nil {
		logger.Fatalf("Failed to start telemetry service: %v", err)
	}

	// Setup HTTP server
	router := gin.New()
	router.Use(gin.Recovery())

	// Register routes
	telemetry.RegisterRoutes(router, service)

	server := &http.Server{
		Addr:    cfg.HTTPPort,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.Infof("Telemetry service starting on %s", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Stop the service
	service.Stop()

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}
