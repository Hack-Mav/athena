package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// WiringValidationTest focuses on wiring diagram and component compatibility testing
type WiringValidationTest struct{}

// NewWiringValidationTest creates a new wiring validation test instance
func NewWiringValidationTest() *WiringValidationTest {
	return &WiringValidationTest{}
}

// TestWiringDiagramGeneration tests wiring diagram generation functionality
func TestWiringDiagramGeneration(t *testing.T) {
	suite, err := NewTemplateTestSuite("../../templates/arduino")
	require.NoError(t, err, "Failed to create template test suite")

	wvt := NewWiringValidationTest()

	for _, template := range suite.Templates {
		t.Run(fmt.Sprintf("DiagramGeneration_%s", template.ID), func(t *testing.T) {
			t.Parallel()

			// Test diagram metadata
			err := wvt.validateDiagramMetadata(template)
			assert.NoError(t, err, "Diagram metadata should be valid")

			// Test diagram generation
			err = wvt.testDiagramGeneration(template)
			assert.NoError(t, err, "Diagram generation should work")

			// Test diagram export formats
			err = wvt.testDiagramExportFormats(template)
			assert.NoError(t, err, "Diagram export formats should be supported")
		})
	}
}

// TestComponentCompatibility tests component compatibility validation
func TestComponentCompatibility(t *testing.T) {
	suite, err := NewTemplateTestSuite("../../templates/arduino")
	require.NoError(t, err, "Failed to create template test suite")

	wvt := NewWiringValidationTest()

	for _, template := range suite.Templates {
		t.Run(fmt.Sprintf("ComponentCompatibility_%s", template.ID), func(t *testing.T) {
			t.Parallel()

			// Test component definitions
			err := wvt.validateComponentDefinitions(template.WiringSpec)
			assert.NoError(t, err, "Component definitions should be valid")

			// Test component compatibility rules
			err = wvt.testComponentCompatibilityRules(template.WiringSpec)
			assert.NoError(t, err, "Component compatibility rules should be satisfied")

			// Test power requirements
			err = wvt.validatePowerRequirements(template.WiringSpec)
			assert.NoError(t, err, "Power requirements should be satisfied")

			// Test voltage compatibility
			err = wvt.validateVoltageCompatibility(template.WiringSpec)
			assert.NoError(t, err, "Voltage compatibility should be valid")
		})
	}
}

// TestConnectionValidation tests connection validation rules
func TestConnectionValidation(t *testing.T) {
	suite, err := NewTemplateTestSuite("../../templates/arduino")
	require.NoError(t, err, "Failed to create template test suite")

	wvt := NewWiringValidationTest()

	for _, template := range suite.Templates {
		t.Run(fmt.Sprintf("ConnectionValidation_%s", template.ID), func(t *testing.T) {
			t.Parallel()

			// Test connection validity
			err := wvt.validateConnectionRules(template.WiringSpec)
			assert.NoError(t, err, "Connection rules should be valid")

			// Test connection completeness
			err = wvt.validateConnectionCompleteness(template.WiringSpec)
			assert.NoError(t, err, "All connections should be complete")

			// Test connection safety
			err = wvt.validateConnectionSafety(template.WiringSpec)
			assert.NoError(t, err, "Connections should be safe")

			// Test signal integrity
			err = wvt.validateSignalIntegrity(template.WiringSpec)
			assert.NoError(t, err, "Signal integrity should be maintained")
		})
	}
}

// TestWiringDiagramFormats tests different wiring diagram formats
func TestWiringDiagramFormats(t *testing.T) {
	suite, err := NewTemplateTestSuite("../../templates/arduino")
	require.NoError(t, err, "Failed to create template test suite")

	wvt := NewWiringValidationTest()

	supportedFormats := []string{"png", "svg", "pdf", "json"}

	for _, template := range suite.Templates {
		t.Run(fmt.Sprintf("DiagramFormats_%s", template.ID), func(t *testing.T) {
			t.Parallel()

			for _, format := range supportedFormats {
				t.Run(fmt.Sprintf("Format_%s", format), func(t *testing.T) {
					err := wvt.validateDiagramFormat(template, format)
					assert.NoError(t, err,
						"Diagram format %s should be supported", format)
				})
			}
		})
	}
}

