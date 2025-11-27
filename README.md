# ATHENA - Arduino Template Hub & Natural-Language Provisioning

ATHENA is a comprehensive platform that makes Arduino prototyping effortless by providing curated templates, unified configuration, natural language processing for firmware generation, and one-click provisioning with optional cloud connectivity and device management.

## Features

- **Template Management**: Curated Arduino templates with metadata and dependency management
- **Natural Language Processing**: Convert project descriptions into working Arduino code
- **One-Click Provisioning**: Compile and flash firmware to Arduino devices automatically
- **Device Management**: Track and monitor deployed Arduino devices
- **OTA Updates**: Remote firmware updates with rollback capabilities
- **Telemetry Collection**: Real-time sensor data collection and visualization
- **CLI & Web Interface**: Both command-line and web-based interfaces

## Architecture

ATHENA follows a microservices architecture with the following components:

- **API Gateway**: Routes requests and handles authentication
- **Template Service**: Manages Arduino templates and metadata
- **NLP Service**: Processes natural language requirements
- **Provisioning Service**: Compiles and flashes Arduino firmware
- **Device Service**: Manages device registry and lifecycle
- **Telemetry Service**: Collects and processes device data
- **OTA Service**: Handles over-the-air firmware updates
- **CLI Tool**: Command-line interface for all operations

## Quick Start

### Prerequisites

- Go 1.21 or later
- Docker and Docker Compose
- Make

### Development Setup

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd athena-platform
   ```

2. Start the development environment:
   ```bash
   make dev-setup
   ```

3. Build all services:
   ```bash
   make build
   ```

4. Run tests:
   ```bash
   make test
   ```

### Using Docker Compose

Start all services with dependencies:
```bash
docker-compose up -d
```

Check service health:
```bash
make health-check
```

View logs:
```bash
docker-compose logs -f
```

## API Endpoints

### API Gateway (Port 8000)
- `GET /health` - Health check
- `GET /api/v1/templates` - List templates
- `POST /api/v1/nlp/parse` - Parse natural language requirements
- `POST /api/v1/provisioning/compile` - Compile template
- `GET /api/v1/devices` - List devices

### Individual Services
- Template Service: Port 8001
- NLP Service: Port 8002
- Provisioning Service: Port 8003
- Device Service: Port 8004
- Telemetry Service: Port 8005
- OTA Service: Port 8006

## CLI Usage

Build the CLI tool:
```bash
make build-cli
```

Basic commands:
```bash
# List available templates
./bin/athena-cli template list

# Generate plan from natural language
./bin/athena-cli plan "Create a temperature sensor with WiFi connectivity"

# Provision a device
./bin/athena-cli provision --template temp-sensor --port /dev/ttyUSB0

# List devices
./bin/athena-cli device list
```

## Configuration

Configuration can be provided via:
1. YAML file (`configs/config.yaml`)
2. Environment variables (prefixed with `ATHENA_`)
3. Command-line flags

### Environment Variables

```bash
export ATHENA_LOG_LEVEL=debug
export ATHENA_DATASTORE_HOST=localhost:8081
export ATHENA_REDIS_ADDR=localhost:6379
export ATHENA_MQTT_BROKER=tcp://localhost:1883
export ATHENA_LLM_API_KEY=your-openai-api-key
```

## Development

### Project Structure

```
├── services/               # Service entry points
│   ├── api-gateway/
│   ├── template-service/
│   ├── nlp-service/
│   ├── provisioning-service/
│   ├── device-service/
│   ├── telemetry-service/
│   ├── ota-service/
│   ├── secrets-service/
│   └── cli/
│   └── platform-lib/     # Shared library
├── internal/               # Internal packages (if exists)
├── pkg/                    # Shared packages (if exists)
├── build/                  # Docker files
├── configs/                # Configuration files
└── .github/workflows/      # CI/CD pipelines
```

### Building Services

Build all services:
```bash
make build
```

Build specific service:
```bash
make build-service SERVICE=template-service
```

### Running Tests

Run all tests:
```bash
make test
```

Run tests for specific service:
```bash
make test-service SERVICE=template-service
```

Generate coverage report:
```bash
make test-coverage
```

### Code Quality

Format code:
```bash
make fmt
```

Run linter:
```bash
make lint
```

Run security scan:
```bash
make security-scan
```

## Deployment

### Local Development
```bash
make dev-setup
```

### Docker Deployment
```bash
make docker-build
docker-compose up -d
```

### Production Deployment
See deployment documentation for Kubernetes manifests and production configuration.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run `make ci` to ensure all checks pass
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:
- Create an issue on GitHub
- Check the documentation
- Review the API specifications