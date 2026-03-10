# Template Validation Tests

This directory contains comprehensive test suites for validating Arduino templates in the ATHENA system.

## Test Suites

### 1. Template Validation Tests (`template_validation_test.go`)

**Purpose**: Core template structure and schema validation

**Coverage**:
- Template compilation across supported Arduino boards
- Parameter schema validation and default value handling
- Required field validation
- Template variable syntax validation
- JSON schema compliance

**Key Functions**:
- `TestTemplateCompilation` - Tests compilation compatibility
- `TestParameterSchemas` - Validates JSON schemas and parameters
- `TestWiringDiagrams` - Tests wiring specification validation

### 2. Arduino Compilation Tests (`arduino_compilation_test.go`)

**Purpose**: Real Arduino compilation and library dependency testing

**Coverage**:
- Actual Arduino CLI compilation (when available)
- Library dependency resolution and compatibility
- Arduino code template syntax validation
- Board-specific features and limitations
- Pin assignment validation

**Key Functions**:
- `TestRealArduinoCompilation` - Real compilation with Arduino CLI
- `TestLibraryDependencies` - Library format and compatibility
- `TestCodeTemplateSyntax` - Arduino code validation
- `TestBoardSpecificFeatures` - Board-specific validation

### 3. Wiring Validation Tests (`wiring_validation_test.go`)

**Purpose**: Wiring diagram generation and component compatibility testing

**Coverage**:
- Wiring diagram generation and export formats
- Component compatibility validation
- Connection safety and integrity checks
- Power and voltage compatibility
- Signal integrity validation

**Key Functions**:
- `TestWiringDiagramGeneration` - Diagram generation testing
- `TestComponentCompatibility` - Component compatibility rules
- `TestConnectionValidation` - Connection safety and validity
- `TestWiringDiagramFormats` - Multiple format support

## Running Tests

### Prerequisites

1. **Go 1.21+** - Required for running the test suites
2. **Arduino CLI** (optional) - For real compilation tests
   ```bash
   # Install Arduino CLI
   curl -fsSL https://raw.githubusercontent.com/arduino/arduino-cli/master/install.sh | sh
   ```

### Running All Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...
```

### Running Specific Test Suites

```bash
# Template validation tests
go test -v -run TestTemplate

# Arduino compilation tests
go test -v -run TestRealArduino

# Wiring validation tests
go test -v -run TestWiring
```

### Running Tests in CI Mode

```bash
# Skip tests that require Arduino CLI
go test -short ./...

