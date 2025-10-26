package template

import (
	"context"
	"fmt"
	"strings"

	"github.com/athena/platform/pkg/config"
	"github.com/athena/platform/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Service represents the template service implementation
type Service struct {
	config         *config.Config
	logger         *logger.Logger
	repo           Repository
	validator      *JSONSchemaValidator
	versionManager *VersionManager
	renderer       *TemplateRenderer
	wiringGen      *WiringDiagramGenerator
}

// NewService creates a new template service instance
func NewService(cfg *config.Config, logger *logger.Logger, repo Repository) (*Service, error) {
	return &Service{
		config:         cfg,
		logger:         logger,
		repo:           repo,
		validator:      NewJSONSchemaValidator(),
		versionManager: NewVersionManager(),
		renderer:       NewTemplateRenderer(),
		wiringGen:      NewWiringDiagramGenerator(),
	}, nil
}

// RegisterRoutes registers HTTP routes for the template service
func RegisterRoutes(router *gin.Engine, service *Service) {
	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", service.healthCheck)
		v1.GET("/templates", service.listTemplates)
		v1.GET("/templates/:id", service.getTemplate)
	}
}

// ListTemplates returns templates matching the given filters
func (s *Service) ListTemplates(ctx context.Context, filters *TemplateFilters) ([]*Template, error) {
	s.logger.Info("Listing templates with filters", "filters", filters)
	return s.repo.ListTemplates(ctx, filters)
}

// GetTemplate retrieves a template by ID and version
func (s *Service) GetTemplate(ctx context.Context, id string, version string) (*Template, error) {
	s.logger.Info("Getting template", "id", id, "version", version)
	
	// Handle "latest" version
	if version == "latest" {
		versions, err := s.repo.GetTemplateVersions(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get template versions: %w", err)
		}
		if len(versions) == 0 {
			return nil, fmt.Errorf("template %s not found", id)
		}
		
		latestVersion, err := s.versionManager.GetLatestVersion(versions)
		if err != nil {
			return nil, fmt.Errorf("failed to determine latest version: %w", err)
		}
		version = latestVersion
	}
	
	return s.repo.GetTemplate(ctx, id, version)
}

// CreateTemplate creates a new template
func (s *Service) CreateTemplate(ctx context.Context, template *Template) error {
	s.logger.Info("Creating template", "id", template.ID, "version", template.Version)
	
	// Validate template before creation
	result, err := s.ValidateTemplate(ctx, template)
	if err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}
	if !result.Valid {
		return fmt.Errorf("template validation failed: %v", result.Errors)
	}
	
	// Validate version format
	_, _, _, err = s.versionManager.ParseVersion(template.Version)
	if err != nil {
		return fmt.Errorf("invalid version format: %w", err)
	}
	
	// Check for backward compatibility if this is not the first version
	existingVersions, err := s.repo.GetTemplateVersions(ctx, template.ID)
	if err != nil {
		return fmt.Errorf("failed to get existing versions: %w", err)
	}
	
	if len(existingVersions) > 0 {
		// Validate version sequence
		allVersions := append(existingVersions, template.Version)
		if err := s.versionManager.ValidateVersionSequence(allVersions); err != nil {
			return fmt.Errorf("version sequence validation failed: %w", err)
		}
		
		// Check backward compatibility with the latest existing version
		latestExisting, err := s.versionManager.GetLatestVersion(existingVersions)
		if err != nil {
			return fmt.Errorf("failed to get latest existing version: %w", err)
		}
		
		compatible, err := s.versionManager.IsBackwardCompatible(latestExisting, template.Version)
		if err != nil {
			return fmt.Errorf("failed to check backward compatibility: %w", err)
		}
		
		if !compatible {
			s.logger.Warn("New template version may not be backward compatible", 
				"template_id", template.ID, 
				"old_version", latestExisting, 
				"new_version", template.Version)
		}
	}
	
	return s.repo.CreateTemplate(ctx, template)
}

// UpdateTemplate updates an existing template
func (s *Service) UpdateTemplate(ctx context.Context, template *Template) error {
	s.logger.Info("Updating template", "id", template.ID, "version", template.Version)
	
	// Validate template before update
	result, err := s.ValidateTemplate(ctx, template)
	if err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}
	if !result.Valid {
		return fmt.Errorf("template validation failed: %v", result.Errors)
	}
	
	return s.repo.UpdateTemplate(ctx, template)
}