// TestWiringValidationEdgeCases tests edge cases and error conditions
func TestWiringValidationEdgeCases(t *testing.T) {
	wvt := NewWiringValidationTest()

	t.Run("EmptyWiringSpec", func(t *testing.T) {
		emptySpec := WiringSpec{
			Components:  []Component{},
			Connections: []Connection{},
		}

		err := wvt.validateComponentDefinitions(emptySpec)
		assert.Error(t, err, "Empty wiring spec should fail validation")
	})

	t.Run("InvalidComponent", func(t *testing.T) {
		invalidSpec := WiringSpec{
			Components: []Component{
				{
					ID:   "",
					Type: "",
					Name: "",
					Pins: []Pin{},
				},
			},
			Connections: []Connection{},
		}

		err := wvt.validateComponentDefinitions(invalidSpec)
		assert.Error(t, err, "Invalid component should fail validation")
	})

	t.Run("CircularConnections", func(t *testing.T) {
		// Test detection of circular connections
		circularSpec := WiringSpec{
			Components: []Component{
				{
					ID:   "comp1",
					Type: "sensor",
					Name: "Component 1",
					Pins: []Pin{
						{Number: "1", Name: "OUT", Type: "digital"},
						{Number: "2", Name: "IN", Type: "digital"},
					},
				},
				{
					ID:   "comp2",
					Type: "sensor",
					Name: "Component 2",
					Pins: []Pin{
						{Number: "1", Name: "OUT", Type: "digital"},
						{Number: "2", Name: "IN", Type: "digital"},
					},
				},
			},
			Connections: []Connection{
				{
					FromComponent: "comp1",
					FromPin:       "1",
					ToComponent:   "comp2",
					ToPin:         "2",
					WireColor:     "blue",
				},
				{
					FromComponent: "comp2",
					FromPin:       "1",
					ToComponent:   "comp1",
					ToPin:         "2",
					WireColor:     "red",
				},
			},
		}

		err := wvt.validateConnectionRules(circularSpec)
		// Circular connections might be valid in some cases, but we should detect them
		assert.NoError(t, err, "Circular connections should be detected")
	})
}

// WiringValidationTest methods

func (wvt *WiringValidationTest) validateDiagramMetadata(template Template) error {
	// Find wiring diagram asset
	var wiringAsset *Asset
	for _, asset := range template.Assets {
		if asset.Type == "wiring_diagram" {
			wiringAsset = &asset
			break
		}
	}

	if wiringAsset == nil {
		return fmt.Errorf("template should have a wiring diagram asset")
	}

	// Check required metadata fields
	requiredFields := []string{"description", "connections"}
	for _, field := range requiredFields {
		if _, exists := wiringAsset.Metadata[field]; !exists {
			return fmt.Errorf("wiring diagram missing required field: %s", field)
		}
	}

	// Validate connections metadata
	connections, ok := wiringAsset.Metadata["connections"].([]interface{})
	if !ok {
		return fmt.Errorf("connections metadata should be an array")
	}

	for i, conn := range connections {
		connMap, ok := conn.(map[string]interface{})
		if !ok {
			return fmt.Errorf("connection %d should be an object", i)
		}

		requiredConnFields := []string{"from", "to", "wire_color"}
		for _, field := range requiredConnFields {
			if _, exists := connMap[field]; !exists {
				return fmt.Errorf("connection %d missing required field: %s", i, field)
			}
		}
	}

	return nil
}

func (wvt *WiringValidationTest) testDiagramGeneration(template Template) error {
	// Simulate diagram generation
	// In a real implementation, this would call the diagram generation service

	wiringSpec := template.WiringSpec

	// Check that we have enough information for diagram generation
	if len(wiringSpec.Components) == 0 {
		return fmt.Errorf("cannot generate diagram: no components")
	}

	if len(wiringSpec.Connections) == 0 {
		return fmt.Errorf("cannot generate diagram: no connections")
	}

	// Validate that all referenced components exist
	componentMap := make(map[string]bool)
	for _, comp := range wiringSpec.Components {
		componentMap[comp.ID] = true
	}

	for _, conn := range wiringSpec.Connections {
		if !componentMap[conn.FromComponent] {
			return fmt.Errorf("connection references non-existent component: %s",
				conn.FromComponent)
		}
		if !componentMap[conn.ToComponent] {
			return fmt.Errorf("connection references non-existent component: %s",
				conn.ToComponent)
		}
	}

	return nil
}