# Run benchmarks
go test -bench=. ./...
```

## Test Configuration

### Environment Variables

- `ARDUINO_CLI_PATH` - Path to Arduino CLI executable (default: `arduino-cli`)
- `TEMPLATE_DIR` - Path to template directory (default: `../../templates/arduino`)
- `TEST_WORKSPACE` - Path to test workspace (default: `./test_workspace`)

### Test Categories

1. **Unit Tests** - Fast tests that don't require external tools
2. **Integration Tests** - Tests that require Arduino CLI
3. **Benchmark Tests** - Performance tests for validation operations

## Template Structure Validation

### Required Template Fields

All templates must include these fields:
- `id` - Unique template identifier
- `name` - Human-readable name
- `version` - Semantic version (e.g., "1.0.0")
- `category` - Template category (sensing, automation, connectivity)
- `description` - Detailed description
- `author` - Template author
- `boards_supported` - List of supported Arduino boards
- `libraries` - Required Arduino libraries
- `schema` - JSON schema for parameters
- `parameters` - Default parameter values
- `assets` - Template assets (code, diagrams, docs)
- `wiring_spec` - Wiring specification

### Schema Validation Rules

1. **Type Validation** - Parameters must match schema types
2. **Constraint Validation** - Minimum/maximum values must be respected
3. **Required Fields** - All required parameters must be present
4. **Default Values** - Defaults must be within constraints

### Wiring Specification Rules

1. **Component Validation** - All components must have valid definitions
2. **Pin Compatibility** - Connected pins must be compatible
3. **Power Requirements** - Total power must not exceed board capacity
4. **Safety Checks** - No short circuits or unsafe connections

## Supported Arduino Boards

### Arduino Boards
- **Arduino Uno** (`arduino:avr:uno`)
  - 13 digital pins (0-13)
  - 6 analog pins (A0-A5)
  - 5V operation
  - 500mA total power

- **Arduino Nano** (`arduino:avr:nano`)
  - 13 digital pins (0-13)
  - 8 analog pins (A0-A7)
  - 5V operation
  - 500mA total power

### ESP Boards
- **ESP32 DevKit** (`esp32:esp32:devkitv1`)
  - 39 digital pins (0-39)
  - 12 analog pins (0-11)
  - 3.3V operation
  - 300mA total power
  - Built-in WiFi and Bluetooth

- **ESP8266 D1 Mini** (`esp8266:esp8266:d1mini`)
  - 16 digital pins (0-16)
  - 1 analog pin (A0)
  - 3.3V operation
  - 200mA total power
  - Built-in WiFi

## Library Compatibility

### Supported Libraries

| Library | Compatible Boards | Version |
|---------|------------------|---------|
| DHT sensor library | Uno, Nano, ESP32 | 1.4.4 |
| Adafruit Unified Sensor | Uno, Nano, ESP32 | 1.1.9 |
| PubSubClient | All boards | 2.8 |
| ArduinoJson | All boards | 6.21.3 |
| WiFi | ESP32, ESP8266 | 2.0.0 |
| WebServer | ESP32, ESP8266 | 2.0.0 |
| EEPROM | All boards | 2.0.0 |

### Library Validation Rules

1. **Version Format** - Must follow semantic versioning
2. **Board Compatibility** - Libraries must be compatible with supported boards
3. **No Duplicates** - No duplicate libraries in template
4. **Valid Names** - Library names must be valid

## Code Template Validation

### Required Arduino Functions

All code templates must include:
- `void setup()` - Initialization function
- `void loop()` - Main program loop

### Template Variable Syntax

Template variables use Go template syntax:
- `{{.parameterName}}` - Parameter substitution
- Must be valid Go template syntax
- All variables must be defined in parameters

### Code Validation Rules

1. **Syntax Validation** - Valid Arduino C++ syntax
2. **Include Statements** - Required library includes
3. **Variable Substitution** - All template variables must be valid
4. **Function Completeness** - Required functions must be present

## Wiring Diagram Validation

### Diagram Formats

Supported export formats:
- **PNG** - Raster image format
- **SVG** - Vector format for scaling
- **PDF** - Printable document format
- **JSON** - Structured data format

### Component Types

- **board** - Arduino development board
- **sensor** - Input sensors (DHT22, ultrasonic, etc.)
- **actuator** - Output devices (relays, LEDs, motors)
- **power** - Power supply components
- **communication** - Communication modules

### Pin Types

- **power** - Power supply pins (VCC, 5V, 3.3V)
- **ground** - Ground pins (GND)
- **digital** - Digital I/O pins
- **analog** - Analog input pins
- **pwm** - PWM-capable digital pins
- **i2c** - I2C communication pins
- **spi** - SPI communication pins

### Connection Rules

1. **Type Compatibility** - Connected pins must be compatible
2. **Voltage Matching** - Power connections must match voltages
3. **No Short Circuits** - Power cannot connect directly to ground
4. **Proper Grounding** - All components need ground connections
5. **Pin Limits** - No pin should have excessive connections

## Troubleshooting

### Common Test Failures

1. **Missing Arduino CLI** - Install Arduino CLI or run with `-short` flag
2. **Library Installation** - Check internet connection for library downloads
3. **Compilation Errors** - Verify code syntax and library versions
4. **Template Validation** - Check JSON schema and parameter definitions

### Debug Mode

Run tests with verbose output for debugging:
```bash
go test -v -run TestName ./...
```

### Test Workspace

Tests use a temporary workspace directory:
- Default: `./test_workspace`
- Cleaned up automatically after tests
- Can be overridden with `TEST_WORKSPACE` environment variable

## Contributing

### Adding New Tests

1. Follow the existing test structure and naming conventions
2. Use table-driven tests for multiple test cases
3. Include both positive and negative test cases
4. Add descriptive test names and comments
5. Update this README for new test coverage

### Test Naming Conventions

- `Test[Feature]` - Main test functions
- `Benchmark[Feature]` - Performance tests
- `validate[Feature]` - Helper validation functions
- `is[Condition]` - Helper condition check functions

### Code Style

- Follow Go standard formatting (`go fmt`)
- Use testify/assert for assertions
- Use testify/require for critical assertions
- Include error messages in assertions
- Add comments for complex validation logic

## Continuous Integration

### GitHub Actions

The test suite is designed to run in CI environments:
- Unit tests run without external dependencies
- Integration tests require Arduino CLI installation
- Use `-short` flag to skip Arduino CLI-dependent tests in CI

### Test Coverage

Aim for >80% test coverage:
- Run `go test -cover ./...` to check coverage
- Use `go test -coverprofile=coverage.out ./...` for detailed reports
- Generate HTML coverage with `go tool cover -html=coverage.out`

## Performance

### Benchmark Results

Run benchmarks to track performance:
```bash
go test -bench=. ./...
```

### Optimization Tips

- Use parallel tests with `t.Parallel()`
- Cache expensive operations
- Minimize file I/O in tests
- Use table-driven tests for efficiency
