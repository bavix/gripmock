package sdk

import (
	"maps"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

type stuberError struct {
	code    codes.Code
	msg     string
	details []map[string]any
}

type expectationBase struct {
	srv    *Server
	svc    string
	method string

	matchers []stuber.InputData
	headers  stuber.InputHeader

	times     int
	priority  int
	session   string
	committed bool
	stubID    uuid.UUID

	stub *stuber.Stub

	id *uuid.UUID

	respHeaders  map[string]string
	respTrailers map[string]string

	sequence []stuber.InputData

	effects []stuber.Effect
}

// Available after calling a terminal method (Return, SendStream, Run).
func (b *expectationBase) StubID() string {
	return b.stubID.String()
}

func (b *expectationBase) init(srv *Server, fullMethod string) {
	svc, method := splitMethodName(fullMethod)
	if svc == "" || method == "" {
		panic("gripmock: invalid full method name: " + fullMethod)
	}

	b.srv = srv
	b.svc = svc
	b.method = method
}

func (b *expectationBase) mustNotBeCommitted(method string) {
	if b.committed {
		panic("gripmock: ." + method + "() must be called before the terminal method " +
			"(Return/ReturnError/SendStream/Run); chaining it after has no effect")
	}
}

func (b *expectationBase) newStubID() uuid.UUID {
	if b.id != nil {
		return *b.id
	}

	return uuid.New()
}

func (b *expectationBase) applyResponseMeta(out *stuber.Output) {
	if len(b.respHeaders) > 0 {
		if out.Headers == nil {
			out.Headers = make(map[string]string, len(b.respHeaders))
		}

		maps.Copy(out.Headers, b.respHeaders)
	}

	if len(b.respTrailers) > 0 {
		if out.Trailers == nil {
			out.Trailers = make(map[string]string, len(b.respTrailers))
		}

		maps.Copy(out.Trailers, b.respTrailers)
	}
}

func (b *expectationBase) mustBeEmbedded() {
	if b.srv.remote != nil {
		panic("gripmock: .Run() installs an in-process handler and is unavailable in remote mode; " +
			"use a static expectation (Return/SendStream) or an embedded server")
	}
}

func (b *expectationBase) streamInputs() []stuber.InputData {
	if len(b.sequence) > 0 {
		return b.sequence
	}

	return []stuber.InputData{mergeInputData(b.matchers...)}
}

func (b *expectationBase) setSequence(matchers []Matcher) {
	seq := make([]stuber.InputData, len(matchers))
	for i, m := range matchers {
		seq[i] = m.compilePayload()
	}

	b.sequence = seq
}

func (b *expectationBase) setResponseHeaders(headers map[string]string) {
	if b.respHeaders == nil {
		b.respHeaders = make(map[string]string, len(headers))
	}

	maps.Copy(b.respHeaders, headers)
}

func (b *expectationBase) setResponseTrailers(trailers map[string]string) {
	if b.respTrailers == nil {
		b.respTrailers = make(map[string]string, len(trailers))
	}

	maps.Copy(b.respTrailers, trailers)
}

func mergeInputHeader(a, b stuber.InputHeader) stuber.InputHeader {
	a.Equals = mergeStrAny(a.Equals, b.Equals)
	a.Contains = mergeStrAny(a.Contains, b.Contains)
	a.Matches = mergeStrAny(a.Matches, b.Matches)
	a.Glob = mergeStrAny(a.Glob, b.Glob)

	if len(b.AnyOf) > 0 {
		a.AnyOf = append(a.AnyOf, b.AnyOf...)
	}

	return a
}

// UnaryExpectation builds a unary mock expectation.
type UnaryExpectation struct {
	expectationBase

	delay time.Duration

	chainIdx int

	kv        map[string]any
	value     *any
	err       *stuberError
	handler   UnaryHandler
	firstStub *stuber.Stub
}

func newUnaryExpectation(srv *Server, fullMethod string) *UnaryExpectation {
	e := &UnaryExpectation{}
	e.init(srv, fullMethod)

	return e
}

// Match accepts key-value pairs (shorthand for Equals on payload) or Matcher values.
func (e *UnaryExpectation) Match(matches ...any) *UnaryExpectation {
	e.mustNotBeCommitted("Match")
	e.matchers = append(e.matchers, compileMatchArgs(matches...)...)

	return e
}

func (e *UnaryExpectation) WithHeader(headers ...Matcher) *UnaryExpectation {
	e.mustNotBeCommitted("WithHeader")

	for _, h := range headers {
		e.headers = mergeInputHeader(e.headers, h.compileHeader())
	}

	return e
}

// Return with optional Delay: Return(Delay(100*ms, "msg", "hello")).
func (e *UnaryExpectation) Return(kv ...any) *UnaryExpectation {
	delay, data := extractDelay(kv, "sdk.Return")
	e.kv = data

	if delay > 0 {
		e.delay = delay
	}

	e.register()

	return e
}

// Delay holds the response back before it is sent.
func (e *UnaryExpectation) Delay(d time.Duration) *UnaryExpectation {
	e.mustNotBeCommitted("Delay")
	e.delay = d

	return e
}

// ReturnProto marshals a proto.Message to JSON and uses it as response.
func (e *UnaryExpectation) ReturnProto(msg proto.Message) *UnaryExpectation {
	e.kv = protoToMap(msg)
	e.register()

	return e
}

// ReturnJSON marshals any value to JSON and uses it as response.
func (e *UnaryExpectation) ReturnJSON(v any) *UnaryExpectation {
	e.kv = jsonToMap(v)
	e.register()

	return e
}

// ReturnErrorWithDetails returns a gRPC error with additional error details.
func (e *UnaryExpectation) ReturnErrorWithDetails(code codes.Code, msg string, details ...map[string]any) *UnaryExpectation {
	c := code
	e.err = &stuberError{code: c, msg: msg, details: details}
	e.register()

	return e
}

// ReturnError returns a gRPC error for the call.
func (e *UnaryExpectation) ReturnError(code codes.Code, msg string) *UnaryExpectation {
	c := code
	e.err = &stuberError{code: c, msg: msg}
	e.register()

	return e
}

func (e *UnaryExpectation) Run(fn UnaryHandler) *UnaryExpectation {
	e.mustBeEmbedded()
	e.handler = fn
	e.register()

	return e
}

// NextWillReturn chains sequential responses: 1st call→Return, 2nd→NextWillReturn, etc.
func (e *UnaryExpectation) NextWillReturn(kv ...any) *UnaryExpectation {
	e.chainIdx++
	e.fixFirstUnlimited()

	delay, data := extractDelay(kv, "sdk.NextWillReturn")

	output := stuber.Output{Data: data}
	if delay > 0 {
		output.Delay = types.Duration(delay)
	}

	e.registerOutput(output, e.priority-e.chainIdx)

	return e
}

func (e *UnaryExpectation) NextWillReturnError(code codes.Code, msg string) *UnaryExpectation {
	e.chainIdx++
	e.fixFirstUnlimited()

	c := code
	output := stuber.Output{Code: &c, Error: msg}
	e.registerOutput(output, e.priority-e.chainIdx)

	return e
}

//nolint:funcorder
func (e *UnaryExpectation) fixFirstUnlimited() {
	if e.firstStub != nil && e.firstStub.Options.Times == 0 {
		e.firstStub.Options.Times = 1
		e.srv.upsertStub(e.firstStub)
	}
}

func (e *UnaryExpectation) Once() *UnaryExpectation {
	e.mustNotBeCommitted("Once")
	e.times = 1

	return e
}

func (e *UnaryExpectation) Twice() *UnaryExpectation {
	e.mustNotBeCommitted("Twice")
	e.times = 2

	return e
}

func (e *UnaryExpectation) Times(n int) *UnaryExpectation {
	e.mustNotBeCommitted("Times")
	e.times = n

	return e
}

func (e *UnaryExpectation) Priority(n int) *UnaryExpectation {
	e.mustNotBeCommitted("Priority")
	e.priority = n

	return e
}

// Session isolates this stub to a specific session (X-Gripmock-Session header).
func (e *UnaryExpectation) Session(id string) *UnaryExpectation {
	e.mustNotBeCommitted("Session")
	e.session = id

	return e
}

func (e *UnaryExpectation) WithID(id uuid.UUID) *UnaryExpectation {
	e.mustNotBeCommitted("WithID")
	e.id = &id

	return e
}

func (e *UnaryExpectation) ReturnHeaders(headers map[string]string) *UnaryExpectation {
	e.mustNotBeCommitted("ReturnHeaders")
	e.setResponseHeaders(headers)

	return e
}

// ReturnTrailers sets response metadata sent with the closing status.
func (e *UnaryExpectation) ReturnTrailers(trailers map[string]string) *UnaryExpectation {
	e.mustNotBeCommitted("ReturnTrailers")
	e.setResponseTrailers(trailers)

	return e
}

// ReturnValue replies with v verbatim, including scalars, arrays and null.
func (e *UnaryExpectation) ReturnValue(v any) *UnaryExpectation {
	e.value = &v
	e.register()

	return e
}

// ReturnStatus replies with a bare gRPC status code and no message.
func (e *UnaryExpectation) ReturnStatus(code codes.Code) *UnaryExpectation {
	return e.ReturnError(code, "")
}

func (e *UnaryExpectation) register() {
	e.committed = true
	output := e.buildOutput()
	e.firstStub = e.registerOutput(output, e.priority)
	e.stub = e.firstStub
	e.stubID = e.firstStub.ID
}

func (e *UnaryExpectation) chainStubID() uuid.UUID {
	if e.chainIdx > 0 {
		return uuid.New()
	}

	return e.newStubID()
}

func (e *UnaryExpectation) registerOutput(output stuber.Output, priority int) *stuber.Stub {
	matcher := mergeInputData(e.matchers...)

	times := e.times
	if e.chainIdx > 0 {
		times = 1
	}

	e.applyResponseMeta(&output)

	stub := &stuber.Stub{
		ID:       e.chainStubID(),
		Service:  e.svc,
		Method:   e.method,
		Input:    matcher,
		Headers:  e.headers,
		Output:   output,
		Priority: priority,
		Session:  e.session,
		Options:  stuber.StubOptions{Times: times},
		Effects:  e.effects,

		UnaryHandler: stuber.UnaryHandler(e.handler),
	}
	e.srv.trackExpectation(stub)

	return stub
}

func (e *UnaryExpectation) buildOutput() stuber.Output {
	if e.err != nil {
		c := e.err.code

		out := stuber.Output{Code: &c, Error: e.err.msg, Details: e.err.details}
		if e.delay > 0 {
			out.Delay = types.Duration(e.delay)
		}

		return out
	}

	data := any(e.kv)
	if e.value != nil {
		data = *e.value
	}

	out := stuber.Output{Data: data}
	if e.delay > 0 {
		out.Delay = types.Duration(e.delay)
	}

	return out
}

// ServerStreamExpectation builds a server-stream mock expectation.
type ServerStreamExpectation struct {
	expectationBase

	delay time.Duration
}

func newServerStreamExpectation(srv *Server, fullMethod string) *ServerStreamExpectation {
	e := &ServerStreamExpectation{}
	e.init(srv, fullMethod)

	return e
}

// Match accepts key-value pairs (shorthand for Equals on payload) or Matcher values.
func (e *ServerStreamExpectation) Match(matches ...any) *ServerStreamExpectation {
	e.mustNotBeCommitted("Match")
	e.matchers = append(e.matchers, compileMatchArgs(matches...)...)

	return e
}

func (e *ServerStreamExpectation) WithHeader(headers ...Matcher) *ServerStreamExpectation {
	e.mustNotBeCommitted("WithHeader")

	for _, h := range headers {
		e.headers = mergeInputHeader(e.headers, h.compileHeader())
	}

	return e
}

func (e *ServerStreamExpectation) SendStream(items ...any) *ServerStreamBuilder {
	stream := make([]any, 0, len(items))
	for _, item := range items {
		stream = append(stream, injectStreamDelay(item))
	}

	stub := e.register(stuber.Output{Stream: stream}, nil)

	return &ServerStreamBuilder{
		srv:          e.srv,
		stub:         stub,
		msgs:         stream,
		svc:          e.svc,
		method:       e.method,
		matcher:      stub.Input,
		headers:      e.headers,
		respHeaders:  e.respHeaders,
		respTrailers: e.respTrailers,
		session:      e.session,
		pri:          e.priority,
		times:        e.times,
		effects:      e.effects,
	}
}

func (e *ServerStreamExpectation) Times(n int) *ServerStreamExpectation {
	e.mustNotBeCommitted("Times")
	e.times = n

	return e
}

func (e *ServerStreamExpectation) Priority(n int) *ServerStreamExpectation {
	e.mustNotBeCommitted("Priority")
	e.priority = n

	return e
}

func (e *ServerStreamExpectation) Once() *ServerStreamExpectation { return e.Times(1) }

func (e *ServerStreamExpectation) Twice() *ServerStreamExpectation { return e.Times(2) } //nolint:mnd

// Run writes the response stream from fn instead of a static Output.Stream.
func (e *ServerStreamExpectation) Run(fn ServerStreamHandler) *ServerStreamExpectation {
	e.mustBeEmbedded()
	e.register(stuber.Output{}, stuber.ServerStreamHandler(fn))

	return e
}

// Session isolates this stub to a specific session (X-Gripmock-Session header).
func (e *ServerStreamExpectation) Session(id string) *ServerStreamExpectation {
	e.mustNotBeCommitted("Session")
	e.session = id

	return e
}

func (e *ServerStreamExpectation) Delay(d time.Duration) *ServerStreamExpectation {
	e.mustNotBeCommitted("Delay")
	e.delay = d

	return e
}

// ReturnError terminates the call with a gRPC status instead of a stream.
func (e *ServerStreamExpectation) ReturnError(code codes.Code, msg string) *ServerStreamExpectation {
	return e.ReturnErrorWithDetails(code, msg)
}

// ReturnStatus terminates the call with a bare gRPC status code and no message.
func (e *ServerStreamExpectation) ReturnStatus(code codes.Code) *ServerStreamExpectation {
	return e.ReturnError(code, "")
}

func (e *ServerStreamExpectation) ReturnErrorWithDetails(
	code codes.Code,
	msg string,
	details ...map[string]any,
) *ServerStreamExpectation {
	e.committed = true

	c := code
	output := stuber.Output{Code: &c, Error: msg, Details: details}

	if e.delay > 0 {
		output.Delay = types.Duration(e.delay)
	}

	e.applyResponseMeta(&output)

	stub := &stuber.Stub{
		ID:       e.newStubID(),
		Service:  e.svc,
		Method:   e.method,
		Input:    mergeInputData(e.matchers...),
		Headers:  e.headers,
		Output:   output,
		Priority: e.priority,
		Session:  e.session,
		Options:  stuber.StubOptions{Times: e.times},
		Effects:  e.effects,
	}
	e.stubID = stub.ID
	e.stub = stub
	e.srv.trackExpectation(stub)

	return e
}

func (e *ServerStreamExpectation) WithID(id uuid.UUID) *ServerStreamExpectation {
	e.mustNotBeCommitted("WithID")
	e.id = &id

	return e
}

func (e *ServerStreamExpectation) ReturnHeaders(headers map[string]string) *ServerStreamExpectation {
	e.mustNotBeCommitted("ReturnHeaders")
	e.setResponseHeaders(headers)

	return e
}

func (e *ServerStreamExpectation) ReturnTrailers(trailers map[string]string) *ServerStreamExpectation {
	e.mustNotBeCommitted("ReturnTrailers")
	e.setResponseTrailers(trailers)

	return e
}

func (e *ServerStreamExpectation) register(output stuber.Output, handler stuber.ServerStreamHandler) *stuber.Stub {
	e.committed = true

	if e.delay > 0 {
		output.Delay = types.Duration(e.delay)
	}

	e.applyResponseMeta(&output)

	id := e.newStubID()
	stub := &stuber.Stub{
		ID:       id,
		Service:  e.svc,
		Method:   e.method,
		Input:    mergeInputData(e.matchers...),
		Headers:  e.headers,
		Output:   output,
		Priority: e.priority,
		Session:  e.session,
		Options:  stuber.StubOptions{Times: e.times},
		Effects:  e.effects,

		ServerStreamHandler: handler,
	}
	e.stubID = id
	e.stub = stub
	e.srv.trackExpectation(stub)

	return stub
}

// ServerStreamBuilder extends ServerStreamExpectation for chaining additional stream messages.
type ServerStreamBuilder struct {
	srv          *Server
	stub         *stuber.Stub
	msgs         []any
	svc          string
	method       string
	matcher      stuber.InputData
	headers      stuber.InputHeader
	respHeaders  map[string]string
	respTrailers map[string]string
	session      string
	pri          int
	times        int
	chainIdx     int
	effects      []stuber.Effect
}

// Send accepts KV pairs or DelayItem: Send(Delay(100*ms, "msg", "hello")).
func (b *ServerStreamBuilder) Send(kv ...any) *ServerStreamBuilder {
	if len(kv) == 1 {
		b.msgs = append(b.msgs, streamElement(kv[0], "sdk.Send"))
	} else {
		b.msgs = append(b.msgs, parseKVPairs(kv, "sdk.Send"))
	}

	b.upsert()

	return b
}

// NextWillReturn registers a new stub with lower priority.
func (b *ServerStreamBuilder) NextWillReturn(kv ...any) *ServerStreamBuilder {
	b.chainIdx++
	b.fixFirstUnlimited()

	matcher := b.matcher
	headers := b.headers

	streamMsg := injectStreamDelay(extractDelayItem(kv))
	output := stuber.Output{Stream: []any{streamMsg}}

	if len(b.respHeaders) > 0 {
		output.Headers = maps.Clone(b.respHeaders)
	}

	if len(b.respTrailers) > 0 {
		output.Trailers = maps.Clone(b.respTrailers)
	}

	stub := &stuber.Stub{
		ID:       uuid.New(),
		Service:  b.svc,
		Method:   b.method,
		Input:    matcher,
		Headers:  headers,
		Output:   output,
		Priority: b.pri - b.chainIdx,
		Session:  b.session,
		Options:  stuber.StubOptions{Times: 1},
		Effects:  b.effects,
	}
	b.srv.trackExpectation(stub)

	return b
}

func (b *ServerStreamBuilder) fixFirstUnlimited() {
	if b.stub.Options.Times == 0 {
		b.stub.Options.Times = 1
		b.srv.upsertStub(b.stub)
	}
}

func (b *ServerStreamBuilder) upsert() {
	b.stub.Output = stuber.Output{Stream: b.msgs}
	b.srv.upsertStub(b.stub)
}

// ClientStreamExpectation builds a client-stream mock expectation.
type ClientStreamExpectation struct {
	expectationBase

	delay   time.Duration
	kv      map[string]any
	value   *any
	err     *stuberError
	handler ClientStreamHandler
}

func newClientStreamExpectation(srv *Server, fullMethod string) *ClientStreamExpectation {
	e := &ClientStreamExpectation{}
	e.init(srv, fullMethod)

	return e
}

// Match accepts key-value pairs (shorthand for Equals on payload) or Matcher values.
func (e *ClientStreamExpectation) Match(matches ...any) *ClientStreamExpectation {
	e.mustNotBeCommitted("Match")
	e.matchers = append(e.matchers, compileMatchArgs(matches...)...)

	return e
}

func (e *ClientStreamExpectation) WithHeader(headers ...Matcher) *ClientStreamExpectation {
	e.mustNotBeCommitted("WithHeader")

	for _, h := range headers {
		e.headers = mergeInputHeader(e.headers, h.compileHeader())
	}

	return e
}

func (e *ClientStreamExpectation) Return(kv ...any) *ClientStreamExpectation {
	delay, data := extractDelay(kv, "sdk.ClientStream.Return")
	e.kv = data

	if delay > 0 {
		e.delay = delay
	}

	e.register()

	return e
}

// ReturnProto marshals a proto.Message to JSON and uses it as response.
func (e *ClientStreamExpectation) ReturnProto(msg proto.Message) *ClientStreamExpectation {
	e.kv = protoToMap(msg)
	e.register()

	return e
}

// ReturnJSON marshals any value to JSON and uses it as response.
func (e *ClientStreamExpectation) ReturnJSON(v any) *ClientStreamExpectation {
	e.kv = jsonToMap(v)
	e.register()

	return e
}

func (e *ClientStreamExpectation) ReturnValue(v any) *ClientStreamExpectation {
	e.value = &v
	e.register()

	return e
}

func (e *ClientStreamExpectation) ReturnError(code codes.Code, msg string) *ClientStreamExpectation {
	return e.ReturnErrorWithDetails(code, msg)
}

// ReturnStatus replies with a bare gRPC status code and no message.
func (e *ClientStreamExpectation) ReturnStatus(code codes.Code) *ClientStreamExpectation {
	return e.ReturnError(code, "")
}

// Delay holds the response back before the reply is sent.
func (e *ClientStreamExpectation) Delay(d time.Duration) *ClientStreamExpectation {
	e.mustNotBeCommitted("Delay")
	e.delay = d

	return e
}

// ReturnErrorWithDetails returns a gRPC error carrying google.rpc.* details.
func (e *ClientStreamExpectation) ReturnErrorWithDetails(
	code codes.Code,
	msg string,
	details ...map[string]any,
) *ClientStreamExpectation {
	c := code
	e.err = &stuberError{code: c, msg: msg, details: details}
	e.register()

	return e
}

// Run drains the request stream with fn and replies with its return value.
func (e *ClientStreamExpectation) Run(fn ClientStreamHandler) *ClientStreamExpectation {
	e.mustBeEmbedded()
	e.handler = fn
	e.register()

	return e
}

func (e *ClientStreamExpectation) Once() *ClientStreamExpectation { return e.Times(1) }

func (e *ClientStreamExpectation) Twice() *ClientStreamExpectation { return e.Times(2) } //nolint:mnd

// Session isolates this stub to a specific session (X-Gripmock-Session header).
func (e *ClientStreamExpectation) Session(id string) *ClientStreamExpectation {
	e.mustNotBeCommitted("Session")
	e.session = id

	return e
}

func (e *ClientStreamExpectation) Times(n int) *ClientStreamExpectation {
	e.mustNotBeCommitted("Times")
	e.times = n

	return e
}

func (e *ClientStreamExpectation) Priority(n int) *ClientStreamExpectation {
	e.mustNotBeCommitted("Priority")
	e.priority = n

	return e
}

func (e *ClientStreamExpectation) WithID(id uuid.UUID) *ClientStreamExpectation {
	e.mustNotBeCommitted("WithID")
	e.id = &id

	return e
}

func (e *ClientStreamExpectation) ReturnHeaders(headers map[string]string) *ClientStreamExpectation {
	e.mustNotBeCommitted("ReturnHeaders")
	e.setResponseHeaders(headers)

	return e
}

func (e *ClientStreamExpectation) ReturnTrailers(trailers map[string]string) *ClientStreamExpectation {
	e.mustNotBeCommitted("ReturnTrailers")
	e.setResponseTrailers(trailers)

	return e
}

func (e *ClientStreamExpectation) MatchSequence(matchers ...Matcher) *ClientStreamExpectation {
	e.mustNotBeCommitted("MatchSequence")
	e.setSequence(matchers)

	return e
}

func (e *ClientStreamExpectation) register() {
	e.committed = true

	var output stuber.Output

	switch {
	case e.err != nil:
		c := e.err.code
		output = stuber.Output{Code: &c, Error: e.err.msg, Details: e.err.details}
	case e.value != nil:
		output = stuber.Output{Data: *e.value}
	default:
		output = stuber.Output{Data: e.kv}
	}

	if e.delay > 0 {
		output.Delay = types.Duration(e.delay)
	}

	e.applyResponseMeta(&output)

	id := e.newStubID()
	e.stubID = id
	stub := &stuber.Stub{
		ID:       id,
		Service:  e.svc,
		Method:   e.method,
		Inputs:   e.streamInputs(),
		Headers:  e.headers,
		Output:   output,
		Priority: e.priority,
		Session:  e.session,
		Options:  stuber.StubOptions{Times: e.times},
		Effects:  e.effects,

		ClientStreamHandler: stuber.ClientStreamHandler(e.handler),
	}
	e.stub = stub
	e.srv.trackExpectation(stub)
}

// BidirectionalExpectation builds a bidi-stream mock expectation.
type BidirectionalExpectation struct {
	expectationBase

	delay time.Duration
}

func newBidiExpectation(srv *Server, fullMethod string) *BidirectionalExpectation {
	e := &BidirectionalExpectation{}
	e.init(srv, fullMethod)

	return e
}

func (e *BidirectionalExpectation) WithHeader(headers ...Matcher) *BidirectionalExpectation {
	e.mustNotBeCommitted("WithHeader")

	for _, h := range headers {
		e.headers = mergeInputHeader(e.headers, h.compileHeader())
	}

	return e
}

// Run executes a custom handler for this bidi expectation.
func (e *BidirectionalExpectation) Run(fn BidirectionalHandler) *BidirectionalExpectation {
	e.mustBeEmbedded()

	return e.register(stuber.Output{}, []stuber.InputData{{}}, stuber.StreamHandler(fn))
}

// Delay holds every message of the stream back by d.
func (e *BidirectionalExpectation) Delay(d time.Duration) *BidirectionalExpectation {
	e.mustNotBeCommitted("Delay")
	e.delay = d

	return e
}

// ReturnError fails the exchange with a gRPC status instead of answering.
func (e *BidirectionalExpectation) ReturnError(code codes.Code, msg string) *BidirectionalExpectation {
	return e.ReturnErrorWithDetails(code, msg)
}

func (e *BidirectionalExpectation) ReturnErrorWithDetails(
	code codes.Code,
	msg string,
	details ...map[string]any,
) *BidirectionalExpectation {
	c := code

	return e.register(stuber.Output{Code: &c, Error: msg, Details: details}, e.streamInputs(), nil)
}

// ReturnStatus fails the exchange with a bare gRPC status code and no message.
func (e *BidirectionalExpectation) ReturnStatus(code codes.Code) *BidirectionalExpectation {
	return e.ReturnError(code, "")
}

func (e *BidirectionalExpectation) Times(n int) *BidirectionalExpectation {
	e.mustNotBeCommitted("Times")
	e.times = n

	return e
}

func (e *BidirectionalExpectation) Priority(n int) *BidirectionalExpectation {
	e.mustNotBeCommitted("Priority")
	e.priority = n

	return e
}

func (e *BidirectionalExpectation) Once() *BidirectionalExpectation { return e.Times(1) }

func (e *BidirectionalExpectation) Twice() *BidirectionalExpectation { return e.Times(2) } //nolint:mnd

// Session isolates this stub to a specific session (X-Gripmock-Session header).
func (e *BidirectionalExpectation) Session(id string) *BidirectionalExpectation {
	e.mustNotBeCommitted("Session")
	e.session = id

	return e
}

func jsonToMap(v any) map[string]any {
	var m map[string]any

	switch val := v.(type) {
	case map[string]any:
		m = val
	default:
		data, err := json.Marshal(val)
		if err == nil {
			_ = json.Unmarshal(data, &m)
		}
	}

	if m == nil {
		m = map[string]any{"_value": v}
	}

	return m
}

func compileMatchArgs(args ...any) []stuber.InputData {
	if len(args) == 0 {
		return nil
	}

	if len(args) == 1 {
		if m, ok := args[0].(Matcher); ok {
			return []stuber.InputData{m.compilePayload()}
		}

		panic("gripmock: Match requires a Matcher or key-value pairs, got a single non-Matcher arg")
	}

	if len(args)%2 != 0 {
		panic("gripmock: Match key-value pairs must be even in number")
	}

	const stride = 2

	out := make([]stuber.InputData, 0, len(args)/stride)

	for i := 0; i < len(args); i += stride {
		key, ok := args[i].(string)
		if !ok {
			panic("gripmock: Match key must be a string")
		}

		out = append(out, Equals(key, args[i+1]).compilePayload())
	}

	return out
}

func protoToMap(msg proto.Message) map[string]any {
	raw, err := protojson.Marshal(msg)
	if err != nil {
		panic("gripmock: failed to marshal proto message: " + err.Error())
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		panic("gripmock: failed to unmarshal proto JSON: " + err.Error())
	}

	return m
}

//nolint:cyclop
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

		if len(in.Glob) > 0 {
			if out.Glob == nil {
				out.Glob = make(map[string]any, len(in.Glob))
			}

			maps.Copy(out.Glob, in.Glob)
		}

		if len(in.AnyOf) > 0 {
			out.AnyOf = append(out.AnyOf, in.AnyOf...)
		}
	}

	return out
}

