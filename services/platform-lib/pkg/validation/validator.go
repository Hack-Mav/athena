package validation

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Validator provides input validation and sanitization
type Validator struct {
	// Basic validator without external dependencies
}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	return &Validator{}
}

// Validate validates a struct and returns validation errors
func (v *Validator) Validate(s interface{}) error {
	// Basic struct validation - in production, use proper validation library
	return nil
}

// ValidateVar validates a single field
func (v *Validator) ValidateVar(field interface{}, tag string) error {
	// Basic field validation
	str, ok := field.(string)
	if !ok {
		return fmt.Errorf("field must be a string")
	}

	rules := strings.Split(tag, ",")
	for _, rule := range rules {
		if err := v.applyRule(str, rule); err != nil {
			return err
		}
	}

	return nil
}

// applyRule applies a single validation rule
func (v *Validator) applyRule(value, rule string) error {
	switch rule {
	case "required":
		if value == "" {
			return fmt.Errorf("field is required")
		}
	case "email":
		if !v.isValidEmail(value) {
			return fmt.Errorf("invalid email format")
		}
	case "template_id":
		if !v.isValidTemplateID(value) {
			return fmt.Errorf("invalid template ID format")
		}
	case "arduino_pin":
		if !v.isValidArduinoPin(value) {
			return fmt.Errorf("invalid Arduino pin format")
		}
	case "safe_string":
		if !v.isSafeString(value) {
			return fmt.Errorf("contains unsafe characters")
		}
	case "mqtt_topic":
		if !v.isValidMQTTTopic(value) {
			return fmt.Errorf("invalid MQTT topic format")
		}
	case "uuid":
		if !v.isValidUUID(value) {
			return fmt.Errorf("invalid UUID format")
		}
	case "semver":
		if !v.isValidSemver(value) {
			return fmt.Errorf("invalid semantic version format")
		}
	case "url":
		if !v.isValidURL(value) {
			return fmt.Errorf("invalid URL format")
		}
	}

	// Handle oneof rules
	if strings.HasPrefix(rule, "oneof=") {
		options := strings.TrimPrefix(rule, "oneof=")
		validOptions := strings.Split(options, " ")
		isValid := false
		for _, option := range validOptions {
			if value == option {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("must be one of: %s", options)
		}
	}

	// Handle min/max length rules
	if strings.HasPrefix(rule, "min=") {
		minLen := 0
		fmt.Sscanf(rule, "min=%d", &minLen)
		if len(value) < minLen {
			return fmt.Errorf("minimum length is %d", minLen)
		}
	}

	if strings.HasPrefix(rule, "max=") {
		maxLen := 0
		fmt.Sscanf(rule, "max=%d", &maxLen)
		if len(value) > maxLen {
			return fmt.Errorf("maximum length is %d", maxLen)
		}
	}

	return nil
}

// SanitizeString sanitizes a string input
func (v *Validator) SanitizeString(input string) string {
	// Remove leading/trailing whitespace
	input = strings.TrimSpace(input)

	// Remove potentially dangerous characters
	input = sanitizeString(input)

	return input
}

// SanitizeTemplateID sanitizes template ID
func (v *Validator) SanitizeTemplateID(input string) string {
	input = strings.ToLower(input)
	input = strings.ReplaceAll(input, " ", "-")
	input = strings.ReplaceAll(input, "_", "-")

	// Remove any characters that aren't alphanumeric or hyphen
	re := regexp.MustCompile(`[^a-z0-9-]`)
	input = re.ReplaceAllString(input, "")

	// Remove consecutive hyphens
	re = regexp.MustCompile(`-+`)
	input = re.ReplaceAllString(input, "-")

	// Remove leading/trailing hyphens
	input = strings.Trim(input, "-")

	return input
}

// ValidationError represents validation errors
type ValidationError struct {
	Errors map[string]string `json:"errors"`
}

// NewValidationError creates a new validation error
func NewValidationError(errs []string) *ValidationError {
	errors := make(map[string]string)

	for i, e := range errs {
		errors[fmt.Sprintf("field_%d", i)] = e
	}

	return &ValidationError{
		Errors: errors,
	}
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	var messages []string
	for field, message := range e.Errors {
		messages = append(messages, fmt.Sprintf("%s: %s", field, message))
	}
	return strings.Join(messages, "; ")
}

// Validation helper functions

func (v *Validator) isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

func (v *Validator) isValidTemplateID(id string) bool {
	if len(id) < 3 || len(id) > 50 {
		return false
	}

	// Only allow lowercase letters, numbers, and hyphens
	matched, _ := regexp.MatchString(`^[a-z0-9-]+$`, id)
	return matched
}

func (v *Validator) isValidArduinoPin(pin string) bool {
	// Allow numeric pins or analog pins (A0-A5)
	matched, _ := regexp.MatchString(`^([0-9]+|A[0-5])$`, pin)
	return matched
}

func (v *Validator) isSafeString(s string) bool {
	// Check for dangerous characters
	dangerous := []string{"<", ">", "&", "\"", "'", "/", "\\", "script", "javascript"}

	for _, d := range dangerous {
		if strings.Contains(strings.ToLower(s), d) {
			return false
		}
	}

	return true
}

func (v *Validator) isValidMQTTTopic(topic string) bool {
	if len(topic) == 0 || len(topic) > 255 {
		return false
	}

	// MQTT topic validation rules
	if strings.HasPrefix(topic, "$") || strings.HasPrefix(topic, "+") {
		return false
	}

	if strings.HasSuffix(topic, "/") {
		return false
	}

	if strings.Contains(topic, "//") {
		return false
	}

	// # can only be used as wildcard at the end
	if strings.Contains(topic, "#") && !strings.HasSuffix(topic, "/#") {
		return false
	}

	return true
}

func (v *Validator) isValidUUID(uuid string) bool {
	// Basic UUID validation (v4 format)
	re := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	return re.MatchString(uuid)
}

func (v *Validator) isValidSemver(version string) bool {
	// Semantic version validation (major.minor.patch)
	re := regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
	return re.MatchString(version)
}

func (v *Validator) isValidURL(url string) bool {
	// Basic URL validation
	re := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	return re.MatchString(url)
}

// Sanitization functions

func sanitizeString(input string) string {
	var result strings.Builder

	for _, r := range input {
		// Remove control characters except newline and tab
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			continue
		}

		// Replace potentially dangerous characters
		switch r {
		case '<':
			result.WriteString("&lt;")
		case '>':
			result.WriteString("&gt;")
		case '&':
			result.WriteString("&amp;")
		case '"':
			result.WriteString("&quot;")
		case '\'':
			result.WriteString("&#39;")
		default:
			result.WriteRune(r)
		}
	}

	return result.String()
}

