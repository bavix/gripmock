package template_system

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/bavix/gripmock/v3/internal/config"
	"github.com/bavix/gripmock/v3/internal/infra/watcher"
	"github.com/bavix/gripmock/v3/internal/infra/yaml2json"
)

// WatcherAdapter adapts the existing StubWatcher to our interface.
type WatcherAdapter struct {
	watcher *watcher.StubWatcher
	config  config.AppConfig
}

// NewWatcherAdapter creates a new watcher adapter.
func NewWatcherAdapter(cfg config.AppConfig) *WatcherAdapter {
	return &WatcherAdapter{
		watcher: watcher.NewStubWatcher(cfg),
		config:  cfg,
	}
}

// Watch implements the Watcher interface.
func (wa *WatcherAdapter) Watch(ctx context.Context, path string, callback func(string) error) error {
	// Create a channel for file changes
	changes, err := wa.watcher.Watch(ctx, filepath.Dir(path))
	if err != nil {
		return err
	}

	// Start watching for changes
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case changedPath := <-changes:
				if changedPath == path {
					if err := callback(changedPath); err != nil {
						// Log error but continue watching
						continue
					}
				}
			}
		}
	}()

	return nil
}

// WatchDirectory implements the Watcher interface.
func (wa *WatcherAdapter) WatchDirectory(ctx context.Context, dirPath string, extensions []string, callback func(string) error) error {
	// Create a channel for file changes
	changes, err := wa.watcher.Watch(ctx, dirPath)
	if err != nil {
		return err
	}

	// Start watching for changes
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case changedPath := <-changes:
				// Check if the changed file has one of the specified extensions
				for _, ext := range extensions {
					if filepath.Ext(changedPath) == ext {
						if err := callback(changedPath); err != nil {
							// Log error but continue watching
							continue
						}

						break
					}
				}
			}
		}
	}()

	return nil
}

// YamlParserAdapter adapts the existing yaml2json converter to our interface.
type YamlParserAdapter struct {
	converter *yaml2json.Convertor
}

// NewYamlParserAdapter creates a new YAML parser adapter.
func NewYamlParserAdapter() *YamlParserAdapter {
	return &YamlParserAdapter{
		converter: yaml2json.New(),
	}
}

// ParseFile implements the YamlParser interface.
func (ypa *YamlParserAdapter) ParseFile(path string) (map[string]any, error) {
	// Read file content
	//nolint:gosec
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Convert YAML to JSON
	jsonData, err := ypa.converter.Execute(path, data)
	if err != nil {
		return nil, err
	}

	// Parse JSON to map
	var result map[string]any
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// NewDefaultTemplateFacade creates a template facade with default adapters.
func NewDefaultTemplateFacade(cfg config.AppConfig) *TemplateFacade {
	return NewTemplateFacade(
		NewWatcherAdapter(cfg),
		NewYamlParserAdapter(),
	)
}
