package template

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestService_ValidateTemplate(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()

	t.Run("Valid template", func(t *testing.T) {
		template := createTestTemplate()

		result, err := service.ValidateTemplate(ctx, template)

		assert.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
	})

	t.Run("Invalid template - missing required fields", func(t *testing.T) {
		template := &Template{
			ID:   "",
			Name: "Test Template",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sensorPin": map[string]interface{}{
						"type": "integer",
					},
				},
			},
		}

		result, err := service.ValidateTemplate(ctx, template)

		assert.NoError(t, err)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
	})

	t.Run("Invalid template - malformed schema", func(t *testing.T) {
		template := createTestTemplate()
		template.Schema = map[string]interface{}{
			"type": "invalid_type",
		}

		result, err := service.ValidateTemplate(ctx, template)

		// The validation may return an error or a result with errors, both are acceptable
		if err != nil {
			assert.Error(t, err)
		} else {
			assert.False(t, result.Valid)
			assert.NotEmpty(t, result.Errors)
		}
	})

	t.Run("Template with warnings", func(t *testing.T) {
		template := createTestTemplate()
		template.Libraries = []LibraryDependency{
			{Name: "", Version: "1.0.0"}, // Empty library name should generate warning
		}

		result, err := service.ValidateTemplate(ctx, template)

		assert.NoError(t, err)
		// Template should still be valid but with warnings
		if result.Valid {
			assert.NotEmpty(t, result.Warnings)
		}
	})
}

func TestService_ValidateParameters(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()

	template := createTestTemplate()

	t.Run("Valid parameters", func(t *testing.T) {
		parameters := map[string]interface{}{
			"sensorPin": 2,
			"interval":  1000,
		}

		result, err := service.ValidateParameters(ctx, template, parameters)

		assert.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
	})

	t.Run("Missing required parameter", func(t *testing.T) {
		parameters := map[string]interface{}{
			"interval": 1000,
			// Missing sensorPin
		}

		result, err := service.ValidateParameters(ctx, template, parameters)

		assert.NoError(t, err)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
		assert.Contains(t, result.Errors[0], "sensorPin")
	})

	t.Run("Invalid parameter type", func(t *testing.T) {
		parameters := map[string]interface{}{
			"sensorPin": "not_a_number", // Should be integer
			"interval":  1000,
		}

		result, err := service.ValidateParameters(ctx, template, parameters)

		assert.NoError(t, err)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
	})

	t.Run("Parameter out of range", func(t *testing.T) {
		parameters := map[string]interface{}{
			"sensorPin": 20, // Exceeds maximum of 13
			"interval":  1000,
		}

		result, err := service.ValidateParameters(ctx, template, parameters)

		assert.NoError(t, err)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
	})

	t.Run("Default value applied", func(t *testing.T) {
		parameters := map[string]interface{}{
			"sensorPin": 2,
			// interval should get default value
		}

		result, err := service.ValidateParameters(ctx, template, parameters)

		assert.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
	})
}

func TestService_ValidateBoardCapabilities(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()

	template := createTestTemplate()

	t.Run("Supported board", func(t *testing.T) {
		parameters := map[string]interface{}{
			"sensorPin": 2,
			"interval":  1000,
		}

		result, err := service.ValidateBoardCapabilities(ctx, template, "arduino-uno", parameters)

		assert.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
	})

	t.Run("Unsupported board", func(t *testing.T) {
		parameters := map[string]interface{}{
			"sensorPin": 2,
			"interval":  1000,
		}

		result, err := service.ValidateBoardCapabilities(ctx, template, "esp8266", parameters)

		assert.NoError(t, err)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
		assert.Contains(t, result.Errors[0], "esp8266")
	})

	t.Run("Pin compatibility check", func(t *testing.T) {
		parameters := map[string]interface{}{
			"sensorPin": 2, // Valid pin for Arduino Uno
			"interval":  1000,
		}

		result, err := service.ValidateBoardCapabilities(ctx, template, "arduino-uno", parameters)

		assert.NoError(t, err)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
	})
}

