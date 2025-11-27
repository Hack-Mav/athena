package nlp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_ParseRequirements(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectError   bool
		validateFunc  func(*testing.T, *ParsedRequirements)
	}{
		{
			name:  "Simple temperature sensor project",
			input: "I want to build a temperature monitor using a DHT22 sensor",
			validateFunc: func(t *testing.T, req *ParsedRequirements) {
				assert.NotEmpty(t, req.Intent)
				assert.Contains(t, req.Intent, "temperature")
				assert.NotEmpty(t, req.RawInput)
			},
		},
		{
			name:  "LED control project",
			input: "Create an Arduino project that blinks an LED on pin 13",
			validateFunc: func(t *testing.T, req *ParsedRequirements) {
				assert.NotEmpty(t, req.Intent)
			},
		},
		{
			name:        "Empty input",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock LLM client
			mockClient := &MockLLMClient{
				response: &LLMResponse{
					Content: `{"intent": "temperature monitoring", "sensors": [{"type": "temperature", "model": "DHT22"}], "actuators": [], "communication": [], "constraints": {}, "board_preference": ""}`,
				},
			}
			parser := NewParser(mockClient)

			ctx := context.Background()
			req, err := parser.ParseRequirements(ctx, tt.input)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, req)

			if tt.validateFunc != nil {
				tt.validateFunc(t, req)
			}
		})
	}
}

func TestParser_ClassifyIntent(t *testing.T) {
	tests := []struct {
		name             string
		requirements     *ParsedRequirements
		expectedCategory string
	}{
		{
			name: "Sensing project",
			requirements: &ParsedRequirements{
				Intent: "Monitor temperature and humidity",
				Sensors: []SensorSpec{
					{Type: "temperature"},
					{Type: "humidity"},
				},
			},
			expectedCategory: "sensing",
		},
		{
			name: "Automation project",
			requirements: &ParsedRequirements{
				Intent: "Control LED based on temperature",
				Sensors: []SensorSpec{
					{Type: "temperature"},
				},
				Actuators: []ActuatorSpec{
					{Type: "led"},
				},
			},
			expectedCategory: "automation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockLLMClient{
				response: &LLMResponse{
					Content: tt.expectedCategory,
				},
			}
			parser := NewParser(mockClient)

			ctx := context.Background()
			category, err := parser.ClassifyIntent(ctx, tt.requirements)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedCategory, category)
		})
	}
}

func TestParser_ExtractTechnicalSpecs(t *testing.T) {
	requirements := &ParsedRequirements{
		Intent: "Temperature monitoring system",
		Sensors: []SensorSpec{
			{Type: "temperature", Model: "DHT22"},
			{Type: "humidity", Model: "DHT22"},
		},
		Actuators: []ActuatorSpec{
			{Type: "led"},
		},
		Communication: []CommSpec{
			{Protocol: "wifi"},
		},
	}

	mockClient := &MockLLMClient{}
	parser := NewParser(mockClient)

	ctx := context.Background()
	specs, err := parser.ExtractTechnicalSpecs(ctx, requirements)

	require.NoError(t, err)
	require.NotNil(t, specs)

	assert.Equal(t, 2, specs["sensor_count"])
	assert.Equal(t, 1, specs["actuator_count"])
	assert.True(t, specs["requires_network"].(bool))
	assert.NotEmpty(t, specs["complexity"])
}

func TestParser_CalculateComplexity(t *testing.T) {
	tests := []struct {
		name         string
		requirements *ParsedRequirements
		expected     string
	}{
		{
			name: "Beginner - simple LED",
			requirements: &ParsedRequirements{
				Actuators: []ActuatorSpec{{Type: "led"}},
			},
			expected: "beginner",
		},
		{
			name: "Intermediate - multiple sensors",
			requirements: &ParsedRequirements{
				Sensors: []SensorSpec{
					{Type: "temperature"},
					{Type: "humidity"},
					{Type: "pressure"},
				},
				Actuators: []ActuatorSpec{
					{Type: "led"},
				},
			},
			expected: "intermediate",
		},
		{
			name: "Advanced - complex system",
			requirements: &ParsedRequirements{
				Sensors: []SensorSpec{
					{Type: "temperature", Threshold: &ThresholdSpec{Value: 25}},
					{Type: "humidity", Threshold: &ThresholdSpec{Value: 60}},
					{Type: "motion"},
				},
				Actuators: []ActuatorSpec{
					{Type: "servo"},
					{Type: "relay"},
				},
				Communication: []CommSpec{
					{Protocol: "wifi"},
					{Protocol: "mqtt"},
				},
			},
			expected: "advanced",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockLLMClient{}
			parser := NewParser(mockClient)

			complexity := parser.calculateComplexity(tt.requirements)
			assert.Equal(t, tt.expected, complexity)
		})
	}
}

func TestParser_EstimatePowerRequirements(t *testing.T) {
	tests := []struct {
		name         string
		requirements *ParsedRequirements
		expected     string
	}{
		{
			name: "Battery - simple sensor",
			requirements: &ParsedRequirements{
				Sensors: []SensorSpec{{Type: "temperature"}},
			},
			expected: "battery",
		},
		{
			name: "USB - WiFi communication",
			requirements: &ParsedRequirements{
				Sensors: []SensorSpec{{Type: "temperature"}},
				Communication: []CommSpec{
					{Protocol: "wifi"},
				},
			},
			expected: "usb",
		},
		{
			name: "External - motor",
			requirements: &ParsedRequirements{
				Actuators: []ActuatorSpec{
					{Type: "motor"},
				},
			},
			expected: "external",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockLLMClient{}
			parser := NewParser(mockClient)

			power := parser.estimatePowerRequirements(tt.requirements)
			assert.Equal(t, tt.expected, power)
		})
	}
}

// MockLLMClient for testing
type MockLLMClient struct {
	response *LLMResponse
	err      error
}

func (m *MockLLMClient) Complete(ctx context.Context, prompt string) (*LLMResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *MockLLMClient) CompleteWithRequest(ctx context.Context, request *LLMRequest) (*LLMResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}
