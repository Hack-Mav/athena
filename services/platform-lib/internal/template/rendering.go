package template

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"text/template"
)

// TemplateRenderer handles Arduino code template rendering
type TemplateRenderer struct {
	funcMap template.FuncMap
}

// NewTemplateRenderer creates a new template renderer with custom functions
func NewTemplateRenderer() *TemplateRenderer {
	return &TemplateRenderer{
		funcMap: template.FuncMap{
			"upper":        strings.ToUpper,
			"lower":        strings.ToLower,
			"title":        strings.Title,
			"replace":      strings.ReplaceAll,
			"contains":     strings.Contains,
			"hasPrefix":    strings.HasPrefix,
			"hasSuffix":    strings.HasSuffix,
			"join":         strings.Join,
			"split":        strings.Split,
			"trim":         strings.TrimSpace,
			"add":          add,
			"sub":          sub,
			"mul":          mul,
			"div":          div,
			"mod":          mod,
			"eq":           eq,
			"ne":           ne,
			"lt":           lt,
			"le":           le,
			"gt":           gt,
			"ge":           ge,
			"and":          and,
			"or":           or,
			"not":          not,
			"default":      defaultValue,
			"pinType":      getPinType,
			"analogPin":    isAnalogPin,
			"digitalPin":   isDigitalPin,
			"pwmPin":       isPWMPin,
			"formatPin":    formatPin,
			"generateID":   generateID,
			"indent":       indent,
			"comment":      comment,
			"defineConst":  defineConstant,
		},
	}
}

// RenderArduinoCode renders Arduino code from a template with parameters
func (tr *TemplateRenderer) RenderArduinoCode(templateCode string, parameters map[string]interface{}) (string, error) {
	// Create a new template with custom functions
	tmpl, err := template.New("arduino").Funcs(tr.funcMap).Parse(templateCode)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Render the template
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, parameters)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// RenderTemplateWithIncludes renders a template with support for includes and partials
func (tr *TemplateRenderer) RenderTemplateWithIncludes(mainTemplate string, includes map[string]string, parameters map[string]interface{}) (string, error) {
	// Create the main template
	tmpl := template.New("main").Funcs(tr.funcMap)

	// Parse all includes first
	for name, content := range includes {
		_, err := tmpl.New(name).Parse(content)
		if err != nil {
			return "", fmt.Errorf("failed to parse include template %s: %w", name, err)
		}
	}

	// Parse the main template
	_, err := tmpl.Parse(mainTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse main template: %w", err)
	}

	// Render the template
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, parameters)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// Custom template functions

// Arithmetic functions
func add(a, b interface{}) (interface{}, error) {
	av, bv, err := convertToNumbers(a, b)
	if err != nil {
		return nil, err
	}
	return av + bv, nil
}

func sub(a, b interface{}) (interface{}, error) {
	av, bv, err := convertToNumbers(a, b)
	if err != nil {
		return nil, err
	}
	return av - bv, nil
}

func mul(a, b interface{}) (interface{}, error) {
	av, bv, err := convertToNumbers(a, b)
	if err != nil {
		return nil, err
	}
	return av * bv, nil
}

func div(a, b interface{}) (interface{}, error) {
	av, bv, err := convertToNumbers(a, b)
	if err != nil {
		return nil, err
	}
	if bv == 0 {
		return nil, fmt.Errorf("division by zero")
	}
	return av / bv, nil
}

func mod(a, b interface{}) (interface{}, error) {
	av, bv, err := convertToIntegers(a, b)
	if err != nil {
		return nil, err
	}
	if bv == 0 {
		return nil, fmt.Errorf("modulo by zero")
	}
	return av % bv, nil
}

// Comparison functions
func eq(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func ne(a, b interface{}) bool {
	return !eq(a, b)
}

func lt(a, b interface{}) (bool, error) {
	av, bv, err := convertToNumbers(a, b)
	if err != nil {
		return false, err
	}
	return av < bv, nil
}

func le(a, b interface{}) (bool, error) {
	av, bv, err := convertToNumbers(a, b)
	if err != nil {
		return false, err
	}
	return av <= bv, nil
}

func gt(a, b interface{}) (bool, error) {
	av, bv, err := convertToNumbers(a, b)
	if err != nil {
		return false, err
	}
	return av > bv, nil
}

func ge(a, b interface{}) (bool, error) {
	av, bv, err := convertToNumbers(a, b)
	if err != nil {
		return false, err
	}
	return av >= bv, nil
}

// Logical functions
func and(a, b bool) bool {
	return a && b
}

func or(a, b bool) bool {
	return a || b
}

func not(a bool) bool {
	return !a
}

// Utility functions
func defaultValue(defaultVal, value interface{}) interface{} {
	if value == nil || value == "" {
		return defaultVal
	}
	return value
}

// Arduino-specific functions
func getPinType(pin interface{}) string {
	pinStr := fmt.Sprintf("%v", pin)
	if strings.HasPrefix(strings.ToUpper(pinStr), "A") {
		return "analog"
	}
	return "digital"
}

func isAnalogPin(pin interface{}) bool {
	return getPinType(pin) == "analog"
}

func isDigitalPin(pin interface{}) bool {
	return getPinType(pin) == "digital"
}

func isPWMPin(pin interface{}) bool {
	pinStr := fmt.Sprintf("%v", pin)
	// Common PWM pins on Arduino Uno
	pwmPins := []string{"3", "5", "6", "9", "10", "11"}
	for _, pwmPin := range pwmPins {
		if pinStr == pwmPin {
			return true
		}
	}
	return false
}

func formatPin(pin interface{}) string {
	pinStr := fmt.Sprintf("%v", pin)
	if strings.HasPrefix(strings.ToUpper(pinStr), "A") {
		return strings.ToUpper(pinStr)
	}
	return pinStr
}

func generateID(prefix string) string {
	// Simple ID generation - in production, you'd use a more sophisticated approach
	return fmt.Sprintf("%s_%d", prefix, len(prefix)*7+42)
}

func indent(spaces int, text string) string {
	indentation := strings.Repeat(" ", spaces)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = indentation + line
		}
	}
	return strings.Join(lines, "\n")
}

func comment(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = "// " + line
		}
	}
	return strings.Join(lines, "\n")
}

func defineConstant(name string, value interface{}) string {
	return fmt.Sprintf("#define %s %v", strings.ToUpper(name), value)
}

// Helper functions
func convertToNumbers(a, b interface{}) (float64, float64, error) {
	av, err := toFloat64(a)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot convert %v to number: %w", a, err)
	}
	bv, err := toFloat64(b)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot convert %v to number: %w", b, err)
	}
	return av, bv, nil
}

func convertToIntegers(a, b interface{}) (int, int, error) {
	av, err := toInt(a)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot convert %v to integer: %w", a, err)
	}
	bv, err := toInt(b)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot convert %v to integer: %w", b, err)
	}
	return av, bv, nil
}

func toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		return strconv.ParseFloat(val, 64)
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}

func toInt(v interface{}) (int, error) {
	switch val := v.(type) {
	case int:
		return val, nil
	case int32:
		return int(val), nil
	case int64:
		return int(val), nil
	case float64:
		return int(val), nil
	case float32:
		return int(val), nil
	case string:
		return strconv.Atoi(val)
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}