package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Template represents the structure of an Arduino template
type Template struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Version        string                 `json:"version"`
	Category       string                 `json:"category"`
	Description    string                 `json:"description"`
	Author         string                 `json:"author"`
	BoardsSupported []string              `json:"boards_supported"`
	Libraries      []Library              `json:"libraries"`
	Schema         map[string]interface{} `json:"schema"`
	Parameters     map[string]interface{} `json:"parameters"`
	Assets         []Asset                `json:"assets"`
	WiringSpec     WiringSpec             `json:"wiring_spec"`
}

type Library struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Asset struct {
	Type     string                 `json:"type"`
	Path     string                 `json:"path"`
	Metadata map[string]interface{} `json:"metadata"`
}

type WiringSpec struct {
	Components  []Component `json:"components"`
	Connections []Connection `json:"connections"`
}

type Component struct {
	ID   string `json:"id"`
	Type string `json:"name"`
	Name string `json:"name"`
	Pins []Pin  `json:"pins"`
}

type Pin struct {
	Number string `json:"number"`
	Name   string `json:"name"`
	Type   string `json:"type"`
}

type Connection struct {
	FromComponent string `json:"from_component"`
	FromPin       string `json:"from_pin"`
	ToComponent   string `json:"to_component"`
	ToPin         string `json:"to_pin"`
	WireColor     string `json:"wire_color"`
}

// Supported Arduino boards for testing
var supportedBoards = []string{
	"arduino:avr:uno",
	"arduino:avr:nano",
	"esp32:esp32:devkitv1",
	"esp8266:esp8266:d1mini",
}

// TemplateTestSuite holds test configuration
type TemplateTestSuite struct {
	TemplateDir string
	Templates   []Template
}

// NewTemplateTestSuite creates a new test suite
func NewTemplateTestSuite(templateDir string) (*TemplateTestSuite, error) {
	suite := &TemplateTestSuite{TemplateDir: templateDir}
	
	err := suite.loadTemplates()
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}
	
	return suite, nil
}

// loadTemplates loads all template JSON files
func (suite *TemplateTestSuite) loadTemplates() error {
	templateFiles, err := filepath.Glob(filepath.Join(suite.TemplateDir, "**", "*.json"))
	if err != nil {
		return err
	}
	
	for _, file := range templateFiles {
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read template file %s: %w", file, err)
		}
		
		var template Template
		if err := json.Unmarshal(data, &template); err != nil {
			return fmt.Errorf("failed to parse template file %s: %w", file, err)
		}
		
		suite.Templates = append(suite.Templates, template)
	}
	
	return nil
}

// TestTemplateCompilation tests template compilation across supported boards
func TestTemplateCompilation(t *testing.T) {
	suite, err := NewTemplateTestSuite("../../templates/arduino")
	require.NoError(t, err, "Failed to create template test suite")
	
	for _, template := range suite.Templates {
		t.Run(fmt.Sprintf("Compilation_%s", template.ID), func(t *testing.T) {
			t.Parallel()
			
			// Test each supported board
			for _, board := range supportedBoards {
				if isBoardSupported(template, board) {
					t.Run(fmt.Sprintf("Board_%s", board), func(t *testing.T) {
						err := simulateCompilation(template, board)
						assert.NoError(t, err, 
							"Template %s should compile successfully on %s", 
							template.ID, board)
					})
				}
			}
		})
	}
}

// TestParameterSchemas validates parameter schemas and default values
func TestParameterSchemas(t *testing.T) {
	suite, err := NewTemplateTestSuite("../../templates/arduino")
	require.NoError(t, err, "Failed to create template test suite")
	
	for _, template := range suite.Templates {
		t.Run(fmt.Sprintf("Schema_%s", template.ID), func(t *testing.T) {
			t.Parallel()
			
			// Validate schema structure
			err := validateSchema(template.Schema)
			assert.NoError(t, err, "Template %s should have valid schema", template.ID)
			
			// Validate parameters against schema
			err = validateParametersAgainstSchema(template.Parameters, template.Schema)
			assert.NoError(t, err, 
				"Template %s parameters should match schema", template.ID)
			
			// Validate required fields
			err = validateRequiredFields(template)
			assert.NoError(t, err, 
				"Template %s should have all required fields", template.ID)
			
			// Test parameter validation
			err = testParameterValidation(template)
			assert.NoError(t, err, 
				"Template %s parameter validation should work", template.ID)
		})
	}
}

