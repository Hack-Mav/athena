package template

import (
	"fmt"
	"strings"
)

// WiringDiagramGenerator generates Mermaid syntax wiring diagrams
type WiringDiagramGenerator struct{}

// NewWiringDiagramGenerator creates a new wiring diagram generator
func NewWiringDiagramGenerator() *WiringDiagramGenerator {
	return &WiringDiagramGenerator{}
}

// GenerateWiringDiagram generates a wiring diagram from template and parameters
func (wdg *WiringDiagramGenerator) GenerateWiringDiagram(template *Template, parameters map[string]interface{}) (*WiringDiagram, error) {
	// Extract components and connections from template and parameters
	components := wdg.extractComponents(template, parameters)
	connections := wdg.extractConnections(template, parameters, components)
	
	// Generate Mermaid syntax
	mermaidSyntax := wdg.generateMermaidSyntax(components, connections)
	
	return &WiringDiagram{
		MermaidSyntax: mermaidSyntax,
		Components:    components,
		Connections:   connections,
		Metadata: map[string]interface{}{
			"template_id":      template.ID,
			"template_version": template.Version,
			"generated_from":   "parameters",
		},
	}, nil
}

// extractComponents extracts hardware components from template and parameters
func (wdg *WiringDiagramGenerator) extractComponents(template *Template, parameters map[string]interface{}) []Component {
	var components []Component
	
	// Always include the Arduino board
	arduino := wdg.createArduinoComponent(template.BoardsSupported)
	components = append(components, arduino)
	
	// Extract components based on template category and parameters
	switch strings.ToLower(template.Category) {
	case "sensing":
		components = append(components, wdg.extractSensorComponents(template, parameters)...)
	case "automation":
		components = append(components, wdg.extractAutomationComponents(template, parameters)...)
	case "display":
		components = append(components, wdg.extractDisplayComponents(template, parameters)...)
	case "communication":
		components = append(components, wdg.extractCommunicationComponents(template, parameters)...)
	default:
		components = append(components, wdg.extractGenericComponents(template, parameters)...)
	}
	
	return components
}

// extractConnections extracts wiring connections from parameters
func (wdg *WiringDiagramGenerator) extractConnections(template *Template, parameters map[string]interface{}, components []Component) []Connection {
	var connections []Connection
	
	// Find Arduino component
	var arduinoComponent *Component
	for i := range components {
		if components[i].Type == "microcontroller" {
			arduinoComponent = &components[i]
			break
		}
	}
	
	if arduinoComponent == nil {
		return connections
	}
	
	// Extract pin connections from parameters
	for paramName, paramValue := range parameters {
		if strings.Contains(strings.ToLower(paramName), "pin") {
			pinStr := fmt.Sprintf("%v", paramValue)
			
			// Find the component this pin connects to
			componentName := wdg.inferComponentFromParameter(paramName, template)
			if componentName != "" {
				connection := Connection{
					FromComponent: arduinoComponent.ID,
					FromPin:       pinStr,
					ToComponent:   componentName,
					ToPin:         wdg.inferComponentPin(paramName, componentName),
					WireColor:     wdg.getWireColor(paramName, pinStr),
				}
				connections = append(connections, connection)
			}
		}
	}
	
	// Add power and ground connections
	connections = append(connections, wdg.generatePowerConnections(components)...)
	
	return connections
}

// generateMermaidSyntax generates Mermaid diagram syntax
func (wdg *WiringDiagramGenerator) generateMermaidSyntax(components []Component, connections []Connection) string {
	var builder strings.Builder
	
	// Start with graph definition
	builder.WriteString("graph TD\n")
	
	// Add component definitions
	for _, component := range components {
		shape := wdg.getMermaidShape(component.Type)
		builder.WriteString(fmt.Sprintf("    %s%s%s\n", 
			component.ID, 
			shape[0], 
			component.Name,
		))
		if len(shape) > 1 {
			builder.WriteString(shape[1])
		}
		builder.WriteString("\n")
	}
	
	// Add connections
	for _, connection := range connections {
		label := ""
		if connection.WireColor != "" {
			label = fmt.Sprintf("|%s wire|", connection.WireColor)
		}
		
		builder.WriteString(fmt.Sprintf("    %s -->%s %s\n", 
			connection.FromComponent, 
			label,
			connection.ToComponent,
		))
	}
	
	// Add styling
	builder.WriteString(wdg.generateMermaidStyling(components))
	
	return builder.String()
}

