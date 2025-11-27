package nlp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Parser handles natural language parsing
type Parser struct {
	llmClient LLMClientInterface
}

// NewParser creates a new parser instance
func NewParser(llmClient LLMClientInterface) *Parser {
	return &Parser{
		llmClient: llmClient,
	}
}

// ParseRequirements extracts structured requirements from natural language input
func (p *Parser) ParseRequirements(ctx context.Context, input string) (*ParsedRequirements, error) {
	prompt := p.buildParsingPrompt(input)

	response, err := p.llmClient.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	// Extract JSON from response
	jsonContent, err := ExtractJSON(response.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JSON from response: %w", err)
	}

	// Parse the JSON response
	var requirements ParsedRequirements
	if err := json.Unmarshal([]byte(jsonContent), &requirements); err != nil {
		return nil, fmt.Errorf("failed to parse requirements JSON: %w", err)
	}

	// Set metadata
	requirements.RawInput = input
	requirements.ParsedAt = time.Now()

	// Validate and normalize the parsed requirements
	if err := p.validateRequirements(&requirements); err != nil {
		return nil, fmt.Errorf("requirements validation failed: %w", err)
	}

	return &requirements, nil
}

// buildParsingPrompt constructs the prompt for requirement extraction
func (p *Parser) buildParsingPrompt(input string) string {
	return fmt.Sprintf(`You are an expert Arduino developer. Parse the following natural language description of an Arduino project and extract structured requirements.

User Input: %s

Extract and return a JSON object with the following structure:
{
  "intent": "Brief description of what the user wants to build",
  "sensors": [
    {
      "type": "sensor type (e.g., temperature, humidity, distance, motion)",
      "model": "specific model if mentioned (e.g., DHT22, HC-SR04)",
      "pin": "pin assignment if mentioned",
      "sample_rate": sample rate in milliseconds (default 1000),
      "threshold": {
        "operator": "gt/lt/eq/between",
        "value": threshold value if mentioned
      }
    }
  ],
  "actuators": [
    {
      "type": "actuator type (e.g., led, servo, relay, motor, buzzer)",
      "pin": "pin assignment if mentioned",
      "initial_state": "initial state (e.g., off, 0, closed)"
    }
  ],
  "communication": [
    {
      "protocol": "communication protocol (e.g., wifi, mqtt, bluetooth, http)",
      "endpoint": "server endpoint if mentioned",
      "port": port number if mentioned,
      "topic": "MQTT topic if applicable"
    }
  ],
  "constraints": {
    "power_source": "battery/usb/external if mentioned",
    "size": "size constraints if mentioned",
    "cost": "budget constraints if mentioned",
    "environment": "indoor/outdoor/waterproof if mentioned"
  },
  "board_preference": "preferred Arduino board (e.g., uno, nano, esp32, esp8266)"
}

Guidelines:
- Extract all sensors, actuators, and communication methods mentioned
- Infer reasonable defaults for unspecified parameters
- If no board is mentioned, leave board_preference empty
- For sensors without specific models, use generic types
- Include threshold information if the user mentions conditions like "when temperature exceeds X"
- Extract communication requirements like WiFi, MQTT, Bluetooth
- Identify constraints like power source, size, budget, environment

Return ONLY the JSON object, no additional text.`, input)
}

// validateRequirements validates and normalizes parsed requirements
func (p *Parser) validateRequirements(req *ParsedRequirements) error {
	if req.Intent == "" {
		return fmt.Errorf("intent cannot be empty")
	}

	// Normalize sensor types
	for i := range req.Sensors {
		req.Sensors[i].Type = strings.ToLower(strings.TrimSpace(req.Sensors[i].Type))
		if req.Sensors[i].SampleRate <= 0 {
			req.Sensors[i].SampleRate = 1000 // Default 1 second
		}
	}

	// Normalize actuator types
	for i := range req.Actuators {
		req.Actuators[i].Type = strings.ToLower(strings.TrimSpace(req.Actuators[i].Type))
	}

	// Normalize communication protocols
	for i := range req.Communication {
		req.Communication[i].Protocol = strings.ToLower(strings.TrimSpace(req.Communication[i].Protocol))
	}

	// Normalize board preference
	if req.BoardPreference != "" {
		req.BoardPreference = strings.ToLower(strings.TrimSpace(req.BoardPreference))
	}

	return nil
}

