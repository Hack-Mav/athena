package main

import (
	"os"

	"github.com/athena/platform-lib/pkg/cli"
	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load("athena-cli")
	if err != nil {
		// For CLI, we can use default config if loading fails
		cfg = config.Default("athena-cli")
	}

	// Initialize logger
	logger := logger.New(cfg.LogLevel, cfg.ServiceName)

	// Create and execute CLI
	rootCmd := cli.NewRootCommand(cfg, logger)
	if err := rootCmd.Execute(); err != nil {
		logger.Error(err)
		os.Exit(1)
	}
}
