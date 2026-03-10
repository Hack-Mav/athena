package provisioning

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/logger"
	"github.com/athena/platform-lib/pkg/template"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockTemplateService is a mock implementation of template service
type MockTemplateService struct {
	mock.Mock
}

func (m *MockTemplateService) GetTemplate(ctx context.Context, id string, version string) (*template.Template, error) {
	args := m.Called(ctx, id, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*template.Template), args.Error(1)
}

func (m *MockTemplateService) ValidateParameters(ctx context.Context, tmpl *template.Template, parameters map[string]interface{}) (*template.ValidationResult, error) {
	args := m.Called(ctx, tmpl, parameters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*template.ValidationResult), args.Error(1)
}

func (m *MockTemplateService) RenderTemplate(ctx context.Context, id string, version string, parameters map[string]interface{}) (*template.RenderedTemplate, error) {
	args := m.Called(ctx, id, version, parameters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*template.RenderedTemplate), args.Error(1)
}

// MockArduinoCLI is a mock implementation of Arduino CLI
type MockArduinoCLI struct {
	mock.Mock
}

func (m *MockArduinoCLI) Execute(ctx context.Context, args []string) (string, error) {
	callArgs := m.Called(ctx, args)
	return callArgs.String(0), callArgs.Error(1)
}

func (m *MockArduinoCLI) ExecuteWithProgress(ctx context.Context, args []string, progressCallback func(string)) (string, error) {
	callArgs := m.Called(ctx, args, mock.Anything)
	return callArgs.String(0), callArgs.Error(1)
}

// ProvisioningTestSuite contains integration tests for the provisioning workflow
type ProvisioningTestSuite struct {
	suite.Suite
	service         *Service
	mockTemplateSvc *MockTemplateService
	mockArduinoCLI  *MockArduinoCLI
	tempDir         string
	workspaceDir    string
	cacheDir        string
	artifactDir     string
}

func (suite *ProvisioningTestSuite) SetupSuite() {
	// Create temporary directories for testing
	tempDir, err := os.MkdirTemp("", "athena-provisioning-test")
	suite.Require().NoError(err)
	suite.tempDir = tempDir

	suite.workspaceDir = filepath.Join(tempDir, "workspace")
	suite.cacheDir = filepath.Join(tempDir, "cache")
	suite.artifactDir = filepath.Join(tempDir, "artifacts")

	err = os.MkdirAll(suite.workspaceDir, 0755)
	suite.Require().NoError(err)
	err = os.MkdirAll(suite.cacheDir, 0755)
	suite.Require().NoError(err)
	err = os.MkdirAll(suite.artifactDir, 0755)
	suite.Require().NoError(err)
}

func (suite *ProvisioningTestSuite) SetupTest() {
	suite.mockTemplateSvc = new(MockTemplateService)
	suite.mockArduinoCLI = new(MockArduinoCLI)

	cfg := &config.Config{
		LogLevel:       "debug",
		ServiceName:    "test-provisioning-service",
		ArduinoCLIPath: "arduino-cli",
	}

	logger := logger.New("debug", "test")

	// Create service with mocked dependencies
	suite.service = &Service{
		config:          cfg,
		logger:          logger,
		cli:             &ArduinoCLI{cliPath: "arduino-cli"},
		boardManager:    NewBoardManager(&ArduinoCLI{cliPath: "arduino-cli"}),
		libraryManager:  NewLibraryManager(&ArduinoCLI{cliPath: "arduino-cli"}),
		compiler:        NewCompiler(&ArduinoCLI{cliPath: "arduino-cli"}, suite.workspaceDir, suite.cacheDir),
		artifactManager: NewArtifactManager(suite.artifactDir),
		flasher:         NewFlasher(&ArduinoCLI{cliPath: "arduino-cli"}),
	}
}

func (suite *ProvisioningTestSuite) TearDownSuite() {
	os.RemoveAll(suite.tempDir)
}