// Helper methods

// createArduinoComponent creates the Arduino microcontroller component
func (wdg *WiringDiagramGenerator) createArduinoComponent(supportedBoards []string) Component {
	boardName := "Arduino Uno" // Default
	if len(supportedBoards) > 0 {
		boardName = wdg.getBoardDisplayName(supportedBoards[0])
	}
	
	pins := wdg.getArduinoPins(supportedBoards)
	
	return Component{
		ID:   "arduino",
		Type: "microcontroller",
		Name: boardName,
		Pins: pins,
		Metadata: map[string]interface{}{
			"supported_boards": supportedBoards,
		},
	}
}

// extractSensorComponents extracts sensor components
func (wdg *WiringDiagramGenerator) extractSensorComponents(template *Template, parameters map[string]interface{}) []Component {
	var components []Component
	
	templateName := strings.ToLower(template.Name)
	templateDesc := strings.ToLower(template.Description)
	
	// DHT22 Temperature/Humidity Sensor
	if strings.Contains(templateName, "dht22") || strings.Contains(templateDesc, "dht22") {
		components = append(components, Component{
			ID:   "dht22",
			Type: "sensor",
			Name: "DHT22 Sensor",
			Pins: []Pin{
				{Number: "1", Name: "VCC", Type: "power", Voltage: "3.3V-5V"},
				{Number: "2", Name: "DATA", Type: "digital", Description: "Data pin"},
				{Number: "3", Name: "NC", Type: "unused", Description: "Not connected"},
				{Number: "4", Name: "GND", Type: "ground"},
			},
		})
	}
	
	// Ultrasonic Distance Sensor
	if strings.Contains(templateName, "ultrasonic") || strings.Contains(templateDesc, "ultrasonic") || 
	   strings.Contains(templateName, "hc-sr04") || strings.Contains(templateDesc, "hc-sr04") {
		components = append(components, Component{
			ID:   "ultrasonic",
			Type: "sensor",
			Name: "HC-SR04 Ultrasonic",
			Pins: []Pin{
				{Number: "VCC", Name: "VCC", Type: "power", Voltage: "5V"},
				{Number: "TRIG", Name: "Trigger", Type: "digital", Description: "Trigger pin"},
				{Number: "ECHO", Name: "Echo", Type: "digital", Description: "Echo pin"},
				{Number: "GND", Name: "GND", Type: "ground"},
			},
		})
	}
	
	// Light Sensor (LDR)
	if strings.Contains(templateName, "light") || strings.Contains(templateDesc, "light") ||
	   strings.Contains(templateName, "ldr") || strings.Contains(templateDesc, "photoresistor") {
		components = append(components, Component{
			ID:   "ldr",
			Type: "sensor",
			Name: "Light Sensor (LDR)",
			Pins: []Pin{
				{Number: "1", Name: "Signal", Type: "analog", Description: "Analog output"},
				{Number: "2", Name: "GND", Type: "ground"},
			},
		})
	}
	
	return components
}

// extractAutomationComponents extracts automation components
func (wdg *WiringDiagramGenerator) extractAutomationComponents(template *Template, parameters map[string]interface{}) []Component {
	var components []Component
	
	templateName := strings.ToLower(template.Name)
	templateDesc := strings.ToLower(template.Description)
	
	// LED
	if strings.Contains(templateName, "led") || strings.Contains(templateDesc, "led") {
		components = append(components, Component{
			ID:   "led",
			Type: "actuator",
			Name: "LED",
			Pins: []Pin{
				{Number: "anode", Name: "Anode (+)", Type: "digital", Description: "Positive terminal"},
				{Number: "cathode", Name: "Cathode (-)", Type: "ground", Description: "Negative terminal"},
			},
		})
	}
	
	// Servo Motor
	if strings.Contains(templateName, "servo") || strings.Contains(templateDesc, "servo") {
		components = append(components, Component{
			ID:   "servo",
			Type: "actuator",
			Name: "Servo Motor",
			Pins: []Pin{
				{Number: "VCC", Name: "VCC", Type: "power", Voltage: "5V"},
				{Number: "GND", Name: "GND", Type: "ground"},
				{Number: "SIGNAL", Name: "Signal", Type: "digital", Description: "PWM control signal"},
			},
		})
	}
	
	// Relay
	if strings.Contains(templateName, "relay") || strings.Contains(templateDesc, "relay") {
		components = append(components, Component{
			ID:   "relay",
			Type: "actuator",
			Name: "Relay Module",
			Pins: []Pin{
				{Number: "VCC", Name: "VCC", Type: "power", Voltage: "5V"},
				{Number: "GND", Name: "GND", Type: "ground"},
				{Number: "IN", Name: "Input", Type: "digital", Description: "Control signal"},
				{Number: "COM", Name: "Common", Type: "switch", Description: "Common terminal"},
				{Number: "NO", Name: "Normally Open", Type: "switch"},
				{Number: "NC", Name: "Normally Closed", Type: "switch"},
			},
		})
	}
	
	return components
}

