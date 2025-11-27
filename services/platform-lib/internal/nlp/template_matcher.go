package nlp

import (
	"context"
	"fmt"
	"strings"
)

// TemplateMatcher handles template selection based on requirements
type TemplateMatcher struct {
	llmClient LLMClientInterface
}

// NewTemplateMatcher creates a new template matcher
func NewTemplateMatcher(llmClient LLMClientInterface) *TemplateMatcher {
	return &TemplateMatcher{
		llmClient: llmClient,
	}
}

// TemplateScore represents a template with its match score
type TemplateScore struct {
	TemplateID   string
	TemplateName string
	Score        float64
	Reasons      []string
}

// MatchTemplates finds the best matching templates for the given requirements
func (tm *TemplateMatcher) MatchTemplates(ctx context.Context, requirements *ParsedRequirements, availableTemplates []TemplateInfo) ([]TemplateScore, error) {
	if len(availableTemplates) == 0 {
		return nil, fmt.Errorf("no templates available")
	}

	var scores []TemplateScore

	for _, template := range availableTemplates {
		score := tm.calculateTemplateScore(requirements, &template)
		if score.Score > 0 {
			scores = append(scores, score)
		}
	}

	// Sort by score (descending)
	scores = tm.sortByScore(scores)

	return scores, nil
}

// SelectBestTemplate selects the best matching template
func (tm *TemplateMatcher) SelectBestTemplate(ctx context.Context, requirements *ParsedRequirements, availableTemplates []TemplateInfo) (*TemplateScore, error) {
	scores, err := tm.MatchTemplates(ctx, requirements, availableTemplates)
	if err != nil {
		return nil, err
	}

	if len(scores) == 0 {
		return nil, fmt.Errorf("no matching templates found")
	}

	return &scores[0], nil
}

// calculateTemplateScore calculates how well a template matches the requirements
func (tm *TemplateMatcher) calculateTemplateScore(requirements *ParsedRequirements, template *TemplateInfo) TemplateScore {
	score := TemplateScore{
		TemplateID:   template.ID,
		TemplateName: template.Name,
		Score:        0,
		Reasons:      []string{},
	}

	// Check category match
	if tm.matchesCategory(requirements, template) {
		score.Score += 30
		score.Reasons = append(score.Reasons, "Category matches project intent")
	}

	// Check sensor compatibility
	sensorScore, sensorReasons := tm.scoreSensorCompatibility(requirements, template)
	score.Score += sensorScore
	score.Reasons = append(score.Reasons, sensorReasons...)

	// Check actuator compatibility
	actuatorScore, actuatorReasons := tm.scoreActuatorCompatibility(requirements, template)
	score.Score += actuatorScore
	score.Reasons = append(score.Reasons, actuatorReasons...)

	// Check communication compatibility
	commScore, commReasons := tm.scoreCommunicationCompatibility(requirements, template)
	score.Score += commScore
	score.Reasons = append(score.Reasons, commReasons...)

	// Check board compatibility
	if tm.matchesBoard(requirements, template) {
		score.Score += 10
		score.Reasons = append(score.Reasons, "Compatible with preferred board")
	}

	return score
}

// matchesCategory checks if template category matches requirements
func (tm *TemplateMatcher) matchesCategory(requirements *ParsedRequirements, template *TemplateInfo) bool {
	intent := strings.ToLower(requirements.Intent)
	category := strings.ToLower(template.Category)

	// Direct category match
	if strings.Contains(intent, category) {
		return true
	}

	// Check for sensor-related keywords
	if category == "sensing" {
		if len(requirements.Sensors) > 0 {
			return true
		}
	}

	// Check for automation keywords
	if category == "automation" {
		if len(requirements.Actuators) > 0 && len(requirements.Sensors) > 0 {
			return true
		}
	}

	// Check for communication keywords
	if category == "communication" {
		if len(requirements.Communication) > 0 {
			return true
		}
	}

	return false
}

// scoreSensorCompatibility scores sensor compatibility
func (tm *TemplateMatcher) scoreSensorCompatibility(requirements *ParsedRequirements, template *TemplateInfo) (float64, []string) {
	score := 0.0
	reasons := []string{}

	if len(requirements.Sensors) == 0 {
		return score, reasons
	}

	matchedSensors := 0
	for _, reqSensor := range requirements.Sensors {
		if tm.templateSupportsSensor(template, reqSensor.Type, reqSensor.Model) {
			matchedSensors++
		}
	}

	if matchedSensors > 0 {
		score = float64(matchedSensors) * 15.0
		reasons = append(reasons, fmt.Sprintf("Supports %d/%d required sensors", matchedSensors, len(requirements.Sensors)))
	}

	return score, reasons
}