// TestWiringDiagrams tests wiring diagram generation and component compatibility
func TestWiringDiagrams(t *testing.T) {
	suite, err := NewTemplateTestSuite("../../templates/arduino")
	require.NoError(t, err, "Failed to create template test suite")
	
	for _, template := range suite.Templates {
		t.Run(fmt.Sprintf("Wiring_%s", template.ID), func(t *testing.T) {
			t.Parallel()
			
			// Validate wiring spec structure
			err := validateWiringSpec(template.WiringSpec)
			assert.NoError(t, err, 
				"Template %s should have valid wiring specification", template.ID)
			
			// Test component compatibility
			err = validateComponentCompatibility(template.WiringSpec)
			assert.NoError(t, err, 
				"Template %s components should be compatible", template.ID)
			
			// Test connection validity
			err = validateConnections(template.WiringSpec)
			assert.NoError(t, err, 
				"Template %s connections should be valid", template.ID)
			
			// Test wiring diagram generation
			err = testWiringDiagramGeneration(template)
			assert.NoError(t, err, 
				"Template %s wiring diagram generation should work", template.ID)
		})
	}
}

// Helper functions

func isBoardSupported(template Template, board string) bool {
	for _, supportedBoard := range template.BoardsSupported {
		if supportedBoard == board {
			return true
		}
	}
	return false
}

func simulateCompilation(template Template, board string) error {
	// Simulate Arduino compilation checks
	
	// Check library compatibility
	for _, lib := range template.Libraries {
		if !isLibraryCompatible(lib, board) {
			return fmt.Errorf("library %s v%s is not compatible with %s", 
				lib.Name, lib.Version, board)
		}
	}
	
	// Check code template syntax
	codeAsset := findCodeAsset(template.Assets)
	if codeAsset != nil {
		if err := validateArduinoCode(codeAsset.Metadata["content"].(string)); err != nil {
			return fmt.Errorf("code validation failed: %w", err)
		}
	}
	
	// Check pin assignments for board compatibility
	if err := validatePinAssignments(template, board); err != nil {
		return fmt.Errorf("pin assignment validation failed: %w", err)
	}
	
	return nil
}

func isLibraryCompatible(lib Library, board string) bool {
	// Simplified library compatibility check
	// In a real implementation, this would check against a library database
	compatibleLibs := map[string][]string{
		"DHT sensor library":      {"arduino:avr:uno", "arduino:avr:nano", "esp32:esp32:devkitv1"},
		"Adafruit Unified Sensor": {"arduino:avr:uno", "arduino:avr:nano", "esp32:esp32:devkitv1"},
		"PubSubClient":            {"arduino:avr:uno", "arduino:avr:nano", "esp32:esp32:devkitv1", "esp8266:esp8266:d1mini"},
		"ArduinoJson":             {"arduino:avr:uno", "arduino:avr:nano", "esp32:esp32:devkitv1", "esp8266:esp8266:d1mini"},
		"WiFi":                    {"esp32:esp32:devkitv1", "esp8266:esp8266:d1mini"},
		"WebServer":               {"esp32:esp32:devkitv1", "esp8266:esp8266:d1mini"},
		"EEPROM":                  {"arduino:avr:uno", "arduino:avr:nano", "esp32:esp32:devkitv1", "esp8266:esp8266:d1mini"},
	}
	
	if boards, exists := compatibleLibs[lib.Name]; exists {
		for _, compatibleBoard := range boards {
			if compatibleBoard == board {
				return true
			}
		}
	}
	return false
}

func findCodeAsset(assets []Asset) *Asset {
	for _, asset := range assets {
		if asset.Type == "code" {
			return &asset
		}
	}
	return nil
}

