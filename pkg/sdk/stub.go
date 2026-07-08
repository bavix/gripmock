package sdk

import (
	"maps"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// StubBuilder defines a fluent builder for creating and registering stubs.
//
// Deprecated: use ExpectUnary, ExpectServerStream, ExpectClientStream,
// ExpectBidirectionalStream instead.
//
// All methods return StubBuilder for chaining.
//
//nolint:interfacebloat
type StubBuilder interface {
	When(input Matcher) StubBuilder
	Match(kv ...any) (StubBuilder, error)
	MustMatch(kv ...any) StubBuilder
	WhenStream(inputs ...Matcher) StubBuilder
	WhenHeaders(headers Matcher) StubBuilder
	Reply(output stuber.Output) StubBuilder
	Return(kv ...any) (StubBuilder, error)
	MustReturn(kv ...any) StubBuilder
	ReplyStream(msgs ...stuber.Output) StubBuilder
	ReplyError(code codes.Code, msg string) StubBuilder
	ReplyErrorWithDetails(code codes.Code, msg string, details ...map[string]any) StubBuilder
	ReplyHeaders(headers map[string]string) StubBuilder
	ReplyHeaderPairs(kv ...string) StubBuilder
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

func (c *stubBuilderCore) When(input Matcher) StubBuilder { //nolint:ireturn
	c.data.input = input.compilePayload()
	c.data.inputs = nil

	return c
}

// Returns error on invalid key-value pairs.
func (c *stubBuilderCore) Match(kv ...any) (StubBuilder, error) { //nolint:ireturn
	if len(kv)%2 != 0 {
		return c, errors.Wrapf(ErrInvalidInput, "sdk.Match: need pairs (key, value), got %d args", len(kv))
	}

	m := Matcher{}

	for i := range len(kv) / 2 {
		key, ok := kv[i*2].(string)
		if !ok {
			return c, errors.Wrapf(ErrInvalidInput, "sdk.Match: key at %d must be string, got %T", i*2, kv[i*2]) //nolint:mnd
		}

		if m.equals == nil {
			m.equals = make(map[string]any)
		}

		m.equals[key] = kv[i*2+1]
	}

	return c.When(m), nil
}

func (c *stubBuilderCore) MustMatch(kv ...any) StubBuilder { //nolint:ireturn
	b, err := c.Match(kv...)
	if err != nil {
		panic(err.Error())
	}

	return b
}

func (c *stubBuilderCore) WhenStream(inputs ...Matcher) StubBuilder { //nolint:ireturn
	ids := make([]stuber.InputData, len(inputs))
	for i, m := range inputs {
		ids[i] = m.compilePayload()
	}

	c.data.inputs = ids
	c.data.input = stuber.InputData{}

	return c
}

func (c *stubBuilderCore) WhenHeaders(headers Matcher) StubBuilder { //nolint:ireturn
	c.data.headers = headers.compileHeader()

	return c
}

func (c *stubBuilderCore) Reply(output stuber.Output) StubBuilder { //nolint:ireturn
	c.data.output = output
	c.data.output.Stream = nil

	return c
}

// Returns error on invalid key-value pairs.
func (c *stubBuilderCore) Return(kv ...any) (StubBuilder, error) { //nolint:ireturn
	output, err := kvToOutputErr(kv, "sdk.Return")
	if err != nil {
		return c, err
	}

	return c.Reply(output), nil
}

// Panics on invalid key-value pairs. Use Return for error handling.
func (c *stubBuilderCore) MustReturn(kv ...any) StubBuilder { //nolint:ireturn
	return c.Reply(kvToOutput(kv, "sdk.MustReturn"))
}

func (c *stubBuilderCore) Unary(inKey string, inVal any, outKey string, outVal any) StubBuilder { //nolint:ireturn
	c.data.input = Equals(inKey, inVal).compilePayload()
	c.data.inputs = nil
	c.data.output = stuber.Output{Data: map[string]any{outKey: outVal}}
	c.data.output.Stream = nil

	return c
}

func (c *stubBuilderCore) ReplyStream(msgs ...stuber.Output) StubBuilder { //nolint:ireturn
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

func (c *stubBuilderCore) ReplyError(code codes.Code, msg string) StubBuilder { //nolint:ireturn
	codeCopy := code
	c.data.output = stuber.Output{Code: &codeCopy, Error: msg}

	return c
}

func (c *stubBuilderCore) ReplyErrorWithDetails(code codes.Code, msg string, details ...map[string]any) StubBuilder { //nolint:ireturn
	codeCopy := code
	c.data.output = stuber.Output{Code: &codeCopy, Error: msg, Details: details}

	return c
}

func (c *stubBuilderCore) ReplyHeaders(headers map[string]string) StubBuilder { //nolint:ireturn
	if c.data.output.Headers == nil {
		c.data.output.Headers = make(map[string]string)
	}

	maps.Copy(c.data.output.Headers, headers)

	return c
}

func (c *stubBuilderCore) ReplyHeaderPairs(kv ...string) StubBuilder { //nolint:ireturn
	headers, err := parseHeaderPairsErr(kv, "sdk.ReplyHeaderPairs")
	panicIfErr(err)

	if c.data.output.Headers == nil {
		c.data.output.Headers = make(map[string]string)
	}

	maps.Copy(c.data.output.Headers, headers)

	return c
}

func (c *stubBuilderCore) Priority(p int) StubBuilder { //nolint:ireturn
	c.data.priority = p

	return c
}

func (c *stubBuilderCore) Times(n int) StubBuilder { //nolint:ireturn
	c.data.options.Times = n

	return c
}

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

func parseKVPairs(kv []any, errPrefix string) map[string]any {
	m, err := parseKVPairsErr(kv, errPrefix)
	panicIfErr(err)

	return m
}

// Merge combines multiple Matcher values (Equals, Contains, Matches, Glob merged).
func Merge(inputs ...Matcher) Matcher {
	return And(inputs...)
}

// mergeInputData combines multiple InputData values (internal helper).
func mergeInputData(inputs ...stuber.InputData) stuber.InputData {
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
