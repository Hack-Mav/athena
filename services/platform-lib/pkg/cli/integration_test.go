package cli

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/athena/platform-lib/pkg/config"
	"github.com/athena/platform-lib/pkg/logger"
)

// MockServiceClient implements ServiceClient interface for testing
type MockServiceClient struct {
	templates map[string]Template
	devices   map[string]Device
	releases  map[string]Release
	metrics   map[string]TelemetryMetrics
}

func NewMockServiceClient() *MockServiceClient {
	return &MockServiceClient{
		templates: map[string]Template{
			"basic-led": {
				ID:          "basic-led",
				Name:        "Basic LED Blink",
				Description: "Simple LED blinking example",
				Category:    "beginner",
				Tags:        []string{"led", "basic"},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			"sensor-dht": {
				ID:          "sensor-dht",
				Name:        "DHT22 Sensor",
				Description: "Temperature and humidity sensor",
				Category:    "sensors",
				Tags:        []string{"dht22", "temperature", "humidity"},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		},
		devices: map[string]Device{
			"device-001": {
				ID:        "device-001",
				Name:      "Arduino Uno 1",
				Board:     "arduino:avr:uno",
				Status:    "online",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		releases: map[string]Release{
			"v1.0.0": {
				ID:          "v1.0.0",
				Version:     "1.0.0",
				Description: "Initial release",
				CreatedAt:   time.Now(),
				ArtifactID:  "artifact-001",
			},
		},
		metrics: map[string]TelemetryMetrics{
			"device-001": {
				DeviceID:  "device-001",
				Timestamp: time.Now(),
				Metrics: map[string]interface{}{
					"temperature": 23.5,
					"humidity":    45.2,
					"status":      "ok",
				},
			},
		},
	}
}

func (m *MockServiceClient) ListTemplates(ctx context.Context) ([]Template, error) {
	var templates []Template
	for _, tmpl := range m.templates {
		templates = append(templates, tmpl)
	}
	return templates, nil
}

func (m *MockServiceClient) GetTemplate(ctx context.Context, id string) (*Template, error) {
	tmpl, exists := m.templates[id]
	if !exists {
		return nil, &httpError{StatusCode: 404, Message: "template not found"}
	}
	return &tmpl, nil
}

func (m *MockServiceClient) Compile(ctx context.Context, req *CompileRequest) (*CompileResponse, error) {
	if req.TemplateID == "" || req.Board == "" {
		return nil, &httpError{StatusCode: 400, Message: "missing required fields"}
	}
	return &CompileResponse{
		ArtifactID: "artifact-" + req.TemplateID,
		Status:     "success",
		Message:    "Compilation successful",
	}, nil
}

func (m *MockServiceClient) Flash(ctx context.Context, req *FlashRequest) (*FlashResponse, error) {
	if req.Port == "" || req.Board == "" || req.ArtifactID == "" {
		return nil, &httpError{StatusCode: 400, Message: "missing required fields"}
	}
	return &FlashResponse{
		Success: true,
		Message: "Flash successful",
	}, nil
}

func (m *MockServiceClient) ListDevices(ctx context.Context) ([]Device, error) {
	var devices []Device
	for _, dev := range m.devices {
		devices = append(devices, dev)
	}
	return devices, nil
}

func (m *MockServiceClient) GetDevice(ctx context.Context, id string) (*Device, error) {
	dev, exists := m.devices[id]
	if !exists {
		return nil, &httpError{StatusCode: 404, Message: "device not found"}
	}
	return &dev, nil
}

func (m *MockServiceClient) GetTelemetryMetrics(ctx context.Context, deviceID string) (*TelemetryMetrics, error) {
	metrics, exists := m.metrics[deviceID]
	if !exists {
		return nil, &httpError{StatusCode: 404, Message: "metrics not found"}
	}
	return &metrics, nil
}

func (m *MockServiceClient) ListReleases(ctx context.Context) ([]Release, error) {
	var releases []Release
	for _, rel := range m.releases {
		releases = append(releases, rel)
	}
	return releases, nil
}

type httpError struct {
	StatusCode int
	Message    string
}

func (e *httpError) Error() string {
	return e.Message
}

func setupTestEnvironment(t *testing.T) (string, func()) {
	tempDir, err := os.MkdirTemp("", "athena-cli-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Set up fake home directory
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)

	cleanup := func() {
		os.Setenv("HOME", originalHome)
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

func TestCLIIntegration(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create test config
	cfg := &config.Config{
		Services: map[string]string{
			"template-service":     "http://localhost:8001",
			"provisioning-service": "http://localhost:8003",
			"device-service":       "http://localhost:8004",
			"telemetry-service":    "http://localhost:8005",
			"ota-service":          "http://localhost:8006",
		},
	}

	logger := logger.New("info", "athena-cli-test")
	mockClient := NewMockServiceClient()

	// Test root command
	t.Run("RootCommand", func(t *testing.T) {
		rootCmd := NewRootCommand(cfg, logger)
		if rootCmd == nil {
			t.Fatal("Root command should not be nil")
		}

		if rootCmd.Use != "athena" {
			t.Errorf("Expected root command use 'athena', got '%s'", rootCmd.Use)
		}

		// Check that all subcommands are added
		expectedCommands := []string{"template", "provision", "device", "plan", "profile", "telemetry", "ota"}
		for _, expected := range expectedCommands {
			found := false
			for _, cmd := range rootCmd.Commands() {
				if cmd.Name() == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected subcommand '%s' not found", expected)
			}
		}
	})

	// Test template commands
	t.Run("TemplateCommands", func(t *testing.T) {
		testTemplateCommands(t, cfg, logger, mockClient)
	})

	// Test profile commands
	t.Run("ProfileCommands", func(t *testing.T) {
		testProfileCommands(t, cfg, logger)
	})

	// Test device commands
	t.Run("DeviceCommands", func(t *testing.T) {
		testDeviceCommands(t, cfg, logger, mockClient)
	})

	// Test telemetry commands
	t.Run("TelemetryCommands", func(t *testing.T) {
		testTelemetryCommands(t, cfg, logger, mockClient)
	})

	// Test OTA commands
	t.Run("OTACommands", func(t *testing.T) {
		testOTACommands(t, cfg, logger, mockClient)
	})
}

func testTemplateCommands(t *testing.T, cfg *config.Config, logger *logger.Logger, client *MockServiceClient) {
	// Test template list
	t.Run("TemplateList", func(t *testing.T) {
		buf := new(bytes.Buffer)
		cmd := newTemplateListCommand(cfg, logger)
		cmd.SetOut(buf)

		// Override the client creation in the command
		// This would require dependency injection in a real implementation
		// For now, we just test the command structure

		err := cmd.Execute()
		if err != nil {
			t.Errorf("Template list command failed: %v", err)
		}

		output := buf.String()
		if output == "" {
			t.Error("Template list command should produce output")
		}
	})

	// Test template inspect
	t.Run("TemplateInspect", func(t *testing.T) {
		buf := new(bytes.Buffer)
		cmd := newTemplateInspectCommand(cfg, logger)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"basic-led"})

		err := cmd.Execute()
		if err != nil {
			t.Errorf("Template inspect command failed: %v", err)
		}
	})

	// Test template select
	t.Run("TemplateSelect", func(t *testing.T) {
		buf := new(bytes.Buffer)
		cmd := newTemplateSelectCommand(cfg, logger)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"basic-led"})

		err := cmd.Execute()
		if err != nil {
			t.Errorf("Template select command failed: %v", err)
		}
	})
}

func testProfileCommands(t *testing.T, cfg *config.Config, logger *logger.Logger) {
	// Test profile show
	t.Run("ProfileShow", func(t *testing.T) {
		buf := new(bytes.Buffer)
		cmd := newProfileShowCommand(cfg, logger)
		cmd.SetOut(buf)

		err := cmd.Execute()
		if err != nil {
			t.Errorf("Profile show command failed: %v", err)
		}

		output := buf.String()
		if output == "" {
			t.Error("Profile show command should produce output")
		}
	})

	// Test profile list
	t.Run("ProfileList", func(t *testing.T) {
		buf := new(bytes.Buffer)
		cmd := newProfileListCommand(cfg, logger)
		cmd.SetOut(buf)

		err := cmd.Execute()
		if err != nil {
			t.Errorf("Profile list command failed: %v", err)
		}

		output := buf.String()
		if output == "" {
			t.Error("Profile list command should produce output")
		}
	})

	// Test profile create
	t.Run("ProfileCreate", func(t *testing.T) {
		buf := new(bytes.Buffer)
		cmd := newProfileCreateCommand(cfg, logger)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"test-profile"})

		err := cmd.Execute()
		if err != nil {
			t.Errorf("Profile create command failed: %v", err)
		}

		output := buf.String()
		if output == "" {
			t.Error("Profile create command should produce output")
		}
	})
}

