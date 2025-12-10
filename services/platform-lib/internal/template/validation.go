package template

import (
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

// JSONSchemaValidator provides JSON Schema validation functionality
type JSONSchemaValidator struct{}

// NewJSONSchemaValidator creates a new JSON Schema validator
func NewJSONSchemaValidator() *JSONSchemaValidator {
	return &JSONSchemaValidator{}
}

// ValidateParameters validates template parameters against a JSON Schema
func (v *JSONSchemaValidator) ValidateParameters(schema map[string]interface{}, parameters map[string]interface{}) (*ValidationResult, error) {
	if schema == nil {
		return &ValidationResult{
			Valid:    true,
			Errors:   []string{},
			Warnings: []string{},
		}, nil
	}

	// Convert schema to gojsonschema format
	schemaLoader := gojsonschema.NewGoLoader(schema)
	documentLoader := gojsonschema.NewGoLoader(parameters)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	validationResult := &ValidationResult{
		Valid:    result.Valid(),
		Errors:   []string{},
		Warnings: []string{},
	}

	// Collect validation errors
	for _, desc := range result.Errors() {
		validationResult.Errors = append(validationResult.Errors, desc.String())
	}

	return validationResult, nil
}

// ValidateTemplate validates a complete template structure
func (v *JSONSchemaValidator) ValidateTemplate(template *Template) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Validate required fields
	if template.ID == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "template ID is required")
	}

	if template.Name == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "template name is required")
	}

	if template.Version == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "template version is required")
	}

	if template.Category == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "template category is required")
	}

	if len(template.BoardsSupported) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "at least one supported board is required")
	}

	// Validate board names
	validBoards := map[string]bool{
		"arduino:avr:uno":           true,
		"arduino:avr:nano":          true,
		"arduino:avr:mega":          true,
		"arduino:avr:leonardo":      true,
		"esp32:esp32:esp32":         true,
		"esp8266:esp8266:nodemcuv2": true,
	}

	for _, board := range template.BoardsSupported {
		if !validBoards[board] {
			result.Warnings = append(result.Warnings, fmt.Sprintf("board '%s' may not be supported", board))
		}
	}

	// Validate library dependencies
	for _, lib := range template.Libraries {
		if lib.Name == "" {
			result.Valid = false
			result.Errors = append(result.Errors, "library name is required")
		}
		if lib.Version == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("library '%s' has no version specified", lib.Name))
		}
	}

	// Validate assets
	for i, asset := range template.Assets {
		if asset.Type == "" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("asset %d: type is required", i))
		}
		if asset.Path == "" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("asset %d: path is required", i))
		}

		// Validate asset types
		validAssetTypes := map[string]bool{
			"wiring_diagram": true,
			"documentation":  true,
			"image":          true,
			"code":           true,
		}
		if !validAssetTypes[asset.Type] {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("asset %d: invalid type '%s'", i, asset.Type))
		}
	}

	// Validate parameters against schema if both exist
	if template.Schema != nil && template.Parameters != nil {
		paramResult, err := v.ValidateParameters(template.Schema, template.Parameters)
		if err != nil {
			return nil, err
		}

		if !paramResult.Valid {
			result.Valid = false
			result.Errors = append(result.Errors, paramResult.Errors...)
		}
		result.Warnings = append(result.Warnings, paramResult.Warnings...)
	}

	return result, nil
}

