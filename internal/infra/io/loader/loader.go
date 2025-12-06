package loader

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/goccy/go-yaml"
	"github.com/rs/zerolog"

	domain "github.com/bavix/gripmock/v3/internal/domain/types"
)

// LoadStubs reads a single file or a directory recursively and returns normalized stubs.
// - YAML and JSON are supported
// - Legacy items (without 'outputs') are logged as deprecated and converted to the minimal form.
func LoadStubs(ctx context.Context, path string) ([]domain.Stub, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, errors.Wrap(err, "stat path")
	}

	if info.IsDir() {
		return loadDir(ctx, path) //nolint:wrapcheck
	}

	stubs, err := loadFile(ctx, path)
	if err != nil {
		return nil, err
	}

	return stubs, nil
}

func loadDir(ctx context.Context, dir string) ([]domain.Stub, error) {
	var out []domain.Stub

	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !isStubFile(p) {
			return nil
		}

		items, err := loadFile(ctx, p)
		if err != nil {
			return err
		}

		out = append(out, items...)

		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "walk dir")
	}

	return out, nil
}

func isStubFile(name string) bool {
	lower := strings.ToLower(name)

	return strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}

func loadFile(ctx context.Context, file string) ([]domain.Stub, error) {
	raw, err := os.ReadFile(file) //nolint:gosec
	if err != nil {
		return nil, errors.Wrap(err, "read file")
	}

	// Accept both JSON and YAML by using yaml package (which can parse JSON too)
	var items []map[string]any
	if err := yaml.Unmarshal(raw, &items); err != nil {
		return nil, errors.Wrap(err, "unmarshal stubs")
	}

	out := make([]domain.Stub, 0, len(items))
	for _, it := range items {
		if _, ok := it["outputs"]; !ok {
			zerolog.Ctx(ctx).Warn().Str("file", file).Msg("[DEPRECATED] legacy stub format detected; please migrate to outputs")

			out = append(out, convertLegacy(it))

			continue
		}

		out = append(out, normalizeStub(it))
	}

	return out, nil
}

// normalizeStub maps a generic map into domain.Stub without deep validation.
//
//nolint:cyclop
func normalizeStub(m map[string]any) domain.Stub {
	stub := domain.Stub{}
	if v, ok := m["id"].(string); ok {
		stub.ID = v
	}

	if v, ok := m["service"].(string); ok {
		stub.Service = v
	}

	if v, ok := m["method"].(string); ok {
		stub.Method = v
	}

	if v, ok := m["priority"].(int); ok {
		stub.Priority = v
	}

	if v, ok := m["times"].(int); ok {
		stub.Times = v
	}

	// domain.Stub.Inputs is []Matcher, but we keep OutputsRaw as primary for runtime.
	if v, ok := m["inputs"].([]any); ok {
		stub.Inputs = parseMatchers(v)
	}

	if v, ok := m["headers"].(map[string]any); ok {
		matcher := parseMatcher(v)
		stub.Headers = &matcher
	}

	if v, ok := m["outputs"].([]any); ok {
		stub.OutputsRaw = parseOutputs(v)
	}

	if v, ok := m["responseHeaders"].(map[string]any); ok {
		stub.ResponseHeaders = parseStringMap(v)
	}

	if v, ok := m["responseTrailers"].(map[string]any); ok {
		stub.ResponseTrailers = parseStringMap(v)
	}

	return stub
}

// convertLegacy maps a generic map into domain.Stub.
func convertLegacy(m map[string]any) domain.Stub {
	stub := domain.Stub{}

	parseBasicFields(m, &stub)
	parseInputFields(m, &stub)
	parseOutputFields(m, &stub)

	return stub
}

// parseBasicFields extracts basic stub fields.
func parseBasicFields(m map[string]any, stub *domain.Stub) {
	if v, ok := m["id"].(string); ok {
		stub.ID = v
	}

	if v, ok := m["service"].(string); ok {
		stub.Service = v
	}

	if v, ok := m["method"].(string); ok {
		stub.Method = v
	}

	if v, ok := m["priority"].(int); ok {
		stub.Priority = v
	}

	if v, ok := m["times"].(int); ok {
		stub.Times = v
	}
}

// parseInputFields extracts input-related fields.
func parseInputFields(m map[string]any, stub *domain.Stub) {
	// Convert legacy input to v4 inputs
	if v, ok := m["input"].(map[string]any); ok {
		stub.Inputs = []domain.Matcher{parseMatcher(v)}
	}

	// Convert legacy headers to v4 headers
	if v, ok := m["headers"].(map[string]any); ok {
		matcher := parseMatcher(v)
		stub.Headers = &matcher
	}
}

// parseOutputFields extracts output-related fields.
func parseOutputFields(m map[string]any, stub *domain.Stub) {
	// Convert legacy output to v4 outputs
	if v, ok := m["output"].(map[string]any); ok {
		stub.OutputsRaw = []map[string]any{v}
	}

	if v, ok := m["responseHeaders"].(map[string]any); ok {
		stub.ResponseHeaders = parseStringMap(v)
	}

	if v, ok := m["responseTrailers"].(map[string]any); ok {
		stub.ResponseTrailers = parseStringMap(v)
	}
}

func parseMatchers(arr []any) []domain.Matcher {
	matchers := make([]domain.Matcher, 0, len(arr))
	for _, m := range arr {
		if mm, ok := m.(map[string]any); ok {
			matchers = append(matchers, parseMatcher(mm))
		}
	}

	return matchers
}

func parseMatcher(m map[string]any) domain.Matcher {
	matcher := domain.Matcher{}

	if v, ok := m["equals"].(map[string]any); ok {
		matcher.Equals = v
	}

	if v, ok := m["contains"].(map[string]any); ok {
		matcher.Contains = v
	}

	if v, ok := m["matches"].(map[string]any); ok {
		matcher.Matches = v
	}

	if v, ok := m["any"].([]any); ok {
		matcher.Any = parseMatchers(v)
	}

	if v, ok := m["ignoreArrayOrder"].(bool); ok {
		matcher.IgnoreArrayOrder = v
	}

	return matcher
}

func parseOutputs(arr []any) []map[string]any {
	outputs := make([]map[string]any, 0, len(arr))
	for _, o := range arr {
		if om, ok := o.(map[string]any); ok {
			outputs = append(outputs, om)
		}
	}

	return outputs
}

func parseStringMap(m map[string]any) map[string]string {
	result := make(map[string]string)

	for k, v := range m {
		if s, ok := v.(string); ok {
			result[k] = s
		}
	}

	return result
}
