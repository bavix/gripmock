package template

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/bavix/gripmock/v3/pkg/plugins"
)

// ErrMaxRecursionDepthExceeded is returned when structure nesting exceeds MaxRecursionDepth.
var ErrMaxRecursionDepthExceeded = errors.New("maximum recursion depth exceeded")

// MaxRecursionDepth is the maximum allowed nesting depth for ProcessMap/ProcessStream.
const MaxRecursionDepth = 250

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
func New(ctx context.Context, reg plugins.Registry) *Engine {
	return &Engine{
		funcs: Functions(ctx, reg),
	}
}

// Render renders a template string with the given data.
func (e *Engine) Render(tmpl string, data Data) (string, error) {
	if tmpl == "" {
		return "", nil
	}

	tmpl = unescapeTemplateQuotes(tmpl)

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

// unescapeTemplateQuotes removes escape sequences from quotes inside template expressions.
func unescapeTemplateQuotes(tmpl string) string {
	var result strings.Builder

	lastIdx := 0

	for {
		startIdx := strings.Index(tmpl[lastIdx:], "{{")
		if startIdx == -1 {
			result.WriteString(tmpl[lastIdx:])

			break
		}

		startIdx += lastIdx

		endIdx := strings.Index(tmpl[startIdx:], "}}")
		if endIdx == -1 {
			result.WriteString(tmpl[lastIdx:])

			break
		}

		endIdx += startIdx + len("}}")

		result.WriteString(tmpl[lastIdx:startIdx])

		templateExpr := tmpl[startIdx:endIdx]
		unescapedExpr := unescapeQuotesInString(templateExpr)
		result.WriteString(unescapedExpr)

		lastIdx = endIdx
	}

	return result.String()
}

// unescapeQuotesInString removes escape sequences from quotes, handling multiple levels.
func unescapeQuotesInString(s string) string {
	result := s

	maxIterations := 10
	for range maxIterations {
		oldResult := result
		result = strings.ReplaceAll(result, `\"`, `"`)

		result = strings.ReplaceAll(result, `\\"`, `"`)
		if result == oldResult {
			break
		}
	}

	return result
}

// IsTemplateString checks if a string contains template syntax.
func IsTemplateString(s string) bool {
	return strings.Contains(s, "{{") && strings.Contains(s, "}}")
}

// HasTemplates checks if the data contains any template strings.
func HasTemplates(data map[string]any) bool {
	return hasTemplatesInMap(data, 0)
}

// HasTemplatesInStream checks if the stream contains any template strings.
func HasTemplatesInStream(stream []any) bool {
	return hasTemplatesInStream(stream, 0)
}

// HasTemplatesInHeaders checks if the headers contain any template strings.
func HasTemplatesInHeaders(headers map[string]string) bool {
	return hasTemplatesInHeaders(headers)
}

// ProcessMap processes templates in a map recursively.
func (e *Engine) ProcessMap(data map[string]any, templateData Data) error {
	return processMapTemplates(data, templateData, e, 0)
}

// ProcessStream processes templates in a stream.
func (e *Engine) ProcessStream(stream []any, templateData Data) error {
	return processStreamField(stream, templateData, e, 0)
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

func processMapTemplates(data map[string]any, templateData Data, engine *Engine, depth int) error {
	if depth > MaxRecursionDepth {
		return ErrMaxRecursionDepthExceeded
	}

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
			if err := processMapTemplates(v, templateData, engine, depth+1); err != nil {
				return err
			}
		case []any:
			if err := processArrayTemplates(v, templateData, engine, depth+1); err != nil {
				return err
			}
		}
	}

	return nil
}

func processStreamField(stream []any, templateData Data, engine *Engine, depth int) error {
	if stream == nil {
		return nil
	}

	if depth > MaxRecursionDepth {
		return ErrMaxRecursionDepthExceeded
	}

	for i, item := range stream {
		if itemMap, ok := item.(map[string]any); ok {
			if err := processMapTemplates(itemMap, templateData, engine, depth+1); err != nil {
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
func processArrayTemplates(arr []any, templateData Data, engine *Engine, depth int) error {
	if depth > MaxRecursionDepth {
		return ErrMaxRecursionDepthExceeded
	}

	for i, item := range arr {
		switch v := item.(type) {
		case map[string]any:
			if err := processMapTemplates(v, templateData, engine, depth+1); err != nil {
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

func hasTemplatesInMap(data map[string]any, depth int) bool {
	if depth > MaxRecursionDepth {
		return false
	}

	for _, value := range data {
		if hasTemplatesInValue(value, depth) {
			return true
		}
	}

	return false
}

func hasTemplatesInStream(stream []any, depth int) bool {
	if depth > MaxRecursionDepth {
		return false
	}

	for _, item := range stream {
		if itemMap, ok := item.(map[string]any); ok && hasTemplatesInMap(itemMap, depth+1) {
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

func hasTemplatesInValue(value any, depth int) bool {
	if depth > MaxRecursionDepth {
		return false
	}

	switch v := value.(type) {
	case string:
		return IsTemplateString(v)
	case map[string]any:
		return hasTemplatesInMap(v, depth+1)
	case []any:
		return slices.ContainsFunc(v, func(item any) bool { return hasTemplatesInValue(item, depth+1) })
	}

	return false
}
