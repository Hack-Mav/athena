# Template Validation Test Runner (PowerShell)
# This script runs all template validation tests with proper setup and cleanup

param(
    [Parameter(Mandatory=$false)]
    [ValidateSet("unit", "integration", "all", "coverage", "benchmark", "ci", "validate", "list", "help")]
    [string]$Command = "help",
    
    [Parameter(Mandatory=$false)]
    [string]$TemplateFile = "",
    
    [Parameter(Mandatory=$false)]
    [string]$TemplateDir = "",
    
    [Parameter(Mandatory=$false)]
    [string]$TestWorkspace = "",
    
    [Parameter(Mandatory=$false)]
    [string]$ArduinoCliPath = ""
)

# Configuration
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$DefaultTemplateDir = Join-Path $ScriptDir "..\templates\arduino"
$DefaultWorkspaceDir = Join-Path $ScriptDir "test_workspace"
$DefaultArduinoCli = "arduino-cli"

# Use defaults if not provided
if ([string]::IsNullOrEmpty($TemplateDir)) {
    $TemplateDir = $TemplateDir = $DefaultTemplateDir
}
if ([string]::IsNullOrEmpty($TestWorkspace)) {
    $TestWorkspace = $DefaultWorkspaceDir
}
if ([string]::IsNullOrEmpty($ArduinoCliPath)) {
    $ArduinoCliPath = $DefaultArduinoCli
}

# Colors for output
$Colors = @{
    Red = "Red"
    Green = "Green"
    Yellow = "Yellow"
    Blue = "Blue"
    White = "White"
}

# Functions
function Write-Log {
    param(
        [string]$Message,
        [string]$Level = "INFO"
    )
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $color = $Colors.White
    
    switch ($Level) {
        "INFO" { $color = $Colors.Blue }
        "SUCCESS" { $color = $Colors.Green }
        "WARNING" { $color = $Colors.Yellow }
        "ERROR" { $color = $Colors.Red }
    }
    
    Write-Host "[$timestamp] [$Level] $Message" -ForegroundColor $color
}

function Write-Info {
    param([string]$Message)
    Write-Log -Message $Message -Level "INFO"
}

function Write-Success {
    param([string]$Message)
    Write-Log -Message $Message -Level "SUCCESS"
}

function Write-Warning {
    param([string]$Message)
    Write-Log -Message $Message -Level "WARNING"
}

function Write-Error {
    param([string]$Message)
    Write-Log -Message $Message -Level "ERROR"
}

# Cleanup function
function Invoke-Cleanup {
    Write-Info "Cleaning up test workspace..."
    
    if (Test-Path $TestWorkspace) {
        Remove-Item -Path $TestWorkspace -Recurse -Force
    }
    
    Write-Success "Cleanup completed"
}

# Setup function
function Invoke-Setup {
    Write-Info "Setting up test environment..."
    
    # Create workspace directory
    if (!(Test-Path $TestWorkspace)) {
        New-Item -Path $TestWorkspace -ItemType Directory -Force | Out-Null
    }
    
    # Check if template directory exists
    if (!(Test-Path $TemplateDir)) {
        Write-Error "Template directory not found: $TemplateDir"
        exit 1
    }
    
    # Count templates
    $templateFiles = Get-ChildItem -Path $TemplateDir -Filter "*.json" -Recurse
    $templateCount = $templateFiles.Count
    Write-Info "Found $templateCount templates to validate"
    
    # Check for Arduino CLI
    $arduinoCliAvailable = $false
    try {
        $null = Get-Command $ArduinoCliPath -ErrorAction Stop
        Write-Success "Arduino CLI found: $((Get-Command $ArduinoCliPath).Source)"
        $arduinoCliAvailable = $true
    }
    catch {
        Write-Warning "Arduino CLI not found, some tests will be skipped"
    }
    
    Write-Success "Setup completed"
    return $arduinoCliAvailable
}

