package nlp

import (
	"context"
	"fmt"
	"strings"
)

// SafetyValidator handles electrical safety validation
type SafetyValidator struct {
	boardSpecs map[string]*BoardSpecification
}

// BoardSpecification represents board electrical specifications
type BoardSpecification struct {
	Name              string
	OperatingVoltage  string
	IOVoltage         string
	MaxCurrentPerPin  float64 // in mA
	MaxTotalCurrent   float64 // in mA
	DigitalPins       []string
	AnalogPins        []string
	PWMPins           []string
	PowerPins         map[string]string // pin -> voltage
}

// NewSafetyValidator creates a new safety validator
func NewSafetyValidator() *SafetyValidator {
	return &SafetyValidator{
		boardSpecs: initializeBoardSpecs(),
	}
}

// ValidateSafety performs comprehensive electrical safety validation
func (sv *SafetyValidator) ValidateSafety(ctx context.Context, plan *ImplementationPlan, boardType string) (*SafetyValidation, error) {
	validation := &SafetyValidation{
		Valid:            true,
		Errors:           []string{},
		Warnings:         []string{},
		VoltageChecks:    []VoltageCheck{},
		CurrentChecks:    []CurrentCheck{},
		PinCompatibility: []PinCompatibilityCheck{},
	}

	// Get board specifications
	boardSpec, exists := sv.boardSpecs[strings.ToLower(boardType)]
	if !exists {
		validation.Warnings = append(validation.Warnings, fmt.Sprintf("Board specifications not found for %s, using generic validation", boardType))
		boardSpec = sv.getGenericBoardSpec()
	}

	// Validate voltage compatibility
	sv.validateVoltageCompatibility(plan, boardSpec, validation)

	// Validate current limits
	sv.validateCurrentLimits(plan, boardSpec, validation)

	// Validate pin compatibility
	sv.validatePinCompatibility(plan, boardSpec, validation)

	// Validate pin conflicts
	sv.validatePinConflicts(plan, validation)

	// Validate component compatibility
	sv.validateComponentCompatibility(plan, validation)

	return validation, nil
}

// validateVoltageCompatibility checks voltage compatibility between components
func (sv *SafetyValidator) validateVoltageCompatibility(plan *ImplementationPlan, boardSpec *BoardSpecification, validation *SafetyValidation) {
	for _, component := range plan.WiringDiagram.Components {
		// Skip the board itself
		if component.Type == "board" || component.Type == "arduino" {
			continue
		}

		// Get component voltage requirements
		requiredVoltage := sv.getComponentVoltageRequirement(component)
		if requiredVoltage == "" {
			continue
		}

		// Check against board voltage
		compatible := sv.isVoltageCompatible(requiredVoltage, boardSpec.IOVoltage)

		check := VoltageCheck{
			Component:       component.Name,
			RequiredVoltage: requiredVoltage,
			SuppliedVoltage: boardSpec.IOVoltage,
			Compatible:      compatible,
		}

		if !compatible {
			check.Message = fmt.Sprintf("Component %s requires %s but board provides %s", component.Name, requiredVoltage, boardSpec.IOVoltage)
			validation.Valid = false
			validation.Errors = append(validation.Errors, check.Message)
		}

		validation.VoltageChecks = append(validation.VoltageChecks, check)
	}
}