func testDeviceCommands(t *testing.T, cfg *config.Config, logger *logger.Logger, client *MockServiceClient) {
	// Test device list
	t.Run("DeviceList", func(t *testing.T) {
		buf := new(bytes.Buffer)
		cmd := newDeviceListCommand(cfg, logger)
		cmd.SetOut(buf)

		err := cmd.Execute()
		if err != nil {
			t.Errorf("Device list command failed: %v", err)
		}

		output := buf.String()
		if output == "" {
			t.Error("Device list command should produce output")
		}
	})

	// Test device get
	t.Run("DeviceGet", func(t *testing.T) {
		buf := new(bytes.Buffer)
		cmd := newDeviceGetCommand(cfg, logger)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"device-001"})

		err := cmd.Execute()
		if err != nil {
			t.Errorf("Device get command failed: %v", err)
		}
	})
}

func testTelemetryCommands(t *testing.T, cfg *config.Config, logger *logger.Logger, client *MockServiceClient) {
	// Test telemetry metrics
	t.Run("TelemetryMetrics", func(t *testing.T) {
		buf := new(bytes.Buffer)
		cmd := newTelemetryMetricsCommand(cfg, logger)
		cmd.SetOut(buf)
		cmd.SetArgs([]string{"--device", "device-001"})

		err := cmd.Execute()
		if err != nil {
			t.Errorf("Telemetry metrics command failed: %v", err)
		}

		output := buf.String()
		if output == "" {
			t.Error("Telemetry metrics command should produce output")
		}
	})
}