// extractDisplayComponents extracts display components
func (wdg *WiringDiagramGenerator) extractDisplayComponents(template *Template, parameters map[string]interface{}) []Component {
	var components []Component
	
	templateName := strings.ToLower(template.Name)
	templateDesc := strings.ToLower(template.Description)
	
	// LCD Display
	if strings.Contains(templateName, "lcd") || strings.Contains(templateDesc, "lcd") {
		components = append(components, Component{
			ID:   "lcd",
			Type: "display",
			Name: "LCD Display",
			Pins: []Pin{
				{Number: "VSS", Name: "VSS", Type: "ground"},
				{Number: "VDD", Name: "VDD", Type: "power", Voltage: "5V"},
				{Number: "V0", Name: "Contrast", Type: "analog"},
				{Number: "RS", Name: "Register Select", Type: "digital"},
				{Number: "E", Name: "Enable", Type: "digital"},
				{Number: "D4", Name: "Data 4", Type: "digital"},
				{Number: "D5", Name: "Data 5", Type: "digital"},
				{Number: "D6", Name: "Data 6", Type: "digital"},
				{Number: "D7", Name: "Data 7", Type: "digital"},
			},
		})
	}
	
	// OLED Display
	if strings.Contains(templateName, "oled") || strings.Contains(templateDesc, "oled") {
		components = append(components, Component{
			ID:   "oled",
			Type: "display",
			Name: "OLED Display",
			Pins: []Pin{
				{Number: "VCC", Name: "VCC", Type: "power", Voltage: "3.3V-5V"},
				{Number: "GND", Name: "GND", Type: "ground"},
				{Number: "SCL", Name: "SCL", Type: "i2c", Description: "I2C Clock"},
				{Number: "SDA", Name: "SDA", Type: "i2c", Description: "I2C Data"},
			},
		})
	}
	
	return components
}

// extractCommunicationComponents extracts communication components
func (wdg *WiringDiagramGenerator) extractCommunicationComponents(template *Template, parameters map[string]interface{}) []Component {
	var components []Component
	
	templateName := strings.ToLower(template.Name)
	templateDesc := strings.ToLower(template.Description)
	
	// WiFi Module (ESP8266)
	if strings.Contains(templateName, "wifi") || strings.Contains(templateDesc, "wifi") ||
	   strings.Contains(templateName, "esp8266") || strings.Contains(templateDesc, "esp8266") {
		components = append(components, Component{
			ID:   "wifi_module",
			Type: "communication",
			Name: "WiFi Module",
			Pins: []Pin{
				{Number: "VCC", Name: "VCC", Type: "power", Voltage: "3.3V"},
				{Number: "GND", Name: "GND", Type: "ground"},
				{Number: "TX", Name: "TX", Type: "serial", Description: "Serial transmit"},
				{Number: "RX", Name: "RX", Type: "serial", Description: "Serial receive"},
				{Number: "RST", Name: "Reset", Type: "digital"},
				{Number: "EN", Name: "Enable", Type: "digital"},
			},
		})
	}
	
	// Bluetooth Module
	if strings.Contains(templateName, "bluetooth") || strings.Contains(templateDesc, "bluetooth") {
		components = append(components, Component{
			ID:   "bluetooth",
			Type: "communication",
			Name: "Bluetooth Module",
			Pins: []Pin{
				{Number: "VCC", Name: "VCC", Type: "power", Voltage: "3.3V-5V"},
				{Number: "GND", Name: "GND", Type: "ground"},
				{Number: "TX", Name: "TX", Type: "serial"},
				{Number: "RX", Name: "RX", Type: "serial"},
			},
		})
	}
	
	return components
}

