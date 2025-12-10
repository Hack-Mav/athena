package template

import (
	"context"
)

// TemplateService defines the interface for template operations
type TemplateService interface {
	// Template management
	ListTemplates(ctx context.Context, filters *TemplateFilters) ([]*Template, error)
	GetTemplate(ctx context.Context, id string, version string) (*Template, error)
	CreateTemplate(ctx context.Context, template *Template) error
	UpdateTemplate(ctx context.Context, template *Template) error
	DeleteTemplate(ctx context.Context, id string, version string) error

	// Template validation and rendering
	ValidateTemplate(ctx context.Context, template *Template) (*ValidationResult, error)
	ValidateParameters(ctx context.Context, template *Template, parameters map[string]interface{}) (*ValidationResult, error)
	ValidateBoardCapabilities(ctx context.Context, template *Template, boardType string, parameters map[string]interface{}) (*ValidationResult, error)
	RenderTemplate(ctx context.Context, id string, version string, parameters map[string]interface{}) (*RenderedTemplate, error)

	// Wiring diagram generation
	GenerateWiringDiagram(ctx context.Context, template *Template, parameters map[string]interface{}) (*WiringDiagram, error)

	// Template search and discovery
	SearchTemplates(ctx context.Context, query string, filters *TemplateFilters) ([]*Template, error)
	GetTemplateVersions(ctx context.Context, id string) ([]string, error)
	GetTemplateCount(ctx context.Context, filters *TemplateFilters) (int64, error)

	// Asset management
	CreateAsset(ctx context.Context, templateID, templateVersion string, asset *Asset) error
	GetAssets(ctx context.Context, templateID, templateVersion string) ([]*Asset, error)
	DeleteAsset(ctx context.Context, templateID, templateVersion, assetType, assetPath string) error
}