func testOTACommands(t *testing.T, cfg *config.Config, logger *logger.Logger, client *MockServiceClient) {
	// Test OTA releases
	t.Run("OTAReleases", func(t *testing.T) {
		buf := new(bytes.Buffer)
		cmd := newOTAReleasesCommand(cfg, logger)
		cmd.SetOut(buf)

		err := cmd.Execute()
		if err != nil {
			t.Errorf("OTA releases command failed: %v", err)
		}

		output := buf.String()
		if output == "" {
			t.Error("OTA releases command should produce output")
		}
	})
}

func TestProfileManager(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Test profile manager operations
	pm, err := NewProfileManager()
	if err != nil {
		t.Fatalf("Failed to create profile manager: %v", err)
	}

	// Test getting current profile
	profile, err := pm.GetCurrentProfile()
	if err != nil {
		t.Errorf("Failed to get current profile: %v", err)
	}
	if profile == nil {
		t.Error("Current profile should not be nil")
	}

	// Test creating a new profile
	newProfile := Profile{
		Name:       "test",
		TemplateID: "basic-led",
		Board:      "arduino:avr:uno",
		Port:       "COM3",
	}

	err = pm.CreateProfile("test", newProfile)
	if err != nil {
		t.Errorf("Failed to create profile: %v", err)
	}

	// Test switching to the new profile
	err = pm.SetCurrentProfile("test")
	if err != nil {
		t.Errorf("Failed to switch profile: %v", err)
	}

	// Test updating current profile
	updates := map[string]interface{}{
		"board": "arduino:avr:nano",
		"port":  "COM4",
	}
	err = pm.UpdateCurrentProfile(updates)
	if err != nil {
		t.Errorf("Failed to update profile: %v", err)
	}

	// Verify the update
	profile, err = pm.GetCurrentProfile()
	if err != nil {
		t.Errorf("Failed to get current profile after update: %v", err)
	}
	if profile.Board != "arduino:avr:nano" {
		t.Errorf("Expected board 'arduino:avr:nano', got '%s'", profile.Board)
	}
	if profile.Port != "COM4" {
		t.Errorf("Expected port 'COM4', got '%s'", profile.Port)
	}

	// Test listing profiles
	profiles := pm.ListProfiles()
	if len(profiles) < 2 {
		t.Error("Should have at least 2 profiles (default and test)")
	}

	// Test deleting a profile
	err = pm.DeleteProfile("test")
	if err != nil {
		t.Errorf("Failed to delete profile: %v", err)
	}

	// Verify deletion
	_, err = pm.GetProfile("test")
	if err == nil {
		t.Error("Deleted profile should not exist")
	}
}

