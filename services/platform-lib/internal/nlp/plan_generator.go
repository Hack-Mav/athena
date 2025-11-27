package nlp

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PlanGenerator generates implementation plans
type PlanGenerator struct {
	llmClient       *LLMClient
	safetyValidator *SafetyValidator
}

// NewPlanGenerator creates a new plan generator
func NewPlanGenerator(llmClient *LLMClient) *PlanGenerator {
	return &PlanGenerator{
		llmClient:       llmClient,
		safetyValidator: NewSafetyValidator(),
	}
}

// GeneratePlan generates a complete implementation plan
func (pg *PlanGenerator) GeneratePlan(ctx context.Context, requirements *ParsedRequirements, template *TemplateInfo, parameters map[string]interface{}, boardType string) (*ImplementationPlan, error) {
	plan := &ImplementationPlan{
		TemplateID:   template.ID,
		TemplateName: template.Name,
		Parameters:   parameters,
		CreatedAt:    time.Now(),
	}

	// Generate wiring diagram
	wiringDiagram, err := pg.generateWiringDiagram(ctx, requirements, template, parameters, boardType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate wiring diagram: %w", err)
	}
	plan.WiringDiagram = wiringDiagram

	// Perform safety validation
	safetyChecks, err := pg.safetyValidator.ValidateSafety(ctx, plan, boardType)
	if err != nil {
		return nil, fmt.Errorf("safety validation failed: %w", err)
	}
	plan.SafetyChecks = safetyChecks

	// Add safety warnings to plan warnings
	plan.Warnings = append(plan.Warnings, safetyChecks.Warnings...)
	if !safetyChecks.Valid {
		plan.Warnings = append(plan.Warnings, "SAFETY ERRORS DETECTED - Review and fix before proceeding")
		plan.Warnings = append(plan.Warnings, safetyChecks.Errors...)
	}

	// Generate bill of materials
	bom, err := pg.generateBOM(ctx, requirements, template, wiringDiagram, boardType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate BOM: %w", err)
	}
	plan.BOM = bom

	// Calculate estimated cost
	plan.EstimatedCost = pg.calculateTotalCost(bom)

	// Generate step-by-step instructions
	instructions, err := pg.generateInstructions(ctx, requirements, template, wiringDiagram, parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to generate instructions: %w", err)
	}
	plan.Instructions = instructions

	// Determine difficulty level
	plan.DifficultyLevel = pg.determineDifficultyLevel(requirements, wiringDiagram)

	return plan, nil
}

// generateWiringDiagram generates a wiring diagram
func (pg *PlanGenerator) generateWiringDiagram(ctx context.Context, requirements *ParsedRequirements, template *TemplateInfo, parameters map[string]interface{}, boardType string) (*WiringDiagram, error) {
	diagram := &WiringDiagram{
		Components:  []Component{},
		Connections: []Connection{},
	}

	// Add the Arduino board as the main component
	board := Component{
		ID:   "board",
		Type: "arduino",
		Name: boardType,
		Pins: []Pin{},
	}
	diagram.Components = append(diagram.Components, board)

	// Add sensors
	for i, sensor := range requirements.Sensors {
		componentID := fmt.Sprintf("sensor_%d", i)
		component := Component{
			ID:   componentID,
			Type: sensor.Type,
			Name: fmt.Sprintf("%s Sensor", strings.Title(sensor.Type)),
			Pins: []Pin{
				{Number: "VCC", Name: "Power", Type: "power", Voltage: "5V"},
				{Number: "GND", Name: "Ground", Type: "ground"},
				{Number: "OUT", Name: "Signal", Type: "digital"},
			},
		}

		if sensor.Model != "" {
			component.Name = sensor.Model
		}

		diagram.Components = append(diagram.Components, component)

		// Add connections
		pin := sensor.Pin
		if pin == "" {
			pin = pg.assignPin(sensor.Type, i, parameters)
		}

		diagram.Connections = append(diagram.Connections, Connection{
			FromComponent: "board",
			FromPin:       pin,
			ToComponent:   componentID,
			ToPin:         "OUT",
			WireColor:     "yellow",
		})

		diagram.Connections = append(diagram.Connections, Connection{
			FromComponent: "board",
			FromPin:       "5V",
			ToComponent:   componentID,
			ToPin:         "VCC",
			WireColor:     "red",
		})

		diagram.Connections = append(diagram.Connections, Connection{
			FromComponent: "board",
			FromPin:       "GND",
			ToComponent:   componentID,
			ToPin:         "GND",
			WireColor:     "black",
		})
	}

	// Add actuators
	for i, actuator := range requirements.Actuators {
		componentID := fmt.Sprintf("actuator_%d", i)
		component := Component{
			ID:   componentID,
			Type: actuator.Type,
			Name: fmt.Sprintf("%s", strings.Title(actuator.Type)),
			Pins: []Pin{},
		}

		// Define pins based on actuator type
		if actuator.Type == "led" {
			component.Pins = []Pin{
				{Number: "ANODE", Name: "Anode", Type: "digital"},
				{Number: "CATHODE", Name: "Cathode", Type: "ground"},
			}
		} else if actuator.Type == "servo" {
			component.Pins = []Pin{
				{Number: "VCC", Name: "Power", Type: "power", Voltage: "5V"},
				{Number: "GND", Name: "Ground", Type: "ground"},
				{Number: "SIG", Name: "Signal", Type: "pwm"},
			}
		} else {
			component.Pins = []Pin{
				{Number: "VCC", Name: "Power", Type: "power"},
				{Number: "GND", Name: "Ground", Type: "ground"},
				{Number: "IN", Name: "Control", Type: "digital"},
			}
		}

		diagram.Components = append(diagram.Components, component)

		// Add connections
		pin := actuator.Pin
		if pin == "" {
			pin = pg.assignPin(actuator.Type, i, parameters)
		}

		if actuator.Type == "led" {
			diagram.Connections = append(diagram.Connections, Connection{
				FromComponent: "board",
				FromPin:       pin,
				ToComponent:   componentID,
				ToPin:         "ANODE",
				WireColor:     "green",
			})
			diagram.Connections = append(diagram.Connections, Connection{
				FromComponent: "board",
				FromPin:       "GND",
				ToComponent:   componentID,
				ToPin:         "CATHODE",
				WireColor:     "black",
			})
		} else {
			diagram.Connections = append(diagram.Connections, Connection{
				FromComponent: "board",
				FromPin:       pin,
				ToComponent:   componentID,
				ToPin:         "IN",
				WireColor:     "blue",
			})
			diagram.Connections = append(diagram.Connections, Connection{
				FromComponent: "board",
				FromPin:       "5V",
				ToComponent:   componentID,
				ToPin:         "VCC",
				WireColor:     "red",
			})
			diagram.Connections = append(diagram.Connections, Connection{
				FromComponent: "board",
				FromPin:       "GND",
				ToComponent:   componentID,
				ToPin:         "GND",
				WireColor:     "black",
			})
		}
	}

	// Generate Mermaid syntax
	diagram.MermaidSyntax = pg.generateMermaidSyntax(diagram)

	return diagram, nil
}

