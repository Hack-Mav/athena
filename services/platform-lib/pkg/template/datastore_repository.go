package template

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
)

// DatastoreRepository implements the Repository interface using Google Cloud Datastore
type DatastoreRepository struct {
	client *datastore.Client
}

// NewDatastoreRepository creates a new Datastore repository
func NewDatastoreRepository(client *datastore.Client) *DatastoreRepository {
	return &DatastoreRepository{
		client: client,
	}
}

// CreateTemplate creates a new template in Datastore
func (r *DatastoreRepository) CreateTemplate(ctx context.Context, template *Template) error {
	if template == nil {
		return fmt.Errorf("template cannot be nil")
	}

	// Check if template already exists
	exists, err := r.TemplateExists(ctx, template.ID, template.Version)
	if err != nil {
		return fmt.Errorf("failed to check template existence: %w", err)
	}
	if exists {
		return fmt.Errorf("template %s version %s already exists", template.ID, template.Version)
	}

	// Convert to entity
	entity, err := template.ToEntity()
	if err != nil {
		return fmt.Errorf("failed to convert template to entity: %w", err)
	}

	// Set timestamps
	now := time.Now()
	entity.CreatedAt = now
	entity.UpdatedAt = now

	// Create Datastore key
	key := datastore.NameKey("Template", fmt.Sprintf("%s#%s", template.ID, template.Version), nil)

	// Store in Datastore
	_, err = r.client.Put(ctx, key, entity)
	if err != nil {
		return fmt.Errorf("failed to store template in Datastore: %w", err)
	}

	// Store assets
	for _, asset := range template.Assets {
		if err := r.CreateAsset(ctx, template.ID, template.Version, &asset); err != nil {
			return fmt.Errorf("failed to store asset: %w", err)
		}
	}

	return nil
}

// GetTemplate retrieves a template by ID and version from Datastore
func (r *DatastoreRepository) GetTemplate(ctx context.Context, id, version string) (*Template, error) {
	if id == "" {
		return nil, fmt.Errorf("template ID cannot be empty")
	}
	if version == "" {
		return nil, fmt.Errorf("template version cannot be empty")
	}

	// Create Datastore key
	key := datastore.NameKey("Template", fmt.Sprintf("%s#%s", id, version), nil)

	// Retrieve from Datastore
	var entity TemplateEntity
	err := r.client.Get(ctx, key, &entity)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return nil, fmt.Errorf("template %s version %s not found", id, version)
		}
		return nil, fmt.Errorf("failed to retrieve template from Datastore: %w", err)
	}

	// Convert to template
	template, err := entity.FromEntity()
	if err != nil {
		return nil, fmt.Errorf("failed to convert entity to template: %w", err)
	}

	// Load assets
	assets, err := r.GetAssets(ctx, id, version)
	if err != nil {
		return nil, fmt.Errorf("failed to load assets: %w", err)
	}
	template.Assets = make([]Asset, len(assets))
	for i, asset := range assets {
		template.Assets[i] = *asset
	}

	return template, nil
}

