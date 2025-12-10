package nlp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ParameterFiller handles automatic parameter filling for templates
type ParameterFiller struct {
	llmClient *LLMClient
}

// NewParameterFiller creates a new parameter filler
func NewParameterFiller(llmClient *LLMClient) *ParameterFiller {
	return &ParameterFiller{
		llmClient: llmClient,
	}
}

// FillParameters automatically fills template parameters based on requirements
func (pf *ParameterFiller) FillParameters(ctx context.Context, requirements *ParsedRequirements, template *TemplateInfo, schema map[string]interface{}) (map[string]interface{}, error) {
	// Start with default parameters from schema
	parameters := pf.extractDefaultParameters(schema)

	// Fill sensor-related parameters
	if err := pf.fillSensorParameters(requirements, parameters); err != nil {
		return nil, fmt.Errorf("failed to fill sensor parameters: %w", err)
	}

	// Fill actuator-related parameters
	if err := pf.fillActuatorParameters(requirements, parameters); err != nil {
		return nil, fmt.Errorf("failed to fill actuator parameters: %w", err)
	}

	// Fill communication parameters
	if err := pf.fillCommunicationParameters(requirements, parameters); err != nil {
		return nil, fmt.Errorf("failed to fill communication parameters: %w", err)
	}

	// Fill timing parameters
	pf.fillTimingParameters(requirements, parameters)

	// Use LLM to fill any remaining complex parameters
	if err := pf.fillRemainingParameters(ctx, requirements, template, schema, parameters); err != nil {
		return nil, fmt.Errorf("failed to fill remaining parameters: %w", err)
	}

	return parameters, nil
}

// extractDefaultParameters extracts default values from JSON schema
func (pf *ParameterFiller) extractDefaultParameters(schema map[string]interface{}) map[string]interface{} {
	parameters := make(map[string]interface{})

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return parameters
	}

	for key, value := range properties {
		propMap, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract default value if present
		if defaultVal, exists := propMap["default"]; exists {
			parameters[key] = defaultVal
		}
	}

	return parameters
}

// fillSensorParameters fills sensor-related parameters
func (pf *ParameterFiller) fillSensorParameters(requirements *ParsedRequirements, parameters map[string]interface{}) error {
	for i, sensor := range requirements.Sensors {
		// Fill pin assignments
		if sensor.Pin != "" {
			pinKey := fmt.Sprintf("%s_pin", sensor.Type)
			if i > 0 {
				pinKey = fmt.Sprintf("%s_pin_%d", sensor.Type, i+1)
			}
			parameters[pinKey] = pf.normalizePinName(sensor.Pin)
		}

		// Fill sample rate
		if sensor.SampleRate > 0 {
			rateKey := fmt.Sprintf("%s_sample_rate", sensor.Type)
			if i > 0 {
				rateKey = fmt.Sprintf("%s_sample_rate_%d", sensor.Type, i+1)
			}
			parameters[rateKey] = sensor.SampleRate
		}

		// Fill threshold values
		if sensor.Threshold != nil {
			thresholdKey := fmt.Sprintf("%s_threshold", sensor.Type)
			if i > 0 {
				thresholdKey = fmt.Sprintf("%s_threshold_%d", sensor.Type, i+1)
			}
			parameters[thresholdKey] = sensor.Threshold.Value
		}

		// Fill sensor model if specified
		if sensor.Model != "" {
			modelKey := fmt.Sprintf("%s_model", sensor.Type)
			parameters[modelKey] = sensor.Model
		}
	}

	return nil
}

// fillActuatorParameters fills actuator-related parameters
func (pf *ParameterFiller) fillActuatorParameters(requirements *ParsedRequirements, parameters map[string]interface{}) error {
	for i, actuator := range requirements.Actuators {
		// Fill pin assignments
		if actuator.Pin != "" {
			pinKey := fmt.Sprintf("%s_pin", actuator.Type)
			if i > 0 {
				pinKey = fmt.Sprintf("%s_pin_%d", actuator.Type, i+1)
			}
			parameters[pinKey] = pf.normalizePinName(actuator.Pin)
		}

		// Fill initial state
		if actuator.InitialState != "" {
			stateKey := fmt.Sprintf("%s_initial_state", actuator.Type)
			if i > 0 {
				stateKey = fmt.Sprintf("%s_initial_state_%d", actuator.Type, i+1)
			}
			parameters[stateKey] = actuator.InitialState
		}
	}

	return nil
}

