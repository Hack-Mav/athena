#!/bin/bash

# Template Validation Test Runner
# This script runs all template validation tests with proper setup and cleanup

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE_DIR="${TEMPLATE_DIR:-$TEST_DIR/../templates/arduino}"
WORKSPACE_DIR="${TEST_WORKSPACE:-$TEST_DIR/test_workspace}"
ARDUINO_CLI="${ARDUINO_CLI_PATH:-arduino-cli}"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up test workspace..."
    if [ -d "$WORKSPACE_DIR" ]; then
        rm -rf "$WORKSPACE_DIR"
    fi
    log_success "Cleanup completed"
}

# Setup function
setup() {
    log_info "Setting up test environment..."
    
    # Create workspace directory
    mkdir -p "$WORKSPACE_DIR"
    
    # Check if template directory exists
    if [ ! -d "$TEMPLATE_DIR" ]; then
        log_error "Template directory not found: $TEMPLATE_DIR"
        exit 1
    fi
    
    # Count templates
    TEMPLATE_COUNT=$(find "$TEMPLATE_DIR" -name "*.json" | wc -l)
    log_info "Found $TEMPLATE_COUNT templates to validate"
    
    # Check for Arduino CLI
    if command -v "$ARDUINO_CLI" &> /dev/null; then
        log_success "Arduino CLI found: $(which $ARDUINO_CLI)"
        ARDUINO_CLI_AVAILABLE=true
    else
        log_warning "Arduino CLI not found, some tests will be skipped"
        ARDUINO_CLI_AVAILABLE=false
    fi
    
    log_success "Setup completed"
}

# Run unit tests
run_unit_tests() {
    log_info "Running unit tests..."
    
    cd "$TEST_DIR"
    
    if go test -v -run "TestTemplate|TestParameter|TestWiring" ./...; then
        log_success "Unit tests passed"
    else
        log_error "Unit tests failed"
        return 1
    fi
}

# Run integration tests
run_integration_tests() {
    log_info "Running integration tests..."
    
    cd "$TEST_DIR"
    
    if [ "$ARDUINO_CLI_AVAILABLE" = true ]; then
        if go test -v -run "TestRealArduino|TestLibrary|TestCode" ./...; then
            log_success "Integration tests passed"
        else
            log_error "Integration tests failed"
            return 1
        fi
    else
        log_warning "Skipping integration tests (Arduino CLI not available)"
    fi
}

# Run all tests
run_all_tests() {
    log_info "Running all tests..."
    
    cd "$TEST_DIR"
    
    if go test -v ./...; then
        log_success "All tests passed"
    else
        log_error "Some tests failed"
        return 1
    fi
}

# Run tests with coverage
run_coverage_tests() {
    log_info "Running tests with coverage..."
    
    cd "$TEST_DIR"
    
    # Run tests with coverage
    if go test -coverprofile=coverage.out ./...; then
        log_success "Coverage tests completed"
        
        # Show coverage percentage
        COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
        log_info "Total coverage: $COVERAGE"
        
        # Generate HTML coverage report
        go tool cover -html=coverage.out -o coverage.html
        log_success "HTML coverage report generated: coverage.html"
    else
        log_error "Coverage tests failed"
        return 1
    fi
}

# Run benchmark tests
run_benchmark_tests() {
    log_info "Running benchmark tests..."
    
    cd "$TEST_DIR"
    
    if go test -bench=. -benchmem ./...; then
        log_success "Benchmark tests completed"
    else
        log_error "Benchmark tests failed"
        return 1
    fi
}

# Run CI tests (skip Arduino CLI dependent tests)
run_ci_tests() {
    log_info "Running CI tests (short mode)..."
    
    cd "$TEST_DIR"
    
    if go test -short -v ./...; then
        log_success "CI tests passed"
    else
        log_error "CI tests failed"
        return 1
    fi
}

# Validate specific template
validate_template() {
    local template_file="$1"
    
    if [ ! -f "$template_file" ]; then
        log_error "Template file not found: $template_file"
        return 1
    fi
    
    log_info "Validating template: $(basename "$template_file")"
    
    # Extract template ID for test filtering
    TEMPLATE_ID=$(jq -r '.id' "$template_file" 2>/dev/null || echo "unknown")
    
    cd "$TEST_DIR"
    
    # Run tests specific to this template
    if go test -v -run ".*$TEMPLATE_ID.*" ./... 2>/dev/null || \
       go test -v -run "TestTemplate|TestParameter|TestWiring" ./...; then
        log_success "Template validation passed: $TEMPLATE_ID"
    else
        log_error "Template validation failed: $TEMPLATE_ID"
        return 1
    fi
}

# List available templates
list_templates() {
    log_info "Available templates:"
    
    find "$TEMPLATE_DIR" -name "*.json" -exec basename {} \; | sort | while read -r template; do
        echo "  - $template"
    done
    
    echo
    log_info "Total templates: $(find "$TEMPLATE_DIR" -name "*.json" | wc -l)"
}

# Show help
show_help() {
    echo "Template Validation Test Runner"
    echo
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo
    echo "Commands:"
    echo "  unit           Run unit tests only"
    echo "  integration    Run integration tests (requires Arduino CLI)"
    echo "  all            Run all tests"
    echo "  coverage       Run tests with coverage report"
    echo "  benchmark      Run benchmark tests"
    echo "  ci             Run CI tests (skip Arduino CLI dependent)"
    echo "  validate FILE  Validate specific template file"
    echo "  list           List available templates"
    echo "  help           Show this help message"
    echo
    echo "Environment Variables:"
    echo "  TEMPLATE_DIR   Template directory (default: ../templates/arduino)"
    echo "  TEST_WORKSPACE Test workspace directory (default: ./test_workspace)"
    echo "  ARDUINO_CLI_PATH Arduino CLI executable (default: arduino-cli)"
    echo
    echo "Examples:"
    echo "  $0 all                    # Run all tests"
    echo "  $0 unit                   # Run unit tests only"
    echo "  $0 validate template.json # Validate specific template"
    echo "  $0 coverage               # Run with coverage report"
    echo "  TEMPLATE_DIR=/path/to/templates $0 all"
}

# Main script logic
main() {
    local command="${1:-help}"
    
    case "$command" in
        "unit")
            setup
            run_unit_tests
            ;;
        "integration")
            setup
            run_integration_tests
            ;;
        "all")
            setup
            run_all_tests
            ;;
        "coverage")
            setup
            run_coverage_tests
            ;;
        "benchmark")
            setup
            run_benchmark_tests
            ;;
        "ci")
            setup
            run_ci_tests
            ;;
        "validate")
            if [ -z "$2" ]; then
                log_error "Please specify a template file"
                show_help
                exit 1
            fi
            setup
            validate_template "$2"
            ;;
        "list")
            list_templates
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            log_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Set up cleanup trap
trap cleanup EXIT

# Run main function with all arguments
main "$@"
