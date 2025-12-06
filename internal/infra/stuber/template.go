package stuber

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"text/template"
	"time"
)

// TemplateData represents the context data available for template rendering.
type TemplateData struct {
	Request      map[string]any `json:"request"`
	Headers      map[string]any `json:"headers"`
	MessageIndex int            `json:"messageIndex"`
	RequestTime  time.Time      `json:"requestTime"`
	State        map[string]any `json:"state"`
	Requests     []any          `json:"requests"`
}

// createTemplate creates a template with custom functions.
func createTemplate() *template.Template {
	return template.New("dynamic").Funcs(TemplateFunctions())
}

// RenderTemplate renders a template string with the given data.
func RenderTemplate(tmpl string, data TemplateData) (string, error) {
	if tmpl == "" {
		return "", nil
	}

	t := createTemplate()

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

// ProcessDynamicOutput processes the output data and applies dynamic templates.
func (o *Output) ProcessDynamicOutput(requestData map[string]any, headers map[string]any, messageIndex int, allMessages []any) error {
	// Create template data
	templateData := TemplateData{
		Request:      requestData,
		Headers:      headers,
		MessageIndex: messageIndex,
		RequestTime:  time.Now(),
		State:        make(map[string]any),
		Requests:     allMessages,
	}

	// Process all output fields
	err := processDataField(o.Data, templateData)
	if err != nil {
		return fmt.Errorf("failed to process data templates: %w", err)
	}

	err = processStreamField(o.Stream, templateData)
	if err != nil {
		return fmt.Errorf("failed to process stream templates: %w", err)
	}

	if renderedError, err := processErrorField(o.Error, templateData); err != nil {
		return fmt.Errorf("failed to process error template: %w", err)
	} else if renderedError != "" {
		o.Error = renderedError
	}

	err = processHeadersField(o.Headers, templateData)
	if err != nil {
		return fmt.Errorf("failed to process header templates: %w", err)
	}

	return nil
}

func processDataField(data map[string]any, templateData TemplateData) error {
	if data == nil {
		return nil
	}

	return processMapTemplates(data, templateData)
}

func processStreamField(stream []any, templateData TemplateData) error {
	if stream == nil {
		return nil
	}

	for i, item := range stream {
		if itemMap, ok := item.(map[string]any); ok {
			err := processMapTemplates(itemMap, templateData)
			if err != nil {
				return fmt.Errorf("failed to process stream template at index %d: %w", i, err)
			}

			stream[i] = itemMap
		}
	}

	return nil
}

func processErrorField(errorStr string, templateData TemplateData) (string, error) {
	if errorStr == "" || !IsTemplateString(errorStr) {
		return errorStr, nil
	}

	rendered, err := RenderTemplate(errorStr, templateData)
	if err != nil {
		return "", fmt.Errorf("failed to process error template: %w", err)
	}

	return rendered, nil
}

func processHeadersField(headers map[string]string, templateData TemplateData) error {
	if headers == nil {
		return nil
	}

	for key, value := range headers {
		if IsTemplateString(value) {
			rendered, err := RenderTemplate(value, templateData)
			if err != nil {
				return fmt.Errorf("failed to process header template for %s: %w", key, err)
			}

			headers[key] = rendered
		}
	}

	return nil
}

// processMapTemplates recursively processes templates in a map.
func processMapTemplates(data map[string]any, templateData TemplateData) error {
	for key, value := range data {
		switch v := value.(type) {
		case string:
			if IsTemplateString(v) {
				rendered, err := RenderTemplate(v, templateData)
				if err != nil {
					return fmt.Errorf("failed to process template for key %s: %w", key, err)
				}

				data[key] = rendered
			}
		case map[string]any:
			err := processMapTemplates(v, templateData)
			if err != nil {
				return err
			}
		case []any:
			err := processArrayTemplates(v, templateData)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func processArrayTemplates(arr []any, templateData TemplateData) error {
	for i, item := range arr {
		switch v := item.(type) {
		case map[string]any:
			err := processMapTemplates(v, templateData)
			if err != nil {
				return err
			}

			arr[i] = v
		case string:
			if IsTemplateString(v) {
				rendered, err := RenderTemplate(v, templateData)
				if err != nil {
					return fmt.Errorf("failed to process template for array item %d: %w", i, err)
				}

				arr[i] = rendered
			}
		}
	}

	return nil
}

// IsTemplateString checks if a string contains template syntax.
func IsTemplateString(s string) bool {
	return strings.Contains(s, "{{") && strings.Contains(s, "}}")
}

// HasTemplates checks if the output contains any template strings.
func (o *Output) HasTemplates() bool {
	return (o.Data != nil && hasTemplatesInMap(o.Data)) ||
		(o.Stream != nil && hasTemplatesInStream(o.Stream)) ||
		(o.Error != "" && IsTemplateString(o.Error)) ||
		(o.Headers != nil && hasTemplatesInHeaders(o.Headers))
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
		if slices.ContainsFunc(v, hasTemplatesInValue) {
			return true
		}
	}

	return false
}