// fillCommunicationParameters fills communication-related parameters
func (pf *ParameterFiller) fillCommunicationParameters(requirements *ParsedRequirements, parameters map[string]interface{}) error {
	for _, comm := range requirements.Communication {
		switch strings.ToLower(comm.Protocol) {
		case "wifi":
			// WiFi parameters will be filled from secrets
			parameters["wifi_enabled"] = true

		case "mqtt":
			if comm.Endpoint != "" {
				parameters["mqtt_server"] = comm.Endpoint
			}
			if comm.Port > 0 {
				parameters["mqtt_port"] = comm.Port
			} else {
				parameters["mqtt_port"] = 1883 // Default MQTT port
			}
			if comm.Topic != "" {
				parameters["mqtt_topic"] = comm.Topic
			}

		case "http", "https":
			if comm.Endpoint != "" {
				parameters["http_endpoint"] = comm.Endpoint
			}
			if comm.Port > 0 {
				parameters["http_port"] = comm.Port
			}

		case "bluetooth", "ble":
			parameters["bluetooth_enabled"] = true
		}
	}

	return nil
}

// fillTimingParameters fills timing-related parameters
func (pf *ParameterFiller) fillTimingParameters(requirements *ParsedRequirements, parameters map[string]interface{}) {
	// Set default loop delay if not specified
	if _, exists := parameters["loop_delay"]; !exists {
		// Use the minimum sensor sample rate as loop delay
		minSampleRate := 1000 // Default 1 second
		for _, sensor := range requirements.Sensors {
			if sensor.SampleRate > 0 && sensor.SampleRate < minSampleRate {
				minSampleRate = sensor.SampleRate
			}
		}
		parameters["loop_delay"] = minSampleRate
	}

	// Set default delays for actuators
	if _, exists := parameters["delay_on"]; !exists {
		parameters["delay_on"] = 1000
	}
	if _, exists := parameters["delay_off"]; !exists {
		parameters["delay_off"] = 1000
	}
}

// fillRemainingParameters uses LLM to fill complex parameters
func (pf *ParameterFiller) fillRemainingParameters(ctx context.Context, requirements *ParsedRequirements, template *TemplateInfo, schema map[string]interface{}, parameters map[string]interface{}) error {
	// Identify unfilled required parameters
	unfilledParams := pf.findUnfilledRequiredParameters(schema, parameters)
	if len(unfilledParams) == 0 {
		return nil // All required parameters are filled
	}

	// Build prompt for LLM
	prompt := pf.buildParameterFillingPrompt(requirements, template, schema, parameters, unfilledParams)

	// Get LLM response
	response, err := pf.llmClient.Complete(ctx, prompt)
	if err != nil {
		return fmt.Errorf("LLM request failed: %w", err)
	}

	// Extract JSON from response
	jsonContent, err := ExtractJSON(response.Content)
	if err != nil {
		return fmt.Errorf("failed to extract JSON from response: %w", err)
	}

	// Parse suggested parameters
	var suggestedParams map[string]interface{}
	if err := json.Unmarshal([]byte(jsonContent), &suggestedParams); err != nil {
		return fmt.Errorf("failed to parse suggested parameters: %w", err)
	}

	// Merge suggested parameters
	for key, value := range suggestedParams {
		if _, exists := parameters[key]; !exists {
			parameters[key] = value
		}
	}

	return nil
}

// findUnfilledRequiredParameters finds required parameters that haven't been filled
func (pf *ParameterFiller) findUnfilledRequiredParameters(schema map[string]interface{}, parameters map[string]interface{}) []string {
	var unfilled []string

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return unfilled
	}

	required, ok := schema["required"].([]interface{})
	if !ok {
		return unfilled
	}

	// Convert required to map for easier lookup
	requiredMap := make(map[string]bool)
	for _, req := range required {
		if reqStr, ok := req.(string); ok {
			requiredMap[reqStr] = true
		}
	}

	// Find unfilled required parameters
	for key := range properties {
		if requiredMap[key] {
			if _, exists := parameters[key]; !exists {
				unfilled = append(unfilled, key)
			}
		}
	}

	return unfilled
}

