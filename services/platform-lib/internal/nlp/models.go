package nlp

import "time"

// ParsedRequirements represents extracted requirements from natural language input
type ParsedRequirements struct {
	Intent          string                 `json:"intent"`
	Sensors         []SensorSpec           `json:"sensors"`
	Actuators       []ActuatorSpec         `json:"actuators"`
	Communication   []CommSpec             `json:"communication"`
	Constraints     map[string]interface{} `json:"constraints"`
	BoardPreference string                 `json:"board_preference"`
	RawInput        string                 `json:"raw_input"`
	ParsedAt        time.Time              `json:"parsed_at"`
}

// SensorSpec represents a sensor specification
type SensorSpec struct {
	Type       string                 `json:"type"`        // e.g., "temperature", "humidity", "distance"
	Model      string                 `json:"model"`       // e.g., "DHT22", "HC-SR04"
	Pin        string                 `json:"pin"`         // e.g., "D2", "A0"
	SampleRate int                    `json:"sample_rate"` // milliseconds
	Threshold  *ThresholdSpec         `json:"threshold,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ActuatorSpec represents an actuator specification
type ActuatorSpec struct {
	Type         string                 `json:"type"`          // e.g., "led", "servo", "relay", "motor"
	Pin          string                 `json:"pin"`           // e.g., "D3", "D4"
	InitialState string                 `json:"initial_state"` // e.g., "off", "0", "closed"
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// CommSpec represents a communication specification
type CommSpec struct {
	Protocol string                 `json:"protocol"` // e.g., "wifi", "mqtt", "bluetooth", "http"
	Endpoint string                 `json:"endpoint,omitempty"`
	Port     int                    `json:"port,omitempty"`
	Topic    string                 `json:"topic,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ThresholdSpec represents a threshold specification for sensors
type ThresholdSpec struct {
	Min      float64 `json:"min,omitempty"`
	Max      float64 `json:"max,omitempty"`
	Operator string  `json:"operator"` // e.g., "gt", "lt", "eq", "between"
	Value    float64 `json:"value,omitempty"`
}

// ImplementationPlan represents a complete implementation plan
type ImplementationPlan struct {
	TemplateID      string                 `json:"template_id"`
	TemplateName    string                 `json:"template_name"`
	Parameters      map[string]interface{} `json:"parameters"`
	WiringDiagram   *WiringDiagram         `json:"wiring_diagram"`
	BOM             []BOMItem              `json:"bom"`
	Instructions    []string               `json:"instructions"`
	Warnings        []string               `json:"warnings"`
	SafetyChecks    *SafetyValidation      `json:"safety_checks"`
	EstimatedCost   float64                `json:"estimated_cost,omitempty"`
	DifficultyLevel string                 `json:"difficulty_level"`
	CreatedAt       time.Time              `json:"created_at"`
}

// WiringDiagram represents a generated wiring diagram
type WiringDiagram struct {
	MermaidSyntax string                 `json:"mermaid_syntax"`
	Components    []Component            `json:"components"`
	Connections   []Connection           `json:"connections"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Component represents a hardware component
type Component struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Name     string                 `json:"name"`
	Pins     []Pin                  `json:"pins"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Pin represents a component pin
type Pin struct {
	Number      string  `json:"number"`
	Name        string  `json:"name"`
	Type        string  `json:"type"` // 'digital', 'analog', 'power', 'ground'
	Voltage     string  `json:"voltage,omitempty"`
	MaxCurrent  float64 `json:"max_current,omitempty"` // in mA
	Description string  `json:"description,omitempty"`
}

// Connection represents a wiring connection
type Connection struct {
	FromComponent string `json:"from_component"`
	FromPin       string `json:"from_pin"`
	ToComponent   string `json:"to_component"`
	ToPin         string `json:"to_pin"`
	WireColor     string `json:"wire_color,omitempty"`
}

// BOMItem represents a bill of materials item
type BOMItem struct {
	Component   string  `json:"component"`
	Quantity    int     `json:"quantity"`
	Description string  `json:"description"`
	PartNumber  string  `json:"part_number,omitempty"`
	Price       float64 `json:"price,omitempty"`
	Supplier    string  `json:"supplier,omitempty"`
	URL         string  `json:"url,omitempty"`
}

// SafetyValidation represents electrical safety validation results
type SafetyValidation struct {
	Valid            bool                    `json:"valid"`
	Errors           []string                `json:"errors,omitempty"`
	Warnings         []string                `json:"warnings,omitempty"`
	VoltageChecks    []VoltageCheck          `json:"voltage_checks"`
	CurrentChecks    []CurrentCheck          `json:"current_checks"`
	PinCompatibility []PinCompatibilityCheck `json:"pin_compatibility"`
}

// VoltageCheck represents a voltage compatibility check
type VoltageCheck struct {
	Component       string `json:"component"`
	RequiredVoltage string `json:"required_voltage"`
	SuppliedVoltage string `json:"supplied_voltage"`
	Compatible      bool   `json:"compatible"`
	Message         string `json:"message,omitempty"`
}

// CurrentCheck represents a current limit check
type CurrentCheck struct {
	Pin             string  `json:"pin"`
	Component       string  `json:"component"`
	RequiredCurrent float64 `json:"required_current"` // in mA
	MaxCurrent      float64 `json:"max_current"`      // in mA
	Safe            bool    `json:"safe"`
	Message         string  `json:"message,omitempty"`
}

// PinCompatibilityCheck represents a pin compatibility check
type PinCompatibilityCheck struct {
	Pin        string `json:"pin"`
	Component  string `json:"component"`
	PinType    string `json:"pin_type"`
	Required   string `json:"required"`
	Compatible bool   `json:"compatible"`
	Message    string `json:"message,omitempty"`
}

// ValidationResult represents validation results
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// LLMRequest represents a request to the LLM provider
type LLMRequest struct {
	Prompt      string                 `json:"prompt"`
	Model       string                 `json:"model"`
	Temperature float64                `json:"temperature"`
	MaxTokens   int                    `json:"max_tokens"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// LLMResponse represents a response from the LLM provider
type LLMResponse struct {
	Content      string                 `json:"content"`
	Model        string                 `json:"model"`
	TokensUsed   int                    `json:"tokens_used"`
	FinishReason string                 `json:"finish_reason"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}
