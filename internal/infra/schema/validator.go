package schema

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Schema validators for legacy and v4 stub arrays.

var (
	// ErrMissingService indicates that the required 'service' field is missing.
	ErrMissingService = errors.New("missing field 'service'")
	// ErrMissingMethod indicates that the required 'method' field is missing.
	ErrMissingMethod = errors.New("missing field 'method'")
	// ErrMissingOrEmptyOutputs indicates that the required 'outputs' field is missing or empty.
	ErrMissingOrEmptyOutputs = errors.New("missing or empty 'outputs'")
)

// ValidateStubV4 performs a lightweight structural validation of the v4 stub list.
// It checks required fields for each item: service, method, outputs (non-empty array).
// This avoids adding a heavy JSON Schema dependency while protecting endpoints.
func ValidateStubV4(raw []byte) error {
	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		return err
	}

	for i, it := range items {
		if _, ok := it["service"]; !ok {
			return fmt.Errorf("item index %d: %w", i, ErrMissingService)
		}

		if _, ok := it["method"]; !ok {
			return fmt.Errorf("item index %d: %w", i, ErrMissingMethod)
		}

		outs, ok := it["outputs"].([]any)
		if !ok || len(outs) == 0 {
			return fmt.Errorf("item index %d: %w", i, ErrMissingOrEmptyOutputs)
		}
	}

	return nil
}

// ValidateLegacy performs a minimal structural check for legacy stubs.
// It ensures presence of 'service' and 'method'.
func ValidateLegacy(raw []byte) error {
	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		return err
	}

	for i, it := range items {
		if _, ok := it["service"]; !ok {
			return fmt.Errorf("item index %d: %w", i, ErrMissingService)
		}

		if _, ok := it["method"]; !ok {
			return fmt.Errorf("item index %d: %w", i, ErrMissingMethod)
		}
	}

	return nil
}