func TestErrorHandling(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cfg := &config.Config{
		Services: map[string]string{
			"template-service": "http://invalid:8001",
		},
	}
	logger := logger.New("info", "athena-cli-test")

	// Test error handling in template inspect with invalid ID
	t.Run("TemplateInspectInvalidID", func(t *testing.T) {
		buf := new(bytes.Buffer)
		cmd := newTemplateInspectCommand(cfg, logger)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"nonexistent"})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for invalid template ID")
		}
	})

	// Test error handling in device get with invalid ID
	t.Run("DeviceGetInvalidID", func(t *testing.T) {
		buf := new(bytes.Buffer)
		cmd := newDeviceGetCommand(cfg, logger)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"nonexistent"})

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for invalid device ID")
		}
	})

	// Test error handling in telemetry metrics with missing device flag
	t.Run("TelemetryMetricsMissingDevice", func(t *testing.T) {
		buf := new(bytes.Buffer)
		cmd := newTelemetryMetricsCommand(cfg, logger)
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for missing device flag")
		}
	})
}

func TestCommandArgumentParsing(t *testing.T) {
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()

	cfg := &config.Config{
		Services: map[string]string{
			"template-service":     "http://localhost:8001",
			"provisioning-service": "http://localhost:8003",
			"device-service":       "http://localhost:8004",
		},
	}
	logger := logger.New("info", "athena-cli-test")

	// Test argument parsing for template inspect
	t.Run("TemplateInspectArgs", func(t *testing.T) {
		cmd := newTemplateInspectCommand(cfg, logger)

		// Test with missing argument
		cmd.SetArgs([]string{})
		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for missing template ID")
		}

		// Test with correct argument
		cmd.SetArgs([]string{"basic-led"})
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		// This will fail due to service not being available, but argument parsing should work
		err = cmd.Execute()
		// We expect an error from the service call, not from argument parsing
		if err != nil && !contains(err.Error(), "failed to get template") {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	// Test argument parsing for device get
	t.Run("DeviceGetArgs", func(t *testing.T) {
		cmd := newDeviceGetCommand(cfg, logger)

		// Test with missing argument
		cmd.SetArgs([]string{})
		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for missing device ID")
		}

		// Test with correct argument
		cmd.SetArgs([]string{"device-001"})
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		err = cmd.Execute()
		// We expect an error from the service call, not from argument parsing
		if err != nil && !contains(err.Error(), "failed to get device") {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	// Test flag parsing for telemetry metrics
	t.Run("TelemetryMetricsFlags", func(t *testing.T) {
		cmd := newTelemetryMetricsCommand(cfg, logger)

		// Test with missing device flag
		cmd.SetArgs([]string{})
		err := cmd.Execute()
		if err == nil {
			t.Error("Expected error for missing device flag")
		}

		// Test with device flag
		cmd.SetArgs([]string{"--device", "device-001"})
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		err = cmd.Execute()
		// We expect an error from the service call, not from flag parsing
		if err != nil && !contains(err.Error(), "failed to get telemetry metrics") {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