func (suite *ProvisioningTestSuite) TestEndToEndProvisioningWorkflow() {
	ctx := context.Background()

	// Create a test template
	testTemplate := &template.Template{
		ID:              "temperature-sensor",
		Version:         "1.0.0",
		Name:            "Temperature Sensor",
		Category:        "sensing",
		Description:     "DHT22 temperature and humidity sensor",
		BoardsSupported: []string{"arduino:avr:uno"},
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"sensorPin": map[string]interface{}{
					"type":    "integer",
					"minimum": 2,
					"maximum": 13,
				},
				"interval": map[string]interface{}{
					"type":    "integer",
					"default": 2000,
				},
			},
			"required": []string{"sensorPin"},
		},
		Parameters: map[string]interface{}{
			"sensorPin": 2,
			"interval":  2000,
		},
		Libraries: []template.LibraryDependency{
			{Name: "DHT sensor library", Version: "1.4.4"},
			{Name: "Adafruit Unified Sensor", Version: "1.1.9"},
		},
		Assets: []template.Asset{
			{
				Type: "code",
				Path: "/templates/temperature_sensor.ino",
				Metadata: map[string]interface{}{
					"content": `#include <DHT.h>
#include <Adafruit_Sensor.h>

#define DHTPIN {{.sensorPin}}
#define DHTTYPE DHT22

DHT dht(DHTPIN, DHTTYPE);

void setup() {
  Serial.begin(9600);
  dht.begin();
}

void loop() {
  delay({{.interval}});
  float h = dht.readHumidity();
  float t = dht.readTemperature();
  
  Serial.print("Humidity: ");
  Serial.print(h);
  Serial.print(" %\t");
  Serial.print("Temperature: ");
  Serial.print(t);
  Serial.println(" *C");
}`,
				},
			},
		},
	}

	parameters := map[string]interface{}{
		"sensorPin": 2,
		"interval":  1000,
	}

	// Mock template service responses
	suite.mockTemplateSvc.On("GetTemplate", ctx, "temperature-sensor", "1.0.0").Return(testTemplate, nil)
	suite.mockTemplateSvc.On("ValidateParameters", ctx, testTemplate, parameters).Return(&template.ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}, nil)

	renderedTemplate := &template.RenderedTemplate{
		Template:   testTemplate,
		Parameters: parameters,
		RenderedCode: `#include <DHT.h>
#include <Adafruit_Sensor.h>

#define DHTPIN 2
#define DHTTYPE DHT22

DHT dht(DHTPIN, DHTTYPE);

void setup() {
  Serial.begin(9600);
  dht.begin();
}

void loop() {
  delay(1000);
  float h = dht.readHumidity();
  float t = dht.readTemperature();
  
  Serial.print("Humidity: ");
  Serial.print(h);
  Serial.print(" %\t");
  Serial.print("Temperature: ");
  Serial.print(t);
  Serial.println(" *C");
}`,
		Assets: testTemplate.Assets,
	}

	suite.mockTemplateSvc.On("RenderTemplate", ctx, "temperature-sensor", "1.0.0", parameters).Return(renderedTemplate, nil)

	// Mock Arduino CLI responses
	suite.mockArduinoCLI.On("Execute", ctx, []string{"board", "list"}).Return(`[
  {
    "fqbn": "arduino:avr:uno",
    "name": "Arduino Uno",
    "platform": "arduino:avr",
    "is_detected": true
  }
]`, nil)

	suite.mockArduinoCLI.On("Execute", ctx, []string{"lib", "install", "DHT sensor library@1.4.4"}).Return("Library installed successfully", nil)
	suite.mockArduinoCLI.On("Execute", ctx, []string{"lib", "install", "Adafruit Unified Sensor@1.1.9"}).Return("Library installed successfully", nil)

	suite.mockArduinoCLI.On("Execute", ctx, []string{"compile", "--fqbn", "arduino:avr:uno", "--build-path", suite.workspaceDir}).Return("Compilation successful", nil)

	suite.mockArduinoCLI.On("ExecuteWithProgress", ctx, []string{"upload", "--fqbn", "arduino:avr:uno", "--port", "/dev/ttyUSB0", filepath.Join(suite.workspaceDir, "temperature_sensor.ino")}, mock.Anything).Return("Upload successful", nil)

	// Test the complete provisioning workflow
	result, err := suite.service.ProvisionDevice(ctx, &ProvisioningRequest{
		TemplateID:      "temperature-sensor",
		TemplateVersion: "1.0.0",
		BoardFQBN:       "arduino:avr:uno",
		Port:            "/dev/ttyUSB0",
		Parameters:      parameters,
	})

	suite.Require().NoError(err)
	suite.Assert().NotNil(result)
	suite.Assert().True(result.Success)
	suite.Assert().Equal("Device provisioned successfully", result.Message)
	suite.Assert().NotEmpty(result.BuildArtifactPath)
	suite.Assert().NotEmpty(result.SerialOutput)

	// Verify mock expectations
	suite.mockTemplateSvc.AssertExpectations(suite.T())
	suite.mockArduinoCLI.AssertExpectations(suite.T())
}

