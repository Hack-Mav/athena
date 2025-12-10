package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/athena/platform-lib/pkg/logger"
	"github.com/athena/platform-lib/pkg/template"
)

// TemplateCache provides caching for template operations
type TemplateCache struct {
	cache  *Cache
	logger logger.Logger
}

// NewTemplateCache creates a new template cache instance
func NewTemplateCache(cache *Cache, logger logger.Logger) *TemplateCache {
	return &TemplateCache{
		cache:  cache,
		logger: logger,
	}
}

// GetTemplate retrieves a template from cache or database
func (tc *TemplateCache) GetTemplate(ctx context.Context, templateID string) (*template.Template, error) {
	// Try cache first
	cacheKey := GenerateTemplateKey(templateID)

	var cachedTemplate template.Template
	if err := tc.cache.Get(ctx, cacheKey, &cachedTemplate); err == nil {
		tc.logger.Debug("Template retrieved from cache", "template_id", templateID)
		return &cachedTemplate, nil
	}

	// Cache miss - this would typically fetch from database
	// For now, return cache miss error
	tc.logger.Debug("Template cache miss", "template_id", templateID)
	return nil, ErrCacheMiss
}

// SetTemplate stores a template in cache
func (tc *TemplateCache) SetTemplate(ctx context.Context, tmpl *template.Template, ttl time.Duration) error {
	if tmpl == nil {
		return ErrInvalidValue
	}

	cacheKey := GenerateTemplateKey(tmpl.ID)
	if err := tc.cache.Set(ctx, cacheKey, tmpl, ttl); err != nil {
		tc.logger.Error("Failed to cache template", "template_id", tmpl.ID, "error", err)
		return err
	}

	tc.logger.Debug("Template cached", "template_id", tmpl.ID, "ttl", ttl)
	return nil
}

// InvalidateTemplate removes a template from cache
func (tc *TemplateCache) InvalidateTemplate(ctx context.Context, templateID string) error {
	cacheKey := GenerateTemplateKey(templateID)
	if err := tc.cache.Delete(ctx, cacheKey); err != nil {
		tc.logger.Error("Failed to invalidate template cache", "template_id", templateID, "error", err)
		return err
	}

	tc.logger.Debug("Template cache invalidated", "template_id", templateID)
	return nil
}

// GetTemplateList retrieves a list of templates from cache
func (tc *TemplateCache) GetTemplateList(ctx context.Context, filter string) ([]*template.Template, error) {
	cacheKey := fmt.Sprintf("%s:list:%s", TemplateKeyPrefix, filter)

	var cachedTemplates []*template.Template
	if err := tc.cache.Get(ctx, cacheKey, &cachedTemplates); err == nil {
		tc.logger.Debug("Template list retrieved from cache", "filter", filter)
		return cachedTemplates, nil
	}

	tc.logger.Debug("Template list cache miss", "filter", filter)
	return nil, ErrCacheMiss
}

// SetTemplateList stores a list of templates in cache
func (tc *TemplateCache) SetTemplateList(ctx context.Context, templates []*template.Template, filter string, ttl time.Duration) error {
	cacheKey := fmt.Sprintf("%s:list:%s", TemplateKeyPrefix, filter)

	if err := tc.cache.Set(ctx, cacheKey, templates, ttl); err != nil {
		tc.logger.Error("Failed to cache template list", "filter", filter, "error", err)
		return err
	}

	tc.logger.Debug("Template list cached", "filter", filter, "count", len(templates), "ttl", ttl)
	return nil
}

// InvalidateTemplateList removes a template list from cache
func (tc *TemplateCache) InvalidateTemplateList(ctx context.Context, filter string) error {
	cacheKey := fmt.Sprintf("%s:list:%s", TemplateKeyPrefix, filter)
	if err := tc.cache.Delete(ctx, cacheKey); err != nil {
		tc.logger.Error("Failed to invalidate template list cache", "filter", filter, "error", err)
		return err
	}

	tc.logger.Debug("Template list cache invalidated", "filter", filter)
	return nil
}

// InvalidateAllTemplates removes all template-related cache entries
func (tc *TemplateCache) InvalidateAllTemplates(ctx context.Context) error {
	// This is a simplified approach - in production, you might want to use Redis SCAN
	// or maintain a set of all template cache keys
	_ = fmt.Sprintf("%s*", TemplateKeyPrefix)

	// For now, we'll clear the entire cache (not ideal for production)
	tc.logger.Warn("Clearing all cache (simplified approach)")
	if err := tc.cache.Clear(ctx); err != nil {
		tc.logger.Error("Failed to clear all template cache", "error", err)
		return err
	}

	tc.logger.Info("All template cache invalidated")
	return nil
}

// CacheStats provides cache statistics
func (tc *TemplateCache) CacheStats(ctx context.Context) (map[string]interface{}, error) {
	// This would typically use Redis INFO command or custom counters
	// For now, return basic stats
	stats := map[string]interface{}{
		"type":    "template_cache",
		"backend": "redis",
		"status":  "active",
	}

	// Check cache health
	if err := tc.cache.Health(ctx); err != nil {
		stats["health"] = "unhealthy"
		stats["error"] = err.Error()
	} else {
		stats["health"] = "healthy"
	}

	return stats, nil
}