// extractGenericComponents extracts generic components based on parameters
func (wdg *WiringDiagramGenerator) extractGenericComponents(template *Template, parameters map[string]interface{}) []Component {
	var components []Component
	
	// Add basic LED if any digital pin is used
	for paramName := range parameters {
		if strings.Contains(strings.ToLower(paramName), "led") {
			components = append(components, Component{
				ID:   "led",
				Type: "actuator",
				Name: "LED",
				Pins: []Pin{
					{Number: "anode", Name: "Anode (+)", Type: "digital"},
					{Number: "cathode", Name: "Cathode (-)", Type: "ground"},
				},
			})
			break
		}
	}
	
	return components
}

// getBoardDisplayName returns a human-readable board name
func (wdg *WiringDiagramGenerator) getBoardDisplayName(boardFQBN string) string {
	boardNames := map[string]string{
		"arduino:avr:uno":              "Arduino Uno",
		"arduino:avr:nano":             "Arduino Nano",
		"arduino:avr:mega":             "Arduino Mega",
		"arduino:avr:leonardo":         "Arduino Leonardo",
		"esp32:esp32:esp32":            "ESP32",
		"esp8266:esp8266:nodemcuv2":    "NodeMCU ESP8266",
	}
	
	if name, exists := boardNames[boardFQBN]; exists {
		return name
	}
	return "Arduino Board"
}

// getArduinoPins returns the pins for an Arduino board
func (wdg *WiringDiagramGenerator) getArduinoPins(supportedBoards []string) []Pin {
	// Default to Arduino Uno pins
	pins := []Pin{
		{Number: "0", Name: "RX", Type: "digital"},
		{Number: "1", Name: "TX", Type: "digital"},
		{Number: "2", Name: "D2", Type: "digital"},
		{Number: "3", Name: "D3", Type: "digital"},
		{Number: "4", Name: "D4", Type: "digital"},
		{Number: "5", Name: "D5", Type: "digital"},
		{Number: "6", Name: "D6", Type: "digital"},
		{Number: "7", Name: "D7", Type: "digital"},
		{Number: "8", Name: "D8", Type: "digital"},
		{Number: "9", Name: "D9", Type: "digital"},
		{Number: "10", Name: "D10", Type: "digital"},
		{Number: "11", Name: "D11", Type: "digital"},
		{Number: "12", Name: "D12", Type: "digital"},
		{Number: "13", Name: "D13", Type: "digital"},
		{Number: "A0", Name: "A0", Type: "analog"},
		{Number: "A1", Name: "A1", Type: "analog"},
		{Number: "A2", Name: "A2", Type: "analog"},
		{Number: "A3", Name: "A3", Type: "analog"},
		{Number: "A4", Name: "A4", Type: "analog"},
		{Number: "A5", Name: "A5", Type: "analog"},
		{Number: "5V", Name: "5V", Type: "power", Voltage: "5V"},
		{Number: "3V3", Name: "3.3V", Type: "power", Voltage: "3.3V"},
		{Number: "GND", Name: "GND", Type: "ground"},
	}
	
	return pins
}

// inferComponentFromParameter infers component name from parameter name
func (wdg *WiringDiagramGenerator) inferComponentFromParameter(paramName string, template *Template) string {
	paramLower := strings.ToLower(paramName)
	
	if strings.Contains(paramLower, "led") {
		return "led"
	} else if strings.Contains(paramLower, "servo") {
		return "servo"
	} else if strings.Contains(paramLower, "sensor") || strings.Contains(paramLower, "dht") {
		return "dht22"
	} else if strings.Contains(paramLower, "ultrasonic") || strings.Contains(paramLower, "trig") || strings.Contains(paramLower, "echo") {
		return "ultrasonic"
	} else if strings.Contains(paramLower, "relay") {
		return "relay"
	} else if strings.Contains(paramLower, "lcd") {
		return "lcd"
	} else if strings.Contains(paramLower, "oled") {
		return "oled"
	}
	
	return "generic_component"
}

