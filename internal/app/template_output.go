package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

var errStreamTemplateNotArray = errors.New("streamTemplate must render an array")

func renderOutputData(engine *template.Engine, output stuber.Output, data template.Data) (any, error) {
	if strings.TrimSpace(output.DataTemplate) != "" {
		value, err := engine.RenderStructured(output.DataTemplate, data)
		if err != nil {
			return nil, fmt.Errorf("failed to process data template: %w", err)
		}

		return value, nil
	}

	value := deepCopyAny(output.Data)
	if dataMap, ok := value.(map[string]any); ok {
		if err := engine.ProcessMap(dataMap, data); err != nil {
			return nil, fmt.Errorf("failed to process dynamic templates: %w", err)
		}

		value = dataMap
	}

	return value, nil
}

func renderOutputStreamTemplate(engine *template.Engine, output stuber.Output, data template.Data) ([]any, bool, error) {
	if strings.TrimSpace(output.StreamTemplate) == "" {
		return nil, false, nil
	}

	value, err := engine.RenderStructured(output.StreamTemplate, data)
	if err != nil {
		return nil, true, fmt.Errorf("failed to process stream template: %w", err)
	}

	stream, ok := value.([]any)
	if !ok {
		return nil, true, fmt.Errorf("%w, got %T", errStreamTemplateNotArray, value)
	}

	return stream, true, nil
}
