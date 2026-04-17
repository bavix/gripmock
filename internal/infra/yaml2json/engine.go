package yaml2json

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/bavix/gripmock/v3/pkg/plugins"
)

type engine struct {
	reg plugins.Registry
}

func newEngine(reg plugins.Registry) *engine {
	return &engine{reg: reg}
}

func (e *engine) Execute(ctx context.Context, name string, data []byte) ([]byte, error) {
	// Check if data contains template markers
	if !containsTemplateMarkers(data) {
		return data, nil
	}

	// Execute template functions at load time (for static templates)
	// and escape runtime templates ({{.Request}}, {{.Headers}}, etc.) for later processing
	executed := e.executeTemplates(ctx, name, data)

	return executed, nil
}

func containsTemplateMarkers(data []byte) bool {
	dataStr := string(data)

	return strings.Contains(dataStr, "{{") && strings.Contains(dataStr, "}}")
}

// runtimeTemplatePattern matches templates that should be processed at runtime.
// These contain request/metadata-bound values and per-call context.
//
//nolint:lll
var runtimeTemplatePattern = regexp.MustCompile(
	`\{\{[^}]*\.(Request|Headers|MessageIndex|Requests|State|RequestTime|Timestamp|StubID|RequestID|AttemptNumber|AttemptIndex|MaxAttempts|TotalAttempts)[^}]*\}\}`,
)

// chainedCallTemplatePattern matches chained call expressions like
// {{foo.Bar.Baz}} which depend on runtime objects and must not be pre-executed.
var chainedCallTemplatePattern = regexp.MustCompile(`\{\{\s*\(?\s*[A-Za-z_][A-Za-z0-9_]*\s*\)?\s*(?:\.[A-Za-z_][A-Za-z0-9_]*)+[^}]*\}\}`)

// executeTemplates executes static template functions at load time and escapes runtime templates.
func (e *engine) executeTemplates(ctx context.Context, name string, data []byte) []byte {
	// Get template functions from registry
	funcs := e.getTemplateFuncs(ctx)

	// Process line by line
	lines := strings.Split(string(data), "\n")

	var result []string

	for _, line := range lines {
		if !containsTemplateMarkers([]byte(line)) {
			result = append(result, line)

			continue
		}

		// Check if this line contains runtime-only templates.
		if runtimeTemplatePattern.MatchString(line) || chainedCallTemplatePattern.MatchString(line) {
			// Escape runtime templates for later processing
			result = append(result, escapeTemplateInLine(line))
		} else {
			// Execute static template functions at load time
			executed := e.executeStaticTemplate(name, line, funcs)
			result = append(result, executed)
		}
	}

	return []byte(strings.Join(result, "\n"))
}

// getTemplateFuncs returns template functions from the registry.
func (e *engine) getTemplateFuncs(ctx context.Context) template.FuncMap {
	if e.reg == nil {
		return template.FuncMap{}
	}

	return getFunctions(ctx, e.reg)
}

// getFunctions returns template functions from the registry, wrapped for use with text/template.
func getFunctions(ctx context.Context, reg plugins.Registry) template.FuncMap {
	raw := reg.Funcs()
	out := make(template.FuncMap, len(raw))

	for name, fn := range raw {
		if typed, ok := fn.(plugins.Func); ok && typed != nil {
			out[name] = func(args ...any) (any, error) {
				callArgs := normalizeArgs(args)

				return typed(ctx, callArgs...)
			}

			continue
		}

		out[name] = fn
	}

	return out
}

// normalizeArgs normalizes arguments for template functions.
func normalizeArgs(args []any) []any {
	if len(args) != 1 {
		return args
	}

	switch v := args[0].(type) {
	case []any:
		return v
	case []float64:
		out := make([]any, len(v))
		for i, val := range v {
			out[i] = val
		}

		return out
	default:
		return args
	}
}

// executeStaticTemplate executes a static template function at load time.
func (e *engine) executeStaticTemplate(name, line string, funcs template.FuncMap) string {
	if len(funcs) == 0 {
		return escapeTemplateInLine(line)
	}

	// Extract and execute the template
	content := extractTemplateContent(line)

	tmpl, err := template.New(name).Funcs(funcs).Parse(content)
	if err != nil {
		return escapeTemplateInLine(line)
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, nil); err != nil {
		return escapeTemplateInLine(line)
	}

	// Replace template with executed result
	return replaceTemplateInLine(line, buf.String())
}

// extractTemplateContent extracts the template content from a line.
func extractTemplateContent(line string) string {
	start := strings.Index(line, "{{")

	end := strings.Index(line, "}}")
	if start == -1 || end == -1 || end < start {
		return line
	}

	return line[start : end+2]
}

// replaceTemplateInLine replaces the template content in a line with the executed result.
func replaceTemplateInLine(line, replacement string) string {
	start := strings.Index(line, "{{")

	end := strings.Index(line, "}}")
	if start == -1 || end == -1 || end < start {
		return line
	}

	before := line[:start]
	after := line[end+2:]

	// Determine if we need quotes around the replacement
	trimmedBefore := strings.TrimSpace(before)
	needsQuotes := strings.HasSuffix(trimmedBefore, ":") || strings.HasSuffix(trimmedBefore, "-")

	if needsQuotes && !isNumericOrJSON(replacement) {
		return fmt.Sprintf("%s\"%s\"%s", before, replacement, after)
	}

	return fmt.Sprintf("%s%s%s", before, replacement, after)
}

// isNumericOrJSON checks if a string is a numeric value or JSON.
func isNumericOrJSON(s string) bool {
	if s == "" {
		return false
	}
	// Check if it's a JSON object/array
	if strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") {
		return true
	}
	// Check if it's a number
	return strings.Contains(s, ".") || strings.HasPrefix(s, "-")
}

// escapeTemplateInLine escapes template syntax so YAML parser treats it as string literals.
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
		tmpl := line[startIdx:endIdx]

		isQuoted := isInsideQuotes(line, startIdx)

		//nolint:nestif
		if isQuoted {
			result.WriteString(before)
			result.WriteString(tmpl)
		} else {
			trimmedBefore := strings.TrimSpace(before)

			if strings.HasSuffix(trimmedBefore, ":") || strings.HasSuffix(trimmedBefore, "-") {
				if strings.Contains(tmpl, `"`) {
					result.WriteString(before)
					result.WriteString(`'`)
					result.WriteString(tmpl)
					result.WriteString(`'`)
				} else {
					result.WriteString(before)
					result.WriteString(`"`)
					result.WriteString(tmpl)
					result.WriteString(`"`)
				}
			} else {
				result.WriteString(before)
				result.WriteString(tmpl)
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