// inferComponentPin infers the component pin from parameter name
func (wdg *WiringDiagramGenerator) inferComponentPin(paramName, componentName string) string {
	paramLower := strings.ToLower(paramName)
	
	switch componentName {
	case "led":
		return "anode"
	case "servo":
		return "SIGNAL"
	case "dht22":
		return "DATA"
	case "ultrasonic":
		if strings.Contains(paramLower, "trig") {
			return "TRIG"
		} else if strings.Contains(paramLower, "echo") {
			return "ECHO"
		}
		return "TRIG"
	case "relay":
		return "IN"
	default:
		return "INPUT"
	}
}

// getWireColor returns appropriate wire color for connection
func (wdg *WiringDiagramGenerator) getWireColor(paramName, pin string) string {
	paramLower := strings.ToLower(paramName)
	
	if strings.Contains(paramLower, "power") || strings.Contains(paramLower, "vcc") || pin == "5V" || pin == "3V3" {
		return "red"
	} else if strings.Contains(paramLower, "ground") || strings.Contains(paramLower, "gnd") || pin == "GND" {
		return "black"
	} else if strings.Contains(paramLower, "data") || strings.Contains(paramLower, "signal") {
		return "yellow"
	} else if strings.Contains(paramLower, "clock") || strings.Contains(paramLower, "scl") {
		return "blue"
	} else if strings.Contains(paramLower, "sda") {
		return "green"
	}
	
	return "gray"
}

// generatePowerConnections generates power and ground connections
func (wdg *WiringDiagramGenerator) generatePowerConnections(components []Component) []Connection {
	var connections []Connection
	
	// Find Arduino component
	var arduinoComponent *Component
	for i := range components {
		if components[i].Type == "microcontroller" {
			arduinoComponent = &components[i]
			break
		}
	}
	
	if arduinoComponent == nil {
		return connections
	}
	
	// Add power and ground connections for each component
	for _, component := range components {
		if component.Type == "microcontroller" {
			continue
		}
		
		// Add power connection
		for _, pin := range component.Pins {
			if pin.Type == "power" {
				powerPin := "5V"
				if pin.Voltage == "3.3V" {
					powerPin = "3V3"
				}
				connections = append(connections, Connection{
					FromComponent: arduinoComponent.ID,
					FromPin:       powerPin,
					ToComponent:   component.ID,
					ToPin:         pin.Number,
					WireColor:     "red",
				})
			} else if pin.Type == "ground" {
				connections = append(connections, Connection{
					FromComponent: arduinoComponent.ID,
					FromPin:       "GND",
					ToComponent:   component.ID,
					ToPin:         pin.Number,
					WireColor:     "black",
				})
			}
		}
	}
	
	return connections
}

// getMermaidShape returns the Mermaid shape for a component type
func (wdg *WiringDiagramGenerator) getMermaidShape(componentType string) []string {
	shapes := map[string][]string{
		"microcontroller": {"[", "]"},
		"sensor":          {"(", ")"},
		"actuator":        {"{", "}"},
		"display":         {"[[", "]]"},
		"communication":   {"((", "))"},
		"power":           {"[/", "/]"},
		"default":         {"[", "]"},
	}
	
	if shape, exists := shapes[componentType]; exists {
		return shape
	}
	return shapes["default"]
}

// generateMermaidStyling generates CSS styling for the Mermaid diagram
func (wdg *WiringDiagramGenerator) generateMermaidStyling(components []Component) string {
	var builder strings.Builder
	
	builder.WriteString("\n    %% Styling\n")
	
	for _, component := range components {
		switch component.Type {
		case "microcontroller":
			builder.WriteString(fmt.Sprintf("    classDef %s fill:#e1f5fe,stroke:#01579b,stroke-width:2px\n", component.ID))
		case "sensor":
			builder.WriteString(fmt.Sprintf("    classDef %s fill:#f3e5f5,stroke:#4a148c,stroke-width:2px\n", component.ID))
		case "actuator":
			builder.WriteString(fmt.Sprintf("    classDef %s fill:#e8f5e8,stroke:#1b5e20,stroke-width:2px\n", component.ID))
		case "display":
			builder.WriteString(fmt.Sprintf("    classDef %s fill:#fff3e0,stroke:#e65100,stroke-width:2px\n", component.ID))
		case "communication":
			builder.WriteString(fmt.Sprintf("    classDef %s fill:#fce4ec,stroke:#880e4f,stroke-width:2px\n", component.ID))
		}
		builder.WriteString(fmt.Sprintf("    class %s %s\n", component.ID, component.ID))
	}
	
	return builder.String()
}