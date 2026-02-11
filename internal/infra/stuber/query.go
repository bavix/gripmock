package stuber

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/bavix/features"
)

const (
	// RequestInternalFlag is a feature flag for internal requests.
	RequestInternalFlag features.Flag = iota
)

// queryJSON is used for JSON unmarshaling to support both "data" (legacy) and "input" formats.
type queryJSON struct {
	ID      *uuid.UUID       `json:"id,omitempty"`
	Service string           `json:"service"`
	Method  string           `json:"method"`
	Headers map[string]any   `json:"headers"`
	Data    map[string]any   `json:"data"`  // Legacy: unary request body
	Input   []map[string]any `json:"input"` // Canonical: supports unary and streaming
}

// Query represents a query for finding stubs.
// Supports both unary (Input with one element) and streaming (Input with multiple elements).
// JSON accepts "data" (legacy, maps to Input[0]) or "input" (array). Prefer "input".
type Query struct {
	ID      *uuid.UUID       `json:"id,omitempty"` // The unique identifier of the stub (optional).
	Service string           `json:"service"`      // The service name to search for.
	Method  string           `json:"method"`       // The method name to search for.
	Headers map[string]any   `json:"headers"`      // The headers to match.
	Input   []map[string]any `json:"input"`        // The input data to match (unary or streaming).

	toggles features.Toggles
}

func toggles(r *http.Request) features.Toggles {
	var flags []features.Flag

	if len(r.Header.Values("X-Gripmock-Requestinternal")) > 0 {
		flags = append(flags, RequestInternalFlag)
	}

	return features.New(flags...)
}

// NewQuery creates a new Query from an HTTP request.
// Supports both legacy "data" (single object) and "input" (array) in JSON body.
func NewQuery(r *http.Request) (Query, error) {
	q := Query{
		toggles: toggles(r),
	}

	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()

	err := decoder.Decode(&q)

	return q, err
}

// NewQueryFromInput creates a Query with the given input data (convenience for programmatic use).
func NewQueryFromInput(service, method string, input []map[string]any, headers map[string]any) Query {
	return Query{
		Service: service,
		Method:  method,
		Input:   input,
		Headers: headers,
	}
}

// UnmarshalJSON implements json.Unmarshaler to support both "data" and "input" in request body.
func (q *Query) UnmarshalJSON(data []byte) error {
	var raw queryJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	q.ID = raw.ID
	q.Service = raw.Service
	q.Method = raw.Method
	q.Headers = raw.Headers

	switch {
	case len(raw.Input) > 0:
		q.Input = raw.Input
	case raw.Data != nil:
		q.Input = []map[string]any{raw.Data}
	default:
		q.Input = nil
	}

	return nil
}

// RequestInternal returns true if the query is marked as internal.
func (q *Query) RequestInternal() bool {
	return q.toggles.Has(RequestInternalFlag)
}

// Data returns the first input element for backward compatibility with legacy unary API.
// Returns nil if Input is empty.
func (q *Query) Data() map[string]any {
	if len(q.Input) == 0 {
		return nil
	}

	return q.Input[0]
}

// QueryBidi represents a query for bidirectional streaming.
// In bidirectional streaming, each message is treated as a separate unary request.
// The server can respond with multiple messages for each request.
type QueryBidi struct {
	ID      *uuid.UUID     `json:"id,omitempty"` // The unique identifier of the stub (optional).
	Service string         `json:"service"`      // The service name to search for.
	Method  string         `json:"method"`       // The method name to search for.
	Headers map[string]any `json:"headers"`      // The headers to match.

	toggles features.Toggles
}

// NewQueryBidi creates a new QueryBidi from an HTTP request.
func NewQueryBidi(r *http.Request) (QueryBidi, error) {
	q := QueryBidi{
		toggles: toggles(r),
	}

	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()

	err := decoder.Decode(&q)

	return q, err
}

// RequestInternal returns true if the query is marked as internal.
func (q QueryBidi) RequestInternal() bool {
	return q.toggles.Has(RequestInternalFlag)
}
