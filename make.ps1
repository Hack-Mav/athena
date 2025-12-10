param(
    [Parameter(Position=0, ValueFromRemainingArguments=$true)]
    [string[]]$Arguments = @()
)

# Parse arguments
$Target = "help"
$Variables = @{}

if ($Arguments.Count -gt 0) {
    $Target = $Arguments[0]
    
    # Parse variable assignments (KEY=value format)
    for ($i = 1; $i -lt $Arguments.Count; $i++) {
        $arg = $Arguments[$i]
        if ($arg -match '^(.+?)=(.+)$') {
            $Variables[$matches[1]] = $matches[2]
        }
    }
}

# Set environment variables for make
foreach ($var in $Variables.GetEnumerator()) {
    [System.Environment]::SetEnvironmentVariable($var.Key, $var.Value)
}

# Function to execute make commands via WSL using Docker Desktop's build environment
function Invoke-MakeCommand {
    param([string]$MakeTarget)
    
    # Extract the base target (first word)
    $baseTarget = ($MakeTarget -split ' ')[0]
    
    # Use Docker build environment which has make
    $dockerCmd = "docker run --rm -v ${PWD}:/workspace -w /workspace alpine/make $MakeTarget"
    
    try {
        Invoke-Expression $dockerCmd
    }
    catch {
        Write-Host "Docker make failed, trying direct Go commands..." -ForegroundColor Yellow
        
        # Fallback to direct Go commands for common targets
        switch ($baseTarget) {
            "help" {
                Write-Host "ATHENA Platform Build System"
                Write-Host ""
                Write-Host "Available targets:"
                Write-Host "  build          - Build all services"
                Write-Host "  build-service  - Build specific service (make build-service SERVICE=template-service)"
                Write-Host "  build-cli      - Build CLI tool"
                Write-Host "  test           - Run all tests"
                Write-Host "  test-service   - Run tests for specific service"
                Write-Host "  lint           - Run linter on all code"
                Write-Host "  fmt            - Format all Go code"
                Write-Host "  clean          - Clean build artifacts"
                Write-Host "  deps           - Download and tidy dependencies"
            }
            "deps" {
                Write-Host "Downloading dependencies..."
                go mod download
                go mod tidy
            }
            "build" {
                Write-Host "Building all services..."
                $services = @("api-gateway", "template-service", "nlp-service", "provisioning-service", "device-service", "telemetry-service", "ota-service")
                New-Item -ItemType Directory -Force -Path "bin"
                
                foreach ($service in $services) {
                    if (Test-Path "services\$service") {
                        Write-Host "Building $service..."
                        go build -o "bin\$service" "./services/$service"
                    }
                }
                
                if (Test-Path "services\cli") {
                    Write-Host "Building athena-cli..."
                    go build -o "bin\athena-cli" "./services/cli"
                }
            }
            "build-service" {
                $service = $Variables["SERVICE"]
                if ([string]::IsNullOrEmpty($service)) {
                    Write-Host "Usage: .\make.ps1 build-service SERVICE=<service-name>"
                    exit 1
                }
                
                Write-Host "Building $service..."
                New-Item -ItemType Directory -Force -Path "bin"
                go build -o "bin\$service" "./services/$service"
            }
            "test" {
                Write-Host "Running all tests..."
                $env:CGO_ENABLED = "1"
                $services = @("api-gateway", "template-service", "nlp-service", "provisioning-service", "device-service", "telemetry-service", "ota-service", "cli")
                
                foreach ($service in $services) {
                    if (Test-Path "services\$service") {
                        Write-Host "Testing $service..."
                        Push-Location "services\$service"
                        go test -v -race ./...
                        Pop-Location
                    }
                }
                
                if (Test-Path "services\platform-lib") {
                    Write-Host "Testing platform-lib..."
                    Push-Location "services\platform-lib"
                    go test -v -race ./...
                    Pop-Location
                }
            }
            "fmt" {
                Write-Host "Formatting Go code..."
                gofmt -s -w .
            }
            "clean" {
                Write-Host "Cleaning build artifacts..."
                Remove-Item -Recurse -Force "bin" -ErrorAction SilentlyContinue
                Remove-Item "coverage.out", "coverage.html" -ErrorAction SilentlyContinue
                go clean ./...
            }
            "lint" {
                Write-Host "Running linter..."
                $services = @("api-gateway", "template-service", "nlp-service", "provisioning-service", "device-service", "telemetry-service", "ota-service", "cli", "platform-lib")
                
                foreach ($service in $services) {
                    if (Test-Path "services\$service") {
                        Write-Host "Linting $service..."
                        Push-Location "services\$service"
                        golangci-lint run ./...
                        Pop-Location
                    }
                }
            }
            "vet" {
                Write-Host "Running go vet..."
                go vet ./...
            }
            "docker-up" {
                Write-Host "Starting development environment..."
                docker-compose up -d
            }
            "docker-down" {
                Write-Host "Stopping development environment..."
                docker-compose down
            }
            "docker-logs" {
                docker-compose logs -f
            }
            "docker-build" {
                Write-Host "Building Docker images..."
                $services = @("api-gateway", "template-service", "nlp-service", "provisioning-service", "device-service", "telemetry-service", "ota-service")
                foreach ($service in $services) {
                    Write-Host "Building $service image..."
                    docker build -f "build/Dockerfile.$service" -t "athena/$service`:latest" .
                }
            }
            "test-service" {
                $service = $Variables["SERVICE"]
                if ([string]::IsNullOrEmpty($service)) {
                    Write-Host "Usage: .\make.ps1 test-service SERVICE=<service-name>"
                    exit 1
                }
                
                Write-Host "Running tests for $service..."
                $env:CGO_ENABLED = "1"
                if (Test-Path "services\$service") {
                    Push-Location "services\$service"
                    go test -v -race ./...
                    Pop-Location
                } else {
                    Write-Host "Service $service not found"
                    exit 1
                }
            }
            default {
                Write-Host "Target '$baseTarget' not implemented in PowerShell fallback"
                Write-Host "Available: help, deps, build, build-service, test, test-service, fmt, clean, lint, vet, docker-up, docker-down, docker-logs, docker-build"
            }
        }
    }
}

# Execute the make command
$varString = ""
foreach ($var in $Variables.GetEnumerator()) {
    $varString += "$($var.Key)=$($var.Value) "
}
Invoke-MakeCommand -MakeTarget "$Target $varString".Trim()