// validateCurrentLimits checks current draw against pin and board limits
func (sv *SafetyValidator) validateCurrentLimits(plan *ImplementationPlan, boardSpec *BoardSpecification, validation *SafetyValidation) {
	totalCurrent := 0.0
	pinCurrentMap := make(map[string]float64)

	for _, connection := range plan.WiringDiagram.Connections {
		// Get component connected to this pin
		component := sv.findComponent(plan.WiringDiagram.Components, connection.ToComponent)
		if component == nil {
			continue
		}

		// Estimate current draw
		currentDraw := sv.estimateComponentCurrent(component)
		if currentDraw == 0 {
			continue
		}

		// Track per-pin current
		pinCurrentMap[connection.FromPin] += currentDraw
		totalCurrent += currentDraw

		// Check per-pin limit
		if pinCurrentMap[connection.FromPin] > boardSpec.MaxCurrentPerPin {
			check := CurrentCheck{
				Pin:             connection.FromPin,
				Component:       component.Name,
				RequiredCurrent: currentDraw,
				MaxCurrent:      boardSpec.MaxCurrentPerPin,
				Safe:            false,
				Message:         fmt.Sprintf("Pin %s current draw (%.1f mA) exceeds maximum (%.1f mA)", connection.FromPin, pinCurrentMap[connection.FromPin], boardSpec.MaxCurrentPerPin),
			}
			validation.CurrentChecks = append(validation.CurrentChecks, check)
			validation.Valid = false
			validation.Errors = append(validation.Errors, check.Message)
		} else {
			check := CurrentCheck{
				Pin:             connection.FromPin,
				Component:       component.Name,
				RequiredCurrent: currentDraw,
				MaxCurrent:      boardSpec.MaxCurrentPerPin,
				Safe:            true,
			}
			validation.CurrentChecks = append(validation.CurrentChecks, check)
		}
	}

	// Check total current
	if totalCurrent > boardSpec.MaxTotalCurrent {
		validation.Valid = false
		validation.Errors = append(validation.Errors, fmt.Sprintf("Total current draw (%.1f mA) exceeds board maximum (%.1f mA)", totalCurrent, boardSpec.MaxTotalCurrent))
	}
}

// validatePinCompatibility checks if pins are used correctly
func (sv *SafetyValidator) validatePinCompatibility(plan *ImplementationPlan, boardSpec *BoardSpecification, validation *SafetyValidation) {
	for _, connection := range plan.WiringDiagram.Connections {
		component := sv.findComponent(plan.WiringDiagram.Components, connection.ToComponent)
		if component == nil {
			continue
		}

		// Determine required pin type
		requiredType := sv.getRequiredPinType(component)
		if requiredType == "" {
			continue
		}

		// Check if pin supports required type
		compatible := sv.isPinCompatible(connection.FromPin, requiredType, boardSpec)

		check := PinCompatibilityCheck{
			Pin:        connection.FromPin,
			Component:  component.Name,
			PinType:    sv.getPinType(connection.FromPin, boardSpec),
			Required:   requiredType,
			Compatible: compatible,
		}

		if !compatible {
			check.Message = fmt.Sprintf("Pin %s (%s) is not compatible with %s (requires %s)", connection.FromPin, check.PinType, component.Name, requiredType)
			validation.Valid = false
			validation.Errors = append(validation.Errors, check.Message)
		}

		validation.PinCompatibility = append(validation.PinCompatibility, check)
	}
}

// validatePinConflicts checks for duplicate pin assignments
func (sv *SafetyValidator) validatePinConflicts(plan *ImplementationPlan, validation *SafetyValidation) {
	pinUsage := make(map[string][]string)

	for _, connection := range plan.WiringDiagram.Connections {
		// Skip power and ground pins
		if sv.isPowerPin(connection.FromPin) {
			continue
		}

		component := sv.findComponent(plan.WiringDiagram.Components, connection.ToComponent)
		if component != nil {
			pinUsage[connection.FromPin] = append(pinUsage[connection.FromPin], component.Name)
		}
	}

	// Check for conflicts
	for pin, components := range pinUsage {
		if len(components) > 1 {
			validation.Valid = false
			validation.Errors = append(validation.Errors, fmt.Sprintf("Pin %s is assigned to multiple components: %v", pin, components))
		}
	}
}

// validateComponentCompatibility checks if components work together
func (sv *SafetyValidator) validateComponentCompatibility(plan *ImplementationPlan, validation *SafetyValidation) {
	// Check for known incompatible combinations
	componentTypes := make(map[string]bool)
	for _, component := range plan.WiringDiagram.Components {
		componentTypes[strings.ToLower(component.Type)] = true
	}

	// Example: Check for I2C address conflicts
	if sv.hasMultipleI2CDevices(plan.WiringDiagram.Components) {
		validation.Warnings = append(validation.Warnings, "Multiple I2C devices detected - ensure they have different addresses")
	}

	// Example: Check for SPI conflicts
	if sv.hasMultipleSPIDevices(plan.WiringDiagram.Components) {
		validation.Warnings = append(validation.Warnings, "Multiple SPI devices detected - ensure proper chip select pin management")
	}
}

