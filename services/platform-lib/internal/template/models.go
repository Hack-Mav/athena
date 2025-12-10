package template

import (
	"encoding/json"
	"time"
)

// Template represents an Arduino project template
type Template struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Version         string                 `json:"version"`
	Category        string                 `json:"category"`
	Description     string                 `json:"description"`
	BoardsSupported []string               `json:"boards_supported"`
	Schema          map[string]interface{} `json:"schema"`
	Parameters      map[string]interface{} `json:"parameters"`
	Libraries       []LibraryDependency    `json:"libraries"`
	Assets          []Asset                `json:"assets"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// LibraryDependency represents an Arduino library dependency
type LibraryDependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	URL     string `json:"url,omitempty"`
}

// Asset represents a template asset (wiring diagram, documentation, etc.)
type Asset struct {
	Type     string                 `json:"type"` // 'wiring_diagram', 'documentation', 'image'
	Path     string                 `json:"path"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TemplateEntity represents the Datastore entity for templates
type TemplateEntity struct {
	ID              string    `datastore:"id"`
	Name            string    `datastore:"name"`
	Version         string    `datastore:"version"`
	Category        string    `datastore:"category"`
	Description     string    `datastore:"description"`
	SchemaJSON      string    `datastore:"schema_json,noindex"`
	ParametersJSON  string    `datastore:"parameters_json,noindex"`
	BoardsSupported []string  `datastore:"boards_supported"`
	LibrariesJSON   string    `datastore:"libraries_json,noindex"`
	CreatedAt       time.Time `datastore:"created_at"`
	UpdatedAt       time.Time `datastore:"updated_at"`
}

// TemplateAssetEntity represents the Datastore entity for template assets
type TemplateAssetEntity struct {
	TemplateID      string    `datastore:"template_id"`
	TemplateVersion string    `datastore:"template_version"`
	AssetType       string    `datastore:"asset_type"`
	AssetPath       string    `datastore:"asset_path"`
	MetadataJSON    string    `datastore:"metadata_json,noindex"`
	CreatedAt       time.Time `datastore:"created_at"`
}

// TemplateFilters represents filters for template queries
type TemplateFilters struct {
	Category        string   `json:"category,omitempty"`
	BoardType       string   `json:"board_type,omitempty"`
	SupportedBoards []string `json:"supported_boards,omitempty"`
	Limit           int      `json:"limit,omitempty"`
	Offset          int      `json:"offset,omitempty"`
}

// ValidationResult represents the result of template validation
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// RenderedTemplate represents a template with rendered parameters
type RenderedTemplate struct {
	Template     *Template              `json:"template"`
	Parameters   map[string]interface{} `json:"parameters"`
	RenderedCode string                 `json:"rendered_code"`
	Assets       []Asset                `json:"assets"`
}

// WiringDiagram represents a generated wiring diagram
type WiringDiagram struct {
	MermaidSyntax string                 `json:"mermaid_syntax"`
	Components    []Component            `json:"components"`
	Connections   []Connection           `json:"connections"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Component represents a hardware component in a wiring diagram
type Component struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Name     string                 `json:"name"`
	Pins     []Pin                  `json:"pins"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Pin represents a component pin
type Pin struct {
	Number      string `json:"number"`
	Name        string `json:"name"`
	Type        string `json:"type"` // 'digital', 'analog', 'power', 'ground'
	Voltage     string `json:"voltage,omitempty"`
	Description string `json:"description,omitempty"`
}

// Connection represents a wiring connection between components
type Connection struct {
	FromComponent string `json:"from_component"`
	FromPin       string `json:"from_pin"`
	ToComponent   string `json:"to_component"`
	ToPin         string `json:"to_pin"`
	WireColor     string `json:"wire_color,omitempty"`
}

// ToEntity converts a Template to a TemplateEntity for Datastore storage
func (t *Template) ToEntity() (*TemplateEntity, error) {
	schemaJSON, err := json.Marshal(t.Schema)
	if err != nil {
		return nil, err
	}

	parametersJSON, err := json.Marshal(t.Parameters)
	if err != nil {
		return nil, err
	}

	librariesJSON, err := json.Marshal(t.Libraries)
	if err != nil {
		return nil, err
	}

	return &TemplateEntity{
		ID:              t.ID,
		Name:            t.Name,
		Version:         t.Version,
		Category:        t.Category,
		Description:     t.Description,
		SchemaJSON:      string(schemaJSON),
		ParametersJSON:  string(parametersJSON),
		BoardsSupported: t.BoardsSupported,
		LibrariesJSON:   string(librariesJSON),
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
	}, nil
}

// FromEntity converts a TemplateEntity to a Template
func (te *TemplateEntity) FromEntity() (*Template, error) {
	var schema map[string]interface{}
	if te.SchemaJSON != "" {
		if err := json.Unmarshal([]byte(te.SchemaJSON), &schema); err != nil {
			return nil, err
		}
	}

	var parameters map[string]interface{}
	if te.ParametersJSON != "" {
		if err := json.Unmarshal([]byte(te.ParametersJSON), &parameters); err != nil {
			return nil, err
		}
	}

	var libraries []LibraryDependency
	if te.LibrariesJSON != "" {
		if err := json.Unmarshal([]byte(te.LibrariesJSON), &libraries); err != nil {
			return nil, err
		}
	}

	return &Template{
		ID:              te.ID,
		Name:            te.Name,
		Version:         te.Version,
		Category:        te.Category,
		Description:     te.Description,
		BoardsSupported: te.BoardsSupported,
		Schema:          schema,
		Parameters:      parameters,
		Libraries:       libraries,
		Assets:          []Asset{}, // Assets are loaded separately
		CreatedAt:       te.CreatedAt,
		UpdatedAt:       te.UpdatedAt,
	}, nil
}

// ToAssetEntity converts an Asset to a TemplateAssetEntity for Datastore storage
func (a *Asset) ToAssetEntity(templateID, templateVersion string) (*TemplateAssetEntity, error) {
	metadataJSON, err := json.Marshal(a.Metadata)
	if err != nil {
		return nil, err
	}

	return &TemplateAssetEntity{
		TemplateID:      templateID,
		TemplateVersion: templateVersion,
		AssetType:       a.Type,
		AssetPath:       a.Path,
		MetadataJSON:    string(metadataJSON),
		CreatedAt:       time.Now(),
	}, nil
}

// FromAssetEntity converts a TemplateAssetEntity to an Asset
func (tae *TemplateAssetEntity) FromAssetEntity() (*Asset, error) {
	var metadata map[string]interface{}
	if tae.MetadataJSON != "" {
		if err := json.Unmarshal([]byte(tae.MetadataJSON), &metadata); err != nil {
			return nil, err
		}
	}

	return &Asset{
		Type:     tae.AssetType,
		Path:     tae.AssetPath,
		Metadata: metadata,
	}, nil
}