func TestService_RenderTemplate(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()

	template := createTestTemplate()
	template.Assets = []Asset{
		{
			Type: "code",
			Path: "/templates/main.ino",
			Metadata: map[string]interface{}{
				"content": `// DHT Sensor Template
// Sensor Pin: {{.sensorPin}}
// Reading Interval: {{.interval}}ms

#include <DHT.h>

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
	}

	parameters := map[string]interface{}{
		"sensorPin": 2,
		"interval":  1000,
	}

	templateID := "test-template-1"
	version := "1.0.0"

	mockRepo.On("GetTemplate", ctx, templateID, version).Return(template, nil)

	result, err := service.RenderTemplate(ctx, templateID, version, parameters)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, template, result.Template)
	assert.Equal(t, parameters, result.Parameters)
	assert.NotEmpty(t, result.RenderedCode)
	assert.Contains(t, result.RenderedCode, "#define DHTPIN 2")
	assert.Contains(t, result.RenderedCode, "delay(1000);")
	assert.Equal(t, template.Assets, result.Assets)

	mockRepo.AssertExpectations(t)
}

func TestService_RenderTemplate_InvalidParameters(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()

	template := createTestTemplate()
	template.Assets = []Asset{
		{
			Type: "code",
			Path: "/templates/main.ino",
			Metadata: map[string]interface{}{
				"content": `// Template with invalid parameter reference
{{.nonExistentParam}}`,
			},
		},
	}

	parameters := map[string]interface{}{
		"sensorPin": 2,
		"interval":  1000,
	}

	templateID := "test-template-1"
	version := "1.0.0"

	mockRepo.On("GetTemplate", ctx, templateID, version).Return(template, nil)

	result, err := service.RenderTemplate(ctx, templateID, version, parameters)

	// Go templates don't error on missing parameters, they render empty strings
	// So we expect the rendering to succeed but with empty content for the missing param
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.RenderedCode, "<no value>")
	assert.Equal(t, template.Assets, result.Assets)

	mockRepo.AssertExpectations(t)
}

func TestService_GenerateWiringDiagram(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()

	template := createTestTemplate()
	template.Assets = []Asset{
		{
			Type: "wiring_spec",
			Path: "/wiring/spec.json",
			Metadata: map[string]interface{}{
				"components": []map[string]interface{}{
					{
						"id":   "arduino",
						"type": "board",
						"name": "Arduino Uno",
						"pins": []map[string]interface{}{
							{"number": "2", "name": "D2", "type": "digital"},
							{"number": "5V", "name": "5V", "type": "power"},
							{"number": "GND", "name": "GND", "type": "ground"},
						},
					},
					{
						"id":   "dht22",
						"type": "sensor",
						"name": "DHT22 Sensor",
						"pins": []map[string]interface{}{
							{"number": "1", "name": "VCC", "type": "power"},
							{"number": "2", "name": "DATA", "type": "digital"},
							{"number": "4", "name": "GND", "type": "ground"},
						},
					},
				},
				"connections": []map[string]interface{}{
					{
						"from_component": "arduino",
						"from_pin":       "2",
						"to_component":   "dht22",
						"to_pin":         "2",
						"wire_color":     "red",
					},
					{
						"from_component": "arduino",
						"from_pin":       "5V",
						"to_component":   "dht22",
						"to_pin":         "1",
						"wire_color":     "black",
					},
					{
						"from_component": "arduino",
						"from_pin":       "GND",
						"to_component":   "dht22",
						"to_pin":         "4",
						"wire_color":     "blue",
					},
				},
			},
		},
	}

	parameters := map[string]interface{}{
		"sensorPin": 2,
		"interval":  1000,
	}

	result, err := service.GenerateWiringDiagram(ctx, template, parameters)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.MermaidSyntax)
	assert.NotEmpty(t, result.Components)
	assert.NotEmpty(t, result.Connections)

	// Check Mermaid syntax
	assert.Contains(t, result.MermaidSyntax, "graph TD")
	assert.Contains(t, result.MermaidSyntax, "arduino")
	assert.Contains(t, result.MermaidSyntax, "dht22")

	// Check components
	assert.Len(t, result.Components, 2)

	arduinoComponent := result.Components[0]
	assert.Equal(t, "arduino", arduinoComponent.ID)
	assert.Equal(t, "board", arduinoComponent.Type)
	assert.NotEmpty(t, arduinoComponent.Pins)

	// Check connections
	assert.Len(t, result.Connections, 3)

	connection := result.Connections[0]
	assert.Equal(t, "arduino", connection.FromComponent)
	assert.Equal(t, "2", connection.FromPin)
	assert.Equal(t, "dht22", connection.ToComponent)
	assert.Equal(t, "2", connection.ToPin)
	assert.Equal(t, "red", connection.WireColor)
}

func TestService_GenerateWiringDiagram_NoWiringSpec(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()

	template := createTestTemplate()
	// No wiring specification assets

	parameters := map[string]interface{}{
		"sensorPin": 2,
		"interval":  1000,
	}

	result, err := service.GenerateWiringDiagram(ctx, template, parameters)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "wiring specification")
}
