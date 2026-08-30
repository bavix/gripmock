package stuber

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	"github.com/bavix/gripmock/v3/internal/infra/types"
)

const (
	SourceFile  = "file"
	SourceRest  = "rest"
	SourceMCP   = "mcp"
	SourceProxy = "proxy"
)

func IsKnownSource(source string) bool {
	switch source {
	case SourceFile, SourceRest, SourceMCP, SourceProxy:
		return true
	default:
		return false
	}
}

// StubOptions holds optional behavior settings for a stub.
type StubOptions struct {
	Times int `json:"times,omitempty" validate:"gte=0"`
}

// Stub represents a gRPC service method and its associated data.
type Stub struct {
	ID       uuid.UUID     `json:"id"`
	Service  string        `json:"service"           validate:"required"`
	Method   string        `json:"method"            validate:"required"`
	Session  string        `json:"session,omitempty"`
	Priority int           `json:"priority"`
	Options  StubOptions   `json:"options,omitempty"` //nolint:modernize
	Headers  InputHeader   `json:"headers"`
	Input    InputData     `json:"input"             validate:"valid_input_config"`
	Inputs   []InputData   `json:"inputs,omitempty"  validate:"valid_input_config"`
	Output   Output        `json:"output"            validate:"valid_output_config"`
	Effects  []Effect      `json:"effects,omitempty" validate:"valid_effects"`
	Source   string        `json:"source,omitempty"`
	Handler  StreamHandler `json:"-"`

	UnaryHandler        UnaryHandler        `json:"-"`
	ServerStreamHandler ServerStreamHandler `json:"-"`
	ClientStreamHandler ClientStreamHandler `json:"-"`

	Used bool `json:"used,omitempty"`
}

// StreamHandler processes a bidirectional stream directly.
type StreamHandler func(ctx context.Context, stream any) error

type UnaryHandler func(ctx context.Context, in any) (any, error)

// ServerStreamHandler writes the response stream itself, given the decoded request.
type ServerStreamHandler func(ctx context.Context, in any, stream any) error

type ClientStreamHandler func(ctx context.Context, messages []any) (any, error)

const (
	EffectActionUpsert = "upsert"
	EffectActionDelete = "delete"
)

// Effect represents a side effect executed after stub match.
type Effect struct {
	Action string         `json:"action"`
	ID     string         `json:"id,omitempty"`
	Stub   map[string]any `json:"stub,omitempty"`
}

// GripMockKey is the reserved map key for per-element stream metadata.
const GripMockKey = "_gripmock"

type GripMockElement struct {
	Delay    types.Delay
	HasDelay bool

	Error    string
	Code     *codes.Code
	Details  []map[string]any
	HasError bool
}

func ExtractGripMock(m map[string]any) GripMockElement {
	var out GripMockElement

	if m == nil {
		return out
	}

	raw, has := m[GripMockKey]
	if !has {
		return out
	}

	delete(m, GripMockKey)

	gk, ok := raw.(map[string]any)
	if !ok {
		return out
	}

	out.Delay, out.HasDelay = gripMockDelay(gk)
	out.Error, out.Code, out.Details, out.HasError = gripMockError(gk)

	return out
}

func gripMockDelay(gk map[string]any) (types.Delay, bool) {
	s, ok := gk["delay"].(string)
	if !ok {
		return "", false
	}

	delay := types.Delay(s)

	_, err := delay.Parse()
	if err != nil {
		return "", false
	}

	return delay, true
}

func gripMockError(gk map[string]any) (string, *codes.Code, []map[string]any, bool) {
	msg, hasMsg := gk["error"].(string)

	var code *codes.Code

	if raw, has := gk["code"]; has {
		if n, ok := toCodeNumber(raw); ok {
			c := codes.Code(n)
			code = &c
		}
	}

	details := toDetailList(gk["details"])

	if !hasMsg && code == nil && len(details) == 0 {
		return "", nil, nil, false
	}

	return msg, code, details, true
}

//nolint:cyclop // one branch per JSON number shape; a table would not be clearer.
func toCodeNumber(raw any) (uint32, bool) {
	switch v := raw.(type) {
	case json.Number:
		n, err := v.Int64()
		if err != nil || n < 0 {
			return 0, false
		}

		return uint32(n), true //nolint:gosec
	case float64:
		if v < 0 {
			return 0, false
		}

		return uint32(v), true
	case int:
		if v < 0 {
			return 0, false
		}

		return uint32(v), true //nolint:gosec
	case int64:
		if v < 0 {
			return 0, false
		}

		return uint32(v), true //nolint:gosec
	case uint32:
		return v, true
	default:
		return 0, false
	}
}

func toDetailList(raw any) []map[string]any {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}

	details := make([]map[string]any, 0, len(items))

	for _, item := range items {
		if detail, ok := item.(map[string]any); ok {
			details = append(details, detail)
		}
	}

	if len(details) == 0 {
		return nil
	}

	return details
}

func ExtractGripMockDelay(m map[string]any) (types.Delay, bool) {
	el := ExtractGripMock(m)

	return el.Delay, el.HasDelay
}

// EffectiveTimes returns the stub's max match count; 0 means unlimited.
func (s *Stub) EffectiveTimes() int {
	return s.Options.Times
}

// Matchers returns the request matchers in one shape: a unary stub has exactly one,
// a client-streaming stub keeps the declared order.
func (s *Stub) Matchers() []InputData {
	if s.Inputs != nil {
		return s.Inputs
	}

	return []InputData{s.Input}
}

