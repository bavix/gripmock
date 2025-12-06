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

// Query represents a query for finding stubs.
// Prefer QueryV2 for better performance and streaming support.
type Query struct {
	ID      *uuid.UUID     `json:"id,omitempty"` // The unique identifier of the stub (optional).
	Service string         `json:"service"`      // The service name to search for.
	Method  string         `json:"method"`       // The method name to search for.
	Headers map[string]any `json:"headers"`      // The headers to match.
	Data    map[string]any `json:"data"`         // The data to match.

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
//
// Parameters:
// - r: The HTTP request to parse.
//
// Returns:
// - Query: The parsed query.
// - error: An error if the request body cannot be parsed.
func NewQuery(r *http.Request) (Query, error) {
	q := Query{
		toggles: toggles(r),
	}

	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()

	err := decoder.Decode(&q)

	return q, err
}

// RequestInternal returns true if the query is marked as internal.
func (q Query) RequestInternal() bool {
	return q.toggles.Has(RequestInternalFlag)
}

// QueryV2 represents a query for finding stubs with improved performance.
// Input is now a slice to support both unary and streaming requests efficiently.
type QueryV2 struct {
	ID      *uuid.UUID       `json:"id,omitempty"` // The unique identifier of the stub (optional).
	Service string           `json:"service"`      // The service name to search for.
	Method  string           `json:"method"`       // The method name to search for.
	Headers map[string]any   `json:"headers"`      // The headers to match.
	Input   []map[string]any `json:"input"`        // The input data to match (supports both unary and streaming).

	toggles features.Toggles
}

// NewQueryV2 creates a new QueryV2 from an HTTP request.
func NewQueryV2(r *http.Request) (QueryV2, error) {
	q := QueryV2{
		toggles: toggles(r),
	}

	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()

	err := decoder.Decode(&q)

	return q, err
}

// RequestInternal returns true if the query is marked as internal.
func (q QueryV2) RequestInternal() bool {
	return q.toggles.Has(RequestInternalFlag)
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