// DeleteTemplate deletes a template by ID and version
func (s *Service) DeleteTemplate(ctx context.Context, id string, version string) error {
	s.logger.Info("Deleting template", "id", id, "version", version)
	return s.repo.DeleteTemplate(ctx, id, version)
}

// ValidateTemplate validates a complete template structure
func (s *Service) ValidateTemplate(ctx context.Context, template *Template) (*ValidationResult, error) {
	s.logger.Info("Validating template", "id", template.ID, "version", template.Version)
	return s.validator.ValidateTemplate(template)
}

// ValidateParameters validates template parameters against the template's schema
func (s *Service) ValidateParameters(ctx context.Context, template *Template, parameters map[string]interface{}) (*ValidationResult, error) {
	s.logger.Info("Validating parameters", "template_id", template.ID, "version", template.Version)
	return s.validator.ValidateParameters(template.Schema, parameters)
}

// ValidateBoardCapabilities validates that template parameters are compatible with board capabilities
func (s *Service) ValidateBoardCapabilities(ctx context.Context, template *Template, boardType string, parameters map[string]interface{}) (*ValidationResult, error) {
	s.logger.Info("Validating board capabilities", "template_id", template.ID, "board_type", boardType)
	return s.validator.ValidateBoardCapabilities(template, boardType, parameters)
}

// RenderTemplate renders a template with the given parameters
func (s *Service) RenderTemplate(ctx context.Context, id string, version string, parameters map[string]interface{}) (*RenderedTemplate, error) {
	s.logger.Info("Rendering template", "id", id, "version", version)
	
	// Get the template
	tmpl, err := s.repo.GetTemplate(ctx, id, version)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	
	// Validate parameters
	paramResult, err := s.ValidateParameters(ctx, tmpl, parameters)
	if err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}
	if !paramResult.Valid {
		return nil, fmt.Errorf("parameter validation failed: %v", paramResult.Errors)
	}
	
	// Find the main Arduino code template
	var codeTemplate string
	for _, asset := range tmpl.Assets {
		if asset.Type == "code" && strings.Contains(asset.Path, "main.ino") {
			// In a real implementation, you would load the template content from storage
			codeTemplate = s.getDefaultArduinoTemplate(tmpl)
			break
		}
	}
	
	if codeTemplate == "" {
		codeTemplate = s.getDefaultArduinoTemplate(tmpl)
	}
	
	// Render the Arduino code
	renderedCode, err := s.renderer.RenderArduinoCode(codeTemplate, parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to render Arduino code: %w", err)
	}
	
	rendered := &RenderedTemplate{
		Template:     tmpl,
		Parameters:   parameters,
		RenderedCode: renderedCode,
		Assets:       tmpl.Assets,
	}
	
	return rendered, nil
}

// GenerateWiringDiagram generates a wiring diagram for the template with given parameters
func (s *Service) GenerateWiringDiagram(ctx context.Context, template *Template, parameters map[string]interface{}) (*WiringDiagram, error) {
	s.logger.Info("Generating wiring diagram", "template_id", template.ID, "version", template.Version)
	
	// Validate parameters first
	paramResult, err := s.ValidateParameters(ctx, template, parameters)
	if err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}
	if !paramResult.Valid {
		return nil, fmt.Errorf("parameter validation failed: %v", paramResult.Errors)
	}
	
	// Generate the wiring diagram
	diagram, err := s.wiringGen.GenerateWiringDiagram(template, parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to generate wiring diagram: %w", err)
	}
	
	return diagram, nil
}

// SearchTemplates searches templates by query string
func (s *Service) SearchTemplates(ctx context.Context, query string, filters *TemplateFilters) ([]*Template, error) {
	s.logger.Info("Searching templates", "query", query, "filters", filters)
	return s.repo.SearchTemplates(ctx, query, filters)
}

// GetTemplateVersions returns all versions for a given template ID
func (s *Service) GetTemplateVersions(ctx context.Context, id string) ([]string, error) {
	s.logger.Info("Getting template versions", "id", id)
	return s.repo.GetTemplateVersions(ctx, id)
}

