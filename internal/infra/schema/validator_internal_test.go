package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateLegacy_OK(t *testing.T) {
	t.Parallel()

	data := []map[string]any{{
		"service": "pkg.Svc", "method": "Foo", "output": map[string]any{"data": map[string]any{"x": 1}},
	}}

	raw, err := json.Marshal(data)
	require.NoError(t, err)
	require.NoError(t, ValidateLegacy(raw))
}

func TestValidateV4_OK(t *testing.T) {
	t.Parallel()

	data := []map[string]any{{
		"service": "pkg.Svc", "method": "Foo", "outputs": []any{map[string]any{"data": map[string]any{"x": 1}}},
	}}

	raw, err := json.Marshal(data)
	require.NoError(t, err)
	require.NoError(t, ValidateStubV4(raw))
}

func TestValidateStubV4_Errors(t *testing.T) {
	t.Parallel()

	t.Run("invalid JSON", func(t *testing.T) {
		t.Parallel()

		err := ValidateStubV4([]byte("{"))
		require.Error(t, err)
	})

	t.Run("missing service", func(t *testing.T) {
		t.Parallel()

		data := []map[string]any{{"method": "Foo", "outputs": []any{map[string]any{}}}}
		raw, _ := json.Marshal(data)

		err := ValidateStubV4(raw)
		require.Error(t, err)
	})

	t.Run("missing method", func(t *testing.T) {
		t.Parallel()

		data := []map[string]any{{"service": "Svc", "outputs": []any{map[string]any{}}}}
		raw, _ := json.Marshal(data)

		err := ValidateStubV4(raw)
		require.Error(t, err)
	})

	t.Run("missing outputs", func(t *testing.T) {
		t.Parallel()

		data := []map[string]any{{"service": "Svc", "method": "Foo"}}
		raw, _ := json.Marshal(data)

		err := ValidateStubV4(raw)
		require.Error(t, err)
	})

	t.Run("empty outputs", func(t *testing.T) {
		t.Parallel()

		data := []map[string]any{{"service": "Svc", "method": "Foo", "outputs": []any{}}}
		raw, _ := json.Marshal(data)

		err := ValidateStubV4(raw)
		require.Error(t, err)
	})
}

func TestValidateLegacy_Errors(t *testing.T) {
	t.Parallel()

	t.Run("invalid JSON", func(t *testing.T) {
		t.Parallel()

		err := ValidateLegacy([]byte("not json"))
		require.Error(t, err)
	})

	t.Run("missing service", func(t *testing.T) {
		t.Parallel()

		data := []map[string]any{{"method": "Foo"}}
		raw, _ := json.Marshal(data)

		err := ValidateLegacy(raw)
		require.Error(t, err)
	})

	t.Run("missing method", func(t *testing.T) {
		t.Parallel()

		data := []map[string]any{{"service": "Svc"}}
		raw, _ := json.Marshal(data)

		err := ValidateLegacy(raw)
		require.Error(t, err)
	})
}