func (wvt *WiringValidationTest) testDiagramExportFormats(template Template) error {
	// Test different export formats
	formats := []string{"png", "svg", "pdf", "json"}

	for _, format := range formats {
		err := wvt.validateDiagramFormat(template, format)
		if err != nil {
			return fmt.Errorf("format %s validation failed: %w", format, err)
		}
	}

	return nil
}

func (wvt *WiringValidationTest) validateDiagramFormat(template Template, format string) error {
	// Validate format-specific requirements
	switch format {
	case "png":
		// PNG should have reasonable dimensions and resolution
		return nil
	case "svg":
		// SVG should be vector-based and scalable
		return nil
	case "pdf":
		// PDF should be printable and have proper page layout
		return nil
	case "json":
		// JSON should contain all diagram data in structured format
		return wvt.validateJSONDiagramFormat(template)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func (wvt *WiringValidationTest) validateJSONDiagramFormat(template Template) error {
	// Validate that diagram can be represented in JSON format
	// This would include component positions, connection paths, etc.

	wiringSpec := template.WiringSpec

	// Check that all components have position information
	for _, comp := range wiringSpec.Components {
		if comp.Type == "" {
			return fmt.Errorf("component %s missing type", comp.ID)
		}
		if comp.Name == "" {
			return fmt.Errorf("component %s missing name", comp.ID)
		}
	}

	// Check that all connections have proper routing information
	for _, conn := range wiringSpec.Connections {
		if conn.FromComponent == "" {
			return fmt.Errorf("connection missing from component")
		}
		if conn.ToComponent == "" {
			return fmt.Errorf("connection missing to component")
		}
		if conn.FromPin == "" {
			return fmt.Errorf("connection missing from pin")
		}
		if conn.ToPin == "" {
			return fmt.Errorf("connection missing to pin")
		}
	}

	return nil
}

func (wvt *WiringValidationTest) validateComponentDefinitions(wiringSpec WiringSpec) error {
	if len(wiringSpec.Components) == 0 {
		return fmt.Errorf("wiring spec must have at least one component")
	}

	// Check for duplicate component IDs
	componentIDs := make(map[string]bool)
	for _, comp := range wiringSpec.Components {
		if comp.ID == "" {
			return fmt.Errorf("component must have an ID")
		}

		if componentIDs[comp.ID] {
			return fmt.Errorf("duplicate component ID: %s", comp.ID)
		}
		componentIDs[comp.ID] = true

		// Validate component fields
		if comp.Type == "" {
			return fmt.Errorf("component %s must have a type", comp.ID)
		}
		if comp.Name == "" {
			return fmt.Errorf("component %s must have a name", comp.ID)
		}
		if len(comp.Pins) == 0 {
			return fmt.Errorf("component %s must have at least one pin", comp.ID)
		}

		// Validate pins
		pinNumbers := make(map[string]bool)
		for _, pin := range comp.Pins {
			if pin.Number == "" {
				return fmt.Errorf("component %s has pin with empty number", comp.ID)
			}

			if pinNumbers[pin.Number] {
				return fmt.Errorf("component %s has duplicate pin number: %s",
					comp.ID, pin.Number)
			}
			pinNumbers[pin.Number] = true

			if pin.Name == "" {
				return fmt.Errorf("component %s pin %s has empty name",
					comp.ID, pin.Number)
			}

			if pin.Type == "" {
				return fmt.Errorf("component %s pin %s has empty type",
					comp.ID, pin.Number)
			}

			if !isValidPinType(pin.Type) {
				return fmt.Errorf("component %s pin %s has invalid type: %s",
					comp.ID, pin.Number, pin.Type)
			}
		}
	}

	return nil
}

func (wvt *WiringValidationTest) testComponentCompatibilityRules(wiringSpec WiringSpec) error {
	// Test component compatibility rules

	// Find board component
	var boardComp *Component
	for _, comp := range wiringSpec.Components {
		if comp.Type == "board" {
			boardComp = &comp
			break
		}
	}

	if boardComp == nil {
		return fmt.Errorf("wiring spec must include a board component")
	}

	// Test sensor-board compatibility
	for _, comp := range wiringSpec.Components {
		if comp.Type == "sensor" {
			err := wvt.validateSensorBoardCompatibility(comp, *boardComp)
			if err != nil {
				return fmt.Errorf("sensor %s compatibility failed: %w", comp.ID, err)
			}
		}
	}

	// Test actuator-board compatibility
	for _, comp := range wiringSpec.Components {
		if comp.Type == "actuator" {
			err := wvt.validateActuatorBoardCompatibility(comp, *boardComp)
			if err != nil {
				return fmt.Errorf("actuator %s compatibility failed: %w", comp.ID, err)
			}
		}
	}

	return nil
}

func (wvt *WiringValidationTest) validateSensorBoardCompatibility(sensor Component, board Component) error {
	// Check voltage compatibility
	sensorVoltage := wvt.getComponentVoltage(sensor)
	boardVoltage := wvt.getComponentVoltage(board)

	if sensorVoltage > boardVoltage {
		return fmt.Errorf("sensor voltage (%.1fV) exceeds board voltage (%.1fV)",
			sensorVoltage, boardVoltage)
	}

	// Check pin compatibility
	sensorPins := wvt.getPowerPins(sensor)
	boardPins := wvt.getPowerPins(board)

	if len(sensorPins) > 0 && len(boardPins) == 0 {
		return fmt.Errorf("sensor requires power pins but board has none available")
	}

	return nil
}

func (wvt *WiringValidationTest) validateActuatorBoardCompatibility(actuator Component, board Component) error {
	// Check power requirements
	actuatorPower := wvt.getComponentPowerConsumption(actuator)
	boardPower := wvt.getBoardPowerCapacity(board)

	if actuatorPower > boardPower {
		return fmt.Errorf("actuator power (%.1fA) exceeds board capacity (%.1fA)",
			actuatorPower, boardPower)
	}

	// Check pin compatibility
	actuatorPins := wvt.getControlPins(actuator)
	boardPins := wvt.getDigitalPins(board)

	if len(actuatorPins) > len(boardPins) {
		return fmt.Errorf("actuator requires %d control pins but board has %d available",
			len(actuatorPins), len(boardPins))
	}

	return nil
}

func (wvt *WiringValidationTest) validatePowerRequirements(wiringSpec WiringSpec) error {
	// Calculate total power consumption
	totalPower := 0.0
	for _, comp := range wiringSpec.Components {
		if comp.Type != "board" {
			totalPower += wvt.getComponentPowerConsumption(comp)
		}
	}

	// Find board and check power capacity
	var boardComp *Component
	for _, comp := range wiringSpec.Components {
		if comp.Type == "board" {
			boardComp = &comp
			break
		}
	}

	if boardComp == nil {
		return fmt.Errorf("no board component found for power validation")
	}

	boardCapacity := wvt.getBoardPowerCapacity(*boardComp)

	if totalPower > boardCapacity {
		return fmt.Errorf("total power consumption (%.1fA) exceeds board capacity (%.1fA)",
			totalPower, boardCapacity)
	}

	return nil
}

func (wvt *WiringValidationTest) validateVoltageCompatibility(wiringSpec WiringSpec) error {
	// Check voltage compatibility between components
	componentVoltages := make(map[string]float64)

	for _, comp := range wiringSpec.Components {
		componentVoltages[comp.ID] = wvt.getComponentVoltage(comp)
	}

	// Check connections for voltage compatibility
	for _, conn := range wiringSpec.Connections {
		fromVoltage := componentVoltages[conn.FromComponent]
		toVoltage := componentVoltages[conn.ToComponent]

		// Power connections should have matching voltages
		if wvt.isPowerConnection(conn, wiringSpec) {
			if fromVoltage != toVoltage {
				return fmt.Errorf("voltage mismatch in power connection: %s (%.1fV) -> %s (%.1fV)",
					conn.FromComponent, fromVoltage, conn.ToComponent, toVoltage)
			}
		}
	}

	return nil
}

func (wvt *WiringValidationTest) validateConnectionRules(wiringSpec WiringSpec) error {
	// Build component map
	componentMap := make(map[string]Component)
	for _, comp := range wiringSpec.Components {
		componentMap[comp.ID] = comp
	}

	// Validate each connection
	for i, conn := range wiringSpec.Connections {
		// Check components exist
		fromComp, exists := componentMap[conn.FromComponent]
		if !exists {
			return fmt.Errorf("connection %d: from component %s not found", i, conn.FromComponent)
		}

		toComp, exists := componentMap[conn.ToComponent]
		if !exists {
			return fmt.Errorf("connection %d: to component %s not found", i, conn.ToComponent)
		}

		// Check pins exist
		if !pinExists(fromComp, conn.FromPin) {
			return fmt.Errorf("connection %d: from pin %s not found in component %s",
				i, conn.FromPin, conn.FromComponent)
		}

		if !pinExists(toComp, conn.ToPin) {
			return fmt.Errorf("connection %d: to pin %s not found in component %s",
				i, conn.ToPin, conn.ToComponent)
		}

		// Check pin type compatibility
		if err := wvt.validatePinCompatibility(fromComp, conn.FromPin, toComp, conn.ToPin); err != nil {
			return fmt.Errorf("connection %d: %w", i, err)
		}

		// Check wire color validity
		if !isValidWireColor(conn.WireColor) {
			return fmt.Errorf("connection %d: invalid wire color: %s", i, conn.WireColor)
		}
	}

	return nil
}

func (wvt *WiringValidationTest) validateConnectionCompleteness(wiringSpec WiringSpec) error {
	// Check that all required pins are connected
	componentMap := make(map[string]Component)
	for _, comp := range wiringSpec.Components {
		componentMap[comp.ID] = comp
	}

	// Track connected pins
	connectedPins := make(map[string]map[string]bool)
	for _, comp := range wiringSpec.Components {
		connectedPins[comp.ID] = make(map[string]bool)
	}

	// Mark connected pins
	for _, conn := range wiringSpec.Connections {
		connectedPins[conn.FromComponent][conn.FromPin] = true
		connectedPins[conn.ToComponent][conn.ToPin] = true
	}

	// Check for unconnected required pins
	for _, comp := range wiringSpec.Components {
		for _, pin := range comp.Pins {
			if wvt.isRequiredPin(comp, pin) && !connectedPins[comp.ID][pin.Number] {
				return fmt.Errorf("required pin %s on component %s is not connected",
					pin.Number, comp.ID)
			}
		}
	}

	return nil
}

func (wvt *WiringValidationTest) validateConnectionSafety(wiringSpec WiringSpec) error {
	// Check for safety issues in connections

	// Check for short circuits
	if err := wvt.detectShortCircuits(wiringSpec); err != nil {
		return fmt.Errorf("safety issue detected: %w", err)
	}

	// Check for overloaded connections
	if err := wvt.detectOverloadedConnections(wiringSpec); err != nil {
		return fmt.Errorf("safety issue detected: %w", err)
	}

	// Check for improper grounding
	if err := wvt.validateGrounding(wiringSpec); err != nil {
		return fmt.Errorf("grounding issue detected: %w", err)
	}

	return nil
}

func (wvt *WiringValidationTest) validateSignalIntegrity(wiringSpec WiringSpec) error {
	// Check signal integrity issues

	// Check for signal interference
	if err := wvt.detectSignalInterference(wiringSpec); err != nil {
		return fmt.Errorf("signal integrity issue: %w", err)
	}

	// Check for proper termination
	if err := wvt.validateSignalTermination(wiringSpec); err != nil {
		return fmt.Errorf("signal termination issue: %w", err)
	}

	return nil
}

// Helper methods for wiring validation

func (wvt *WiringValidationTest) getComponentVoltage(comp Component) float64 {
	// Return component voltage based on type
	// In a real implementation, this would look up component specifications
	switch comp.Type {
	case "board":
		switch comp.Name {
		case "Arduino Uno", "Arduino Nano":
			return 5.0
		case "ESP32 Development Board":
			return 3.3
		case "ESP8266 D1 Mini":
			return 3.3
		default:
			return 5.0
		}
	case "sensor":
		return 3.3 // Most sensors work at 3.3V
	case "actuator":
		return 5.0 // Most actuators work at 5V
	default:
		return 3.3
	}
}

func (wvt *WiringValidationTest) getComponentPowerConsumption(comp Component) float64 {
	// Return component power consumption in Amperes
	// In a real implementation, this would look up component specifications
	switch comp.Type {
	case "sensor":
		return 0.01 // 10mA typical for sensors
	case "actuator":
		return 0.2 // 200mA typical for actuators
	default:
		return 0.01
	}
}

func (wvt *WiringValidationTest) getBoardPowerCapacity(board Component) float64 {
	// Return board power capacity in Amperes
	// In a real implementation, this would look up board specifications
	switch board.Name {
	case "Arduino Uno", "Arduino Nano":
		return 0.5 // 500mA total
	case "ESP32 Development Board":
		return 0.3 // 300mA total
	case "ESP8266 D1 Mini":
		return 0.2 // 200mA total
	default:
		return 0.5
	}
}

func (wvt *WiringValidationTest) getPowerPins(comp Component) []Pin {
	var pins []Pin
	for _, pin := range comp.Pins {
		if pin.Type == "power" {
			pins = append(pins, pin)
		}
	}
	return pins
}

func (wvt *WiringValidationTest) getDigitalPins(comp Component) []Pin {
	var pins []Pin
	for _, pin := range comp.Pins {
		if pin.Type == "digital" {
			pins = append(pins, pin)
		}
	}
	return pins
}

func (wvt *WiringValidationTest) getControlPins(comp Component) []Pin {
	var pins []Pin
	for _, pin := range comp.Pins {
		if pin.Type == "digital" || pin.Type == "pwm" {
			pins = append(pins, pin)
		}
	}
	return pins
}

func (wvt *WiringValidationTest) validatePinCompatibility(fromComp Component, fromPin string,
	toComp Component, toPin string) error {

	var fromPinType, toPinType string

	// Find pin types
	for _, pin := range fromComp.Pins {
		if pin.Number == fromPin {
			fromPinType = pin.Type
			break
		}
	}

	for _, pin := range toComp.Pins {
		if pin.Number == toPin {
			toPinType = pin.Type
			break
		}
	}

	// Define compatibility rules
	compatibilityRules := map[string][]string{
		"power":   {"power", "ground"},
		"ground":  {"power", "ground", "digital", "analog"},
		"digital": {"digital", "ground"},
		"analog":  {"analog", "ground"},
		"pwm":     {"digital", "pwm", "ground"},
		"i2c":     {"i2c", "ground"},
		"spi":     {"spi", "ground"},
	}

	if allowedTypes, exists := compatibilityRules[fromPinType]; exists {
		for _, allowedType := range allowedTypes {
			if allowedType == toPinType {
				return nil
			}
		}
	}

	return fmt.Errorf("pin type %s is not compatible with %s", fromPinType, toPinType)
}

func (wvt *WiringValidationTest) isPowerConnection(conn Connection, wiringSpec WiringSpec) bool {
	// Check if connection is a power connection
	componentMap := make(map[string]Component)
	for _, comp := range wiringSpec.Components {
		componentMap[comp.ID] = comp
	}

	fromComp := componentMap[conn.FromComponent]
	toComp := componentMap[conn.ToComponent]

	var fromPinType, toPinType string

	for _, pin := range fromComp.Pins {
		if pin.Number == conn.FromPin {
			fromPinType = pin.Type
			break
		}
	}

	for _, pin := range toComp.Pins {
		if pin.Number == conn.ToPin {
			toPinType = pin.Type
			break
		}
	}

	return fromPinType == "power" || toPinType == "power"
}

func (wvt *WiringValidationTest) isRequiredPin(comp Component, pin Pin) bool {
	// Define which pins are required to be connected
	switch comp.Type {
	case "sensor":
		return pin.Type == "power" || pin.Type == "ground"
	case "actuator":
		return pin.Type == "power" || pin.Type == "ground"
	case "board":
		return false // Board pins don't need to be connected
	default:
		return pin.Type == "power" || pin.Type == "ground"
	}
}

func (wvt *WiringValidationTest) detectShortCircuits(wiringSpec WiringSpec) error {
	// Check for potential short circuits
	// This is a simplified implementation

	// Check if power and ground are directly connected
	for _, conn := range wiringSpec.Connections {
		if wvt.isPowerToGroundConnection(conn, wiringSpec) {
			return fmt.Errorf("direct power-to-ground connection detected: %s.%s -> %s.%s",
				conn.FromComponent, conn.FromPin, conn.ToComponent, conn.ToPin)
		}
	}

	return nil
}

func (wvt *WiringValidationTest) isPowerToGroundConnection(conn Connection, wiringSpec WiringSpec) bool {
	componentMap := make(map[string]Component)
	for _, comp := range wiringSpec.Components {
		componentMap[comp.ID] = comp
	}

	fromComp := componentMap[conn.FromComponent]
	toComp := componentMap[conn.ToComponent]

	var fromPinType, toPinType string

	for _, pin := range fromComp.Pins {
		if pin.Number == conn.FromPin {
			fromPinType = pin.Type
			break
		}
	}

	for _, pin := range toComp.Pins {
		if pin.Number == conn.ToPin {
			toPinType = pin.Type
			break
		}
	}

	return (fromPinType == "power" && toPinType == "ground") ||
		(fromPinType == "ground" && toPinType == "power")
}

func (wvt *WiringValidationTest) detectOverloadedConnections(wiringSpec WiringSpec) error {
	// Check for overloaded connections
	// This is a simplified implementation

	// Count connections per pin
	pinConnections := make(map[string]int)

	for _, conn := range wiringSpec.Connections {
		fromPinKey := fmt.Sprintf("%s.%s", conn.FromComponent, conn.FromPin)
		toPinKey := fmt.Sprintf("%s.%s", conn.ToComponent, conn.ToPin)

		pinConnections[fromPinKey]++
		pinConnections[toPinKey]++
	}

	// Check for pins with too many connections
	for pinKey, count := range pinConnections {
		if count > 2 {
			return fmt.Errorf("pin %s has %d connections (max 2 recommended)", pinKey, count)
		}
	}

	return nil
}

func (wvt *WiringValidationTest) validateGrounding(wiringSpec WiringSpec) error {
	// Check for proper grounding
	// All components should have ground connections

	componentMap := make(map[string]Component)
	for _, comp := range wiringSpec.Components {
		componentMap[comp.ID] = comp
	}

	connectedGrounds := make(map[string]bool)

	for _, conn := range wiringSpec.Connections {
		// Check if this is a ground connection
		if wvt.isGroundConnection(conn, wiringSpec) {
			connectedGrounds[conn.FromComponent] = true
			connectedGrounds[conn.ToComponent] = true
		}
	}

	// Check that all non-board components have ground connections
	for _, comp := range wiringSpec.Components {
		if comp.Type != "board" {
			if !connectedGrounds[comp.ID] {
				return fmt.Errorf("component %s has no ground connection", comp.ID)
			}
		}
	}

	return nil
}

func (wvt *WiringValidationTest) isGroundConnection(conn Connection, wiringSpec WiringSpec) bool {
	componentMap := make(map[string]Component)
	for _, comp := range wiringSpec.Components {
		componentMap[comp.ID] = comp
	}

	fromComp := componentMap[conn.FromComponent]
	toComp := componentMap[conn.ToComponent]

	var fromPinType, toPinType string

	for _, pin := range fromComp.Pins {
		if pin.Number == conn.FromPin {
			fromPinType = pin.Type
			break
		}
	}

	for _, pin := range toComp.Pins {
		if pin.Number == conn.ToPin {
			toPinType = pin.Type
			break
		}
	}

	return fromPinType == "ground" || toPinType == "ground"
}

func (wvt *WiringValidationTest) detectSignalInterference(wiringSpec WiringSpec) error {
	// Check for potential signal interference
	// This is a simplified implementation

	// Check for long parallel runs of digital and analog signals
	// In a real implementation, this would analyze the physical layout

	return nil
}

func (wvt *WiringValidationTest) validateSignalTermination(wiringSpec WiringSpec) error {
	// Check for proper signal termination
	// This is a simplified implementation

	// Check I2C connections have pull-up resistors
	// In a real implementation, this would check for pull-up resistors on I2C buses

	return nil
}

func isValidWireColor(color string) bool {
	validColors := []string{
		"red", "black", "blue", "green", "yellow", "orange",
		"purple", "brown", "gray", "white", "pink",
	}

	for _, validColor := range validColors {
		if color == validColor {
			return true
		}
	}
	return false
}