func validateArduinoCode(code string) error {
	// Basic Arduino code validation
	// In a real implementation, this would use the Arduino CLI for actual compilation
	
	// Check for required Arduino functions
	requiredFunctions := []string{"void setup()", "void loop()"}
	for _, funcName := range requiredFunctions {
		if !contains(code, funcName) {
			return fmt.Errorf("missing required function: %s", funcName)
		}
	}
	
	// Check for template variable syntax
	if !contains(code, "{{.") || !contains(code, "}}") {
		return fmt.Errorf("code should contain template variables")
	}
	
	return nil
}

func validatePinAssignments(template Template, board string) error {
	// Validate pin assignments based on board type
	pinRanges := map[string]map[string]int{
		"arduino:avr:uno": {
			"digital": 13,
			"analog":  5,
		},
		"arduino:avr:nano": {
			"digital": 13,
			"analog":  8,
		},
		"esp32:esp32:devkitv1": {
			"digital": 39,
			"analog":  12,
		},
		"esp8266:esp8266:d1mini": {
			"digital": 16,
			"analog":  1,
		},
	}
	
	// Check pin parameters against board limits
	for paramName, paramValue := range template.Parameters {
		if contains(paramName, "Pin") {
			if pinNum, ok := paramValue.(float64); ok {
				if rangeInfo, exists := pinRanges[board]; exists {
					if int(pinNum) > rangeInfo["digital"] {
						return fmt.Errorf("pin %d exceeds board limit for %s", 
							int(pinNum), board)
					}
				}
			}
		}
	}
	
	return nil
}

func validateSchema(schema map[string]interface{}) error {
	// Check schema structure
	if schema["type"] != "object" {
		return fmt.Errorf("schema type must be 'object'")
	}
	
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("schema must have 'properties' field")
	}
	
	// Validate each property
	for propName, propSchema := range properties {
		propMap, ok := propSchema.(map[string]interface{})
		if !ok {
			return fmt.Errorf("property %s must be an object", propName)
		}
		
		// Check required fields for property
		if _, exists := propMap["type"]; !exists {
			return fmt.Errorf("property %s must have a 'type' field", propName)
		}
		
		// Validate type
		propType := propMap["type"].(string)
		if !isValidType(propType) {
			return fmt.Errorf("property %s has invalid type: %s", propName, propType)
		}
	}
	
	return nil
}

func isValidType(typeStr string) bool {
	validTypes := []string{"string", "integer", "number", "boolean", "array", "object"}
	for _, validType := range validTypes {
		if typeStr == validType {
			return true
		}
	}
	return false
}

func validateParametersAgainstSchema(parameters map[string]interface{}, schema map[string]interface{}) error {
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("schema must have 'properties' field")
	}
	
	// Check each parameter against schema
	for paramName, paramValue := range parameters {
		propSchema, exists := properties[paramName]
		if !exists {
			return fmt.Errorf("parameter %s not defined in schema", paramName)
		}
		
		propMap := propSchema.(map[string]interface{})
		propType := propMap["type"].(string)
		
		// Validate parameter type
		if err := validateParameterType(paramName, paramValue, propType); err != nil {
			return err
		}
		
		// Validate constraints
		if err := validateParameterConstraints(paramName, paramValue, propMap); err != nil {
			return err
		}
	}
	
	return nil
}

func validateParameterType(paramName string, value interface{}, expectedType string) error {
	switch expectedType {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("parameter %s must be a string", paramName)
		}
	case "integer":
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("parameter %s must be an integer", paramName)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("parameter %s must be a boolean", paramName)
		}
	case "number":
		if _, ok := value.(float64); !ok {
			return fmt.Errorf("parameter %s must be a number", paramName)
		}
	}
	
	return nil
}

func validateParameterConstraints(paramName string, value interface{}, propMap map[string]interface{}) error {
	switch v := value.(type) {
	case float64:
		// Check minimum
		if min, exists := propMap["minimum"]; exists {
			if minVal, ok := min.(float64); ok {
				if v < minVal {
					return fmt.Errorf("parameter %s value %f is below minimum %f", 
						paramName, v, minVal)
				}
			}
		}
		
		// Check maximum
		if max, exists := propMap["maximum"]; exists {
			if maxVal, ok := max.(float64); ok {
				if v > maxVal {
					return fmt.Errorf("parameter %s value %f is above maximum %f", 
						paramName, v, maxVal)
				}
			}
		}
	case string:
		// Check minimum length
		if minLen, exists := propMap["minLength"]; exists {
			if minLenVal, ok := minLen.(float64); ok {
				if len(v) < int(minLenVal) {
					return fmt.Errorf("parameter %s length %d is below minimum %d", 
						paramName, len(v), int(minLenVal))
				}
			}
		}
		
		// Check maximum length
		if maxLen, exists := propMap["maxLength"]; exists {
			if maxLenVal, ok := maxLen.(float64); ok {
				if len(v) > int(maxLenVal) {
					return fmt.Errorf("parameter %s length %d is above maximum %d", 
						paramName, len(v), int(maxLenVal))
				}
			}
		}
	}
	
	return nil
}

