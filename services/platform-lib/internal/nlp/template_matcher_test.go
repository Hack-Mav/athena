package nlp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateMatcher_MatchTemplates(t *testing.T) {
	requirements := &ParsedRequirements{
		Intent: "Temperature monitoring with LED indicator",
		Sensors: []SensorSpec{
			{Type: "temperature", Model: "DHT22"},
		},
		Actuators: []ActuatorSpec{
			{Type: "led"},
		},
	}

	templates := []TemplateInfo{
		{
			ID:              "temp-monitor",
			Name:            "Temperature Monitor",
			Category:        "sensing",
			Description:     "Monitor temperature using DHT22 sensor",
			BoardsSupported: []string{"uno", "nano"},
			RequiredSensors: []string{"temperature", "DHT22"},
		},
		{
			ID:              "led-blink",
			Name:            "LED Blink",
			Category:        "automation",
			Description:     "Simple LED blinking",
			BoardsSupported: []string{"uno"},
		},
		{
			ID:              "motor-control",
			Name:            "Motor Control",
			Category:        "automation",
			Description:     "Control DC motor",
			BoardsSupported: []string{"uno"},
		},
	}

	mockClient := &MockLLMClient{}
	matcher := NewTemplateMatcher(mockClient)

	ctx := context.Background()
	scores, err := matcher.MatchTemplates(ctx, requirements, templates)

	require.NoError(t, err)
	require.NotEmpty(t, scores)

	// Temperature monitor should score highest
	assert.Equal(t, "temp-monitor", scores[0].TemplateID)
	assert.Greater(t, scores[0].Score, 0.0)
}

func TestTemplateMatcher_SelectBestTemplate(t *testing.T) {
	requirements := &ParsedRequirements{
		Intent: "WiFi temperature sensor",
		Sensors: []SensorSpec{
			{Type: "temperature"},
		},
		Communication: []CommSpec{
			{Protocol: "wifi"},
		},
		BoardPreference: "esp32",
	}

	templates := []TemplateInfo{
		{
			ID:              "wifi-temp",
			Name:            "WiFi Temperature Sensor",
			Category:        "communication",
			Description:     "Temperature sensor with WiFi connectivity",
			BoardsSupported: []string{"esp32", "esp8266"},
			Libraries:       []string{"WiFi", "DHT"},
		},
		{
			ID:              "basic-temp",
			Name:            "Basic Temperature Sensor",
			Category:        "sensing",
			Description:     "Simple temperature sensor",
			BoardsSupported: []string{"uno"},
		},
	}

	mockClient := &MockLLMClient{}
	matcher := NewTemplateMatcher(mockClient)

	ctx := context.Background()
	best, err := matcher.SelectBestTemplate(ctx, requirements, templates)

	require.NoError(t, err)
	require.NotNil(t, best)

	// WiFi template should be selected
	assert.Equal(t, "wifi-temp", best.TemplateID)
}

func TestTemplateMatcher_CalculateTemplateScore(t *testing.T) {
	tests := []struct {
		name         string
		requirements *ParsedRequirements
		template     *TemplateInfo
		minScore     float64
	}{
		{
			name: "Perfect match",
			requirements: &ParsedRequirements{
				Intent: "Temperature monitoring",
				Sensors: []SensorSpec{
					{Type: "temperature", Model: "DHT22"},
				},
			},
			template: &TemplateInfo{
				ID:              "temp-monitor",
				Name:            "Temperature Monitor",
				Category:        "sensing",
				Description:     "Monitor temperature using DHT22 sensor",
				RequiredSensors: []string{"temperature", "DHT22"},
			},
			minScore: 40.0,
		},
		{
			name: "Partial match",
			requirements: &ParsedRequirements{
				Intent: "LED control",
				Actuators: []ActuatorSpec{
					{Type: "led"},
				},
			},
			template: &TemplateInfo{
				ID:          "automation",
				Name:        "Automation Template",
				Category:    "automation",
				Description: "General automation with LED and relay",
			},
			minScore: 15.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockLLMClient{}
			matcher := NewTemplateMatcher(mockClient)

			score := matcher.calculateTemplateScore(tt.requirements, tt.template)

			assert.GreaterOrEqual(t, score.Score, tt.minScore)
			assert.NotEmpty(t, score.Reasons)
		})
	}
}

func TestTemplateMatcher_MatchesCategory(t *testing.T) {
	tests := []struct {
		name         string
		requirements *ParsedRequirements
		template     *TemplateInfo
		expected     bool
	}{
		{
			name: "Direct category match",
			requirements: &ParsedRequirements{
				Intent: "Build a sensing project",
			},
			template: &TemplateInfo{
				Category: "sensing",
			},
			expected: true,
		},
		{
			name: "Sensor-based match",
			requirements: &ParsedRequirements{
				Intent: "Monitor environment",
				Sensors: []SensorSpec{
					{Type: "temperature"},
				},
			},
			template: &TemplateInfo{
				Category: "sensing",
			},
			expected: true,
		},
		{
			name: "No match",
			requirements: &ParsedRequirements{
				Intent: "Display data",
			},
			template: &TemplateInfo{
				Category: "sensing",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockLLMClient{}
			matcher := NewTemplateMatcher(mockClient)

			matches := matcher.matchesCategory(tt.requirements, tt.template)
			assert.Equal(t, tt.expected, matches)
		})
	}
}

func TestTemplateMatcher_TemplateSupportsSensor(t *testing.T) {
	template := &TemplateInfo{
		Name:            "DHT22 Temperature Monitor",
		Description:     "Monitor temperature and humidity using DHT22 sensor",
		RequiredSensors: []string{"DHT22", "temperature", "humidity"},
	}

	mockClient := &MockLLMClient{}
	matcher := NewTemplateMatcher(mockClient)

	tests := []struct {
		sensorType  string
		sensorModel string
		expected    bool
	}{
		{"temperature", "", true},
		{"temperature", "DHT22", true},
		{"humidity", "", true},
		{"pressure", "", false},
		{"", "DHT22", true},
		{"motion", "PIR", false},
	}

	for _, tt := range tests {
		result := matcher.templateSupportsSensor(template, tt.sensorType, tt.sensorModel)
		assert.Equal(t, tt.expected, result, "sensorType=%s, sensorModel=%s", tt.sensorType, tt.sensorModel)
	}
}

func TestTemplateMatcher_TemplateSupportsCommunication(t *testing.T) {
	template := &TemplateInfo{
		Name:        "WiFi Sensor",
		Description: "Sensor with WiFi connectivity and MQTT support",
		Libraries:   []string{"WiFi", "PubSubClient"},
	}

	mockClient := &MockLLMClient{}
	matcher := NewTemplateMatcher(mockClient)

	tests := []struct {
		protocol string
		expected bool
	}{
		{"wifi", true},
		{"mqtt", true},
		{"bluetooth", false},
		{"http", false},
	}

	for _, tt := range tests {
		result := matcher.templateSupportsCommunication(template, tt.protocol)
		assert.Equal(t, tt.expected, result, "protocol=%s", tt.protocol)
	}
}