// generateBOM generates a bill of materials
func (pg *PlanGenerator) generateBOM(ctx context.Context, requirements *ParsedRequirements, template *TemplateInfo, diagram *WiringDiagram, boardType string) ([]BOMItem, error) {
	bom := []BOMItem{}

	// Add the Arduino board
	bom = append(bom, BOMItem{
		Component:   boardType,
		Quantity:    1,
		Description: fmt.Sprintf("Arduino %s board", boardType),
		Price:       pg.estimateComponentPrice(boardType),
	})

	// Add sensors
	for _, sensor := range requirements.Sensors {
		item := BOMItem{
			Component:   sensor.Type,
			Quantity:    1,
			Description: fmt.Sprintf("%s sensor", sensor.Type),
			Price:       pg.estimateComponentPrice(sensor.Type),
		}
		if sensor.Model != "" {
			item.Component = sensor.Model
			item.Description = fmt.Sprintf("%s %s sensor", sensor.Model, sensor.Type)
		}
		bom = append(bom, item)
	}

	// Add actuators
	for _, actuator := range requirements.Actuators {
		item := BOMItem{
			Component:   actuator.Type,
			Quantity:    1,
			Description: fmt.Sprintf("%s", actuator.Type),
			Price:       pg.estimateComponentPrice(actuator.Type),
		}
		bom = append(bom, item)
	}

	// Add common components
	bom = append(bom, BOMItem{
		Component:   "Breadboard",
		Quantity:    1,
		Description: "400-point breadboard",
		Price:       3.0,
	})

	bom = append(bom, BOMItem{
		Component:   "Jumper Wires",
		Quantity:    1,
		Description: "Set of jumper wires (male-to-male)",
		Price:       5.0,
	})

	bom = append(bom, BOMItem{
		Component:   "USB Cable",
		Quantity:    1,
		Description: "USB A to B cable for programming",
		Price:       3.0,
	})

	// Add resistors if LEDs are present
	hasLED := false
	for _, actuator := range requirements.Actuators {
		if actuator.Type == "led" {
			hasLED = true
			break
		}
	}
	if hasLED {
		bom = append(bom, BOMItem{
			Component:   "Resistors",
			Quantity:    5,
			Description: "220Ω resistors for LEDs",
			Price:       0.5,
		})
	}

	return bom, nil
}

