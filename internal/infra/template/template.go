package template

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
	"time"
)

// Data represents the context data available for template rendering.
type Data struct {
	Request      map[string]any `json:"request"`
	Headers      map[string]any `json:"headers"`
	MessageIndex int            `json:"messageIndex"`
	RequestTime  time.Time      `json:"requestTime"`
	// Timestamp is a backward-compatibility alias for RequestTime
	Timestamp     time.Time      `json:"timestamp"`
	State         map[string]any `json:"state"`
	Requests      []any          `json:"requests"`
	AttemptNumber int            `json:"attemptNumber"` // Current attempt number (1-based)
	// AttemptIndex is a backward-compatibility alias for AttemptNumber
	AttemptIndex int `json:"attemptIndex"`
	MaxAttempts  int `json:"maxAttempts"` // Maximum number of attempts for this stub
	// TotalAttempts is a backward-compatibility alias for MaxAttempts
	TotalAttempts int    `json:"totalAttempts"`
	StubID        string `json:"stubId"` // Unique identifier of the stub
	// RequestID is a backward-compatibility alias mapped to StubID
	RequestID string `json:"requestId"`
}

// Engine provides template rendering functionality.
type Engine struct {
	funcs template.FuncMap
}

// New creates a new template engine with custom functions.
func New() *Engine {
	return &Engine{
		funcs: Functions(),
	}
}

// Render renders a template string with the given data.
func (e *Engine) Render(tmpl string, data Data) (string, error) {
	if tmpl == "" {
		return "", nil
	}

	t := e.createTemplate()

	parsed, err := t.Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := parsed.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// IsTemplateString checks if a string contains template syntax.
func IsTemplateString(s string) bool {
	return strings.Contains(s, "{{") && strings.Contains(s, "}}")
}

// HasTemplates checks if the data contains any template strings.
func HasTemplates(data map[string]any) bool {
	return hasTemplatesInMap(data)
}

// HasTemplatesInStream checks if the stream contains any template strings.
func HasTemplatesInStream(stream []any) bool {
	return hasTemplatesInStream(stream)
}

// HasTemplatesInHeaders checks if the headers contain any template strings.
func HasTemplatesInHeaders(headers map[string]string) bool {
	return hasTemplatesInHeaders(headers)
}

// ProcessMap processes templates in a map recursively.
func (e *Engine) ProcessMap(data map[string]any, templateData Data) error {
	return processMapTemplates(data, templateData, e)
}

// ProcessStream processes templates in a stream.
func (e *Engine) ProcessStream(stream []any, templateData Data) error {
	return processStreamField(stream, templateData, e)
}

// ProcessHeaders processes templates in headers.
func (e *Engine) ProcessHeaders(headers map[string]string, templateData Data) error {
	return processHeadersField(headers, templateData, e)
}

// ProcessError processes error template.
func (e *Engine) ProcessError(errorStr string, templateData Data) (string, error) {
	return processErrorField(errorStr, templateData, e)
}

// createTemplate creates a template with custom functions.
func (e *Engine) createTemplate() *template.Template {
	return template.New("dynamic").Funcs(e.funcs)
}

func processMapTemplates(data map[string]any, templateData Data, engine *Engine) error {
	for key, value := range data {
		switch v := value.(type) {
		case string:
			if IsTemplateString(v) {
				rendered, err := engine.Render(v, templateData)
				if err != nil {
					return fmt.Errorf("failed to process template for key %s: %w", key, err)
				}

				data[key] = rendered
			}
		case map[string]any:
			err := processMapTemplates(v, templateData, engine)
			if err != nil {
				return err
			}
		case []any:
			err := processArrayTemplates(v, templateData, engine)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func processStreamField(stream []any, templateData Data, engine *Engine) error {
	if stream == nil {
		return nil
	}

	for i, item := range stream {
		if itemMap, ok := item.(map[string]any); ok {
			err := processMapTemplates(itemMap, templateData, engine)
			if err != nil {
				return fmt.Errorf("failed to process stream template at index %d: %w", i, err)
			}

			stream[i] = itemMap
		}
	}

	return nil
}

func processErrorField(errorStr string, templateData Data, engine *Engine) (string, error) {
	if errorStr == "" || !IsTemplateString(errorStr) {
		return errorStr, nil
	}

	rendered, err := engine.Render(errorStr, templateData)
	if err != nil {
		return "", fmt.Errorf("failed to process error template: %w", err)
	}

	return rendered, nil
}

func processHeadersField(headers map[string]string, templateData Data, engine *Engine) error {
	if headers == nil {
		return nil
	}

	for key, value := range headers {
		if IsTemplateString(value) {
			rendered, err := engine.Render(value, templateData)
			if err != nil {
				return fmt.Errorf("failed to process header template for %s: %w", key, err)
			}

			headers[key] = rendered
		}
	}

	return nil
}

// processArrayTemplates recursively processes templates in an array.
func processArrayTemplates(arr []any, templateData Data, engine *Engine) error {
	for i, item := range arr {
		switch v := item.(type) {
		case map[string]any:
			err := processMapTemplates(v, templateData, engine)
			if err != nil {
				return err
			}

			arr[i] = v
		case string:
			if IsTemplateString(v) {
				rendered, err := engine.Render(v, templateData)
				if err != nil {
					return fmt.Errorf("failed to process template for array item %d: %w", i, err)
				}

				arr[i] = rendered
			}
		}
	}

	return nil
}

func hasTemplatesInMap(data map[string]any) bool {
	for _, value := range data {
		if hasTemplatesInValue(value) {
			return true
		}
	}

	return false
}

func hasTemplatesInStream(stream []any) bool {
	for _, item := range stream {
		if itemMap, ok := item.(map[string]any); ok && hasTemplatesInMap(itemMap) {
			return true
		}
	}

	return false
}

func hasTemplatesInHeaders(headers map[string]string) bool {
	for _, value := range headers {
		if IsTemplateString(value) {
			return true
		}
	}

	return false
}

func hasTemplatesInValue(value any) bool {
	switch v := value.(type) {
	case string:
		return IsTemplateString(v)
	case map[string]any:
		return hasTemplatesInMap(v)
	case []any:
		for _, item := range v {
			if hasTemplatesInValue(item) {
				return true
			}
		}
	}

	return false
}