func validateRequiredFields(template Template) error {
	required := []string{"id", "name", "version", "category", "description", "author", "boards_supported", "libraries", "schema", "parameters", "assets", "wiring_spec"}
	
	templateValue := reflect.ValueOf(template)
	for _, field := range required {
		fieldValue := templateValue.FieldByNameFunc(func(name string) bool {
			return toSnakeCase(name) == field
		})
		
		if !fieldValue.IsValid() || fieldValue.IsZero() {
			return fmt.Errorf("missing required field: %s", field)
		}
	}
	
	return nil
}

func testParameterValidation(template Template) error {
	// Test parameter validation with invalid values
	schema := template.Schema
	properties := schema["properties"].(map[string]interface{})
	
	for propName, propSchema := range properties {
		propMap := propSchema.(map[string]interface{})
		propType := propMap["type"].(string)
		
		// Test with invalid type
		invalidValue := getInvalidValue(propType)
		err := validateParameterType(propName, invalidValue, propType)
		if err == nil {
			return fmt.Errorf("parameter validation should fail for invalid type in %s", propName)
		}
		
		// Test with out-of-range values
		if propType == "integer" || propType == "number" {
			if min, exists := propMap["minimum"]; exists {
				minVal := min.(float64)
				err := validateParameterConstraints(propName, minVal-1, propMap)
				if err == nil {
					return fmt.Errorf("parameter validation should fail for value below minimum in %s", propName)
				}
			}
		}
	}
	
	return nil
}

func getInvalidValue(typeStr string) interface{} {
	switch typeStr {
	case "string":
		return 123
	case "integer":
		return "invalid"
	case "boolean":
		return "not_boolean"
	case "number":
		return "invalid_number"
	default:
		return nil
	}
}

func validateWiringSpec(wiringSpec WiringSpec) error {
	// Check components
	if len(wiringSpec.Components) == 0 {
		return fmt.Errorf("wiring spec must have at least one component")
	}
	
	// Check each component
	for _, component := range wiringSpec.Components {
		if component.ID == "" {
			return fmt.Errorf("component must have an ID")
		}
		if component.Type == "" {
			return fmt.Errorf("component must have a type")
		}
		if component.Name == "" {
			return fmt.Errorf("component must have a name")
		}
		if len(component.Pins) == 0 {
			return fmt.Errorf("component must have at least one pin")
		}
		
		// Check pins
		for _, pin := range component.Pins {
			if pin.Number == "" {
				return fmt.Errorf("pin must have a number")
			}
			if pin.Name == "" {
				return fmt.Errorf("pin must have a name")
			}
			if pin.Type == "" {
				return fmt.Errorf("pin must have a type")
			}
			if !isValidPinType(pin.Type) {
				return fmt.Errorf("invalid pin type: %s", pin.Type)
			}
		}
	}
	
	return nil
}

func isValidPinType(pinType string) bool {
	validTypes := []string{"power", "ground", "digital", "analog", "pwm", "i2c", "spi"}
	for _, validType := range validTypes {
		if pinType == validType {
			return true
		}
	}
	return false
}

func validateComponentCompatibility(wiringSpec WiringSpec) error {
	// Check for component compatibility
	// This is a simplified check - in reality, this would be more complex
	
	// Check that we have a board component
	hasBoard := false
	for _, component := range wiringSpec.Components {
		if component.Type == "board" {
			hasBoard = true
			break
		}
	}
	
	if !hasBoard {
		return fmt.Errorf("wiring spec must include a board component")
	}
	
	return nil
}