// generateInstructions generates step-by-step assembly instructions
func (pg *PlanGenerator) generateInstructions(ctx context.Context, requirements *ParsedRequirements, template *TemplateInfo, diagram *WiringDiagram, parameters map[string]interface{}) ([]string, error) {
	instructions := []string{}

	// Step 1: Gather components
	instructions = append(instructions, "Gather all components from the bill of materials")

	// Step 2: Board setup
	instructions = append(instructions, "Place the Arduino board on the breadboard or work surface")

	// Step 3: Connect sensors
	for i, sensor := range requirements.Sensors {
		pin := sensor.Pin
		if pin == "" {
			pin = pg.assignPin(sensor.Type, i, parameters)
		}
		instructions = append(instructions, fmt.Sprintf("Connect %s sensor to pin %s (VCC to 5V, GND to GND)", sensor.Type, pin))
	}

	// Step 4: Connect actuators
	for i, actuator := range requirements.Actuators {
		pin := actuator.Pin
		if pin == "" {
			pin = pg.assignPin(actuator.Type, i, parameters)
		}
		if actuator.Type == "led" {
			instructions = append(instructions, fmt.Sprintf("Connect LED to pin %s through a 220Ω resistor (cathode to GND)", pin))
		} else {
			instructions = append(instructions, fmt.Sprintf("Connect %s to pin %s (VCC to 5V, GND to GND)", actuator.Type, pin))
		}
	}

	// Step 5: Double-check connections
	instructions = append(instructions, "Double-check all connections against the wiring diagram")

	// Step 6: Upload code
	instructions = append(instructions, "Connect Arduino to computer via USB cable")
	instructions = append(instructions, "Open Arduino IDE and upload the generated code")

	// Step 7: Test
	instructions = append(instructions, "Open Serial Monitor at 9600 baud to view output")
	instructions = append(instructions, "Verify that sensors are reading correctly and actuators are responding")

	// Step 8: Troubleshooting
	instructions = append(instructions, "If issues occur, check power connections and verify pin assignments match the code")

	return instructions, nil
}

// Helper methods

func (pg *PlanGenerator) assignPin(componentType string, index int, parameters map[string]interface{}) string {
	// Try to get pin from parameters
	pinKey := fmt.Sprintf("%s_pin", componentType)
	if index > 0 {
		pinKey = fmt.Sprintf("%s_pin_%d", componentType, index+1)
	}

	if pin, ok := parameters[pinKey]; ok {
		if pinStr, ok := pin.(string); ok {
			return pinStr
		}
	}

	// Assign default pins based on component type
	switch componentType {
	case "temperature", "humidity", "dht22", "dht11":
		return "D2"
	case "distance", "ultrasonic":
		return "D3"
	case "motion", "pir":
		return "D4"
	case "led":
		return fmt.Sprintf("D%d", 9+index)
	case "servo":
		return fmt.Sprintf("D%d", 5+index)
	case "relay":
		return fmt.Sprintf("D%d", 7+index)
	default:
		return fmt.Sprintf("D%d", 2+index)
	}
}

func (pg *PlanGenerator) estimateComponentPrice(component string) float64 {
	component = strings.ToLower(component)

	switch {
	case strings.Contains(component, "uno"):
		return 25.0
	case strings.Contains(component, "nano"):
		return 15.0
	case strings.Contains(component, "esp32"):
		return 10.0
	case strings.Contains(component, "esp8266"):
		return 5.0
	case strings.Contains(component, "dht22"):
		return 5.0
	case strings.Contains(component, "dht11"):
		return 3.0
	case strings.Contains(component, "ultrasonic") || strings.Contains(component, "hc-sr04"):
		return 2.0
	case strings.Contains(component, "pir") || strings.Contains(component, "motion"):
		return 3.0
	case strings.Contains(component, "led"):
		return 0.5
	case strings.Contains(component, "servo"):
		return 8.0
	case strings.Contains(component, "relay"):
		return 3.0
	case strings.Contains(component, "temperature"):
		return 4.0
	case strings.Contains(component, "humidity"):
		return 4.0
	case strings.Contains(component, "distance"):
		return 2.0
	default:
		return 5.0
	}
}

func (pg *PlanGenerator) calculateTotalCost(bom []BOMItem) float64 {
	total := 0.0
	for _, item := range bom {
		total += item.Price * float64(item.Quantity)
	}
	return total
}

func (pg *PlanGenerator) determineDifficultyLevel(requirements *ParsedRequirements, diagram *WiringDiagram) string {
	score := 0

	// Count components
	score += len(requirements.Sensors)
	score += len(requirements.Actuators)
	score += len(requirements.Communication) * 2

	// Count connections
	score += len(diagram.Connections) / 3

	if score <= 5 {
		return "beginner"
	} else if score <= 10 {
		return "intermediate"
	}
	return "advanced"
}

func (pg *PlanGenerator) generateMermaidSyntax(diagram *WiringDiagram) string {
	var sb strings.Builder

	sb.WriteString("graph LR\n")

	// Add components
	for _, component := range diagram.Components {
		sb.WriteString(fmt.Sprintf("    %s[%s]\n", component.ID, component.Name))
	}

	// Add connections
	for _, connection := range diagram.Connections {
		sb.WriteString(fmt.Sprintf("    %s -->|%s| %s\n", connection.FromComponent, connection.FromPin, connection.ToComponent))
	}

	return sb.String()
}