func (suite *ProvisioningTestSuite) TestProvisioningWithInvalidParameters() {
	ctx := context.Background()

	testTemplate := &template.Template{
		ID:              "temperature-sensor",
		Version:         "1.0.0",
		Name:            "Temperature Sensor",
		BoardsSupported: []string{"arduino:avr:uno"},
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"sensorPin": map[string]interface{}{
					"type":    "integer",
					"minimum": 2,
					"maximum": 13,
				},
			},
			"required": []string{"sensorPin"},
		},
	}

	// Invalid parameters - missing required sensorPin
	parameters := map[string]interface{}{
		"interval": 1000,
	}

	suite.mockTemplateSvc.On("GetTemplate", ctx, "temperature-sensor", "1.0.0").Return(testTemplate, nil)
	suite.mockTemplateSvc.On("ValidateParameters", ctx, testTemplate, parameters).Return(&template.ValidationResult{
		Valid:    false,
		Errors:   []string{"Missing required parameter: sensorPin"},
		Warnings: []string{},
	}, nil)

	result, err := suite.service.ProvisionDevice(ctx, &ProvisioningRequest{
		TemplateID:      "temperature-sensor",
		TemplateVersion: "1.0.0",
		BoardFQBN:       "arduino:avr:uno",
		Port:            "/dev/ttyUSB0",
		Parameters:      parameters,
	})

	suite.Require().NoError(err)
	suite.Assert().NotNil(result)
	suite.Assert().False(result.Success)
	suite.Assert().Contains(result.Message, "validation failed")
	suite.Assert().Contains(result.Errors, "Missing required parameter: sensorPin")
}

func (suite *ProvisioningTestSuite) TestProvisioningWithUnsupportedBoard() {
	ctx := context.Background()

	testTemplate := &template.Template{
		ID:              "temperature-sensor",
		Version:         "1.0.0",
		Name:            "Temperature Sensor",
		BoardsSupported: []string{"arduino:avr:uno"}, // Only supports Uno
	}

	parameters := map[string]interface{}{
		"sensorPin": 2,
		"interval":  1000,
	}

	suite.mockTemplateSvc.On("GetTemplate", ctx, "temperature-sensor", "1.0.0").Return(testTemplate, nil)

	// Try to provision with unsupported board
	result, err := suite.service.ProvisionDevice(ctx, &ProvisioningRequest{
		TemplateID:      "temperature-sensor",
		TemplateVersion: "1.0.0",
		BoardFQBN:       "arduino:avr:mega", // Mega is not supported
		Port:            "/dev/ttyUSB0",
		Parameters:      parameters,
	})

	suite.Require().NoError(err)
	suite.Assert().NotNil(result)
	suite.Assert().False(result.Success)
	suite.Assert().Contains(result.Message, "board not supported")
}

