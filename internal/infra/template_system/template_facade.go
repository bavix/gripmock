package template_system

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TemplateFacade provides a unified interface for template processing.
type TemplateFacade struct {
	watcher    Watcher
	yamlParser YamlParser
	template   *TemplateProcessor
	mu         sync.RWMutex
	cache      map[string]TemplateCacheEntry
}

// Watcher interface for file watching.
type Watcher interface {
	Watch(ctx context.Context, path string, callback func(string) error) error
	WatchDirectory(ctx context.Context, dirPath string, extensions []string, callback func(string) error) error
}

// YamlParser interface for YAML parsing.
type YamlParser interface {
	ParseFile(path string) (map[string]any, error)
}

// TemplateCacheEntry represents a cached template entry.
type TemplateCacheEntry struct {
	Content    map[string]any
	LastUpdate time.Time
	Hash       string
}

// TemplateProcessor handles template processing.
type TemplateProcessor struct {
	mu sync.RWMutex
}

// NewTemplateFacade creates a new template facade.
func NewTemplateFacade(watcher Watcher, yamlParser YamlParser) *TemplateFacade {
	return &TemplateFacade{
		watcher:    watcher,
		yamlParser: yamlParser,
		template:   &TemplateProcessor{},
		cache:      make(map[string]TemplateCacheEntry),
	}
}

// LoadAndWatch loads a template file and watches for changes.
func (tf *TemplateFacade) LoadAndWatch(ctx context.Context, filePath string) error {
	// Load initial content
	if err := tf.loadTemplate(filePath); err != nil {
		return fmt.Errorf("failed to load template %s: %w", filePath, err)
	}

	// Start watching for changes
	return tf.watcher.Watch(ctx, filePath, func(path string) error {
		return tf.loadTemplate(path)
	})
}

// loadTemplate loads and processes a template file.
//
//nolint:funcorder
func (tf *TemplateFacade) loadTemplate(filePath string) error {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	// Parse YAML to JSON
	content, err := tf.yamlParser.ParseFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to parse YAML file %s: %w", filePath, err)
	}

	// Process templates in the content
	processedContent, err := tf.template.ProcessTemplates(content)
	if err != nil {
		return fmt.Errorf("failed to process templates in %s: %w", filePath, err)
	}

	// Cache the result
	tf.cache[filePath] = TemplateCacheEntry{
		Content:    processedContent,
		LastUpdate: time.Now(),
		Hash:       tf.generateHash(processedContent),
	}

	return nil
}

// GetTemplate returns cached template content.
func (tf *TemplateFacade) GetTemplate(filePath string) (map[string]any, error) {
	tf.mu.RLock()
	defer tf.mu.RUnlock()

	entry, exists := tf.cache[filePath]
	if !exists {
		return nil, fmt.Errorf("template %s not found in cache", filePath) //nolint:err113
	}

	return entry.Content, nil
}

// GetTemplateWithContext returns template content with runtime context.
func (tf *TemplateFacade) GetTemplateWithContext(filePath string, context map[string]any) (map[string]any, error) {
	content, err := tf.GetTemplate(filePath)
	if err != nil {
		return nil, err
	}

	// Apply runtime context to templates
	return tf.template.ApplyContext(content, context)
}

// ProcessString processes a string template with context.
func (tf *TemplateFacade) ProcessString(template string, context map[string]any) (string, error) {
	return tf.template.ProcessString(template, context)
}

// ProcessMap processes a map template with context.
func (tf *TemplateFacade) ProcessMap(template map[string]any, context map[string]any) (map[string]any, error) {
	return tf.template.ProcessMap(template, context)
}

// WatchDirectory watches a directory for template changes.
func (tf *TemplateFacade) WatchDirectory(ctx context.Context, dirPath string, extensions []string) error {
	return tf.watcher.WatchDirectory(ctx, dirPath, extensions, func(path string) error {
		return tf.loadTemplate(path)
	})
}

// GetCacheStats returns cache statistics.
func (tf *TemplateFacade) GetCacheStats() map[string]any {
	tf.mu.RLock()
	defer tf.mu.RUnlock()

	stats := map[string]any{
		"totalEntries": len(tf.cache),
		"entries":      make(map[string]any),
	}

	for path, entry := range tf.cache {
		if entries, ok := stats["entries"].(map[string]any); ok {
			entries[path] = map[string]any{
				"lastUpdate": entry.LastUpdate,
				"hash":       entry.Hash,
			}
		}
	}

	return stats
}

