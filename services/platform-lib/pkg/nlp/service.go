package nlp

import (
	"context"
	"fmt"
)

// Service represents the NLP planner service
type Service struct {
	llmClient       *LLMClient
	parser          *Parser
	templateMatcher *TemplateMatcher
	parameterFiller *ParameterFiller
	planGenerator   *PlanGenerator
	safetyValidator *SafetyValidator
}

// ServiceConfig represents NLP service configuration
type ServiceConfig struct {
	LLMEndpoint    string  `json:"llm_endpoint"`
	LLMAPIKey      string  `json:"llm_api_key"`
	LLMModel       string  `json:"llm_model"`
	LLMTemperature float64 `json:"llm_temperature"`
	LLMMaxTokens   int     `json:"llm_max_tokens"`
	LLMTimeout     int     `json:"llm_timeout"`
}

// NewService creates a new NLP service instance
func NewService(config *ServiceConfig) (*Service, error) {
	// Use provided configuration or defaults
	if config == nil {
		config = &ServiceConfig{
			LLMEndpoint:    "http://localhost:11434/v1/chat/completions",
			LLMModel:       "gpt-3.5-turbo",
			LLMTemperature: 0.7,
			LLMMaxTokens:   2000,
			LLMTimeout:     30,
		}
	}

	// Create LLM client
	llmClient := NewLLMClient(&LLMConfig{
		Endpoint:    config.LLMEndpoint,
		APIKey:      config.LLMAPIKey,
		Model:       config.LLMModel,
		Temperature: config.LLMTemperature,
		MaxTokens:   config.LLMMaxTokens,
		Timeout:     config.LLMTimeout,
	})

	// Create parser
	parser := NewParser(llmClient)

	// Create template matcher
	templateMatcher := NewTemplateMatcher(llmClient)

	// Create parameter filler
	parameterFiller := NewParameterFiller(llmClient)

	// Create plan generator
	planGenerator := NewPlanGenerator(llmClient)

	// Create safety validator
	safetyValidator := NewSafetyValidator()

	return &Service{
		llmClient:       llmClient,
		parser:          parser,
		templateMatcher: templateMatcher,
		parameterFiller: parameterFiller,
		planGenerator:   planGenerator,
		safetyValidator: safetyValidator,
	}, nil
}

// ParseRequirements parses natural language input and extracts requirements
func (s *Service) ParseRequirements(ctx context.Context, input string) (*ParsedRequirements, error) {
	if input == "" {
		return nil, fmt.Errorf("input cannot be empty")
	}

	// Parse requirements using LLM
	requirements, err := s.parser.ParseRequirements(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to parse requirements: %w", err)
	}

	return requirements, nil
}

// ClassifyIntent classifies the project intent
func (s *Service) ClassifyIntent(ctx context.Context, requirements *ParsedRequirements) (string, error) {
	category, err := s.parser.ClassifyIntent(ctx, requirements)
	if err != nil {
		return "", fmt.Errorf("failed to classify intent: %w", err)
	}

	return category, nil
}

// ExtractTechnicalSpecs extracts detailed technical specifications
func (s *Service) ExtractTechnicalSpecs(ctx context.Context, requirements *ParsedRequirements) (map[string]interface{}, error) {
	specs, err := s.parser.ExtractTechnicalSpecs(ctx, requirements)
	if err != nil {
		return nil, fmt.Errorf("failed to extract technical specs: %w", err)
	}

	return specs, nil
}

// SelectTemplate selects the best matching template for requirements
func (s *Service) SelectTemplate(ctx context.Context, requirements *ParsedRequirements, availableTemplates []TemplateInfo) (*TemplateScore, error) {
	template, err := s.templateMatcher.SelectBestTemplate(ctx, requirements, availableTemplates)
	if err != nil {
		return nil, fmt.Errorf("failed to select template: %w", err)
	}

	return template, nil
}

// FillTemplateParameters fills template parameters based on requirements
func (s *Service) FillTemplateParameters(ctx context.Context, requirements *ParsedRequirements, template *TemplateInfo, schema map[string]interface{}) (map[string]interface{}, error) {
	parameters, err := s.parameterFiller.FillParameters(ctx, requirements, template, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to fill parameters: %w", err)
	}

	return parameters, nil
}

// ValidateTemplateParameters validates filled parameters against board capabilities
func (s *Service) ValidateTemplateParameters(ctx context.Context, parameters map[string]interface{}, boardType string) (*ValidationResult, error) {
	result, err := s.parameterFiller.ValidateParameters(ctx, parameters, boardType)
	if err != nil {
		return nil, fmt.Errorf("failed to validate parameters: %w", err)
	}

	return result, nil
}

// GeneratePlan generates a complete implementation plan
func (s *Service) GeneratePlan(ctx context.Context, requirements *ParsedRequirements, template *TemplateInfo, parameters map[string]interface{}, boardType string) (*ImplementationPlan, error) {
	plan, err := s.planGenerator.GeneratePlan(ctx, requirements, template, parameters, boardType)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	return plan, nil
}

// ValidatePlan validates an implementation plan for safety
func (s *Service) ValidatePlan(ctx context.Context, plan *ImplementationPlan, boardType string) (*ValidationResult, error) {
	safetyValidation, err := s.safetyValidator.ValidateSafety(ctx, plan, boardType)
	if err != nil {
		return nil, fmt.Errorf("safety validation failed: %w", err)
	}

	result := &ValidationResult{
		Valid:    safetyValidation.Valid,
		Errors:   safetyValidation.Errors,
		Warnings: safetyValidation.Warnings,
	}

	return result, nil
}
