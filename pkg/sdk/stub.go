package sdk

import (
	"maps"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

// StubBuilder defines a fluent builder for creating and registering stubs.
//
// All methods return StubBuilder for chaining.
type StubBuilder interface {
	When(input stuber.InputData) StubBuilder
	Match(kv ...any) StubBuilder
	WhenStream(inputs ...stuber.InputData) StubBuilder
	WhenHeaders(headers stuber.InputHeader) StubBuilder
	Reply(output stuber.Output) StubBuilder
	Return(kv ...any) StubBuilder
	ReplyStream(msgs ...stuber.Output) StubBuilder
	ReplyError(code codes.Code, msg string) StubBuilder
	ReplyErrorWithDetails(code codes.Code, msg string, details ...map[string]any) StubBuilder
	ReplyHeaders(headers map[string]string) StubBuilder
	ReplyHeaderPairs(kv ...string) StubBuilder
	Delay(d time.Duration) StubBuilder
	Priority(p int) StubBuilder
	Times(n int) StubBuilder
	Unary(inKey string, inVal any, outKey string, outVal any) StubBuilder
	Commit() error
}

type stubBuilderData struct {
	input    stuber.InputData
	inputs   []stuber.InputData
	headers  stuber.InputHeader
	output   stuber.Output
	priority int
	options  stuber.StubOptions
}

type stubBuilderCore struct {
	service  string
	method   string
	data     stubBuilderData
	onCommit func(stub *stuber.Stub) error
}

func (c *stubBuilderCore) When(input stuber.InputData) StubBuilder {
	c.data.input = input
	c.data.inputs = nil

	return c
}

// Match adds input data from key-value pairs.
// Panics on invalid key-value pairs (odd number of args, non-string keys).
func (c *stubBuilderCore) Match(kv ...any) StubBuilder {
	return c.When(kvToInput(kv, "sdk.Match"))
}

func (c *stubBuilderCore) WhenStream(inputs ...stuber.InputData) StubBuilder {
	c.data.inputs = inputs
	c.data.input = stuber.InputData{}

	return c
}

func (c *stubBuilderCore) WhenHeaders(headers stuber.InputHeader) StubBuilder {
	c.data.headers = headers

	return c
}

func (c *stubBuilderCore) Reply(output stuber.Output) StubBuilder {
	c.data.output = output
	c.data.output.Stream = nil

	return c
}

// Return sets output data from key-value pairs.
// Panics on invalid key-value pairs (odd number of args, non-string keys).
func (c *stubBuilderCore) Return(kv ...any) StubBuilder {
	return c.Reply(kvToOutput(kv, "sdk.Return"))
}

func (c *stubBuilderCore) Unary(inKey string, inVal any, outKey string, outVal any) StubBuilder {
	c.data.input = Equals(inKey, inVal)
	c.data.inputs = nil
	c.data.output = Data(outKey, outVal)
	c.data.output.Stream = nil

	return c
}

func (c *stubBuilderCore) ReplyStream(msgs ...stuber.Output) StubBuilder {
	stream := make([]any, 0, len(msgs))
	for _, o := range msgs {
		if len(o.Stream) > 0 {
			stream = append(stream, o.Stream...)
		} else if o.Data != nil {
			stream = append(stream, o.Data)
		}
	}

	c.data.output = stuber.Output{Stream: stream}

	return c
}

func (c *stubBuilderCore) ReplyError(code codes.Code, msg string) StubBuilder {
	codeCopy := code
	c.data.output = stuber.Output{Code: &codeCopy, Error: msg}

	return c
}

func (c *stubBuilderCore) ReplyErrorWithDetails(code codes.Code, msg string, details ...map[string]any) StubBuilder {
	codeCopy := code
	c.data.output = stuber.Output{Code: &codeCopy, Error: msg, Details: details}

	return c
}

func (c *stubBuilderCore) ReplyHeaders(headers map[string]string) StubBuilder {
	if c.data.output.Headers == nil {
		c.data.output.Headers = make(map[string]string)
	}
	maps.Copy(c.data.output.Headers, headers)

	return c
}

func (c *stubBuilderCore) ReplyHeaderPairs(kv ...string) StubBuilder {
	headers, err := parseHeaderPairsErr(kv, "sdk.ReplyHeaderPairs")
	if err != nil {
		panic(err)
	}

	if c.data.output.Headers == nil {
		c.data.output.Headers = make(map[string]string)
	}
	maps.Copy(c.data.output.Headers, headers)

	return c
}

func (c *stubBuilderCore) Delay(d time.Duration) StubBuilder {
	c.data.output.Delay = types.Duration(d)

	return c
}

func (c *stubBuilderCore) Priority(p int) StubBuilder {
	c.data.priority = p

	return c
}

func (c *stubBuilderCore) Times(n int) StubBuilder {
	c.data.options.Times = n

	return c
}

// Commit registers the stub. Returns error if registration fails.
func (c *stubBuilderCore) Commit() error {
	return c.onCommit(c.newStub())
}

func (c *stubBuilderCore) newStub() *stuber.Stub {
	return &stuber.Stub{
		ID:       uuid.New(),
		Service:  c.service,
		Method:   c.method,
		Input:    c.data.input,
		Inputs:   c.data.inputs,
		Headers:  c.data.headers,
		Output:   c.data.output,
		Priority: c.data.priority,
		Options:  c.data.options,
	}
}

// Equals returns InputData for exact match.
func Equals(key string, value any) stuber.InputData {
	return stuber.InputData{Equals: map[string]any{key: value}}
}

// Contains returns InputData for partial match.
func Contains(key string, value any) stuber.InputData {
	return stuber.InputData{Contains: map[string]any{key: value}}
}

// Matches returns InputData for regex match.
func Matches(key, pattern string) stuber.InputData {
	return stuber.InputData{Matches: map[string]any{key: pattern}}
}

// Glob returns InputData for glob pattern match (wildcards: *, ?, [chars]).
func Glob(key, pattern string) stuber.InputData {
	return stuber.InputData{Glob: map[string]any{key: pattern}}
}

// parseKVPairs converts key-value pairs to map. Panics on invalid input.
func parseKVPairs(kv []any, errPrefix string) map[string]any {
	m, err := parseKVPairsErr(kv, errPrefix)
	if err != nil {
		panic(err)
	}

	return m
}

// Map returns InputData from key-value pairs (all Equals).
func Map(kv ...any) stuber.InputData {
	if len(kv) == 0 {
		return stuber.InputData{}
	}

	return stuber.InputData{Equals: parseKVPairs(kv, "sdk.Map")}
}

// HeaderEquals returns InputHeader for exact header match.
func HeaderEquals(key string, value any) stuber.InputHeader {
	return stuber.InputHeader{Equals: map[string]any{key: value}}
}

// HeaderContains returns InputHeader for partial header match.
func HeaderContains(key string, value any) stuber.InputHeader {
	return stuber.InputHeader{Contains: map[string]any{key: value}}
}

// HeaderMatches returns InputHeader for regex header match.
func HeaderMatches(key, pattern string) stuber.InputHeader {
	return stuber.InputHeader{Matches: map[string]any{key: pattern}}
}

// HeaderGlob returns InputHeader for glob pattern header match.
func HeaderGlob(key, pattern string) stuber.InputHeader {
	return stuber.InputHeader{Glob: map[string]any{key: pattern}}
}

// HeaderMap returns InputHeader from key-value pairs (all Equals).
func HeaderMap(kv ...any) stuber.InputHeader {
	if len(kv) == 0 {
		return stuber.InputHeader{}
	}

	return stuber.InputHeader{Equals: parseKVPairs(kv, "sdk.HeaderMap")}
}

// IgnoreArrayOrder wraps InputData with IgnoreArrayOrder=true for array field matching.
func IgnoreArrayOrder(input stuber.InputData) stuber.InputData {
	input.IgnoreArrayOrder = true

	return input
}

// AnyOf returns InputData with alternative matchers (OR logic).
// At least one alternative must match for the stub to be selected.
func AnyOf(inputs ...stuber.InputData) stuber.InputData {
	elements := make([]stuber.AnyOfElement, len(inputs))
	for i, in := range inputs {
		elements[i] = stuber.AnyOfElement{
			IgnoreArrayOrder: in.IgnoreArrayOrder,
			Equals:           in.Equals,
			Contains:         in.Contains,
			Matches:          in.Matches,
			Glob:             in.Glob,
		}
	}

	return stuber.InputData{AnyOf: elements}
}

// HeaderAnyOf returns InputHeader with alternative header matchers (OR logic).
func HeaderAnyOf(headers ...stuber.InputHeader) stuber.InputHeader {
	elements := make([]stuber.AnyOfHeaderElement, len(headers))
	for i, h := range headers {
		elements[i] = stuber.AnyOfHeaderElement{
			Equals:   h.Equals,
			Contains: h.Contains,
			Matches:  h.Matches,
			Glob:     h.Glob,
		}
	}

	return stuber.InputHeader{AnyOf: elements}
}

// Merge combines multiple values of the same type into one.
//
//nolint:cyclop
func Merge[T stuber.InputData | stuber.InputHeader | stuber.Output](inputs ...T) T {
	var out T
	for _, in := range inputs {
		switch v := any(in).(type) {
		case stuber.InputData:
			o := any(out).(stuber.InputData)
			if v.IgnoreArrayOrder {
				o.IgnoreArrayOrder = true
			}
			if len(v.Equals) > 0 {
				if o.Equals == nil {
					o.Equals = make(map[string]any, len(v.Equals))
				}
				maps.Copy(o.Equals, v.Equals)
			}
			if len(v.Contains) > 0 {
				if o.Contains == nil {
					o.Contains = make(map[string]any, len(v.Contains))
				}
				maps.Copy(o.Contains, v.Contains)
			}
			if len(v.Matches) > 0 {
				if o.Matches == nil {
					o.Matches = make(map[string]any, len(v.Matches))
				}
				maps.Copy(o.Matches, v.Matches)
			}
			if len(v.Glob) > 0 {
				if o.Glob == nil {
					o.Glob = make(map[string]any, len(v.Glob))
				}
				maps.Copy(o.Glob, v.Glob)
			}
			if len(v.AnyOf) > 0 {
				o.AnyOf = append(o.AnyOf, v.AnyOf...)
			}
			out = any(o).(T)

		case stuber.InputHeader:
			o := any(out).(stuber.InputHeader)
			if len(v.Equals) > 0 {
				if o.Equals == nil {
					o.Equals = make(map[string]any, len(v.Equals))
				}
				maps.Copy(o.Equals, v.Equals)
			}
			if len(v.Contains) > 0 {
				if o.Contains == nil {
					o.Contains = make(map[string]any, len(v.Contains))
				}
				maps.Copy(o.Contains, v.Contains)
			}
			if len(v.Matches) > 0 {
				if o.Matches == nil {
					o.Matches = make(map[string]any, len(v.Matches))
				}
				maps.Copy(o.Matches, v.Matches)
			}
			if len(v.Glob) > 0 {
				if o.Glob == nil {
					o.Glob = make(map[string]any, len(v.Glob))
				}
				maps.Copy(o.Glob, v.Glob)
			}
			if len(v.AnyOf) > 0 {
				o.AnyOf = append(o.AnyOf, v.AnyOf...)
			}
			out = any(o).(T)

		case stuber.Output:
			o := any(out).(stuber.Output)
			if v.Data != nil {
				if outMap, ok := o.Data.(map[string]any); ok {
					if addMap, ok := v.Data.(map[string]any); ok {
						maps.Copy(outMap, addMap)
					} else {
						o.Data = v.Data
					}
				} else if o.Data == nil {
					o.Data = v.Data
				}
			}
			if len(v.Headers) > 0 {
				if o.Headers == nil {
					o.Headers = make(map[string]string, len(v.Headers))
				}
				maps.Copy(o.Headers, v.Headers)
			}
			if v.Error != "" {
				o.Error = v.Error
				o.Code = v.Code
			}
			if v.Delay != 0 {
				o.Delay = v.Delay
			}
			if len(v.Stream) > 0 {
				o.Stream = append(o.Stream, v.Stream...)
			}
			if len(v.Details) > 0 {
				o.Details = append(o.Details, v.Details...)
			}
			out = any(o).(T)
		}
	}

	return out
}

func kvToInput(kv []any, errPrefix string) stuber.InputData {
	if len(kv) == 0 {
		return stuber.InputData{}
	}

	return stuber.InputData{Equals: parseKVPairs(kv, errPrefix)}
}

func kvToOutput(kv []any, errPrefix string) stuber.Output {
	if len(kv) == 0 {
		return stuber.Output{}
	}

	return stuber.Output{Data: parseKVPairs(kv, errPrefix)}
}

// Data returns Output with Data map from key-value pairs.
func Data(kv ...any) stuber.Output {
	if len(kv) == 0 {
		return stuber.Output{}
	}

	return stuber.Output{Data: parseKVPairs(kv, "sdk.Data")}
}

// ReplyHeader returns Output with one response header. Use with Merge.
func ReplyHeader(key, value string) stuber.Output {
	return stuber.Output{Headers: map[string]string{key: value}}
}

// ReplyDelay returns Output with delay. Use with Merge.
func ReplyDelay(d time.Duration) stuber.Output {
	return stuber.Output{Delay: types.Duration(d)}
}

// ReplyErr returns Output with error response. Use with Merge.
func ReplyErr(code codes.Code, msg string) stuber.Output {
	c := code

	return stuber.Output{Code: &c, Error: msg}
}

// ReplyErrWithDetails returns Output with error response and gRPC status details. Use with Merge.
func ReplyErrWithDetails(code codes.Code, msg string, details ...map[string]any) stuber.Output {
	c := code

	return stuber.Output{Code: &c, Error: msg, Details: details}
}

// StreamItem returns Output with one stream message (for server streaming). Use with Merge or ReplyStream.
func StreamItem(kv ...any) stuber.Output {
	if len(kv) == 0 {
		return stuber.Output{Stream: []any{map[string]any{}}}
	}

	return stuber.Output{Stream: []any{parseKVPairs(kv, "sdk.StreamItem")}}
}

// StreamDelayItem returns Output with one stream message and a per-element
// _gripmock delay. The delay applies before this message, overriding any global
// output.delay. Use with Merge or ReplyStream.
func StreamDelayItem(delay time.Duration, kv ...any) stuber.Output {
	m := parseKVPairs(kv, "sdk.StreamDelayItem")
	m[stuber.GripmockKey] = map[string]any{
		"delay": delay.String(),
	}

	return stuber.Output{Stream: []any{m}}
}