// ClassifyIntent classifies the project intent into categories
func (p *Parser) ClassifyIntent(ctx context.Context, requirements *ParsedRequirements) (string, error) {
	prompt := fmt.Sprintf(`Classify the following Arduino project intent into ONE of these categories:
- sensing: Projects focused on reading sensor data
- automation: Projects that control devices based on conditions
- monitoring: Projects that track and report data over time
- communication: Projects focused on data transmission
- display: Projects that show information on screens
- robotics: Projects involving movement and navigation
- wearable: Projects designed to be worn
- audio: Projects involving sound generation or processing
- data_logging: Projects that record data for later analysis

Project Intent: %s

Sensors: %v
Actuators: %v
Communication: %v

Return ONLY the category name, nothing else.`, requirements.Intent, p.getSensorTypes(requirements), p.getActuatorTypes(requirements), p.getCommunicationProtocols(requirements))

	response, err := p.llmClient.Complete(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("intent classification failed: %w", err)
	}

	category := strings.ToLower(strings.TrimSpace(response.Content))

	// Validate category
	validCategories := map[string]bool{
		"sensing": true, "automation": true, "monitoring": true,
		"communication": true, "display": true, "robotics": true,
		"wearable": true, "audio": true, "data_logging": true,
	}

	if !validCategories[category] {
		// Default to sensing if invalid
		return "sensing", nil
	}

	return category, nil
}

// ExtractTechnicalSpecs extracts detailed technical specifications
func (p *Parser) ExtractTechnicalSpecs(ctx context.Context, requirements *ParsedRequirements) (map[string]interface{}, error) {
	specs := make(map[string]interface{})

	// Extract sensor specifications
	if len(requirements.Sensors) > 0 {
		specs["sensors"] = requirements.Sensors
		specs["sensor_count"] = len(requirements.Sensors)
	}

	// Extract actuator specifications
	if len(requirements.Actuators) > 0 {
		specs["actuators"] = requirements.Actuators
		specs["actuator_count"] = len(requirements.Actuators)
	}

	// Extract communication specifications
	if len(requirements.Communication) > 0 {
		specs["communication"] = requirements.Communication
		specs["requires_network"] = p.requiresNetwork(requirements.Communication)
	}

	// Extract constraints
	if requirements.Constraints != nil {
		specs["constraints"] = requirements.Constraints
	}

	// Determine complexity
	complexity := p.calculateComplexity(requirements)
	specs["complexity"] = complexity

	// Determine power requirements
	powerReq := p.estimatePowerRequirements(requirements)
	specs["power_requirements"] = powerReq

	return specs, nil
}

// Helper methods

func (p *Parser) getSensorTypes(req *ParsedRequirements) []string {
	types := make([]string, len(req.Sensors))
	for i, sensor := range req.Sensors {
		types[i] = sensor.Type
	}
	return types
}

func (p *Parser) getActuatorTypes(req *ParsedRequirements) []string {
	types := make([]string, len(req.Actuators))
	for i, actuator := range req.Actuators {
		types[i] = actuator.Type
	}
	return types
}

func (p *Parser) getCommunicationProtocols(req *ParsedRequirements) []string {
	protocols := make([]string, len(req.Communication))
	for i, comm := range req.Communication {
		protocols[i] = comm.Protocol
	}
	return protocols
}

func (p *Parser) requiresNetwork(comms []CommSpec) bool {
	for _, comm := range comms {
		if comm.Protocol == "wifi" || comm.Protocol == "mqtt" ||
			comm.Protocol == "http" || comm.Protocol == "https" {
			return true
		}
	}
	return false
}

func (p *Parser) calculateComplexity(req *ParsedRequirements) string {
	score := 0

	// Add points for sensors
	score += len(req.Sensors)

	// Add points for actuators
	score += len(req.Actuators)

	// Add points for communication
	score += len(req.Communication) * 2

	// Add points for thresholds and conditions
	for _, sensor := range req.Sensors {
		if sensor.Threshold != nil {
			score++
		}
	}

	if score <= 3 {
		return "beginner"
	} else if score <= 7 {
		return "intermediate"
	}
	return "advanced"
}

func (p *Parser) estimatePowerRequirements(req *ParsedRequirements) string {
	// Check for high-power components
	for _, actuator := range req.Actuators {
		if actuator.Type == "motor" || actuator.Type == "servo" || actuator.Type == "relay" {
			return "external" // Requires external power supply
		}
	}

	// Check for network communication
	for _, comm := range req.Communication {
		if comm.Protocol == "wifi" {
			return "usb" // WiFi requires stable power
		}
	}

	// Default to USB power
	if len(req.Sensors) > 2 || len(req.Actuators) > 1 {
		return "usb"
	}

	return "battery" // Simple projects can run on battery
}
