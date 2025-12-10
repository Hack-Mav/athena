# PowerShell Make Script for ATHENA

This PowerShell script (`make.ps1`) provides a Windows-compatible replacement for the Unix `make` command in the ATHENA project.

## Why This Script Exists

The original Makefile uses Unix-style shell syntax that doesn't work properly on Windows, especially with the Node.js version of `make`. This script provides a PowerShell-native implementation of the most common make targets.

## Usage

```powershell
# Show help
.\make.ps1 help

# Build all services
.\make.ps1 build

# Build a specific service
.\make.ps1 build-service SERVICE=api-gateway

# Run tests
.\make.ps1 test

# Test a specific service
.\make.ps1 test-service SERVICE=api-gateway

# Format code
.\make.ps1 fmt

# Run linter
.\make.ps1 lint

# Clean build artifacts
.\make.ps1 clean

# Docker commands
.\make.ps1 docker-up
.\make.ps1 docker-down
.\make.ps1 docker-build
```

## Available Targets

- **help** - Show available commands
- **deps** - Download and tidy dependencies
- **build** - Build all services
- **build-service** - Build specific service (requires SERVICE parameter)
- **test** - Run all tests
- **test-service** - Run tests for specific service (requires SERVICE parameter)
- **fmt** - Format Go code
- **lint** - Run golangci-lint on all services
- **vet** - Run go vet on all code
- **clean** - Clean build artifacts
- **docker-up** - Start development environment
- **docker-down** - Stop development environment
- **docker-logs** - Show logs from all services
- **docker-build** - Build all Docker images

## How It Works

The script first attempts to use Docker's `alpine/make` image to run the original Makefile. If that fails (which it typically does on Windows), it falls back to PowerShell-native implementations of the commands.

## Requirements

- PowerShell
- Go
- Docker (for Docker commands)
- golangci-lint (for linting)

## Installation

The script is already included in the project. No additional installation required. Just run it from the project root directory.
