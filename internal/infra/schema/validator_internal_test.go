package schema

import (
	"encoding/json"
	"testing"
)

func TestValidateLegacy_OK(t *testing.T) {
	t.Parallel()

	data := []map[string]any{{
		"service": "pkg.Svc", "method": "Foo", "output": map[string]any{"data": map[string]any{"x": 1}},
	}}

	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	if err := ValidateLegacy(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateV4_OK(t *testing.T) {
	t.Parallel()

	data := []map[string]any{{
		"service": "pkg.Svc", "method": "Foo", "outputs": []any{map[string]any{"data": map[string]any{"x": 1}}},
	}}

	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	if err := ValidateStubV4(raw); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
