# ATHENA Platform Makefile

# Variables
DOCKER_COMPOSE = docker-compose
GO = go
GOFMT = gofmt
GOLINT = golangci-lint
GOTEST = go test
GOBUILD = go build

# Service names
SERVICES = api-gateway template-service nlp-service provisioning-service device-service telemetry-service ota-service
CLI_SERVICE = cli

# Docker image prefix
IMAGE_PREFIX = athena

# Build directory
BUILD_DIR = build
BIN_DIR = bin

# Default target
.PHONY: all
all: build

# Help target
.PHONY: help
help:
	@echo "ATHENA Platform Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  build          - Build all services"
	@echo "  build-service  - Build specific service (make build-service SERVICE=template-service)"
	@echo "  build-cli      - Build CLI tool"
	@echo "  test           - Run all tests"
	@echo "  test-service   - Run tests for specific service"
	@echo "  lint           - Run linter on all code"
	@echo "  fmt            - Format all Go code"
	@echo "  clean          - Clean build artifacts"
	@echo "  docker-build   - Build all Docker images"
	@echo "  docker-up      - Start development environment"
	@echo "  docker-down    - Stop development environment"
	@echo "  docker-logs    - Show logs from all services"
	@echo "  deps           - Download and tidy dependencies"
	@echo "  generate       - Generate code (protobuf, mocks, etc.)"

# Build targets
.PHONY: build
build: deps $(SERVICES) $(CLI_SERVICE)

.PHONY: $(SERVICES)
$(SERVICES):
	@echo "Building $@..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/$@ ./cmd/$@

.PHONY: $(CLI_SERVICE)
$(CLI_SERVICE):
	@echo "Building athena-cli..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/athena-cli ./cmd/cli

.PHONY: build-service
build-service:
	@if [ -z "$(SERVICE)" ]; then \
		echo "Usage: make build-service SERVICE=<service-name>"; \
		exit 1; \
	fi
	@echo "Building $(SERVICE)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/$(SERVICE) ./cmd/$(SERVICE)

.PHONY: build-cli
build-cli: $(CLI_SERVICE)

# Test targets
.PHONY: test
test:
	@echo "Running all tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

.PHONY: test-service
test-service:
	@if [ -z "$(SERVICE)" ]; then \
		echo "Usage: make test-service SERVICE=<service-name>"; \
		exit 1; \
	fi
	@echo "Running tests for $(SERVICE)..."
	$(GOTEST) -v -race ./internal/$(SERVICE)/... ./cmd/$(SERVICE)/...

.PHONY: test-coverage
test-coverage: test
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Code quality targets
.PHONY: lint
lint:
	@echo "Running linter..."
	$(GOLINT) run ./...

.PHONY: fmt
fmt:
	@echo "Formatting Go code..."
	$(GOFMT) -s -w .

.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

# Dependency management
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

.PHONY: deps-update
deps-update:
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy

# Docker targets
.PHONY: docker-build
docker-build:
	@echo "Building Docker images..."
	@for service in $(SERVICES); do \
		echo "Building $$service image..."; \
		docker build -f $(BUILD_DIR)/Dockerfile.$$service -t $(IMAGE_PREFIX)/$$service:latest .; \
	done

.PHONY: docker-up
docker-up:
	@echo "Starting development environment..."
	$(DOCKER_COMPOSE) up -d

.PHONY: docker-down
docker-down:
	@echo "Stopping development environment..."
	$(DOCKER_COMPOSE) down

.PHONY: docker-logs
docker-logs:
	$(DOCKER_COMPOSE) logs -f

.PHONY: docker-clean
docker-clean:
	@echo "Cleaning Docker resources..."
	$(DOCKER_COMPOSE) down -v --remove-orphans
	docker system prune -f

# Development targets
.PHONY: dev-setup
dev-setup: deps docker-up
	@echo "Development environment setup complete!"
	@echo "Services will be available at:"
	@echo "  API Gateway: http://localhost:8000"
	@echo "  MinIO Console: http://localhost:9001 (athena/dev_password)"

.PHONY: dev-reset
dev-reset: docker-clean dev-setup

# Code generation
.PHONY: generate
generate:
	@echo "Generating code..."
	$(GO) generate ./...

# Clean targets
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	rm -f coverage.out coverage.html
	$(GO) clean ./...

.PHONY: clean-all
clean-all: clean docker-clean

# Install targets
.PHONY: install
install: build
	@echo "Installing binaries..."
	@for service in $(SERVICES); do \
		cp $(BIN_DIR)/$$service $(GOPATH)/bin/; \
	done
	cp $(BIN_DIR)/athena-cli $(GOPATH)/bin/

# CI/CD targets
.PHONY: ci
ci: deps fmt vet lint test

.PHONY: release
release: ci build docker-build
	@echo "Release build complete!"

# Database migration targets
.PHONY: migrate-up
migrate-up:
	@echo "Running database migrations..."
	# Add migration commands here when needed

.PHONY: migrate-down
migrate-down:
	@echo "Rolling back database migrations..."
	# Add rollback commands here when needed

# Monitoring targets
.PHONY: health-check
health-check:
	@echo "Checking service health..."
	@for port in 8000 8001 8002 8003 8004 8005 8006; do \
		echo -n "Port $$port: "; \
		curl -s -o /dev/null -w "%{http_code}" http://localhost:$$port/health || echo "DOWN"; \
	done