# Run unit tests
function Invoke-UnitTests {
    Write-Info "Running unit tests..."
    
    Push-Location $ScriptDir
    
    try {
        $result = go test -v -run "TestTemplate|TestParameter|TestWiring" ./...
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Unit tests passed"
        }
        else {
            Write-Error "Unit tests failed"
            return $false
        }
    }
    finally {
        Pop-Location
    }
    
    return $true
}

# Run integration tests
function Invoke-IntegrationTests {
    Write-Info "Running integration tests..."
    
    Push-Location $ScriptDir
    
    try {
        $result = go test -v -run "TestRealArduino|TestLibrary|TestCode" ./...
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Integration tests passed"
        }
        else {
            Write-Error "Integration tests failed"
            return $false
        }
    }
    finally {
        Pop-Location
    }
    
    return $true
}

# Run all tests
function Invoke-AllTests {
    Write-Info "Running all tests..."
    
    Push-Location $ScriptDir
    
    try {
        $result = go test -v ./...
        if ($LASTEXITCODE -eq 0) {
            Write-Success "All tests passed"
        }
        else {
            Write-Error "Some tests failed"
            return $false
        }
    }
    finally {
        Pop-Location
    }
    
    return $true
}

# Run tests with coverage
function Invoke-CoverageTests {
    Write-Info "Running tests with coverage..."
    
    Push-Location $ScriptDir
    
    try {
        $result = go test -coverprofile=coverage.out ./...
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Coverage tests completed"
            
            # Show coverage percentage
            $coverageOutput = go tool cover -func=coverage.out
            $totalLine = $coverageOutput | Where-Object { $_ -match "total:" }
            if ($totalLine) {
                $coverage = ($totalLine -split '\s+')[2]
                Write-Info "Total coverage: $coverage"
            }
            
            # Generate HTML coverage report
            go tool cover -html=coverage.out -o coverage.html
            Write-Success "HTML coverage report generated: coverage.html"
        }
        else {
            Write-Error "Coverage tests failed"
            return $false
        }
    }
    finally {
        Pop-Location
    }
    
    return $true
}

# Run benchmark tests
function Invoke-BenchmarkTests {
    Write-Info "Running benchmark tests..."
    
    Push-Location $ScriptDir
    
    try {
        $result = go test -bench=. -benchmem ./...
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Benchmark tests completed"
        }
        else {
            Write-Error "Benchmark tests failed"
            return $false
        }
    }
    finally {
        Pop-Location
    }
    
    return $true
}

# Run CI tests (skip Arduino CLI dependent tests)
function Invoke-CITests {
    Write-Info "Running CI tests (short mode)..."
    
    Push-Location $ScriptDir
    
    try {
        $result = go test -short -v ./...
        if ($LASTEXITCODE -eq 0) {
            Write-Success "CI tests passed"
        }
        else {
            Write-Error "CI tests failed"
            return $false
        }
    }
    finally {
        Pop-Location
    }
    
    return $true
}

# Validate specific template
function Invoke-ValidateTemplate {
    param([string]$TemplatePath)
    
    if (!(Test-Path $TemplatePath)) {
        Write-Error "Template file not found: $TemplatePath"
        return $false
    }
    
    Write-Info "Validating template: $(Split-Path $TemplatePath -Leaf)"
    
    # Extract template ID for test filtering
    try {
        $templateContent = Get-Content $TemplatePath -Raw | ConvertFrom-Json
        $templateId = $templateContent.id
    }
    catch {
        $templateId = "unknown"
        Write-Warning "Could not extract template ID from JSON"
    }
    
    Push-Location $ScriptDir
    
    try {
        # Run tests specific to this template
        $testResult = go test -v -run ".*$templateId.*" ./... 2>$null
        if ($LASTEXITCODE -ne 0) {
            # Fallback to general tests if specific tests don't exist
            $testResult = go test -v -run "TestTemplate|TestParameter|TestWiring" ./...
        }
        
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Template validation passed: $templateId"
        }
        else {
            Write-Error "Template validation failed: $templateId"
            return $false
        }
    }
    finally {
        Pop-Location
    }
    
    return $true
}