func (suite *ProvisioningTestSuite) TestProvisioningWithCompilationFailure() {
	ctx := context.Background()

	testTemplate := &template.Template{
		ID:              "temperature-sensor",
		Version:         "1.0.0",
		Name:            "Temperature Sensor",
		BoardsSupported: []string{"arduino:avr:uno"},
	}

	parameters := map[string]interface{}{
		"sensorPin": 2,
		"interval":  1000,
	}

	suite.mockTemplateSvc.On("GetTemplate", ctx, "temperature-sensor", "1.0.0").Return(testTemplate, nil)
	suite.mockTemplateSvc.On("ValidateParameters", ctx, testTemplate, parameters).Return(&template.ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}, nil)

	renderedTemplate := &template.RenderedTemplate{
		Template:     testTemplate,
		Parameters:   parameters,
		RenderedCode: "// Invalid Arduino code that will fail compilation\ninvalid_syntax_here",
		Assets:       testTemplate.Assets,
	}

	suite.mockTemplateSvc.On("RenderTemplate", ctx, "temperature-sensor", "1.0.0", parameters).Return(renderedTemplate, nil)

	suite.mockArduinoCLI.On("Execute", ctx, []string{"board", "list"}).Return(`[{"fqbn": "arduino:avr:uno", "name": "Arduino Uno"}]`, nil)
	suite.mockArduinoCLI.On("Execute", ctx, []string{"compile", "--fqbn", "arduino:avr:uno", "--build-path", suite.workspaceDir}).Return("", fmt.Errorf("compilation failed: invalid syntax"))

	result, err := suite.service.ProvisionDevice(ctx, &ProvisioningRequest{
		TemplateID:      "temperature-sensor",
		TemplateVersion: "1.0.0",
		BoardFQBN:       "arduino:avr:uno",
		Port:            "/dev/ttyUSB0",
		Parameters:      parameters,
	})

	suite.Require().NoError(err)
	suite.Assert().NotNil(result)
	suite.Assert().False(result.Success)
	suite.Assert().Contains(result.Message, "compilation failed")
}

func (suite *ProvisioningTestSuite) TestProvisioningWithDeviceNotConnected() {
	ctx := context.Background()

	testTemplate := &template.Template{
		ID:              "temperature-sensor",
		Version:         "1.0.0",
		Name:            "Temperature Sensor",
		BoardsSupported: []string{"arduino:avr:uno"},
	}

	parameters := map[string]interface{}{
		"sensorPin": 2,
		"interval":  1000,
	}

	suite.mockTemplateSvc.On("GetTemplate", ctx, "temperature-sensor", "1.0.0").Return(testTemplate, nil)
	suite.mockTemplateSvc.On("ValidateParameters", ctx, testTemplate, parameters).Return(&template.ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}, nil)

	renderedTemplate := &template.RenderedTemplate{
		Template:     testTemplate,
		Parameters:   parameters,
		RenderedCode: "// Valid Arduino code\nvoid setup() {}\nvoid loop() {}",
		Assets:       testTemplate.Assets,
	}

	suite.mockTemplateSvc.On("RenderTemplate", ctx, "temperature-sensor", "1.0.0", parameters).Return(renderedTemplate, nil)

	suite.mockArduinoCLI.On("Execute", ctx, []string{"board", "list"}).Return(`[{"fqbn": "arduino:avr:uno", "name": "Arduino Uno"}]`, nil)
	suite.mockArduinoCLI.On("Execute", ctx, []string{"compile", "--fqbn", "arduino:avr:uno", "--build-path", suite.workspaceDir}).Return("Compilation successful", nil)
	suite.mockArduinoCLI.On("ExecuteWithProgress", ctx, []string{"upload", "--fqbn", "arduino:avr:uno", "--port", "/dev/ttyUSB0", mock.Anything}, mock.Anything).Return("", fmt.Errorf("device not found on port /dev/ttyUSB0"))

	result, err := suite.service.ProvisionDevice(ctx, &ProvisioningRequest{
		TemplateID:      "temperature-sensor",
		TemplateVersion: "1.0.0",
		BoardFQBN:       "arduino:avr:uno",
		Port:            "/dev/ttyUSB0",
		Parameters:      parameters,
	})

	suite.Require().NoError(err)
	suite.Assert().NotNil(result)
	suite.Assert().False(result.Success)
	suite.Assert().Contains(result.Message, "device not found")
}