// Helper methods

func (sv *SafetyValidator) getComponentVoltageRequirement(component Component) string {
	// Extract voltage from metadata or infer from component type
	if voltage, ok := component.Metadata["voltage"].(string); ok {
		return voltage
	}

	// Infer from component type
	componentType := strings.ToLower(component.Type)
	switch {
	case strings.Contains(componentType, "5v"):
		return "5V"
	case strings.Contains(componentType, "3.3v") || strings.Contains(componentType, "3v3"):
		return "3.3V"
	case strings.Contains(componentType, "led"):
		return "3.3V" // Most LEDs work with 3.3V
	case strings.Contains(componentType, "sensor"):
		return "3.3V" // Most modern sensors are 3.3V
	}

	return ""
}

func (sv *SafetyValidator) isVoltageCompatible(required, supplied string) bool {
	required = strings.ToUpper(strings.TrimSpace(required))
	supplied = strings.ToUpper(strings.TrimSpace(supplied))

	// Exact match
	if required == supplied {
		return true
	}

	// 3.3V components can work with 5V through level shifters (warning, not error)
	// 5V components generally don't work with 3.3V

	return false
}

func (sv *SafetyValidator) estimateComponentCurrent(component *Component) float64 {
	// Estimate current draw based on component type
	componentType := strings.ToLower(component.Type)

	switch {
	case strings.Contains(componentType, "led"):
		return 20.0 // 20mA typical for LED
	case strings.Contains(componentType, "servo"):
		return 500.0 // 500mA typical for small servo
	case strings.Contains(componentType, "motor"):
		return 1000.0 // 1A typical for small motor
	case strings.Contains(componentType, "relay"):
		return 70.0 // 70mA typical for relay coil
	case strings.Contains(componentType, "sensor"):
		return 5.0 // 5mA typical for sensor
	case strings.Contains(componentType, "display"):
		return 50.0 // 50mA typical for small display
	}

	return 0.0
}

func (sv *SafetyValidator) findComponent(components []Component, id string) *Component {
	for i := range components {
		if components[i].ID == id {
			return &components[i]
		}
	}
	return nil
}

func (sv *SafetyValidator) getRequiredPinType(component *Component) string {
	componentType := strings.ToLower(component.Type)

	switch {
	case strings.Contains(componentType, "servo"):
		return "pwm"
	case strings.Contains(componentType, "analog") || strings.Contains(componentType, "potentiometer"):
		return "analog"
	case strings.Contains(componentType, "led") || strings.Contains(componentType, "relay"):
		return "digital"
	}

	return "digital" // Default to digital
}

func (sv *SafetyValidator) isPinCompatible(pin, requiredType string, boardSpec *BoardSpecification) bool {
	pin = strings.ToUpper(pin)

	switch requiredType {
	case "analog":
		return sv.contains(boardSpec.AnalogPins, pin)
	case "pwm":
		return sv.contains(boardSpec.PWMPins, pin)
	case "digital":
		return sv.contains(boardSpec.DigitalPins, pin) || sv.contains(boardSpec.PWMPins, pin)
	}

	return true
}

func (sv *SafetyValidator) getPinType(pin string, boardSpec *BoardSpecification) string {
	pin = strings.ToUpper(pin)

	if sv.contains(boardSpec.AnalogPins, pin) {
		return "analog"
	}
	if sv.contains(boardSpec.PWMPins, pin) {
		return "pwm"
	}
	if sv.contains(boardSpec.DigitalPins, pin) {
		return "digital"
	}

	return "unknown"
}

func (sv *SafetyValidator) isPowerPin(pin string) bool {
	pin = strings.ToUpper(pin)
	return pin == "VCC" || pin == "5V" || pin == "3.3V" || pin == "3V3" || pin == "GND" || pin == "GROUND"
}