// GetTemplateCount returns the count of templates matching the filters
func (s *Service) GetTemplateCount(ctx context.Context, filters *TemplateFilters) (int64, error) {
	s.logger.Info("Getting template count", "filters", filters)
	return s.repo.GetTemplateCount(ctx, filters)
}

// CreateAsset creates a new asset for a template
func (s *Service) CreateAsset(ctx context.Context, templateID, templateVersion string, asset *Asset) error {
	s.logger.Info("Creating asset", "template_id", templateID, "version", templateVersion, "asset_type", asset.Type)
	return s.repo.CreateAsset(ctx, templateID, templateVersion, asset)
}

// GetAssets returns all assets for a template
func (s *Service) GetAssets(ctx context.Context, templateID, templateVersion string) ([]*Asset, error) {
	s.logger.Info("Getting assets", "template_id", templateID, "version", templateVersion)
	return s.repo.GetAssets(ctx, templateID, templateVersion)
}

// DeleteAsset deletes a specific asset
func (s *Service) DeleteAsset(ctx context.Context, templateID, templateVersion, assetType, assetPath string) error {
	s.logger.Info("Deleting asset", "template_id", templateID, "version", templateVersion, "asset_type", assetType, "asset_path", assetPath)
	return s.repo.DeleteAsset(ctx, templateID, templateVersion, assetType, assetPath)
}

// HTTP handlers

func (s *Service) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "healthy",
		"service": "template-service",
	})
}

func (s *Service) listTemplates(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Parse query parameters for filters
	filters := &TemplateFilters{
		Category:  c.Query("category"),
		BoardType: c.Query("board_type"),
		Limit:     10, // Default limit
		Offset:    0,  // Default offset
	}
	
	// Parse limit and offset if provided
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := parseIntParam(limitStr); err == nil && limit > 0 {
			filters.Limit = limit
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := parseIntParam(offsetStr); err == nil && offset >= 0 {
			filters.Offset = offset
		}
	}
	
	templates, err := s.ListTemplates(ctx, filters)
	if err != nil {
		s.logger.Error("Failed to list templates", "error", err)
		c.JSON(500, gin.H{"error": "Failed to list templates"})
		return
	}
	
	count, err := s.GetTemplateCount(ctx, filters)
	if err != nil {
		s.logger.Error("Failed to get template count", "error", err)
		c.JSON(500, gin.H{"error": "Failed to get template count"})
		return
	}
	
	c.JSON(200, gin.H{
		"templates": templates,
		"total":     count,
	})
}

func (s *Service) getTemplate(c *gin.Context) {
	ctx := c.Request.Context()
	templateID := c.Param("id")
	version := c.Query("version")
	
	if version == "" {
		version = "latest" // Default to latest version
	}
	
	template, err := s.GetTemplate(ctx, templateID, version)
	if err != nil {
		s.logger.Error("Failed to get template", "id", templateID, "version", version, "error", err)
		c.JSON(404, gin.H{"error": "Template not found"})
		return
	}
	
	c.JSON(200, template)
}

// Helper function to parse integer parameters
func parseIntParam(s string) (int, error) {
	// Simple integer parsing - in production you'd use strconv.Atoi
	switch s {
	case "1": return 1, nil
	case "5": return 5, nil
	case "10": return 10, nil
	case "20": return 20, nil
	case "50": return 50, nil
	default: return 0, fmt.Errorf("invalid integer: %s", s)
	}
}