func (suite *ProvisioningTestSuite) TestProvisioningProgressTracking() {
	ctx := context.Background()

	testTemplate := &template.Template{
		ID:              "temperature-sensor",
		Version:         "1.0.0",
		Name:            "Temperature Sensor",
		BoardsSupported: []string{"arduino:avr:uno"},
	}

	parameters := map[string]interface{}{
		"sensorPin": 2,
		"interval":  1000,
	}

	suite.mockTemplateSvc.On("GetTemplate", ctx, "temperature-sensor", "1.0.0").Return(testTemplate, nil)
	suite.mockTemplateSvc.On("ValidateParameters", ctx, testTemplate, parameters).Return(&template.ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}, nil)

	renderedTemplate := &template.RenderedTemplate{
		Template:     testTemplate,
		Parameters:   parameters,
		RenderedCode: "// Valid Arduino code\nvoid setup() {}\nvoid loop() {}",
		Assets:       testTemplate.Assets,
	}

	suite.mockTemplateSvc.On("RenderTemplate", ctx, "temperature-sensor", "1.0.0", parameters).Return(renderedTemplate, nil)

	suite.mockArduinoCLI.On("Execute", ctx, []string{"board", "list"}).Return(`[{"fqbn": "arduino:avr:uno", "name": "Arduino Uno"}]`, nil)
	suite.mockArduinoCLI.On("Execute", ctx, []string{"compile", "--fqbn", "arduino:avr:uno", "--build-path", suite.workspaceDir}).Return("Compilation successful", nil)

	// Mock progress tracking during upload
	progressUpdates := []string{"Starting upload...", "Uploading sketch...", "Upload complete!"}
	suite.mockArduinoCLI.On("ExecuteWithProgress", ctx, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		progressCallback := args.Get(2).(func(string))
		// Simulate progress updates
		for _, update := range progressUpdates {
			progressCallback(update)
			time.Sleep(10 * time.Millisecond) // Small delay to simulate real progress
		}
	}).Return("Upload successful", nil)

	// Track progress updates
	var receivedProgress []string
	progressCallback := func(progress string) {
		receivedProgress = append(receivedProgress, progress)
	}

	result, err := suite.service.ProvisionDeviceWithProgress(ctx, &ProvisioningRequest{
		TemplateID:      "temperature-sensor",
		TemplateVersion: "1.0.0",
		BoardFQBN:       "arduino:avr:uno",
		Port:            "/dev/ttyUSB0",
		Parameters:      parameters,
	}, progressCallback)

	suite.Require().NoError(err)
	suite.Assert().NotNil(result)
	suite.Assert().True(result.Success)
	suite.Assert().NotEmpty(receivedProgress)

	// Verify all progress updates were received
	for _, expectedUpdate := range progressUpdates {
		suite.Assert().Contains(receivedProgress, expectedUpdate)
	}
}

// Additional comprehensive integration tests