func (sv *SafetyValidator) hasMultipleI2CDevices(components []Component) bool {
	count := 0
	for _, component := range components {
		if strings.Contains(strings.ToLower(component.Type), "i2c") {
			count++
		}
	}
	return count > 1
}

func (sv *SafetyValidator) hasMultipleSPIDevices(components []Component) bool {
	count := 0
	for _, component := range components {
		if strings.Contains(strings.ToLower(component.Type), "spi") {
			count++
		}
	}
	return count > 1
}

func (sv *SafetyValidator) contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

func (sv *SafetyValidator) getGenericBoardSpec() *BoardSpecification {
	return &BoardSpecification{
		Name:              "Generic Arduino",
		OperatingVoltage:  "5V",
		IOVoltage:         "5V",
		MaxCurrentPerPin:  40.0,
		MaxTotalCurrent:   200.0,
		DigitalPins:       []string{"D0", "D1", "D2", "D3", "D4", "D5", "D6", "D7", "D8", "D9", "D10", "D11", "D12", "D13"},
		AnalogPins:        []string{"A0", "A1", "A2", "A3", "A4", "A5"},
		PWMPins:           []string{"D3", "D5", "D6", "D9", "D10", "D11"},
		PowerPins:         map[string]string{"5V": "5V", "3.3V": "3.3V", "GND": "0V"},
	}
}

// initializeBoardSpecs initializes board specifications database
func initializeBoardSpecs() map[string]*BoardSpecification {
	specs := make(map[string]*BoardSpecification)

	// Arduino Uno
	specs["uno"] = &BoardSpecification{
		Name:              "Arduino Uno",
		OperatingVoltage:  "5V",
		IOVoltage:         "5V",
		MaxCurrentPerPin:  40.0,
		MaxTotalCurrent:   200.0,
		DigitalPins:       []string{"D0", "D1", "D2", "D3", "D4", "D5", "D6", "D7", "D8", "D9", "D10", "D11", "D12", "D13"},
		AnalogPins:        []string{"A0", "A1", "A2", "A3", "A4", "A5"},
		PWMPins:           []string{"D3", "D5", "D6", "D9", "D10", "D11"},
		PowerPins:         map[string]string{"5V": "5V", "3.3V": "3.3V", "GND": "0V"},
	}

	// Arduino Nano
	specs["nano"] = specs["uno"] // Similar to Uno

	// ESP32
	specs["esp32"] = &BoardSpecification{
		Name:              "ESP32",
		OperatingVoltage:  "3.3V",
		IOVoltage:         "3.3V",
		MaxCurrentPerPin:  40.0,
		MaxTotalCurrent:   500.0,
		DigitalPins:       []string{"D0", "D1", "D2", "D3", "D4", "D5", "D12", "D13", "D14", "D15", "D16", "D17", "D18", "D19", "D21", "D22", "D23", "D25", "D26", "D27", "D32", "D33"},
		AnalogPins:        []string{"A0", "A3", "A4", "A5", "A6", "A7", "A10", "A11", "A12", "A13", "A14", "A15"},
		PWMPins:           []string{"D2", "D4", "D5", "D12", "D13", "D14", "D15", "D16", "D17", "D18", "D19", "D21", "D22", "D23", "D25", "D26", "D27", "D32", "D33"},
		PowerPins:         map[string]string{"3.3V": "3.3V", "GND": "0V"},
	}

	// ESP8266
	specs["esp8266"] = &BoardSpecification{
		Name:              "ESP8266",
		OperatingVoltage:  "3.3V",
		IOVoltage:         "3.3V",
		MaxCurrentPerPin:  12.0,
		MaxTotalCurrent:   200.0,
		DigitalPins:       []string{"D0", "D1", "D2", "D3", "D4", "D5", "D6", "D7", "D8"},
		AnalogPins:        []string{"A0"},
		PWMPins:           []string{"D1", "D2", "D3", "D4", "D5", "D6", "D7", "D8"},
		PowerPins:         map[string]string{"3.3V": "3.3V", "GND": "0V"},
	}

	return specs
}
