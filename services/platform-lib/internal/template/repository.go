package template

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Repository defines the interface for template data operations
type Repository interface {
	// Template CRUD operations
	CreateTemplate(ctx context.Context, template *Template) error
	GetTemplate(ctx context.Context, id, version string) (*Template, error)
	UpdateTemplate(ctx context.Context, template *Template) error
	DeleteTemplate(ctx context.Context, id, version string) error

	// Template querying
	ListTemplates(ctx context.Context, filters *TemplateFilters) ([]*Template, error)
	SearchTemplates(ctx context.Context, query string, filters *TemplateFilters) ([]*Template, error)
	GetTemplateVersions(ctx context.Context, id string) ([]string, error)

	// Asset operations
	CreateAsset(ctx context.Context, templateID, templateVersion string, asset *Asset) error
	GetAssets(ctx context.Context, templateID, templateVersion string) ([]*Asset, error)
	DeleteAsset(ctx context.Context, templateID, templateVersion, assetType, assetPath string) error

	// Utility operations
	TemplateExists(ctx context.Context, id, version string) (bool, error)
	GetTemplateCount(ctx context.Context, filters *TemplateFilters) (int64, error)
}

// MemoryRepository provides an in-memory implementation of the Repository interface
// This is useful for testing and development
type MemoryRepository struct {
	templates map[string]*Template
	assets    map[string][]*Asset // key: templateID#version
}

// NewMemoryRepository creates a new in-memory repository
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		templates: make(map[string]*Template),
		assets:    make(map[string][]*Asset),
	}
}

// CreateTemplate creates a new template in memory
func (r *MemoryRepository) CreateTemplate(ctx context.Context, template *Template) error {
	if template == nil {
		return fmt.Errorf("template cannot be nil")
	}

	key := fmt.Sprintf("%s#%s", template.ID, template.Version)

	// Check if template already exists
	if _, exists := r.templates[key]; exists {
		return fmt.Errorf("template %s version %s already exists", template.ID, template.Version)
	}

	// Set timestamps
	now := time.Now()
	template.CreatedAt = now
	template.UpdatedAt = now

	// Store template
	r.templates[key] = template

	// Store assets
	if len(template.Assets) > 0 {
		assets := make([]*Asset, len(template.Assets))
		for i := range template.Assets {
			assets[i] = &template.Assets[i]
		}
		r.assets[key] = assets
	}

	return nil
}

// GetTemplate retrieves a template by ID and version
func (r *MemoryRepository) GetTemplate(ctx context.Context, id, version string) (*Template, error) {
	if id == "" {
		return nil, fmt.Errorf("template ID cannot be empty")
	}
	if version == "" {
		return nil, fmt.Errorf("template version cannot be empty")
	}

	key := fmt.Sprintf("%s#%s", id, version)
	template, exists := r.templates[key]
	if !exists {
		return nil, fmt.Errorf("template %s version %s not found", id, version)
	}

	// Load assets
	if assets, hasAssets := r.assets[key]; hasAssets {
		template.Assets = make([]Asset, len(assets))
		for i, asset := range assets {
			template.Assets[i] = *asset
		}
	}

	return template, nil
}

// UpdateTemplate updates an existing template
func (r *MemoryRepository) UpdateTemplate(ctx context.Context, template *Template) error {
	if template == nil {
		return fmt.Errorf("template cannot be nil")
	}

	key := fmt.Sprintf("%s#%s", template.ID, template.Version)

	// Check if template exists
	if _, exists := r.templates[key]; !exists {
		return fmt.Errorf("template %s version %s not found", template.ID, template.Version)
	}

	// Update timestamp
	template.UpdatedAt = time.Now()

	// Store updated template
	r.templates[key] = template

	// Update assets
	if len(template.Assets) > 0 {
		assets := make([]*Asset, len(template.Assets))
		for i := range template.Assets {
			assets[i] = &template.Assets[i]
		}
		r.assets[key] = assets
	}

	return nil
}

// DeleteTemplate deletes a template by ID and version
func (r *MemoryRepository) DeleteTemplate(ctx context.Context, id, version string) error {
	if id == "" {
		return fmt.Errorf("template ID cannot be empty")
	}
	if version == "" {
		return fmt.Errorf("template version cannot be empty")
	}

	key := fmt.Sprintf("%s#%s", id, version)

	// Check if template exists
	if _, exists := r.templates[key]; !exists {
		return fmt.Errorf("template %s version %s not found", id, version)
	}

	// Delete template and assets
	delete(r.templates, key)
	delete(r.assets, key)

	return nil
}

// ListTemplates returns templates matching the given filters
func (r *MemoryRepository) ListTemplates(ctx context.Context, filters *TemplateFilters) ([]*Template, error) {
	var result []*Template

	for _, template := range r.templates {
		if r.matchesFilters(template, filters) {
			// Load assets
			key := fmt.Sprintf("%s#%s", template.ID, template.Version)
			if assets, hasAssets := r.assets[key]; hasAssets {
				template.Assets = make([]Asset, len(assets))
				for i, asset := range assets {
					template.Assets[i] = *asset
				}
			}
			result = append(result, template)
		}
	}

	// Apply pagination
	if filters != nil {
		if filters.Offset > 0 && filters.Offset < len(result) {
			result = result[filters.Offset:]
		}
		if filters.Limit > 0 && filters.Limit < len(result) {
			result = result[:filters.Limit]
		}
	}

	return result, nil
}

