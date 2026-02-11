package yaml2json

import (
	"strings"

	"github.com/bavix/gripmock/v3/pkg/plugins"
)

type engine struct {
	reg plugins.Registry
}

func newEngine(reg plugins.Registry) *engine {
	return &engine{reg: reg}
}

func (e *engine) Execute(name string, data []byte) ([]byte, error) {
	_ = name

	// Check if data contains template markers that need escaping for YAML parser
	if !containsTemplateMarkers(data) {
		return data, nil
	}

	// Escape template syntax so YAML parser treats it as a string literal
	// This allows templates to be parsed as strings and processed later at runtime
	escaped := escapeTemplatesForYAML(data)

	return escaped, nil
}

func containsTemplateMarkers(data []byte) bool {
	dataStr := string(data)

	return strings.Contains(dataStr, "{{") && strings.Contains(dataStr, "}}")
}

// escapeTemplatesForYAML escapes template syntax so YAML parser treats it as string literals
// This allows {{ }} syntax to be preserved as strings for later runtime processing.
func escapeTemplatesForYAML(data []byte) []byte {
	dataStr := string(data)
	lines := strings.Split(dataStr, "\n")

	var result []string

	for _, line := range lines {
		if strings.Contains(line, "{{") && strings.Contains(line, "}}") {
			escaped := escapeTemplateInLine(line)
			result = append(result, escaped)
		} else {
			result = append(result, line)
		}
	}

	return []byte(strings.Join(result, "\n"))
}

func escapeTemplateInLine(line string) string {
	var result strings.Builder

	lastIdx := 0

	for {
		startIdx := strings.Index(line[lastIdx:], "{{")
		if startIdx == -1 {
			result.WriteString(line[lastIdx:])

			break
		}

		startIdx += lastIdx

		endIdx := strings.Index(line[startIdx:], "}}")
		if endIdx == -1 {
			result.WriteString(line[lastIdx:])

			break
		}

		endIdx += startIdx + len("}}")

		before := line[lastIdx:startIdx]
		template := line[startIdx:endIdx]

		isQuoted := isInsideQuotes(line, startIdx)

		//nolint:nestif
		if isQuoted {
			result.WriteString(before)
			result.WriteString(template)
		} else {
			trimmedBefore := strings.TrimSpace(before)

			if strings.HasSuffix(trimmedBefore, ":") || strings.HasSuffix(trimmedBefore, "-") {
				if strings.Contains(template, `"`) {
					result.WriteString(before)
					result.WriteString(`'`)
					result.WriteString(template)
					result.WriteString(`'`)
				} else {
					result.WriteString(before)
					result.WriteString(`"`)
					result.WriteString(template)
					result.WriteString(`"`)
				}
			} else {
				result.WriteString(before)
				result.WriteString(template)
			}
		}

		lastIdx = endIdx
	}

	return result.String()
}

// isInsideQuotes checks if position is inside a quoted string (single or double quotes).
func isInsideQuotes(line string, pos int) bool {
	before := line[:pos]

	doubleQuoteCount := 0
	singleQuoteCount := 0
	escaped := false

	for i := range len(before) {
		if escaped {
			escaped = false

			continue
		}

		if before[i] == '\\' {
			escaped = true

			continue
		}

		switch before[i] {
		case '"':
			doubleQuoteCount++
		case '\'':
			singleQuoteCount++
		}
	}

	return doubleQuoteCount%2 == 1 || singleQuoteCount%2 == 1
}