// UpdateTemplate updates an existing template in Datastore
func (r *DatastoreRepository) UpdateTemplate(ctx context.Context, template *Template) error {
	if template == nil {
		return fmt.Errorf("template cannot be nil")
	}

	// Check if template exists
	exists, err := r.TemplateExists(ctx, template.ID, template.Version)
	if err != nil {
		return fmt.Errorf("failed to check template existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("template %s version %s not found", template.ID, template.Version)
	}

	// Convert to entity
	entity, err := template.ToEntity()
	if err != nil {
		return fmt.Errorf("failed to convert template to entity: %w", err)
	}

	// Update timestamp
	entity.UpdatedAt = time.Now()

	// Create Datastore key
	key := datastore.NameKey("Template", fmt.Sprintf("%s#%s", template.ID, template.Version), nil)

	// Update in Datastore
	_, err = r.client.Put(ctx, key, entity)
	if err != nil {
		return fmt.Errorf("failed to update template in Datastore: %w", err)
	}

	return nil
}

// DeleteTemplate deletes a template by ID and version from Datastore
func (r *DatastoreRepository) DeleteTemplate(ctx context.Context, id, version string) error {
	if id == "" {
		return fmt.Errorf("template ID cannot be empty")
	}
	if version == "" {
		return fmt.Errorf("template version cannot be empty")
	}

	// Check if template exists
	exists, err := r.TemplateExists(ctx, id, version)
	if err != nil {
		return fmt.Errorf("failed to check template existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("template %s version %s not found", id, version)
	}

	// Create Datastore key
	key := datastore.NameKey("Template", fmt.Sprintf("%s#%s", id, version), nil)

	// Delete from Datastore
	err = r.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete template from Datastore: %w", err)
	}

	// Delete associated assets
	assets, err := r.GetAssets(ctx, id, version)
	if err != nil {
		return fmt.Errorf("failed to get assets for deletion: %w", err)
	}

	for _, asset := range assets {
		if err := r.DeleteAsset(ctx, id, version, asset.Type, asset.Path); err != nil {
			return fmt.Errorf("failed to delete asset: %w", err)
		}
	}

	return nil
}

// ListTemplates returns templates matching the given filters from Datastore
func (r *DatastoreRepository) ListTemplates(ctx context.Context, filters *TemplateFilters) ([]*Template, error) {
	query := datastore.NewQuery("Template")

	// Apply filters
	if filters != nil {
		if filters.Category != "" {
			query = query.Filter("category =", filters.Category)
		}
		if filters.BoardType != "" {
			query = query.Filter("boards_supported =", filters.BoardType)
		}
		if len(filters.SupportedBoards) > 0 {
			// For multiple board filters, we need to use IN operator or multiple queries
			// For simplicity, we'll filter the first supported board
			query = query.Filter("boards_supported =", filters.SupportedBoards[0])
		}
		if filters.Limit > 0 {
			query = query.Limit(filters.Limit)
		}
		if filters.Offset > 0 {
			query = query.Offset(filters.Offset)
		}
	}

	// Order by creation date (newest first)
	query = query.Order("-created_at")

	// Execute query
	var entities []TemplateEntity
	_, err := r.client.GetAll(ctx, query, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to query templates from Datastore: %w", err)
	}

	// Convert entities to templates
	var templates []*Template
	for _, entity := range entities {
		template, err := entity.FromEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert entity to template: %w", err)
		}

		// Load assets
		assets, err := r.GetAssets(ctx, template.ID, template.Version)
		if err != nil {
			return nil, fmt.Errorf("failed to load assets for template %s: %w", template.ID, err)
		}
		template.Assets = make([]Asset, len(assets))
		for i, asset := range assets {
			template.Assets[i] = *asset
		}

		templates = append(templates, template)
	}

	return templates, nil
}

// SearchTemplates searches templates by query string in Datastore
func (r *DatastoreRepository) SearchTemplates(ctx context.Context, query string, filters *TemplateFilters) ([]*Template, error) {
	// Note: Datastore doesn't support full-text search natively
	// For production, you would typically use Google Cloud Search API or Elasticsearch
	// Here we'll implement a simple approach by getting all templates and filtering in memory

	// Get all templates first
	allTemplates, err := r.ListTemplates(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get templates for search: %w", err)
	}

	// Filter by query
	var result []*Template
	queryLower := strings.ToLower(query)

	for _, template := range allTemplates {
		if r.matchesQuery(template, queryLower) {
			result = append(result, template)
		}
	}

	return result, nil
}

// GetTemplateVersions returns all versions for a given template ID from Datastore
func (r *DatastoreRepository) GetTemplateVersions(ctx context.Context, id string) ([]string, error) {
	if id == "" {
		return nil, fmt.Errorf("template ID cannot be empty")
	}

	query := datastore.NewQuery("Template").
		Filter("id =", id).
		Project("version").
		Order("-created_at")

	var entities []TemplateEntity
	_, err := r.client.GetAll(ctx, query, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to query template versions from Datastore: %w", err)
	}

	var versions []string
	for _, entity := range entities {
		versions = append(versions, entity.Version)
	}

	return versions, nil
}

