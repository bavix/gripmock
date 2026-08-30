package app

import (
	"maps"

	"github.com/cockroachdb/errors"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

type renderOptions struct {
	skipData     bool
	renderStream bool
}

func renderOutput(
	engine *template.Engine,
	output stuber.Output,
	templateData template.Data,
	opts renderOptions,
) (stuber.Output, error) {
	rendered, err := renderPayload(engine, output, templateData, opts)
	if err != nil {
		return stuber.Output{}, err
	}

	if rendered.Headers, err = renderMetadata(engine, output.Headers, templateData, "header"); err != nil {
		return stuber.Output{}, err
	}

	if rendered.Trailers, err = renderMetadata(engine, output.Trailers, templateData, "trailer"); err != nil {
		return stuber.Output{}, err
	}

	if output.Error != "" && template.IsTemplateString(output.Error) {
		errorStr, err := engine.ProcessError(output.Error, templateData)
		if err != nil {
			return stuber.Output{}, errors.Wrap(err, "failed to process error template")
		}

		rendered.Error = errorStr
	}

	return rendered, nil
}

func renderPayload(
	engine *template.Engine,
	output stuber.Output,
	templateData template.Data,
	opts renderOptions,
) (stuber.Output, error) {
	rendered := output

	var err error

	if output.HasTemplate() {
		rendered.Template = false
		rendered.Data, rendered.Stream = nil, nil

		if document, isStream := output.Document(); isStream {
			rendered.Stream, err = renderDocumentStream(engine, document, templateData)
		} else if document != "" {
			rendered.Data, err = renderDocumentData(engine, document, templateData)
		}

		if err != nil {
			return stuber.Output{}, err
		}

		return rendered, nil
	}

	if output.IsServerStream() {
		if opts.renderStream {
			if rendered.Stream, err = renderStreamElements(engine, output.Messages(), templateData); err != nil {
				return stuber.Output{}, err
			}
		}

		return rendered, nil
	}

	if !opts.skipData {
		if rendered.Data, err = renderData(engine, output.Data, templateData); err != nil {
			return stuber.Output{}, err
		}
	}

	return rendered, nil
}

func renderData(engine *template.Engine, data any, templateData template.Data) (any, error) {
	rendered, err := engine.ProcessValue(copyForTemplates(data), templateData)
	if err != nil {
		return nil, errors.Wrap(err, errMsgProcessTemplates)
	}

	return rendered, nil
}

func renderStreamElements(engine *template.Engine, stream []any, templateData template.Data) ([]any, error) {
	if stream == nil {
		return nil, nil
	}

	out := make([]any, len(stream))

	for i, item := range stream {
		element, err := engine.ProcessValue(copyForTemplates(item), templateData)
		if err != nil {
			return nil, errors.Wrap(err, "failed to process stream template")
		}

		out[i] = element
	}

	return out, nil
}

func renderMetadata(
	engine *template.Engine,
	values map[string]string,
	templateData template.Data,
	kind string,
) (map[string]string, error) {
	if !template.HasTemplatesInHeaders(values) {
		return values, nil
	}

	rendered := maps.Clone(values)
	if err := engine.ProcessHeaders(rendered, templateData); err != nil {
		return nil, errors.Wrapf(err, "failed to process %s templates", kind)
	}

	return rendered, nil
}