// ClearCache clears the template cache.
func (tf *TemplateFacade) ClearCache() {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	tf.cache = make(map[string]TemplateCacheEntry)
}

// generateHash generates a hash for content.
func (tf *TemplateFacade) generateHash(content map[string]any) string {
	// Simple hash implementation - in production you might want more sophisticated hashing
	return fmt.Sprintf("%v", content)
}

// ProcessTemplates processes templates in the given content.
func (tp *TemplateProcessor) ProcessTemplates(content map[string]any) (map[string]any, error) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	// Deep copy to avoid modifying original
	result := make(map[string]any)

	for k, v := range content {
		processed, err := tp.processValue(v)
		if err != nil {
			return nil, fmt.Errorf("failed to process key %s: %w", k, err)
		}

		result[k] = processed
	}

	return result, nil
}

// ApplyContext applies runtime context to processed templates.
func (tp *TemplateProcessor) ApplyContext(content map[string]any, context map[string]any) (map[string]any, error) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	result := make(map[string]any)

	for k, v := range content {
		processed, err := tp.applyContextToValue(v, context)
		if err != nil {
			return nil, fmt.Errorf("failed to apply context to key %s: %w", k, err)
		}

		result[k] = processed
	}

	return result, nil
}

// ProcessString processes a string template.
func (tp *TemplateProcessor) ProcessString(template string, context map[string]any) (string, error) {
	// Simple template processing - in production you might want more sophisticated templating
	// This is a placeholder for actual template processing logic
	return template, nil
}

// ProcessMap processes a map template.
func (tp *TemplateProcessor) ProcessMap(template map[string]any, context map[string]any) (map[string]any, error) {
	return tp.ApplyContext(template, context)
}

// processValue recursively processes a value.
func (tp *TemplateProcessor) processValue(v any) (any, error) {
	switch val := v.(type) {
	case string:
		return tp.processString(val)
	case map[string]any:
		return tp.processMap(val)
	case []any:
		return tp.processSlice(val)
	default:
		return v, nil
	}
}

// applyContextToValue recursively applies context to a value.
func (tp *TemplateProcessor) applyContextToValue(v any, context map[string]any) (any, error) {
	switch val := v.(type) {
	case string:
		return tp.applyContextToString(val, context)
	case map[string]any:
		return tp.applyContextToMap(val, context)
	case []any:
		return tp.applyContextToSlice(val, context)
	default:
		return v, nil
	}
}

// processString processes a string value.
func (tp *TemplateProcessor) processString(s string) (string, error) {
	// Placeholder for string template processing
	return s, nil
}

// processMap processes a map value.
func (tp *TemplateProcessor) processMap(m map[string]any) (map[string]any, error) {
	result := make(map[string]any)

	for k, v := range m {
		processed, err := tp.processValue(v)
		if err != nil {
			return nil, err
		}

		result[k] = processed
	}

	return result, nil
}

// processSlice processes a slice value.
func (tp *TemplateProcessor) processSlice(s []any) ([]any, error) {
	result := make([]any, len(s))
	for i, v := range s {
		processed, err := tp.processValue(v)
		if err != nil {
			return nil, err
		}

		result[i] = processed
	}

	return result, nil
}

// applyContextToString applies context to a string.
func (tp *TemplateProcessor) applyContextToString(s string, context map[string]any) (string, error) {
	// Placeholder for context application to strings
	return s, nil
}

// applyContextToMap applies context to a map.
func (tp *TemplateProcessor) applyContextToMap(m map[string]any, context map[string]any) (map[string]any, error) {
	result := make(map[string]any)

	for k, v := range m {
		processed, err := tp.applyContextToValue(v, context)
		if err != nil {
			return nil, err
		}

		result[k] = processed
	}

	return result, nil
}

// applyContextToSlice applies context to a slice.
func (tp *TemplateProcessor) applyContextToSlice(s []any, context map[string]any) ([]any, error) {
	result := make([]any, len(s))
	for i, v := range s {
		processed, err := tp.applyContextToValue(v, context)
		if err != nil {
			return nil, err
		}

		result[i] = processed
	}

	return result, nil
}
