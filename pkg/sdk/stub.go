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
	Match(kv ...any) (StubBuilder, error)
	MustMatch(kv ...any) StubBuilder
	WhenStream(inputs ...stuber.InputData) StubBuilder
	WhenHeaders(headers stuber.InputHeader) StubBuilder
	Reply(output stuber.Output) StubBuilder
	Return(kv ...any) (StubBuilder, error)
	MustReturn(kv ...any) StubBuilder
	ReplyStream(msgs ...stuber.Output) StubBuilder
	ReplyError(code codes.Code, msg string) StubBuilder
	ReplyErrorWithDetails(code codes.Code, msg string, details ...map[string]any) StubBuilder
	ReplyHeaders(headers map[string]string) StubBuilder
	ReplyHeaderPairs(kv ...string) StubBuilder
	Delay(d time.Duration) StubBuilder
	IgnoreArrayOrder() StubBuilder
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
// Returns error on invalid key-value pairs.
func (c *stubBuilderCore) Match(kv ...any) (StubBuilder, error) {
	input, err := kvToInputErr(kv, "sdk.Match")
	if err != nil {
		return c, err
	}

	return c.When(input), nil
}

// MustMatch adds input data from key-value pairs.
// Panics on invalid key-value pairs. Use Match for error handling.
func (c *stubBuilderCore) MustMatch(kv ...any) StubBuilder {
	return c.When(kvToInput(kv, "sdk.MustMatch"))
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
// Returns error on invalid key-value pairs.
func (c *stubBuilderCore) Return(kv ...any) (StubBuilder, error) {
	output, err := kvToOutputErr(kv, "sdk.Return")
	if err != nil {
		return c, err
	}

	return c.Reply(output), nil
}

// MustReturn sets output data from key-value pairs.
// Panics on invalid key-value pairs. Use Return for error handling.
func (c *stubBuilderCore) MustReturn(kv ...any) StubBuilder {
	return c.Reply(kvToOutput(kv, "sdk.MustReturn"))
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
	panicIfErr(err)

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

func (c *stubBuilderCore) IgnoreArrayOrder() StubBuilder {
	c.data.input.IgnoreArrayOrder = true
	for i := range c.data.inputs {
		c.data.inputs[i].IgnoreArrayOrder = true
	}
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

// parseKVPairs converts key-value pairs to map. Panics on invalid input.
func parseKVPairs(kv []any, errPrefix string) map[string]any {
	m, err := parseKVPairsErr(kv, errPrefix)
	panicIfErr(err)

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

// IgnoreOrder returns InputData with only IgnoreArrayOrder=true. Use with Merge: Merge(Equals(...), IgnoreOrder()).
func IgnoreOrder() stuber.InputData {
	return stuber.InputData{IgnoreArrayOrder: true}
}

// Merge combines multiple InputData into one (Equals, Contains, Matches merged; IgnoreArrayOrder OR'd).
func Merge(inputs ...stuber.InputData) stuber.InputData {
	out := stuber.InputData{}
	for _, in := range inputs {
		if in.IgnoreArrayOrder {
			out.IgnoreArrayOrder = true
		}

		if len(in.Equals) > 0 {
			if out.Equals == nil {
				out.Equals = make(map[string]any, len(in.Equals))
			}
			maps.Copy(out.Equals, in.Equals)
		}

		if len(in.Contains) > 0 {
			if out.Contains == nil {
				out.Contains = make(map[string]any, len(in.Contains))
			}
			maps.Copy(out.Contains, in.Contains)
		}

		if len(in.Matches) > 0 {
			if out.Matches == nil {
				out.Matches = make(map[string]any, len(in.Matches))
			}
			maps.Copy(out.Matches, in.Matches)
		}
	}

	return out
}

// MergeHeaders combines multiple InputHeader into one.
func MergeHeaders(headers ...stuber.InputHeader) stuber.InputHeader {
	out := stuber.InputHeader{}
	for _, h := range headers {
		if len(h.Equals) > 0 {
			if out.Equals == nil {
				out.Equals = make(map[string]any, len(h.Equals))
			}
			maps.Copy(out.Equals, h.Equals)
		}

		if len(h.Contains) > 0 {
			if out.Contains == nil {
				out.Contains = make(map[string]any, len(h.Contains))
			}
			maps.Copy(out.Contains, h.Contains)
		}

		if len(h.Matches) > 0 {
			if out.Matches == nil {
				out.Matches = make(map[string]any, len(h.Matches))
			}
			maps.Copy(out.Matches, h.Matches)
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

func kvToInputErr(kv []any, errPrefix string) (stuber.InputData, error) {
	if len(kv) == 0 {
		return stuber.InputData{}, nil
	}

	m, err := parseKVPairsErr(kv, errPrefix)
	if err != nil {
		return stuber.InputData{}, err
	}

	return stuber.InputData{Equals: m}, nil
}

func kvToOutput(kv []any, errPrefix string) stuber.Output {
	if len(kv) == 0 {
		return stuber.Output{}
	}
	return stuber.Output{Data: parseKVPairs(kv, errPrefix)}
}

func kvToOutputErr(kv []any, errPrefix string) (stuber.Output, error) {
	if len(kv) == 0 {
		return stuber.Output{}, nil
	}

	m, err := parseKVPairsErr(kv, errPrefix)
	if err != nil {
		return stuber.Output{}, err
	}

	return stuber.Output{Data: m}, nil
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

// StreamItemWithDelay returns Output with one stream message and a per-element
// _gripmock delay. The delay applies before this message, overriding any global
// output.delay. Use with MergeOutput or ReplyStream.
func StreamItemWithDelay(delay time.Duration, kv ...any) stuber.Output {
	m := parseKVPairs(kv, "sdk.StreamItemWithDelay")
	m[stuber.GripmockKey] = map[string]any{
		"delay": delay.String(),
	}
	return stuber.Output{Stream: []any{m}}
}

// MergeOutput combines multiple Output into one (Data/Headers merged; Error/Delay/Stream from first non-zero).
func MergeOutput(outputs ...stuber.Output) stuber.Output {
	out := stuber.Output{}
	for _, o := range outputs {
		if o.Data != nil {
			if outMap, ok := out.Data.(map[string]any); ok {
				if addMap, ok := o.Data.(map[string]any); ok {
					maps.Copy(outMap, addMap)
				} else {
					out.Data = o.Data
				}
			} else if out.Data == nil {
				out.Data = o.Data
			}
		}

		if len(o.Headers) > 0 {
			if out.Headers == nil {
				out.Headers = make(map[string]string, len(o.Headers))
			}
			maps.Copy(out.Headers, o.Headers)
		}

		if o.Error != "" {
			out.Error = o.Error
			out.Code = o.Code
		}

		if o.Delay != 0 {
			out.Delay = o.Delay
		}

		if len(o.Stream) > 0 {
			out.Stream = append(out.Stream, o.Stream...)
		}

		if len(o.Details) > 0 {
			out.Details = append(out.Details, o.Details...)
		}
	}

	return out
}
