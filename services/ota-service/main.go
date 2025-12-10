package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/athena/platform-lib/pkg/ota"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.Load("ota-service")
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := logger.New(cfg.LogLevel, cfg.ServiceName)

	// Initialize service - simplified for now
	// TODO: Implement proper service with all dependencies
	_, err = ota.NewService(cfg, logger, nil, nil, nil, nil)
	if err != nil {
		logger.Error("Failed to initialize OTA service", "error", err)
		os.Exit(1)
	}

	// Setup HTTP server
	router := gin.New()
	router.Use(gin.Recovery())

	// Register routes - basic health check for now
	// TODO: Implement OTA routes
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "ota-service"})
	})

	server := &http.Server{
		Addr:    cfg.HTTPPort,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.Infof("OTA service starting on %s", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start server", "error", err)
			os.Exit(1)
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
		os.Exit(1)
	}

	logger.Info("Server exited")
}
