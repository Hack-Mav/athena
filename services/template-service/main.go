package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/errors"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/athena/platform-lib/pkg/template"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.Load("template-service")
	if err != nil {
		errors.HandleConfigError("Failed to load configuration", err)
	}

	// Initialize logger
	logger := logger.New(cfg.LogLevel, cfg.ServiceName)

	// Initialize Datastore client
	ctx := context.Background()
	datastoreClient, err := datastore.NewClient(ctx, cfg.DatastoreProject)
	if err != nil {
		errors.HandleDBError("Failed to create Datastore client", err)
	}
	defer datastoreClient.Close()

	// Initialize service with Datastore repository
	repo := template.NewDatastoreRepository(datastoreClient)
	service, err := template.NewService(cfg, logger, repo)
	if err != nil {
		errors.HandleServiceError("Failed to initialize template service", err)
	}

	// Setup HTTP server
	router := gin.New()
	router.Use(gin.Recovery())

	// Register routes
	template.RegisterRoutes(router, service)

	server := &http.Server{
		Addr:    cfg.HTTPPort,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.Infof("Template service starting on %s", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errors.HandleNetworkError("Failed to start server", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exited")
}
