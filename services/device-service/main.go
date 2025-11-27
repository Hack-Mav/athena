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
	"github.com/athena/platform-lib/pkg/device"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.Load("device-service")
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

	// Initialize repository
	repository := device.NewDatastoreRepository(datastoreClient)

	// Initialize service
	service, err := device.NewService(cfg, logger, repository)
	if err != nil {
		logger.Fatalf("Failed to initialize device service: %v", err)
	}

	// Setup HTTP server
	router := gin.New()
	router.Use(gin.Recovery())

	// Register routes
	device.RegisterRoutes(router, service)

	server := &http.Server{
		Addr:    cfg.HTTPPort,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.Infof("Device service starting on %s", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown the service first
	if err := service.Shutdown(); err != nil {
		logger.Errorf("Failed to shutdown service gracefully: %v", err)
	}

	// Then shutdown the HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}