// scoreActuatorCompatibility scores actuator compatibility
func (tm *TemplateMatcher) scoreActuatorCompatibility(requirements *ParsedRequirements, template *TemplateInfo) (float64, []string) {
	score := 0.0
	reasons := []string{}

	if len(requirements.Actuators) == 0 {
		return score, reasons
	}

	matchedActuators := 0
	for _, reqActuator := range requirements.Actuators {
		if tm.templateSupportsActuator(template, reqActuator.Type) {
			matchedActuators++
		}
	}

	if matchedActuators > 0 {
		score = float64(matchedActuators) * 15.0
		reasons = append(reasons, fmt.Sprintf("Supports %d/%d required actuators", matchedActuators, len(requirements.Actuators)))
	}

	return score, reasons
}

// scoreCommunicationCompatibility scores communication compatibility
func (tm *TemplateMatcher) scoreCommunicationCompatibility(requirements *ParsedRequirements, template *TemplateInfo) (float64, []string) {
	score := 0.0
	reasons := []string{}

	if len(requirements.Communication) == 0 {
		return score, reasons
	}

	matchedComm := 0
	for _, reqComm := range requirements.Communication {
		if tm.templateSupportsCommunication(template, reqComm.Protocol) {
			matchedComm++
		}
	}

	if matchedComm > 0 {
		score = float64(matchedComm) * 20.0
		reasons = append(reasons, fmt.Sprintf("Supports %d/%d communication protocols", matchedComm, len(requirements.Communication)))
	}

	return score, reasons
}

// matchesBoard checks if template supports the preferred board
func (tm *TemplateMatcher) matchesBoard(requirements *ParsedRequirements, template *TemplateInfo) bool {
	if requirements.BoardPreference == "" {
		return true // No preference specified
	}

	boardPref := strings.ToLower(requirements.BoardPreference)
	for _, supportedBoard := range template.BoardsSupported {
		if strings.Contains(strings.ToLower(supportedBoard), boardPref) {
			return true
		}
	}

	return false
}

// templateSupportsSensor checks if template supports a sensor type
func (tm *TemplateMatcher) templateSupportsSensor(template *TemplateInfo, sensorType, sensorModel string) bool {
	description := strings.ToLower(template.Description)
	name := strings.ToLower(template.Name)

	// Check sensor type
	if strings.Contains(description, strings.ToLower(sensorType)) ||
		strings.Contains(name, strings.ToLower(sensorType)) {
		return true
	}

	// Check sensor model if specified
	if sensorModel != "" {
		if strings.Contains(description, strings.ToLower(sensorModel)) ||
			strings.Contains(name, strings.ToLower(sensorModel)) {
			return true
		}
	}

	// Check required sensors list
	for _, reqSensor := range template.RequiredSensors {
		if strings.Contains(strings.ToLower(reqSensor), strings.ToLower(sensorType)) {
			return true
		}
		if sensorModel != "" && strings.Contains(strings.ToLower(reqSensor), strings.ToLower(sensorModel)) {
			return true
		}
	}

	return false
}

// templateSupportsActuator checks if template supports an actuator type
func (tm *TemplateMatcher) templateSupportsActuator(template *TemplateInfo, actuatorType string) bool {
	description := strings.ToLower(template.Description)
	name := strings.ToLower(template.Name)

	if strings.Contains(description, strings.ToLower(actuatorType)) ||
		strings.Contains(name, strings.ToLower(actuatorType)) {
		return true
	}

	return false
}

// templateSupportsCommunication checks if template supports a communication protocol
func (tm *TemplateMatcher) templateSupportsCommunication(template *TemplateInfo, protocol string) bool {
	description := strings.ToLower(template.Description)
	name := strings.ToLower(template.Name)

	if strings.Contains(description, strings.ToLower(protocol)) ||
		strings.Contains(name, strings.ToLower(protocol)) {
		return true
	}

	// Check libraries for communication support
	for _, lib := range template.Libraries {
		libName := strings.ToLower(lib)
		if strings.Contains(libName, strings.ToLower(protocol)) {
			return true
		}

		// Check for specific library names
		if protocol == "wifi" && (strings.Contains(libName, "wifi") || strings.Contains(libName, "esp")) {
			return true
		}
		if protocol == "mqtt" && strings.Contains(libName, "pubsub") {
			return true
		}
		if protocol == "bluetooth" && (strings.Contains(libName, "ble") || strings.Contains(libName, "bluetooth")) {
			return true
		}
	}

	return false
}

// sortByScore sorts template scores in descending order
func (tm *TemplateMatcher) sortByScore(scores []TemplateScore) []TemplateScore {
	if len(scores) <= 1 {
		return scores
	}

	// Simple bubble sort
	sorted := make([]TemplateScore, len(scores))
	copy(sorted, scores)

	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Score < sorted[j+1].Score {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	return sorted
}

// TemplateInfo represents template information for matching
type TemplateInfo struct {
	ID              string
	Name            string
	Category        string
	Description     string
	BoardsSupported []string
	RequiredSensors []string
	Libraries       []string
}