// DeclaresStreamMatchers reports whether the stub declared `inputs`, which selects the
// stream matcher: every message must match, and an empty list matches nothing.
func (s *Stub) DeclaresStreamMatchers() bool {
	return s.Inputs != nil
}

// IsClientStream returns true if this stub is for client streaming requests (has Inputs data).
func (s *Stub) IsClientStream() bool {
	return len(s.Inputs) > 0
}

// IsServerStream returns true if this stub answers with a stream: literal messages
// in Output.Stream, or a template in that slot.
func (s *Stub) IsServerStream() bool {
	return s.Output.IsServerStream()
}

// IsBidirectional returns true if this stub can handle bidirectional streaming.
func (s *Stub) IsBidirectional() bool {
	return s.IsClientStream() && s.IsServerStream()
}

// Key returns the unique identifier of the stub.
func (s *Stub) Key() uuid.UUID {
	return s.ID
}

// Score returns the priority score of the stub.
func (s *Stub) Score() int {
	return s.Priority
}

// InputData represents the input data of a gRPC request.
type InputData struct {
	Equals           map[string]any `json:"equals"`
	Contains         map[string]any `json:"contains"`
	Matches          map[string]any `json:"matches"`
	Glob             map[string]any `json:"glob,omitempty"`
	AnyOf            []AnyOfElement `json:"anyOf,omitempty"`
	IgnoreArrayOrder bool           `json:"ignoreArrayOrder,omitempty"`
}

// AnyOfElement is a flat alternative matcher inside InputData.
type AnyOfElement struct {
	IgnoreArrayOrder bool           `json:"ignoreArrayOrder,omitempty"`
	Equals           map[string]any `json:"equals"`
	Contains         map[string]any `json:"contains"`
	Matches          map[string]any `json:"matches"`
	Glob             map[string]any `json:"glob,omitempty"`
}

// GetEquals returns the data to match exactly.
func (i InputData) GetEquals() map[string]any {
	return i.Equals
}

// GetContains returns the data to match partially.
func (i InputData) GetContains() map[string]any {
	return i.Contains
}

// GetMatches returns the data to match using regular expressions.
func (i InputData) GetMatches() map[string]any {
	return i.Matches
}

// InputHeader represents the headers of a gRPC request.
type InputHeader struct {
	Equals   map[string]any       `json:"equals"`
	Contains map[string]any       `json:"contains"`
	Matches  map[string]any       `json:"matches"`
	Glob     map[string]any       `json:"glob,omitempty"`
	AnyOf    []AnyOfHeaderElement `json:"anyOf,omitempty"`
}

// AnyOfHeaderElement is a flat alternative matcher inside InputHeader.
type AnyOfHeaderElement struct {
	Equals   map[string]any `json:"equals"`
	Contains map[string]any `json:"contains"`
	Matches  map[string]any `json:"matches"`
	Glob     map[string]any `json:"glob,omitempty"`
}

// GetEquals returns the headers to match exactly.
func (i InputHeader) GetEquals() map[string]any {
	return i.Equals
}

// GetContains returns the headers to match partially.
func (i InputHeader) GetContains() map[string]any {
	return i.Contains
}

// GetMatches returns the headers to match using regular expressions.
func (i InputHeader) GetMatches() map[string]any {
	return i.Matches
}

// Len returns the total number of headers to match.
func (i InputHeader) Len() int {
	n := len(i.Equals) + len(i.Matches) + len(i.Contains) + len(i.Glob)

	for _, alt := range i.AnyOf {
		n += len(alt.Equals) + len(alt.Matches) + len(alt.Contains) + len(alt.Glob)
	}

	return n
}

// Output represents the output data of a gRPC response. With Template set, Data or
// Stream holds a Go template instead of a literal payload; which of the two holds it
// decides whether the stub answers with one message or with a stream.
type Output struct {
	Headers  map[string]string `json:"headers"`
	Trailers map[string]string `json:"trailers,omitempty"`
	Data     any               `json:"data,omitempty"`
	Stream   any               `json:"stream,omitempty"`
	Template bool              `json:"template,omitempty"`
	Error    string            `json:"error"`
	Code     *codes.Code       `json:"code,omitempty"`
	Details  []map[string]any  `json:"details,omitempty"`
	Delay    types.Delay       `json:"delay,omitempty"`
}

// Messages returns the response messages in one shape: a unary payload is the only
// element, a stream keeps its order, a template has none until it is rendered.
func (o Output) Messages() []any {
	if o.Template {
		return nil
	}

	if messages, ok := o.Stream.([]any); ok {
		return messages
	}

	if o.Stream != nil || o.Data == nil {
		return nil
	}

	return []any{o.Data}
}

// IsServerStream reports whether the stub answers with a stream. It is declared by
// the wire shape (stream vs data), never inferred from the payload.
func (o Output) IsServerStream() bool {
	return o.Stream != nil
}

// Document returns the template text and whether it renders the stream.
func (o Output) Document() (string, bool) {
	if text, ok := o.document(o.Stream); ok {
		return text, true
	}

	text, _ := o.document(o.Data)

	return text, false
}

// HasTemplate reports whether the output is computed from a template.
func (o Output) HasTemplate() bool {
	return o.Template
}

func (o Output) document(slot any) (string, bool) {
	if !o.Template {
		return "", false
	}

	text, ok := slot.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return "", false
	}

	return text, true
}
