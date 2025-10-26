# Simple ATHENA Build Script for Windows

Write-Host "ATHENA Platform Build System" -ForegroundColor Green

# Create bin directory
if (-not (Test-Path "bin")) {
    New-Item -ItemType Directory -Path "bin" | Out-Null
    Write-Host "Created bin directory" -ForegroundColor Green
}

# Update dependencies
Write-Host "Updating dependencies..." -ForegroundColor Yellow
go mod download
go mod tidy

# Build services
$services = @(
    "api-gateway",
    "template-service", 
    "nlp-service",
    "provisioning-service",
    "device-service",
    "telemetry-service",
    "ota-service"
)

Write-Host "Building services..." -ForegroundColor Yellow
foreach ($service in $services) {
    Write-Host "Building $service..." -ForegroundColor Cyan
    go build -o "bin\$service.exe" ".\cmd\$service"
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ Built $service" -ForegroundColor Green
    } else {
        Write-Host "✗ Failed to build $service" -ForegroundColor Red
    }
}

# Build CLI
Write-Host "Building CLI..." -ForegroundColor Cyan
go build -o "bin\athena-cli.exe" ".\cmd\cli"
if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Built athena-cli" -ForegroundColor Green
} else {
    Write-Host "✗ Failed to build athena-cli" -ForegroundColor Red
}

Write-Host "Build complete!" -ForegroundColor Green