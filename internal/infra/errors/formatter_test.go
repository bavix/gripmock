package errors_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/errors"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

type mockResult struct {
	found   *stuber.Stub
	similar *stuber.Stub
}

func (m *mockResult) Found() *stuber.Stub   { return m.found }
func (m *mockResult) Similar() *stuber.Stub { return m.similar }

func formatError(t *testing.T, q stuber.Query, r errors.Result) string {
	t.Helper()

	err := errors.NewStubNotFoundFormatter().Format(q, r)
	require.Error(t, err)

	return err.Error()
}

func TestFormatter_NoSimilar(t *testing.T) {
	t.Parallel()

	msg := formatError(t, stuber.Query{
		Service: "svc.Test",
		Method:  "Do",
		Input:   []map[string]any{{"key": "value"}},
	}, &stuber.Result{})

	require.Contains(t, msg, "No matching stub found")
	require.Contains(t, msg, "Service: svc.Test")
	require.Contains(t, msg, "Method: Do")
	require.Contains(t, msg, "Request input:")
	require.Contains(t, msg, `"key": "value"`)
	require.Contains(t, msg, "No similar stubs found.")
}

func TestFormatter_EmptyInput(t *testing.T) {
	t.Parallel()

	msg := formatError(t, stuber.Query{Service: "svc", Method: "m", Input: []map[string]any{}}, &stuber.Result{})
	require.Contains(t, msg, "Request input:")
	require.Contains(t, msg, "(empty)")
}

func TestFormatter_StreamInput(t *testing.T) {
	t.Parallel()

	msg := formatError(t, stuber.Query{
		Service: "svc",
		Method:  "m",
		Input: []map[string]any{
			{"a": 1},
			{"b": 2},
		},
	}, &stuber.Result{})

	require.Contains(t, msg, "Request input (stream):")
	require.Contains(t, msg, "[0]")
	require.Contains(t, msg, "[1]")
}

func TestFormatter_RequestHeaders(t *testing.T) {
	t.Parallel()

	msg := formatError(t, stuber.Query{
		Service: "svc",
		Method:  "m",
		Headers: map[string]any{"x-trace-id": "abc"},
	}, &stuber.Result{})

	require.Contains(t, msg, "Request headers:")
	require.Contains(t, msg, `"x-trace-id": "abc"`)
}

func TestFormatter_SimilarUnary_WithAnyOf(t *testing.T) {
	t.Parallel()

	msg := formatError(t, stuber.Query{Service: "svc", Method: "m", Input: []map[string]any{{"k": "v"}}}, &mockResult{
		similar: &stuber.Stub{
			ID:       uuid.New(),
			Priority: 7,
			Options:  stuber.StubOptions{Times: 2},
			Input: stuber.InputData{
				Equals: map[string]any{"k": "expected"},
				AnyOf: []stuber.AnyOfElement{
					{Contains: map[string]any{"status": "open"}},
					{Contains: map[string]any{"status": "closed"}},
				},
			},
			Output: stuber.Output{Data: map[string]any{"ignored": true}},
		},
	})

	require.Contains(t, msg, "Closest match:")
	require.Contains(t, msg, "priority: 7")
	require.Contains(t, msg, "times: 2")
	require.Contains(t, msg, "input rules:")
	require.Contains(t, msg, "input.equals:")
	require.Contains(t, msg, "input.anyOf: selected [0], hidden: 1")
	require.Contains(t, msg, "input.anyOf[0].contains:")
	require.NotContains(t, msg, "input.anyOf[1]")
	require.NotContains(t, msg, "id:")
	require.NotContains(t, msg, "output:")
}

func TestFormatter_SimilarHeaders_WithAnyOf(t *testing.T) {
	t.Parallel()

	msg := formatError(t, stuber.Query{Service: "svc", Method: "m"}, &mockResult{
		similar: &stuber.Stub{
			Priority: 3,
			Headers: stuber.InputHeader{
				Equals: map[string]any{"x-env": "prod"},
				AnyOf: []stuber.AnyOfHeaderElement{
					{Matches: map[string]any{"x-user": "^admin-.*"}},
					{Matches: map[string]any{"x-user": "^dev-.*"}},
				},
			},
		},
	})

	require.Contains(t, msg, "headers:")
	require.Contains(t, msg, "headers.equals:")
	require.Contains(t, msg, "headers.anyOf: selected [0], hidden: 1")
	require.Contains(t, msg, "headers.anyOf[0].matches:")
	require.NotContains(t, msg, "headers.anyOf[1]")
}

func TestFormatter_SimilarClientStream(t *testing.T) {
	t.Parallel()

	msg := formatError(t, stuber.Query{
		Service: "svc",
		Method:  "m",
		Input:   []map[string]any{{"step": 1}, {"step": 2}},
	}, &mockResult{
		similar: &stuber.Stub{
			Priority: 5,
			Inputs: []stuber.InputData{
				{Equals: map[string]any{"step": 1}},
				{AnyOf: []stuber.AnyOfElement{{Matches: map[string]any{"step": "^2$"}}}},
			},
		},
	})

	require.Contains(t, msg, "input rules (stream):")
	require.Contains(t, msg, "inputs[0].equals:")
	require.Contains(t, msg, "inputs[1].anyOf: selected [0], hidden: 0")
	require.Contains(t, msg, "inputs[1].anyOf[0].matches:")
}

func TestFormatter_FiltersNonSerializable(t *testing.T) {
	t.Parallel()

	msg := formatError(t, stuber.Query{
		Service: "svc",
		Method:  "m",
		Input: []map[string]any{{
			"ok":   "value",
			"bad":  func() {},
			"list": []any{"a", func() {}},
		}},
	}, &stuber.Result{})

	require.Contains(t, msg, `"ok": "value"`)
	require.Contains(t, msg, `"list": [`)
	require.NotContains(t, msg, `"bad"`)
}
