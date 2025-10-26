# ATHENA Platform Build Script for Windows

param(
    [string]$Target = "all",
    [string]$Service = "",
    [switch]$Clean = $false,
    [switch]$Test = $false,
    [switch]$Help = $false
)

# Configuration
$BinDir = "bin"
$Services = @(
    "api-gateway",
    "template-service", 
    "nlp-service",
    "provisioning-service",
    "device-service",
    "telemetry-service",
    "ota-service"
)
$CLI = "cli"

function Show-Help {
    Write-Host "ATHENA Platform Build System for Windows" -ForegroundColor Green
    Write-Host ""
    Write-Host "Usage: .\build.ps1 [options]" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Options:" -ForegroundColor Yellow
    Write-Host "  -Target <target>    Build target (all, service, cli, clean, test)" -ForegroundColor White
    Write-Host "  -Service <name>     Specific service to build (when Target=service)" -ForegroundColor White
    Write-Host "  -Clean              Clean build artifacts" -ForegroundColor White
    Write-Host "  -Test               Run tests" -ForegroundColor White
    Write-Host "  -Help               Show this help" -ForegroundColor White
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Yellow
    Write-Host "  .\build.ps1                           # Build all services" -ForegroundColor White
    Write-Host "  .\build.ps1 -Target service -Service template-service" -ForegroundColor White
    Write-Host "  .\build.ps1 -Target cli               # Build CLI only" -ForegroundColor White
    Write-Host "  .\build.ps1 -Clean                    # Clean artifacts" -ForegroundColor White
    Write-Host "  .\build.ps1 -Test                     # Run tests" -ForegroundColor White
}

function New-BinDirectory {
    if (-not (Test-Path $BinDir)) {
        New-Item -ItemType Directory -Path $BinDir | Out-Null
        Write-Host "Created $BinDir directory" -ForegroundColor Green
    }
}

function Build-Service {
    param([string]$ServiceName)
    
    Write-Host "Building $ServiceName..." -ForegroundColor Yellow
    $OutputPath = "$BinDir\$ServiceName.exe"
    $SourcePath = ".\cmd\$ServiceName"
    
    try {
        go build -o $OutputPath $SourcePath
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ Built $ServiceName successfully" -ForegroundColor Green
        } else {
            Write-Host "✗ Failed to build $ServiceName" -ForegroundColor Red
            return $false
        }
    } catch {
        Write-Host "✗ Error building $ServiceName`: $_" -ForegroundColor Red
        return $false
    }
    return $true
}

function Build-CLI {
    Write-Host "Building athena-cli..." -ForegroundColor Yellow
    $OutputPath = "$BinDir\athena-cli.exe"
    $SourcePath = ".\cmd\cli"
    
    try {
        go build -o $OutputPath $SourcePath
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ Built athena-cli successfully" -ForegroundColor Green
        } else {
            Write-Host "✗ Failed to build athena-cli" -ForegroundColor Red
            return $false
        }
    } catch {
        Write-Host "✗ Error building athena-cli: $_" -ForegroundColor Red
        return $false
    }
    return $true
}

function Build-All {
    Write-Host "Building all services..." -ForegroundColor Cyan
    New-BinDirectory
    
    $success = $true
    
    # Build all services
    foreach ($service in $Services) {
        if (-not (Build-Service $service)) {
            $success = $false
        }
    }
    
    # Build CLI
    if (-not (Build-CLI)) {
        $success = $false
    }
    
    if ($success) {
        Write-Host "✓ All builds completed successfully!" -ForegroundColor Green
    } else {
        Write-Host "✗ Some builds failed" -ForegroundColor Red
        exit 1
    }
}

function Clean-Artifacts {
    Write-Host "Cleaning build artifacts..." -ForegroundColor Yellow
    
    if (Test-Path $BinDir) {
        Remove-Item -Recurse -Force $BinDir
        Write-Host "✓ Removed $BinDir directory" -ForegroundColor Green
    }
    
    # Clean Go cache
    go clean ./...
    Write-Host "✓ Cleaned Go cache" -ForegroundColor Green
    
    # Remove coverage files
    if (Test-Path "coverage.out") {
        Remove-Item "coverage.out"
        Write-Host "✓ Removed coverage.out" -ForegroundColor Green
    }
    
    if (Test-Path "coverage.html") {
        Remove-Item "coverage.html"
        Write-Host "✓ Removed coverage.html" -ForegroundColor Green
    }
}

function Run-Tests {
    Write-Host "Running tests..." -ForegroundColor Yellow
    
    try {
        go test -v -race -coverprofile=coverage.out ./...
        if ($LASTEXITCODE -eq 0) {
            Write-Host "✓ All tests passed!" -ForegroundColor Green
            
            # Generate coverage report
            go tool cover -html=coverage.out -o coverage.html
            Write-Host "✓ Coverage report generated: coverage.html" -ForegroundColor Green
        } else {
            Write-Host "✗ Some tests failed" -ForegroundColor Red
            exit 1
        }
    } catch {
        Write-Host "✗ Error running tests: $_" -ForegroundColor Red
        exit 1
    }
}

function Update-Dependencies {
    Write-Host "Updating dependencies..." -ForegroundColor Yellow
    
    go mod download
    go mod tidy
    
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ Dependencies updated successfully" -ForegroundColor Green
    } else {
        Write-Host "✗ Failed to update dependencies" -ForegroundColor Red
        exit 1
    }
}

# Main execution
if ($Help) {
    Show-Help
    exit 0
}

if ($Clean) {
    Clean-Artifacts
    exit 0
}

if ($Test) {
    Run-Tests
    exit 0
}

# Update dependencies first
Update-Dependencies

switch ($Target.ToLower()) {
    "all" {
        Build-All
    }
    "service" {
        if (-not $Service) {
            Write-Host "Error: -Service parameter required when Target=service" -ForegroundColor Red
            Write-Host "Available services: $($Services -join ', ')" -ForegroundColor Yellow
            exit 1
        }
        if ($Services -notcontains $Service) {
            Write-Host "Error: Unknown service '$Service'" -ForegroundColor Red
            Write-Host "Available services: $($Services -join ', ')" -ForegroundColor Yellow
            exit 1
        }
        New-BinDirectory
        Build-Service $Service
    }
    "cli" {
        New-BinDirectory
        Build-CLI
    }
    "clean" {
        Clean-Artifacts
    }
    "test" {
        Run-Tests
    }
    default {
        Write-Host "Error: Unknown target '$Target'" -ForegroundColor Red
        Show-Help
        exit 1
    }
}