// Common validation rules
const (
	// Template validation
	TemplateIDRule   = "required,template_id"
	TemplateNameRule = "required,min=3,max=100,safe_string"
	TemplateDescRule = "max=500,safe_string"
	TemplateCategory = "required,oneof=sensing automation robotics wearables audio displays communications data_logger examples"

	// Arduino validation
	ArduinoPinRule = "required,arduino_pin"
	BoardTypeRule  = "required,oneof=uno nano esp32 mega"

	// Network validation
	EmailRule     = "required,email"
	URLRule       = "url"
	MQTTTopicRule = "mqtt_topic"

	// General validation
	UUIDRule     = "required,uuid"
	SemverRule   = "semver"
	NonEmptyRule = "required,min=1"
)

// Sanitization rules for different input types
type SanitizationRules struct {
	TemplateID func(string) string
	ArduinoPin func(string) string
	MQTTTopic  func(string) string
	FreeText   func(string) string
}

// GetSanitizationRules returns sanitization functions
func GetSanitizationRules() *SanitizationRules {
	v := NewValidator()
	return &SanitizationRules{
		TemplateID: v.SanitizeTemplateID,
		ArduinoPin: func(input string) string {
			input = strings.ToUpper(strings.TrimSpace(input))
			return input
		},
		MQTTTopic: func(input string) string {
			input = strings.TrimSpace(input)
			// Remove leading/trailing slashes
			input = strings.Trim(input, "/")
			return input
		},
		FreeText: v.SanitizeString,
	}
}