# List available templates
function Invoke-ListTemplates {
    Write-Info "Available templates:"
    
    $templateFiles = Get-ChildItem -Path $TemplateDir -Filter "*.json" -Recurse | Sort-Object Name
    
    foreach ($file in $templateFiles) {
        Write-Host "  - $($file.Name)"
    }
    
    Write-Host ""
    $templateCount = $templateFiles.Count
    Write-Info "Total templates: $templateCount"
}

# Show help
function Show-Help {
    Write-Host "Template Validation Test Runner"
    Write-Host ""
    Write-Host "Usage: .\run_tests.ps1 -Command <COMMAND> [OPTIONS]"
    Write-Host ""
    Write-Host "Commands:"
    Write-Host "  unit           Run unit tests only"
    Write-Host "  integration    Run integration tests (requires Arduino CLI)"
    Write-Host "  all            Run all tests"
    Write-Host "  coverage       Run tests with coverage report"
    Write-Host "  benchmark      Run benchmark tests"
    Write-Host "  ci             Run CI tests (skip Arduino CLI dependent)"
    Write-Host "  validate FILE  Validate specific template file"
    Write-Host "  list           List available templates"
    Write-Host "  help           Show this help message"
    Write-Host ""
    Write-Host "Parameters:"
    Write-Host "  -Command       Command to run (default: help)"
    Write-Host "  -TemplateFile  Template file to validate (for validate command)"
    Write-Host "  -TemplateDir   Template directory (default: ..\templates\arduino)"
    Write-Host "  -TestWorkspace Test workspace directory (default: .\test_workspace)"
    Write-Host "  -ArduinoCliPath Arduino CLI executable (default: arduino-cli)"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  .\run_tests.ps1 -Command all"
    Write-Host "  .\run_tests.ps1 -Command unit"
    Write-Host "  .\run_tests.ps1 -Command validate -TemplateFile template.json"
    Write-Host "  .\run_tests.ps1 -Command coverage"
    Write-Host "  .\run_tests.ps1 -Command all -TemplateDir C:\path\to\templates"
}

# Main script logic
function Main {
    # Set up cleanup trap
    try {
        switch ($Command) {
            "unit" {
                $arduinoCliAvailable = Invoke-Setup
                if (!(Invoke-UnitTests)) {
                    exit 1
                }
            }
            "integration" {
                $arduinoCliAvailable = Invoke-Setup
                if ($arduinoCliAvailable) {
                    if (!(Invoke-IntegrationTests)) {
                        exit 1
                    }
                }
                else {
                    Write-Warning "Skipping integration tests (Arduino CLI not available)"
                }
            }
            "all" {
                $arduinoCliAvailable = Invoke-Setup
                if (!(Invoke-AllTests)) {
                    exit 1
                }
            }
            "coverage" {
                $arduinoCliAvailable = Invoke-Setup
                if (!(Invoke-CoverageTests)) {
                    exit 1
                }
            }
            "benchmark" {
                $arduinoCliAvailable = Invoke-Setup
                if (!(Invoke-BenchmarkTests)) {
                    exit 1
                }
            }
            "ci" {
                $arduinoCliAvailable = Invoke-Setup
                if (!(Invoke-CITests)) {
                    exit 1
                }
            }
            "validate" {
                if ([string]::IsNullOrEmpty($TemplateFile)) {
                    Write-Error "Please specify a template file using -TemplateFile"
                    Show-Help
                    exit 1
                }
                $arduinoCliAvailable = Invoke-Setup
                if (!(Invoke-ValidateTemplate -TemplatePath $TemplateFile)) {
                    exit 1
                }
            }
            "list" {
                Invoke-ListTemplates
            }
            "help" {
                Show-Help
            }
            default {
                Write-Error "Unknown command: $Command"
                Show-Help
                exit 1
            }
        }
    }
    catch {
        Write-Error "Script failed: $($_.Exception.Message)"
        exit 1
    }
    finally {
        Invoke-Cleanup
    }
}

# Run main function
Main