func (suite *ProvisioningTestSuite) TestProvisioningWithLibraryInstallationFailure() {
	ctx := context.Background()

	testTemplate := &template.Template{
		ID:              "complex-sensor",
		Version:         "1.0.0",
		Name:            "Complex Sensor Template",
		BoardsSupported: []string{"arduino:avr:uno"},
		Libraries: []template.LibraryDependency{
			{Name: "NonExistentLibrary", Version: "1.0.0"},
			{Name: "AnotherLibrary", Version: "2.0.0"},
		},
	}

	parameters := map[string]interface{}{
		"sensorPin": 2,
	}

	suite.mockTemplateSvc.On("GetTemplate", ctx, "complex-sensor", "1.0.0").Return(testTemplate, nil)
	suite.mockTemplateSvc.On("ValidateParameters", ctx, testTemplate, parameters).Return(&template.ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}, nil)

	renderedTemplate := &template.RenderedTemplate{
		Template:     testTemplate,
		Parameters:   parameters,
		RenderedCode: "// Valid Arduino code\nvoid setup() {}\nvoid loop() {}",
		Assets:       testTemplate.Assets,
	}

	suite.mockTemplateSvc.On("RenderTemplate", ctx, "complex-sensor", "1.0.0", parameters).Return(renderedTemplate, nil)

	suite.mockArduinoCLI.On("Execute", ctx, []string{"board", "list"}).Return(`[{"fqbn": "arduino:avr:uno", "name": "Arduino Uno"}]`, nil)

	// Mock library installation failures
	suite.mockArduinoCLI.On("Execute", ctx, []string{"lib", "install", "NonExistentLibrary@1.0.0"}).Return("", fmt.Errorf("library not found"))
	suite.mockArduinoCLI.On("Execute", ctx, []string{"lib", "install", "AnotherLibrary@2.0.0"}).Return("Library installed successfully", nil)

	result, err := suite.service.ProvisionDevice(ctx, &ProvisioningRequest{
		TemplateID:      "complex-sensor",
		TemplateVersion: "1.0.0",
		BoardFQBN:       "arduino:avr:uno",
		Port:            "/dev/ttyUSB0",
		Parameters:      parameters,
	})

	suite.Require().NoError(err)
	suite.Assert().NotNil(result)
	suite.Assert().False(result.Success)
	suite.Assert().Contains(result.Message, "library installation failed")
}

func (suite *ProvisioningTestSuite) TestProvisioningWithMultipleBoardTypes() {
	ctx := context.Background()

	testTemplate := &template.Template{
		ID:              "multi-board-sensor",
		Version:         "1.0.0",
		Name:            "Multi-Board Sensor",
		BoardsSupported: []string{"arduino:avr:uno", "arduino:avr:mega", "esp32:esp32:devkitv1"},
	}

	parameters := map[string]interface{}{
		"sensorPin": 2,
	}

	suite.mockTemplateSvc.On("GetTemplate", ctx, "multi-board-sensor", "1.0.0").Return(testTemplate, nil)
	suite.mockTemplateSvc.On("ValidateParameters", ctx, testTemplate, parameters).Return(&template.ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}, nil)

	renderedTemplate := &template.RenderedTemplate{
		Template:     testTemplate,
		Parameters:   parameters,
		RenderedCode: "// Valid Arduino code\nvoid setup() {}\nvoid loop() {}",
		Assets:       testTemplate.Assets,
	}

	suite.mockTemplateSvc.On("RenderTemplate", ctx, "multi-board-sensor", "1.0.0", parameters).Return(renderedTemplate, nil)

	// Test with different board types
	testCases := []struct {
		boardFQBN string
		boardName string
	}{
		{"arduino:avr:uno", "Arduino Uno"},
		{"arduino:avr:mega", "Arduino Mega"},
		{"esp32:esp32:devkitv1", "ESP32 DevKit"},
	}

	for _, tc := range testCases {
		suite.mockArduinoCLI.On("Execute", ctx, []string{"board", "list"}).Return(fmt.Sprintf(`[{"fqbn": "%s", "name": "%s"}]`, tc.boardFQBN, tc.boardName), nil)
		suite.mockArduinoCLI.On("Execute", ctx, []string{"compile", "--fqbn", tc.boardFQBN, "--build-path", suite.workspaceDir}).Return("Compilation successful", nil)
		suite.mockArduinoCLI.On("ExecuteWithProgress", ctx, []string{"upload", "--fqbn", tc.boardFQBN, "--port", "/dev/ttyUSB0", mock.Anything}, mock.Anything).Return("Upload successful", nil)

		result, err := suite.service.ProvisionDevice(ctx, &ProvisioningRequest{
			TemplateID:      "multi-board-sensor",
			TemplateVersion: "1.0.0",
			BoardFQBN:       tc.boardFQBN,
			Port:            "/dev/ttyUSB0",
			Parameters:      parameters,
		})

		suite.Require().NoError(err)
		suite.Assert().NotNil(result)
		suite.Assert().True(result.Success)
		suite.Assert().Contains(result.Message, "provisioned successfully")
	}
}