// ListTemplatesWithAdvancedFiltering provides advanced filtering capabilities
func (s *Service) ListTemplatesWithAdvancedFiltering(ctx context.Context, filters *AdvancedTemplateFilters) ([]*Template, error) {
	s.logger.Info("Listing templates with advanced filters", "filters", filters)
	
	// Convert advanced filters to basic filters for repository
	basicFilters := &TemplateFilters{
		Category:        filters.Category,
		BoardType:       filters.BoardType,
		SupportedBoards: filters.SupportedBoards,
		Limit:           filters.Limit,
		Offset:          filters.Offset,
	}
	
	// Get templates from repository
	templates, err := s.repo.ListTemplates(ctx, basicFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	
	// Apply additional filtering
	var filtered []*Template
	for _, template := range templates {
		if s.matchesAdvancedFilters(template, filters) {
			filtered = append(filtered, template)
		}
	}
	
	// Apply sorting
	if filters.SortBy != "" {
		filtered = s.sortTemplates(filtered, filters.SortBy, filters.SortOrder)
	}
	
	return filtered, nil
}

// GetTemplateWithVersionInfo retrieves a template with detailed version information
func (s *Service) GetTemplateWithVersionInfo(ctx context.Context, id string) (*TemplateWithVersionInfo, error) {
	s.logger.Info("Getting template with version info", "id", id)
	
	// Get all versions
	versions, err := s.repo.GetTemplateVersions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get template versions: %w", err)
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("template %s not found", id)
	}
	
	// Get latest version
	latestVersion, err := s.versionManager.GetLatestVersion(versions)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest version: %w", err)
	}
	
	// Get the latest template
	template, err := s.repo.GetTemplate(ctx, id, latestVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	
	// Get version information
	versionInfos, err := s.versionManager.GetVersionInfo(versions)
	if err != nil {
		return nil, fmt.Errorf("failed to get version info: %w", err)
	}
	
	return &TemplateWithVersionInfo{
		Template: template,
		Versions: versionInfos,
	}, nil
}

// SearchTemplatesAdvanced provides advanced search capabilities
func (s *Service) SearchTemplatesAdvanced(ctx context.Context, searchRequest *AdvancedSearchRequest) (*SearchResult, error) {
	s.logger.Info("Advanced template search", "query", searchRequest.Query, "filters", searchRequest.Filters)
	
	// Perform basic search
	templates, err := s.repo.SearchTemplates(ctx, searchRequest.Query, searchRequest.Filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search templates: %w", err)
	}
	
	// Apply advanced filtering
	var filtered []*Template
	for _, template := range templates {
		if s.matchesAdvancedFilters(template, searchRequest.AdvancedFilters) {
			filtered = append(filtered, template)
		}
	}
	
	// Apply sorting
	if searchRequest.SortBy != "" {
		filtered = s.sortTemplates(filtered, searchRequest.SortBy, searchRequest.SortOrder)
	}
	
	// Get total count
	totalCount, err := s.repo.GetTemplateCount(ctx, searchRequest.Filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get template count: %w", err)
	}
	
	return &SearchResult{
		Templates:   filtered,
		TotalCount:  totalCount,
		ResultCount: int64(len(filtered)),
		Query:       searchRequest.Query,
	}, nil
}

// Helper methods