// ValidateBoardCapabilities validates that template parameters are compatible with board capabilities
func (v *JSONSchemaValidator) ValidateBoardCapabilities(template *Template, boardType string, parameters map[string]interface{}) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Check if board is supported by template
	boardSupported := false
	for _, supportedBoard := range template.BoardsSupported {
		if supportedBoard == boardType {
			boardSupported = true
			break
		}
	}

	if !boardSupported {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("board '%s' is not supported by this template", boardType))
		return result, nil
	}

	// Get board capabilities
	boardCaps := getBoardCapabilities(boardType)
	if boardCaps == nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("unknown board capabilities for '%s'", boardType))
		return result, nil
	}

	// Validate pin assignments
	usedPins := make(map[string]bool)
	for paramName, paramValue := range parameters {
		if strings.Contains(strings.ToLower(paramName), "pin") {
			if pinStr, ok := paramValue.(string); ok {
				// Check if pin exists on board
				if !boardCaps.HasPin(pinStr) {
					result.Valid = false
					result.Errors = append(result.Errors, fmt.Sprintf("pin '%s' does not exist on board '%s'", pinStr, boardType))
					continue
				}

				// Check for pin conflicts
				if usedPins[pinStr] {
					result.Valid = false
					result.Errors = append(result.Errors, fmt.Sprintf("pin '%s' is used multiple times", pinStr))
					continue
				}
				usedPins[pinStr] = true

				// Check pin capabilities
				if strings.Contains(strings.ToLower(paramName), "analog") && !boardCaps.IsAnalogPin(pinStr) {
					result.Valid = false
					result.Errors = append(result.Errors, fmt.Sprintf("pin '%s' does not support analog operations", pinStr))
				}

				if strings.Contains(strings.ToLower(paramName), "pwm") && !boardCaps.IsPWMPin(pinStr) {
					result.Warnings = append(result.Warnings, fmt.Sprintf("pin '%s' may not support PWM operations", pinStr))
				}
			}
		}
	}

	return result, nil
}

// BoardCapabilities represents the capabilities of an Arduino board
type BoardCapabilities struct {
	Name        string   `json:"name"`
	DigitalPins []string `json:"digital_pins"`
	AnalogPins  []string `json:"analog_pins"`
	PWMPins     []string `json:"pwm_pins"`
	I2CPins     []string `json:"i2c_pins"`
	SPIPins     []string `json:"spi_pins"`
	Voltage     string   `json:"voltage"`
	MaxCurrent  int      `json:"max_current_ma"`
}

// HasPin checks if the board has the specified pin
func (bc *BoardCapabilities) HasPin(pin string) bool {
	for _, p := range bc.DigitalPins {
		if p == pin {
			return true
		}
	}
	for _, p := range bc.AnalogPins {
		if p == pin {
			return true
		}
	}
	return false
}

// IsAnalogPin checks if the specified pin supports analog operations
func (bc *BoardCapabilities) IsAnalogPin(pin string) bool {
	for _, p := range bc.AnalogPins {
		if p == pin {
			return true
		}
	}
	return false
}

// IsPWMPin checks if the specified pin supports PWM operations
func (bc *BoardCapabilities) IsPWMPin(pin string) bool {
	for _, p := range bc.PWMPins {
		if p == pin {
			return true
		}
	}
	return false
}

// getBoardCapabilities returns the capabilities for a specific board type
func getBoardCapabilities(boardType string) *BoardCapabilities {
	capabilities := map[string]*BoardCapabilities{
		"arduino:avr:uno": {
			Name:        "Arduino Uno",
			DigitalPins: []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13"},
			AnalogPins:  []string{"A0", "A1", "A2", "A3", "A4", "A5"},
			PWMPins:     []string{"3", "5", "6", "9", "10", "11"},
			I2CPins:     []string{"A4", "A5"},
			SPIPins:     []string{"10", "11", "12", "13"},
			Voltage:     "5V",
			MaxCurrent:  500,
		},
		"arduino:avr:nano": {
			Name:        "Arduino Nano",
			DigitalPins: []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12", "13"},
			AnalogPins:  []string{"A0", "A1", "A2", "A3", "A4", "A5", "A6", "A7"},
			PWMPins:     []string{"3", "5", "6", "9", "10", "11"},
			I2CPins:     []string{"A4", "A5"},
			SPIPins:     []string{"10", "11", "12", "13"},
			Voltage:     "5V",
			MaxCurrent:  500,
		},
		"esp32:esp32:esp32": {
			Name:        "ESP32",
			DigitalPins: []string{"0", "1", "2", "3", "4", "5", "12", "13", "14", "15", "16", "17", "18", "19", "21", "22", "23", "25", "26", "27", "32", "33"},
			AnalogPins:  []string{"32", "33", "34", "35", "36", "39"},
			PWMPins:     []string{"0", "1", "2", "3", "4", "5", "12", "13", "14", "15", "16", "17", "18", "19", "21", "22", "23", "25", "26", "27"},
			I2CPins:     []string{"21", "22"},
			SPIPins:     []string{"18", "19", "23", "5"},
			Voltage:     "3.3V",
			MaxCurrent:  1200,
		},
	}

	return capabilities[boardType]
}
