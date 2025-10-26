package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/athena/platform/internal/ota"
	"github.com/athena/platform/pkg/config"
	"github.com/athena/platform/pkg/logger"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.Load("ota-service")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := logger.New(cfg.LogLevel, cfg.ServiceName)

	// Initialize service
	service, err := ota.NewService(cfg, logger)
	if err != nil {
		logger.Fatalf("Failed to initialize OTA service: %v", err)
	}

	// Setup HTTP server
	router := gin.New()
	router.Use(gin.Recovery())
	
	// Register routes
	ota.RegisterRoutes(router, service)

	server := &http.Server{
		Addr:    cfg.HTTPPort,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.Infof("OTA service starting on %s", cfg.HTTPPort)
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}