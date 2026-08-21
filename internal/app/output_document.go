package app

import (
	"io"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/goccy/go-json"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

const (
	maxDocumentMessages = 10_000
	maxDocumentBytes    = 8 << 20
	documentEchoLimit   = 512
)

var (
	errDocumentNotSingle       = errors.New("a data template must render exactly one JSON value")
	errDocumentTooManyMessages = errors.New("stream template rendered too many messages")
	errDocumentTooLarge        = errors.New("response template rendered too much text")
	errDocumentGripMockKey     = errors.New("_gripmock is only allowed in stream messages")
)

func renderDocument(engine *template.Engine, document string, templateData template.Data) ([]any, string, error) {
	text, err := engine.RenderLimited(document, templateData, maxDocumentBytes)
	if err != nil {
		if errors.Is(err, template.ErrRenderLimit) {
			return nil, "", errors.Wrapf(errDocumentTooLarge, "more than %d bytes", maxDocumentBytes)
		}

		return nil, "", errors.Wrap(err, "failed to render output template")
	}

	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()

	var values []any

	for {
		var value any

		if err := decoder.Decode(&value); err != nil {
			if errors.Is(err, io.EOF) {
				return values, text, nil
			}

			return nil, text, errors.Wrapf(err, "output template did not render valid JSON (rendered=%s)", echoDocument(text))
		}

		if len(values) == maxDocumentMessages {
			return nil, text, errors.Wrapf(errDocumentTooManyMessages, "more than %d", maxDocumentMessages)
		}

		values = append(values, value)
	}
}

func renderDocumentData(engine *template.Engine, document string, templateData template.Data) (any, error) {
	values, text, err := renderDocument(engine, document, templateData)
	if err != nil {
		return nil, err
	}

	if len(values) != 1 {
		return nil, errors.Wrapf(errDocumentNotSingle, "rendered %d (rendered=%s)", len(values), echoDocument(text))
	}

	if message, isObject := values[0].(map[string]any); isObject {
		if _, marked := message[stuber.GripMockKey]; marked {
			return nil, errDocumentGripMockKey
		}
	}

	return values[0], nil
}

func renderDocumentStream(engine *template.Engine, document string, templateData template.Data) ([]any, error) {
	values, _, err := renderDocument(engine, document, templateData)
	if err != nil {
		return nil, err
	}

	if len(values) == 1 {
		if messages, isArray := values[0].([]any); isArray {
			values = messages
		}
	}

	if len(values) > maxDocumentMessages {
		return nil, errors.Wrapf(errDocumentTooManyMessages, "%d > %d", len(values), maxDocumentMessages)
	}

	return values, nil
}

func echoDocument(text string) string {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) <= documentEchoLimit {
		return trimmed
	}

	return trimmed[:documentEchoLimit] + "…"
}
