package nlp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSafetyValidator_ValidateSafety(t *testing.T) {
	tests := []struct {
		name        string
		plan        *ImplementationPlan
		boardType   string
		expectValid bool
		expectError bool
	}{
		{
			name: "Safe configuration",
			plan: &ImplementationPlan{
				WiringDiagram: &WiringDiagram{
					Components: []Component{
						{ID: "board", Type: "arduino", Name: "Arduino Uno"},
						{ID: "led1", Type: "led", Name: "LED"},
					},
					Connections: []Connection{
						{FromComponent: "board", FromPin: "D13", ToComponent: "led1", ToPin: "ANODE"},
						{FromComponent: "board", FromPin: "GND", ToComponent: "led1", ToPin: "CATHODE"},
					},
				},
			},
			boardType:   "uno",
			expectValid: true,
		},
		{
			name: "Pin conflict",
			plan: &ImplementationPlan{
				WiringDiagram: &WiringDiagram{
					Components: []Component{
						{ID: "board", Type: "arduino", Name: "Arduino Uno"},
						{ID: "led1", Type: "led", Name: "LED 1"},
						{ID: "led2", Type: "led", Name: "LED 2"},
					},
					Connections: []Connection{
						{FromComponent: "board", FromPin: "D13", ToComponent: "led1", ToPin: "ANODE"},
						{FromComponent: "board", FromPin: "D13", ToComponent: "led2", ToPin: "ANODE"},
					},
				},
			},
			boardType:   "uno",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewSafetyValidator()
			ctx := context.Background()

			validation, err := validator.ValidateSafety(ctx, tt.plan, tt.boardType)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, validation)

			assert.Equal(t, tt.expectValid, validation.Valid)
			if !tt.expectValid {
				assert.NotEmpty(t, validation.Errors)
			}
		})
	}
}

func TestSafetyValidator_EstimateComponentCurrent(t *testing.T) {
	tests := []struct {
		name            string
		component       *Component
		expectedCurrent float64
	}{
		{
			name:            "LED",
			component:       &Component{Type: "led"},
			expectedCurrent: 20.0,
		},
		{
			name:            "Servo",
			component:       &Component{Type: "servo"},
			expectedCurrent: 500.0,
		},
		{
			name:            "Motor",
			component:       &Component{Type: "motor"},
			expectedCurrent: 1000.0,
		},
		{
			name:            "Sensor",
			component:       &Component{Type: "sensor"},
			expectedCurrent: 5.0,
		},
		{
			name:            "Unknown",
			component:       &Component{Type: "unknown"},
			expectedCurrent: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewSafetyValidator()
			current := validator.estimateComponentCurrent(tt.component)
			assert.Equal(t, tt.expectedCurrent, current)
		})
	}
}

func TestSafetyValidator_GetComponentVoltageRequirement(t *testing.T) {
	tests := []struct {
		name     string
		component Component
		expected string
	}{
		{
			name: "Explicit 5V",
			component: Component{
				Type: "sensor",
				Metadata: map[string]interface{}{
					"voltage": "5V",
				},
			},
			expected: "5V",
		},
		{
			name: "3.3V from type",
			component: Component{
				Type: "sensor_3.3v",
			},
			expected: "3.3V",
		},
		{
			name: "LED default",
			component: Component{
				Type: "led",
			},
			expected: "3.3V",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewSafetyValidator()
			voltage := validator.getComponentVoltageRequirement(tt.component)
			assert.Equal(t, tt.expected, voltage)
		})
	}
}

func TestSafetyValidator_IsPinCompatible(t *testing.T) {
	validator := NewSafetyValidator()
	boardSpec := validator.boardSpecs["uno"]

	tests := []struct {
		name         string
		pin          string
		requiredType string
		expected     bool
	}{
		{
			name:         "Digital pin for digital",
			pin:          "D2",
			requiredType: "digital",
			expected:     true,
		},
		{
			name:         "PWM pin for PWM",
			pin:          "D3",
			requiredType: "pwm",
			expected:     true,
		},
		{
			name:         "Analog pin for analog",
			pin:          "A0",
			requiredType: "analog",
			expected:     true,
		},
		{
			name:         "Digital pin for PWM",
			pin:          "D2",
			requiredType: "pwm",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compatible := validator.isPinCompatible(tt.pin, tt.requiredType, boardSpec)
			assert.Equal(t, tt.expected, compatible)
		})
	}
}

func TestSafetyValidator_BoardSpecifications(t *testing.T) {
	validator := NewSafetyValidator()

	tests := []struct {
		boardType string
		exists    bool
	}{
		{"uno", true},
		{"nano", true},
		{"esp32", true},
		{"esp8266", true},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.boardType, func(t *testing.T) {
			_, exists := validator.boardSpecs[tt.boardType]
			assert.Equal(t, tt.exists, exists)
		})
	}
}

func TestSafetyValidator_ValidateVoltageCompatibility(t *testing.T) {
	validator := NewSafetyValidator()
	boardSpec := validator.boardSpecs["uno"]

	plan := &ImplementationPlan{
		WiringDiagram: &WiringDiagram{
			Components: []Component{
				{ID: "board", Type: "arduino", Name: "Arduino Uno"},
				{
					ID:   "sensor1",
					Type: "sensor",
					Name: "5V Sensor",
					Metadata: map[string]interface{}{
						"voltage": "5V",
					},
				},
			},
		},
	}

	validation := &SafetyValidation{
		Valid:         true,
		VoltageChecks: []VoltageCheck{},
	}

	validator.validateVoltageCompatibility(plan, boardSpec, validation)

	assert.NotEmpty(t, validation.VoltageChecks)
	assert.True(t, validation.Valid)
}

func TestSafetyValidator_ValidatePinConflicts(t *testing.T) {
	validator := NewSafetyValidator()

	plan := &ImplementationPlan{
		WiringDiagram: &WiringDiagram{
			Components: []Component{
				{ID: "board", Type: "arduino"},
				{ID: "led1", Type: "led", Name: "LED 1"},
				{ID: "led2", Type: "led", Name: "LED 2"},
			},
			Connections: []Connection{
				{FromComponent: "board", FromPin: "D13", ToComponent: "led1"},
				{FromComponent: "board", FromPin: "D13", ToComponent: "led2"},
			},
		},
	}

	validation := &SafetyValidation{
		Valid:  true,
		Errors: []string{},
	}

	validator.validatePinConflicts(plan, validation)

	assert.False(t, validation.Valid)
	assert.NotEmpty(t, validation.Errors)
	assert.Contains(t, validation.Errors[0], "D13")
}

func TestSafetyValidator_GetRequiredPinType(t *testing.T) {
	validator := NewSafetyValidator()

	tests := []struct {
		component *Component
		expected  string
	}{
		{&Component{Type: "servo"}, "pwm"},
		{&Component{Type: "analog_sensor"}, "analog"},
		{&Component{Type: "led"}, "digital"},
		{&Component{Type: "relay"}, "digital"},
		{&Component{Type: "unknown"}, "digital"},
	}

	for _, tt := range tests {
		result := validator.getRequiredPinType(tt.component)
		assert.Equal(t, tt.expected, result, "component type: %s", tt.component.Type)
	}
}