// buildParameterFillingPrompt builds a prompt for parameter filling
func (pf *ParameterFiller) buildParameterFillingPrompt(requirements *ParsedRequirements, template *TemplateInfo, schema map[string]interface{}, currentParams map[string]interface{}, unfilledParams []string) string {
	return fmt.Sprintf(`You are an expert Arduino developer. Fill in the missing template parameters based on the user requirements.

User Requirements:
Intent: %s
Sensors: %v
Actuators: %v
Communication: %v

Template: %s
Category: %s

Current Parameters: %v

Unfilled Required Parameters: %v

Parameter Schema: %v

Provide sensible default values for the unfilled parameters based on the user requirements and template context.
Return ONLY a JSON object with the parameter names as keys and their values.

Example:
{
  "device_name": "Temperature Monitor",
  "update_interval": 5000,
  "enable_logging": true
}`, requirements.Intent, pf.formatSensors(requirements.Sensors), pf.formatActuators(requirements.Actuators), pf.formatCommunication(requirements.Communication), template.Name, template.Category, currentParams, unfilledParams, schema)
}

// ValidateParameters validates filled parameters against board capabilities
func (pf *ParameterFiller) ValidateParameters(ctx context.Context, parameters map[string]interface{}, boardType string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Validate pin assignments
	usedPins := make(map[string]bool)
	for key, value := range parameters {
		if strings.HasSuffix(key, "_pin") {
			pinStr, ok := value.(string)
			if !ok {
				continue
			}

			// Check for duplicate pin assignments
			if usedPins[pinStr] {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Pin %s is assigned multiple times", pinStr))
			}
			usedPins[pinStr] = true

			// Validate pin exists on board
			if !pf.isValidPin(pinStr, boardType) {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Pin %s is not valid for board %s", pinStr, boardType))
			}
		}
	}

	// Validate timing parameters
	for key, value := range parameters {
		if strings.Contains(key, "delay") || strings.Contains(key, "interval") || strings.Contains(key, "rate") {
			if intVal, ok := value.(int); ok {
				if intVal < 0 {
					result.Valid = false
					result.Errors = append(result.Errors, fmt.Sprintf("Parameter %s cannot be negative", key))
				}
				if intVal < 10 {
					result.Warnings = append(result.Warnings, fmt.Sprintf("Parameter %s is very small (%d ms), may cause performance issues", key, intVal))
				}
			}
		}
	}

	return result, nil
}

// Helper methods

func (pf *ParameterFiller) normalizePinName(pin string) string {
	pin = strings.TrimSpace(pin)
	pin = strings.ToUpper(pin)

	// Remove common prefixes
	pin = strings.TrimPrefix(pin, "PIN")
	pin = strings.TrimSpace(pin)

	return pin
}

func (pf *ParameterFiller) isValidPin(pin string, boardType string) bool {
	// Simplified validation - in production, this would check against board specifications
	pin = strings.ToUpper(pin)

	// Common Arduino pins
	if strings.HasPrefix(pin, "D") || strings.HasPrefix(pin, "A") {
		return true
	}

	// Numeric pins
	if len(pin) > 0 && pin[0] >= '0' && pin[0] <= '9' {
		return true
	}

	return false
}

func (pf *ParameterFiller) formatSensors(sensors []SensorSpec) string {
	if len(sensors) == 0 {
		return "none"
	}

	var parts []string
	for _, s := range sensors {
		if s.Model != "" {
			parts = append(parts, fmt.Sprintf("%s (%s)", s.Type, s.Model))
		} else {
			parts = append(parts, s.Type)
		}
	}
	return strings.Join(parts, ", ")
}

func (pf *ParameterFiller) formatActuators(actuators []ActuatorSpec) string {
	if len(actuators) == 0 {
		return "none"
	}

	var parts []string
	for _, a := range actuators {
		parts = append(parts, a.Type)
	}
	return strings.Join(parts, ", ")
}

func (pf *ParameterFiller) formatCommunication(comms []CommSpec) string {
	if len(comms) == 0 {
		return "none"
	}

	var parts []string
	for _, c := range comms {
		parts = append(parts, c.Protocol)
	}
	return strings.Join(parts, ", ")
}