func validateConnections(wiringSpec WiringSpec) error {
	// Build component map for validation
	componentMap := make(map[string]Component)
	for _, component := range wiringSpec.Components {
		componentMap[component.ID] = component
	}
	
	// Validate each connection
	for _, connection := range wiringSpec.Connections {
		// Check components exist
		fromComponent, exists := componentMap[connection.FromComponent]
		if !exists {
			return fmt.Errorf("from component %s not found", connection.FromComponent)
		}
		
		toComponent, exists := componentMap[connection.ToComponent]
		if !exists {
			return fmt.Errorf("to component %s not found", connection.FromComponent)
		}
		
		// Check pins exist
		if !pinExists(fromComponent, connection.FromPin) {
			return fmt.Errorf("from pin %s not found in component %s", 
				connection.FromPin, connection.FromComponent)
		}
		
		if !pinExists(toComponent, connection.ToPin) {
			return fmt.Errorf("to pin %s not found in component %s", 
				connection.ToPin, connection.ToComponent)
		}
		
		// Check pin type compatibility
		if err := validatePinCompatibility(fromComponent, connection.FromPin, 
			toComponent, connection.ToPin); err != nil {
			return err
		}
	}
	
	return nil
}

func pinExists(component Component, pinNumber string) bool {
	for _, pin := range component.Pins {
		if pin.Number == pinNumber {
			return true
		}
	}
	return false
}

func validatePinCompatibility(fromComponent Component, fromPin string, 
	toComponent Component, toPin string) error {
	
	var fromPinType, toPinType string
	
	// Find pin types
	for _, pin := range fromComponent.Pins {
		if pin.Number == fromPin {
			fromPinType = pin.Type
			break
		}
	}
	
	for _, pin := range toComponent.Pins {
		if pin.Number == toPin {
			toPinType = pin.Type
			break
		}
	}
	
	// Check compatibility rules
	if fromPinType == "power" && toPinType != "power" {
		return nil // Power can connect to non-power
	}
	
	if fromPinType == "ground" && toPinType != "ground" {
		return nil // Ground can connect to non-ground
	}
	
	if fromPinType == "digital" && toPinType == "digital" {
		return nil // Digital to digital is OK
	}
	
	if fromPinType == "analog" && toPinType == "analog" {
		return nil // Analog to analog is OK
	}
	
	// More complex compatibility rules would go here
	
	return nil
}

func testWiringDiagramGeneration(template Template) error {
	// Test that wiring diagram can be generated
	// This would typically call a diagram generation service
	
	// For now, just validate the structure
	if len(template.WiringSpec.Components) == 0 {
		return fmt.Errorf("cannot generate diagram: no components")
	}
	
	if len(template.WiringSpec.Connections) == 0 {
		return fmt.Errorf("cannot generate diagram: no connections")
	}
	
	// Check that we have the necessary metadata for diagram generation
	for _, asset := range template.Assets {
		if asset.Type == "wiring_diagram" {
			if asset.Metadata == nil {
				return fmt.Errorf("wiring diagram asset missing metadata")
			}
			
			if _, exists := asset.Metadata["description"]; !exists {
				return fmt.Errorf("wiring diagram asset missing description")
			}
			
			if _, exists := asset.Metadata["connections"]; !exists {
				return fmt.Errorf("wiring diagram asset missing connections")
			}
			
			break
		}
	}
	
	return nil
}

// Utility functions
func contains(s string, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    len(s) > len(substr) && 
		    (s[:len(substr)] == substr || 
		     s[len(s)-len(substr):] == substr || 
		     findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toSnakeCase(s string) string {
	result := ""
	for i, c := range s {
		if i > 0 && c >= 'A' && c <= 'Z' {
			result += "_"
		}
		result += string(c)
	}
	return result
}

// Benchmark tests
func BenchmarkTemplateValidation(b *testing.B) {
	suite, err := NewTemplateTestSuite("../../templates/arduino")
	if err != nil {
		b.Fatal(err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, template := range suite.Templates {
			validateSchema(template.Schema)
			validateParametersAgainstSchema(template.Parameters, template.Schema)
			validateWiringSpec(template.WiringSpec)
		}
	}
}