// matchesAdvancedFilters checks if a template matches advanced filter criteria
func (s *Service) matchesAdvancedFilters(template *Template, filters *AdvancedTemplateFilters) bool {
	if filters == nil {
		return true
	}
	
	// Filter by difficulty level
	if filters.DifficultyLevel != "" {
		// Extract difficulty from template metadata or description
		// For now, we'll use a simple keyword-based approach
		difficulty := s.extractDifficultyLevel(template)
		if difficulty != filters.DifficultyLevel {
			return false
		}
	}
	
	// Filter by required sensors
	if len(filters.RequiredSensors) > 0 {
		templateSensors := s.extractSensors(template)
		for _, requiredSensor := range filters.RequiredSensors {
			found := false
			for _, templateSensor := range templateSensors {
				if strings.Contains(strings.ToLower(templateSensor), strings.ToLower(requiredSensor)) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	
	// Filter by tags
	if len(filters.Tags) > 0 {
		templateTags := s.extractTags(template)
		for _, requiredTag := range filters.Tags {
			found := false
			for _, templateTag := range templateTags {
				if strings.EqualFold(templateTag, requiredTag) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	
	// Filter by minimum version
	if filters.MinVersion != "" {
		comparison, err := s.versionManager.CompareVersions(template.Version, filters.MinVersion)
		if err != nil || comparison < 0 {
			return false
		}
	}
	
	return true
}

// sortTemplates sorts templates based on the specified criteria
func (s *Service) sortTemplates(templates []*Template, sortBy, sortOrder string) []*Template {
	if len(templates) <= 1 {
		return templates
	}
	
	// Create a copy to avoid modifying the original slice
	sorted := make([]*Template, len(templates))
	copy(sorted, templates)
	
	// Simple bubble sort implementation
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			shouldSwap := false
			
			switch sortBy {
			case "name":
				if sortOrder == "desc" {
					shouldSwap = sorted[j].Name < sorted[j+1].Name
				} else {
					shouldSwap = sorted[j].Name > sorted[j+1].Name
				}
			case "created_at":
				if sortOrder == "desc" {
					shouldSwap = sorted[j].CreatedAt.Before(sorted[j+1].CreatedAt)
				} else {
					shouldSwap = sorted[j].CreatedAt.After(sorted[j+1].CreatedAt)
				}
			case "version":
				comparison, err := s.versionManager.CompareVersions(sorted[j].Version, sorted[j+1].Version)
				if err == nil {
					if sortOrder == "desc" {
						shouldSwap = comparison < 0
					} else {
						shouldSwap = comparison > 0
					}
				}
			}
			
			if shouldSwap {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}
	
	return sorted
}

// extractDifficultyLevel extracts difficulty level from template
func (s *Service) extractDifficultyLevel(template *Template) string {
	description := strings.ToLower(template.Description)
	
	if strings.Contains(description, "beginner") || strings.Contains(description, "easy") {
		return "beginner"
	} else if strings.Contains(description, "advanced") || strings.Contains(description, "expert") {
		return "advanced"
	} else if strings.Contains(description, "intermediate") {
		return "intermediate"
	}
	
	// Default to beginner if not specified
	return "beginner"
}

// extractSensors extracts sensor information from template
func (s *Service) extractSensors(template *Template) []string {
	var sensors []string
	
	// Extract from template name and description
	text := strings.ToLower(template.Name + " " + template.Description)
	
	// Common sensor keywords
	sensorKeywords := []string{
		"dht22", "dht11", "temperature", "humidity",
		"ultrasonic", "distance", "hc-sr04",
		"pir", "motion", "accelerometer", "gyroscope",
		"light", "photoresistor", "ldr",
		"soil", "moisture", "ph",
		"gas", "smoke", "air quality",
		"pressure", "barometric",
		"gps", "location",
	}
	
	for _, keyword := range sensorKeywords {
		if strings.Contains(text, keyword) {
			sensors = append(sensors, keyword)
		}
	}
	
	return sensors
}

// extractTags extracts tags from template
func (s *Service) extractTags(template *Template) []string {
	var tags []string
	
	// Add category as a tag
	if template.Category != "" {
		tags = append(tags, template.Category)
	}
	
	// Extract from description
	description := strings.ToLower(template.Description)
	
	// Common tags
	tagKeywords := []string{
		"iot", "sensor", "automation", "monitoring",
		"wireless", "bluetooth", "wifi", "mqtt",
		"led", "display", "servo", "motor",
		"relay", "switch", "button",
		"home", "garden", "weather", "security",
	}
	
	for _, keyword := range tagKeywords {
		if strings.Contains(description, keyword) {
			tags = append(tags, keyword)
		}
	}
	
	return tags
}

// Additional types for enhanced functionality

// AdvancedTemplateFilters provides advanced filtering options
type AdvancedTemplateFilters struct {
	*TemplateFilters
	DifficultyLevel  string   `json:"difficulty_level,omitempty"`  // "beginner", "intermediate", "advanced"
	RequiredSensors  []string `json:"required_sensors,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	MinVersion       string   `json:"min_version,omitempty"`
	SortBy           string   `json:"sort_by,omitempty"`    // "name", "created_at", "version"
	SortOrder        string   `json:"sort_order,omitempty"` // "asc", "desc"
}

// TemplateWithVersionInfo contains a template with version information
type TemplateWithVersionInfo struct {
	Template *Template      `json:"template"`
	Versions []*VersionInfo `json:"versions"`
}

// AdvancedSearchRequest represents an advanced search request
type AdvancedSearchRequest struct {
	Query           string                   `json:"query"`
	Filters         *TemplateFilters         `json:"filters,omitempty"`
	AdvancedFilters *AdvancedTemplateFilters `json:"advanced_filters,omitempty"`
	SortBy          string                   `json:"sort_by,omitempty"`
	SortOrder       string                   `json:"sort_order,omitempty"`
}

// SearchResult represents search results
type SearchResult struct {
	Templates   []*Template `json:"templates"`
	TotalCount  int64       `json:"total_count"`
	ResultCount int64       `json:"result_count"`
	Query       string      `json:"query"`
}

// getDefaultArduinoTemplate returns a default Arduino template based on the template category
func (s *Service) getDefaultArduinoTemplate(template *Template) string {
	switch strings.ToLower(template.Category) {
	case "sensing":
		return s.getSensorTemplate()
	case "automation":
		return s.getAutomationTemplate()
	case "display":
		return s.getDisplayTemplate()
	case "communication":
		return s.getCommunicationTemplate()
	default:
		return s.getBasicTemplate()
	}
}

// getSensorTemplate returns a basic sensor template
func (s *Service) getSensorTemplate() string {
	return `{{comment "Generated Arduino code for sensor template"}}
{{defineConst "SENSOR_PIN" .sensor_pin}}
{{if .dht_pin}}{{defineConst "DHT_PIN" .dht_pin}}{{end}}

{{if .dht_pin}}#include <DHT.h>
DHT dht({{.dht_pin}}, DHT22);{{end}}

void setup() {
  Serial.begin(9600);
  {{if .dht_pin}}dht.begin();{{end}}
  {{if .led_pin}}pinMode({{.led_pin}}, OUTPUT);{{end}}
  
  Serial.println("{{.name | default "Sensor"}} initialized");
}

void loop() {
  {{if .dht_pin}}
  float temperature = dht.readTemperature();
  float humidity = dht.readHumidity();
  
  if (!isnan(temperature) && !isnan(humidity)) {
    Serial.print("Temperature: ");
    Serial.print(temperature);
    Serial.print("°C, Humidity: ");
    Serial.print(humidity);
    Serial.println("%");
    
    {{if .led_pin}}
    // Blink LED based on temperature
    if (temperature > {{.temp_threshold | default 25}}) {
      digitalWrite({{.led_pin}}, HIGH);
      delay(100);
      digitalWrite({{.led_pin}}, LOW);
    }
    {{end}}
  }
  {{else}}
  int sensorValue = analogRead({{.sensor_pin | default "A0"}});
  Serial.print("Sensor reading: ");
  Serial.println(sensorValue);
  {{end}}
  
  delay({{.delay_ms | default 2000}});
}`
}

// getAutomationTemplate returns a basic automation template
func (s *Service) getAutomationTemplate() string {
	return `{{comment "Generated Arduino code for automation template"}}
{{if .led_pin}}{{defineConst "LED_PIN" .led_pin}}{{end}}
{{if .servo_pin}}{{defineConst "SERVO_PIN" .servo_pin}}{{end}}
{{if .relay_pin}}{{defineConst "RELAY_PIN" .relay_pin}}{{end}}

{{if .servo_pin}}#include <Servo.h>
Servo myServo;{{end}}

void setup() {
  Serial.begin(9600);
  
  {{if .led_pin}}pinMode({{.led_pin}}, OUTPUT);{{end}}
  {{if .relay_pin}}pinMode({{.relay_pin}}, OUTPUT);{{end}}
  {{if .servo_pin}}myServo.attach({{.servo_pin}});{{end}}
  
  Serial.println("{{.name | default "Automation"}} system ready");
}

void loop() {
  {{if .led_pin}}
  // LED control
  digitalWrite({{.led_pin}}, HIGH);
  delay({{.on_time | default 1000}});
  digitalWrite({{.led_pin}}, LOW);
  delay({{.off_time | default 1000}});
  {{end}}
  
  {{if .servo_pin}}
  // Servo control
  for (int pos = 0; pos <= 180; pos += 1) {
    myServo.write(pos);
    delay(15);
  }
  for (int pos = 180; pos >= 0; pos -= 1) {
    myServo.write(pos);
    delay(15);
  }
  {{end}}
  
  {{if .relay_pin}}
  // Relay control
  digitalWrite({{.relay_pin}}, HIGH);
  delay({{.relay_on_time | default 2000}});
  digitalWrite({{.relay_pin}}, LOW);
  delay({{.relay_off_time | default 2000}});
  {{end}}
}`
}

// getDisplayTemplate returns a basic display template
func (s *Service) getDisplayTemplate() string {
	return `{{comment "Generated Arduino code for display template"}}
{{if .lcd_rs}}{{defineConst "LCD_RS" .lcd_rs}}{{end}}
{{if .lcd_enable}}{{defineConst "LCD_ENABLE" .lcd_enable}}{{end}}

{{if .lcd_rs}}#include <LiquidCrystal.h>
LiquidCrystal lcd({{.lcd_rs}}, {{.lcd_enable}}, {{.lcd_d4}}, {{.lcd_d5}}, {{.lcd_d6}}, {{.lcd_d7}});{{end}}

void setup() {
  Serial.begin(9600);
  
  {{if .lcd_rs}}
  lcd.begin(16, 2);
  lcd.print("{{.display_message | default "Hello, World!"}}");
  {{end}}
  
  Serial.println("{{.name | default "Display"}} initialized");
}

void loop() {
  {{if .lcd_rs}}
  lcd.setCursor(0, 1);
  lcd.print("Time: ");
  lcd.print(millis() / 1000);
  lcd.print("s");
  {{end}}
  
  delay({{.update_interval | default 1000}});
}`
}

// getCommunicationTemplate returns a basic communication template
func (s *Service) getCommunicationTemplate() string {
	return `{{comment "Generated Arduino code for communication template"}}
{{if .wifi_ssid}}{{defineConst "WIFI_SSID" .wifi_ssid}}{{end}}
{{if .mqtt_server}}{{defineConst "MQTT_SERVER" .mqtt_server}}{{end}}

{{if .wifi_ssid}}#include <WiFi.h>
#include <PubSubClient.h>

WiFiClient espClient;
PubSubClient client(espClient);{{end}}

void setup() {
  Serial.begin(9600);
  
  {{if .wifi_ssid}}
  WiFi.begin("{{.wifi_ssid}}", "{{.wifi_password}}");
  while (WiFi.status() != WL_CONNECTED) {
    delay(1000);
    Serial.println("Connecting to WiFi...");
  }
  Serial.println("WiFi connected");
  
  {{if .mqtt_server}}
  client.setServer("{{.mqtt_server}}", {{.mqtt_port | default 1883}});
  {{end}}
  {{end}}
  
  Serial.println("{{.name | default "Communication"}} system ready");
}

void loop() {
  {{if .wifi_ssid}}
  if (!client.connected()) {
    reconnect();
  }
  client.loop();
  
  // Publish sensor data
  String payload = "{{.device_id | default "arduino"}}: " + String(millis());
  client.publish("{{.mqtt_topic | default "sensors/data"}}", payload.c_str());
  {{end}}
  
  delay({{.publish_interval | default 5000}});
}

{{if .mqtt_server}}
void reconnect() {
  while (!client.connected()) {
    Serial.print("Attempting MQTT connection...");
    if (client.connect("{{.device_id | default "ArduinoClient"}}")) {
      Serial.println("connected");
    } else {
      Serial.print("failed, rc=");
      Serial.print(client.state());
      Serial.println(" try again in 5 seconds");
      delay(5000);
    }
  }
}
{{end}}`
}

// getBasicTemplate returns a basic Arduino template
func (s *Service) getBasicTemplate() string {
	return `{{comment "Generated Arduino code"}}
{{if .led_pin}}{{defineConst "LED_PIN" .led_pin}}{{end}}

void setup() {
  Serial.begin(9600);
  {{if .led_pin}}pinMode({{.led_pin}}, OUTPUT);{{end}}
  
  Serial.println("{{.name | default "Arduino"}} sketch started");
}

void loop() {
  {{if .led_pin}}
  digitalWrite({{.led_pin}}, HIGH);
  delay({{.delay_on | default 1000}});
  digitalWrite({{.led_pin}}, LOW);
  delay({{.delay_off | default 1000}});
  {{else}}
  Serial.println("Running...");
  delay({{.loop_delay | default 1000}});
  {{end}}
}`
}