// CreateAsset creates a new asset for a template in Datastore
func (r *DatastoreRepository) CreateAsset(ctx context.Context, templateID, templateVersion string, asset *Asset) error {
	if templateID == "" || templateVersion == "" {
		return fmt.Errorf("template ID and version cannot be empty")
	}
	if asset == nil {
		return fmt.Errorf("asset cannot be nil")
	}

	// Check if template exists
	exists, err := r.TemplateExists(ctx, templateID, templateVersion)
	if err != nil {
		return fmt.Errorf("failed to check template existence: %w", err)
	}
	if !exists {
		return fmt.Errorf("template %s version %s not found", templateID, templateVersion)
	}

	// Convert to entity
	entity, err := asset.ToAssetEntity(templateID, templateVersion)
	if err != nil {
		return fmt.Errorf("failed to convert asset to entity: %w", err)
	}

	// Create Datastore key
	keyName := fmt.Sprintf("%s#%s#%s#%s", templateID, templateVersion, asset.Type, asset.Path)
	key := datastore.NameKey("TemplateAsset", keyName, nil)

	// Store in Datastore
	_, err = r.client.Put(ctx, key, entity)
	if err != nil {
		return fmt.Errorf("failed to store asset in Datastore: %w", err)
	}

	return nil
}

// GetAssets returns all assets for a template from Datastore
func (r *DatastoreRepository) GetAssets(ctx context.Context, templateID, templateVersion string) ([]*Asset, error) {
	if templateID == "" || templateVersion == "" {
		return nil, fmt.Errorf("template ID and version cannot be empty")
	}

	query := datastore.NewQuery("TemplateAsset").
		Filter("template_id =", templateID).
		Filter("template_version =", templateVersion).
		Order("created_at")

	var entities []TemplateAssetEntity
	_, err := r.client.GetAll(ctx, query, &entities)
	if err != nil {
		return nil, fmt.Errorf("failed to query assets from Datastore: %w", err)
	}

	var assets []*Asset
	for _, entity := range entities {
		asset, err := entity.FromAssetEntity()
		if err != nil {
			return nil, fmt.Errorf("failed to convert asset entity: %w", err)
		}
		assets = append(assets, asset)
	}

	return assets, nil
}

// DeleteAsset deletes a specific asset from Datastore
func (r *DatastoreRepository) DeleteAsset(ctx context.Context, templateID, templateVersion, assetType, assetPath string) error {
	if templateID == "" || templateVersion == "" {
		return fmt.Errorf("template ID and version cannot be empty")
	}

	// Create Datastore key
	keyName := fmt.Sprintf("%s#%s#%s#%s", templateID, templateVersion, assetType, assetPath)
	key := datastore.NameKey("TemplateAsset", keyName, nil)

	// Delete from Datastore
	err := r.client.Delete(ctx, key)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return fmt.Errorf("asset not found: type=%s, path=%s", assetType, assetPath)
		}
		return fmt.Errorf("failed to delete asset from Datastore: %w", err)
	}

	return nil
}

// TemplateExists checks if a template exists in Datastore
func (r *DatastoreRepository) TemplateExists(ctx context.Context, id, version string) (bool, error) {
	if id == "" || version == "" {
		return false, fmt.Errorf("template ID and version cannot be empty")
	}

	// Create Datastore key
	key := datastore.NameKey("Template", fmt.Sprintf("%s#%s", id, version), nil)

	// Check existence
	var entity TemplateEntity
	err := r.client.Get(ctx, key, &entity)
	if err != nil {
		if err == datastore.ErrNoSuchEntity {
			return false, nil
		}
		return false, fmt.Errorf("failed to check template existence in Datastore: %w", err)
	}

	return true, nil
}

// GetTemplateCount returns the count of templates matching the filters from Datastore
func (r *DatastoreRepository) GetTemplateCount(ctx context.Context, filters *TemplateFilters) (int64, error) {
	query := datastore.NewQuery("Template")

	// Apply filters
	if filters != nil {
		if filters.Category != "" {
			query = query.Filter("category =", filters.Category)
		}
		if filters.BoardType != "" {
			query = query.Filter("boards_supported =", filters.BoardType)
		}
		if len(filters.SupportedBoards) > 0 {
			query = query.Filter("boards_supported =", filters.SupportedBoards[0])
		}
	}

	// Count only
	count, err := r.client.Count(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to count templates in Datastore: %w", err)
	}

	return int64(count), nil
}

// Helper methods

// matchesQuery checks if a template matches the search query
func (r *DatastoreRepository) matchesQuery(template *Template, query string) bool {
	if query == "" {
		return true
	}

	// Simple text search in name, description, and category
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