func (suite *ProvisioningTestSuite) TestProvisioningWithTemplateNotFound() {
	ctx := context.Background()

	parameters := map[string]interface{}{
		"sensorPin": 2,
	}

	suite.mockTemplateSvc.On("GetTemplate", ctx, "non-existent-template", "1.0.0").Return(nil, fmt.Errorf("template not found"))

	result, err := suite.service.ProvisionDevice(ctx, &ProvisioningRequest{
		TemplateID:      "non-existent-template",
		TemplateVersion: "1.0.0",
		BoardFQBN:       "arduino:avr:uno",
		Port:            "/dev/ttyUSB0",
		Parameters:      parameters,
	})

	suite.Require().NoError(err)
	suite.Assert().NotNil(result)
	suite.Assert().False(result.Success)
	suite.Assert().Contains(result.Message, "template not found")
}

func (suite *ProvisioningTestSuite) TestProvisioningWithArtifactManagement() {
	ctx := context.Background()

	testTemplate := &template.Template{
		ID:              "artifact-test",
		Version:         "1.0.0",
		Name:            "Artifact Test Template",
		BoardsSupported: []string{"arduino:avr:uno"},
	}

	parameters := map[string]interface{}{
		"sensorPin": 2,
	}

	suite.mockTemplateSvc.On("GetTemplate", ctx, "artifact-test", "1.0.0").Return(testTemplate, nil)
	suite.mockTemplateSvc.On("ValidateParameters", ctx, testTemplate, parameters).Return(&template.ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}, nil)

	renderedTemplate := &template.RenderedTemplate{
		Template:     testTemplate,
		Parameters:   parameters,
		RenderedCode: "// Valid Arduino code\nvoid setup() {}\nvoid loop() {}",
		Assets:       testTemplate.Assets,
	}

	suite.mockTemplateSvc.On("RenderTemplate", ctx, "artifact-test", "1.0.0", parameters).Return(renderedTemplate, nil)

	suite.mockArduinoCLI.On("Execute", ctx, []string{"board", "list"}).Return(`[{"fqbn": "arduino:avr:uno", "name": "Arduino Uno"}]`, nil)
	suite.mockArduinoCLI.On("Execute", ctx, []string{"compile", "--fqbn", "arduino:avr:uno", "--build-path", suite.workspaceDir}).Return("Compilation successful", nil)
	suite.mockArduinoCLI.On("ExecuteWithProgress", ctx, []string{"upload", "--fqbn", "arduino:avr:uno", "--port", "/dev/ttyUSB0", mock.Anything}, mock.Anything).Return("Upload successful", nil)

	result, err := suite.service.ProvisionDevice(ctx, &ProvisioningRequest{
		TemplateID:      "artifact-test",
		TemplateVersion: "1.0.0",
		BoardFQBN:       "arduino:avr:uno",
		Port:            "/dev/ttyUSB0",
		Parameters:      parameters,
	})

	suite.Require().NoError(err)
	suite.Assert().NotNil(result)
	suite.Assert().True(result.Success)
	suite.Assert().NotEmpty(result.BuildArtifactPath)

	// Verify artifact file exists
	_, err = os.Stat(result.BuildArtifactPath)
	suite.Require().NoError(err)

	// Verify result has expected metadata
	suite.Assert().NotEmpty(result.Message)
	suite.Assert().Empty(result.Errors)
}

func TestProvisioningIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ProvisioningTestSuite))
}
