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
	"github.com/athena/platform-lib/pkg/secrets"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.Load("secrets-service")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger := logger.New(cfg.LogLevel, cfg.ServiceName)

	// Initialize Datastore client
	ctx := context.Background()
	datastoreClient, err := datastore.NewClient(ctx, cfg.DatastoreProject)
	if err != nil {
		logger.Fatal("Failed to create datastore client", "error", err)
	}
	defer datastoreClient.Close()

	// Initialize repositories
	repository := secrets.NewDatastoreRepository(datastoreClient)
	authRepository := secrets.NewDatastoreAuthRepository(datastoreClient)

	// Initialize services
	service, err := secrets.NewService(cfg, logger, repository)
	if err != nil {
		logger.Fatal("Failed to create secrets service", "error", err)
	}

	authService, err := secrets.NewAuthService(cfg, logger, authRepository)
	if err != nil {
		logger.Fatal("Failed to create auth service", "error", err)
	}

	// Set up HTTP server
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Add CORS middleware for development
	if cfg.Environment == "development" {
		router.Use(func(c *gin.Context) {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Principal")

			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}

			c.Next()
		})
	}

	// Register routes
	secrets.RegisterRoutes(router, service)
	secrets.RegisterAuthRoutes(router, authService)

	// Start HTTP server
	server := &http.Server{
		Addr:    cfg.HTTPPort,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting secrets service", "port", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", "error", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down secrets service...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", "error", err)
	}

	logger.Info("Secrets service stopped")
}