// SearchTemplates searches templates by query string
func (r *MemoryRepository) SearchTemplates(ctx context.Context, query string, filters *TemplateFilters) ([]*Template, error) {
	var result []*Template

	for _, template := range r.templates {
		if r.matchesQuery(template, query) && r.matchesFilters(template, filters) {
			// Load assets
			templateKey := fmt.Sprintf("%s#%s", template.ID, template.Version)
			if assets, hasAssets := r.assets[templateKey]; hasAssets {
				template.Assets = make([]Asset, len(assets))
				for i, asset := range assets {
					template.Assets[i] = *asset
				}
			}
			result = append(result, template)
		}
	}

	// Apply pagination
	if filters != nil {
		if filters.Offset > 0 && filters.Offset < len(result) {
			result = result[filters.Offset:]
		}
		if filters.Limit > 0 && filters.Limit < len(result) {
			result = result[:filters.Limit]
		}
	}

	return result, nil
}

// GetTemplateVersions returns all versions for a given template ID
func (r *MemoryRepository) GetTemplateVersions(ctx context.Context, id string) ([]string, error) {
	if id == "" {
		return nil, fmt.Errorf("template ID cannot be empty")
	}

	var versions []string
	for _, template := range r.templates {
		if template.ID == id {
			versions = append(versions, template.Version)
		}
	}

	return versions, nil
}

// CreateAsset creates a new asset for a template
func (r *MemoryRepository) CreateAsset(ctx context.Context, templateID, templateVersion string, asset *Asset) error {
	if templateID == "" || templateVersion == "" {
		return fmt.Errorf("template ID and version cannot be empty")
	}
	if asset == nil {
		return fmt.Errorf("asset cannot be nil")
	}

	templateKey := fmt.Sprintf("%s#%s", templateID, templateVersion)

	// Check if template exists
	if _, exists := r.templates[templateKey]; !exists {
		return fmt.Errorf("template %s version %s not found", templateID, templateVersion)
	}

	// Add asset
	r.assets[templateKey] = append(r.assets[templateKey], asset)

	return nil
}

// GetAssets returns all assets for a template
func (r *MemoryRepository) GetAssets(ctx context.Context, templateID, templateVersion string) ([]*Asset, error) {
	if templateID == "" || templateVersion == "" {
		return nil, fmt.Errorf("template ID and version cannot be empty")
	}

	key := fmt.Sprintf("%s#%s", templateID, templateVersion)
	assets, exists := r.assets[key]
	if !exists {
		return []*Asset{}, nil
	}

	return assets, nil
}

// DeleteAsset deletes a specific asset
func (r *MemoryRepository) DeleteAsset(ctx context.Context, templateID, templateVersion, assetType, assetPath string) error {
	if templateID == "" || templateVersion == "" {
		return fmt.Errorf("template ID and version cannot be empty")
	}

	key := fmt.Sprintf("%s#%s", templateID, templateVersion)
	assets, exists := r.assets[key]
	if !exists {
		return fmt.Errorf("no assets found for template %s version %s", templateID, templateVersion)
	}

	// Find and remove the asset
	for i, asset := range assets {
		if asset.Type == assetType && asset.Path == assetPath {
			r.assets[key] = append(assets[:i], assets[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("asset not found: type=%s, path=%s", assetType, assetPath)
}

// TemplateExists checks if a template exists
func (r *MemoryRepository) TemplateExists(ctx context.Context, id, version string) (bool, error) {
	if id == "" || version == "" {
		return false, fmt.Errorf("template ID and version cannot be empty")
	}

	key := fmt.Sprintf("%s#%s", id, version)
	_, exists := r.templates[key]
	return exists, nil
}

// GetTemplateCount returns the count of templates matching the filters
func (r *MemoryRepository) GetTemplateCount(ctx context.Context, filters *TemplateFilters) (int64, error) {
	count := int64(0)
	for _, template := range r.templates {
		if r.matchesFilters(template, filters) {
			count++
		}
	}
	return count, nil
}

// Helper methods

// matchesFilters checks if a template matches the given filters
func (r *MemoryRepository) matchesFilters(template *Template, filters *TemplateFilters) bool {
	if filters == nil {
		return true
	}

	// Filter by category
	if filters.Category != "" && template.Category != filters.Category {
		return false
	}

	// Filter by board type
	if filters.BoardType != "" {
		boardSupported := false
		for _, board := range template.BoardsSupported {
			if board == filters.BoardType {
				boardSupported = true
				break
			}
		}
		if !boardSupported {
			return false
		}
	}

	// Filter by supported boards
	if len(filters.SupportedBoards) > 0 {
		hasMatchingBoard := false
		for _, filterBoard := range filters.SupportedBoards {
			for _, templateBoard := range template.BoardsSupported {
				if templateBoard == filterBoard {
					hasMatchingBoard = true
					break
				}
			}
			if hasMatchingBoard {
				break
			}
		}
		if !hasMatchingBoard {
			return false
		}
	}

	return true
}

// matchesQuery checks if a template matches the search query
func (r *MemoryRepository) matchesQuery(template *Template, query string) bool {
	if query == "" {
		return true
	}

	// Simple text search in name, description, and category
	query = strings.ToLower(query)

	if strings.Contains(strings.ToLower(template.Name), query) {
		return true
	}

	if strings.Contains(strings.ToLower(template.Description), query) {
		return true
	}

	if strings.Contains(strings.ToLower(template.Category), query) {
		return true
	}

	return false
}