func (e *BidirectionalExpectation) WithID(id uuid.UUID) *BidirectionalExpectation {
	e.mustNotBeCommitted("WithID")
	e.id = &id

	return e
}

func (e *BidirectionalExpectation) ReturnHeaders(headers map[string]string) *BidirectionalExpectation {
	e.mustNotBeCommitted("ReturnHeaders")
	e.setResponseHeaders(headers)

	return e
}

func (e *BidirectionalExpectation) ReturnTrailers(trailers map[string]string) *BidirectionalExpectation {
	e.mustNotBeCommitted("ReturnTrailers")
	e.setResponseTrailers(trailers)

	return e
}

func (e *BidirectionalExpectation) Match(matches ...any) *BidirectionalExpectation {
	e.mustNotBeCommitted("Match")
	e.matchers = append(e.matchers, compileMatchArgs(matches...)...)

	return e
}

// MatchSequence matches the incoming messages positionally.
func (e *BidirectionalExpectation) MatchSequence(matchers ...Matcher) *BidirectionalExpectation {
	e.mustNotBeCommitted("MatchSequence")
	e.setSequence(matchers)

	return e
}

func (e *BidirectionalExpectation) SendStream(items ...any) *BidirectionalExpectation {
	stream := make([]any, 0, len(items))
	for _, item := range items {
		stream = append(stream, injectStreamDelay(item))
	}

	return e.register(stuber.Output{Stream: stream}, e.streamInputs(), nil)
}

func (e *BidirectionalExpectation) register(
	output stuber.Output,
	inputs []stuber.InputData,
	handler stuber.StreamHandler,
) *BidirectionalExpectation {
	e.committed = true

	if e.delay > 0 {
		output.Delay = types.Duration(e.delay)
	}

	e.applyResponseMeta(&output)

	id := e.newStubID()
	e.stubID = id
	stub := &stuber.Stub{
		ID:       id,
		Service:  e.svc,
		Method:   e.method,
		Inputs:   inputs,
		Headers:  e.headers,
		Output:   output,
		Priority: e.priority,
		Session:  e.session,
		Options:  stuber.StubOptions{Times: e.times},
		Handler:  handler,
		Effects:  e.effects,
	}
	e.stub = stub
	e.srv.trackExpectation(stub)

	return e
}
