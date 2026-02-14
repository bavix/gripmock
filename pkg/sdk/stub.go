package sdk

import (
	"fmt"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

// StubBuilder builds stubs for gRPC methods.
type StubBuilder interface {
	When(input stuber.InputData) StubBuilder
	WhenStream(inputs ...stuber.InputData) StubBuilder
	WhenHeaders(headers stuber.InputHeader) StubBuilder
	Reply(output stuber.Output) StubBuilder
	ReplyStream(msgs ...stuber.Output) StubBuilder
	ReplyError(code codes.Code, msg string) StubBuilder
	ReplyHeaders(headers map[string]string) StubBuilder
	ReplyHeaderPairs(kv ...string) StubBuilder
	Delay(d time.Duration) StubBuilder
	IgnoreArrayOrder() StubBuilder
	Priority(p int) StubBuilder
	Times(n int) StubBuilder // Max matches; 0 = unlimited.
	Commit()
}

type stubBuilder struct {
	mock     *embeddedMock
	service  string
	method   string
	input    stuber.InputData
	inputs   []stuber.InputData
	headers  stuber.InputHeader
	output   stuber.Output
	priority int
	options  stuber.StubOptions
}

func (sb *stubBuilder) When(input stuber.InputData) StubBuilder {
	sb.input = input
	sb.inputs = nil
	return sb
}

func (sb *stubBuilder) WhenStream(inputs ...stuber.InputData) StubBuilder {
	sb.inputs = inputs
	sb.input = stuber.InputData{}
	return sb
}

func (sb *stubBuilder) WhenHeaders(headers stuber.InputHeader) StubBuilder {
	sb.headers = headers
	return sb
}

func (sb *stubBuilder) Reply(output stuber.Output) StubBuilder {
	sb.output = output
	sb.output.Stream = nil
	return sb
}

func (sb *stubBuilder) ReplyStream(msgs ...stuber.Output) StubBuilder {
	stream := make([]any, 0, len(msgs))
	for _, o := range msgs {
		if o.Data != nil {
			stream = append(stream, o.Data)
		}
	}
	sb.output = stuber.Output{Stream: stream}
	return sb
}

func (sb *stubBuilder) ReplyError(code codes.Code, msg string) StubBuilder {
	codeCopy := code
	sb.output = stuber.Output{Code: &codeCopy, Error: msg}
	return sb
}

func (sb *stubBuilder) ReplyHeaders(headers map[string]string) StubBuilder {
	if sb.output.Headers == nil {
		sb.output.Headers = make(map[string]string)
	}
	for k, v := range headers {
		sb.output.Headers[k] = v
	}
	return sb
}

// ReplyHeaderPairs sets response headers from key-value pairs (e.g. "x-custom", "value").
func (sb *stubBuilder) ReplyHeaderPairs(kv ...string) StubBuilder {
	if len(kv)%2 != 0 {
		panic(fmt.Sprintf("sdk.ReplyHeaderPairs: need pairs (key, value), got %d args", len(kv)))
	}
	if sb.output.Headers == nil {
		sb.output.Headers = make(map[string]string)
	}
	for i := 0; i < len(kv); i += 2 {
		sb.output.Headers[kv[i]] = kv[i+1]
	}
	return sb
}

func (sb *stubBuilder) Delay(d time.Duration) StubBuilder {
	sb.output.Delay = types.Duration(d)
	return sb
}

func (sb *stubBuilder) IgnoreArrayOrder() StubBuilder {
	sb.input.IgnoreArrayOrder = true
	for i := range sb.inputs {
		sb.inputs[i].IgnoreArrayOrder = true
	}
	return sb
}

func (sb *stubBuilder) Priority(p int) StubBuilder {
	sb.priority = p
	return sb
}

func (sb *stubBuilder) Times(n int) StubBuilder {
	sb.options.Times = n
	return sb
}

func (sb *stubBuilder) Commit() {
	stub := &stuber.Stub{
		Service:  sb.service,
		Method:   sb.method,
		Input:    sb.input,
		Inputs:   sb.inputs,
		Headers:  sb.headers,
		Output:   sb.output,
		Priority: sb.priority,
		Options:  sb.options,
	}
	sb.mock.budgerigar.PutMany(stub)
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

// Map returns InputData from key-value pairs (all Equals).
func Map(kv ...any) stuber.InputData {
	if len(kv)%2 != 0 {
		panic(fmt.Sprintf("sdk.Map: need pairs (key, value), got %d args", len(kv)))
	}
	m := make(map[string]any)
	for i := 0; i < len(kv); i += 2 {
		k, ok := kv[i].(string)
		if !ok {
			panic(fmt.Sprintf("sdk.Map: key at %d must be string, got %T", i, kv[i]))
		}
		m[k] = kv[i+1]
	}
	return stuber.InputData{Equals: m}
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
	if len(kv)%2 != 0 {
		panic(fmt.Sprintf("sdk.HeaderMap: need pairs (key, value), got %d args", len(kv)))
	}
	m := make(map[string]any)
	for i := 0; i < len(kv); i += 2 {
		k, ok := kv[i].(string)
		if !ok {
			panic(fmt.Sprintf("sdk.HeaderMap: key at %d must be string, got %T", i, kv[i]))
		}
		m[k] = kv[i+1]
	}
	return stuber.InputHeader{Equals: m}
}

// IgnoreArrayOrder wraps InputData with IgnoreArrayOrder=true for array field matching.
func IgnoreArrayOrder(input stuber.InputData) stuber.InputData {
	input.IgnoreArrayOrder = true
	return input
}

// Data returns Output with Data map from key-value pairs.
func Data(kv ...any) stuber.Output {
	if len(kv)%2 != 0 {
		panic(fmt.Sprintf("sdk.Data: need pairs (key, value), got %d args", len(kv)))
	}
	m := make(map[string]any)
	for i := 0; i < len(kv); i += 2 {
		k, ok := kv[i].(string)
		if !ok {
			panic(fmt.Sprintf("sdk.Data: key at %d must be string, got %T", i, kv[i]))
		}
		m[k] = kv[i+1]
	}
	return stuber.Output{Data: m}
}
