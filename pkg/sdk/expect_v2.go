package sdk

import (
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

// expectationBase holds fields shared across all 4 expectation types.
type expectationBase struct {
	srv    *Server
	svc    string
	method string

	matchers []stuber.InputData
	headers  stuber.InputHeader

	times     int
	priority  int
	committed bool
	stubID    uuid.UUID

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

func mergeInputHeader(a, b stuber.InputHeader) stuber.InputHeader {
	a.Equals = mergeStrAny(a.Equals, b.Equals)
	a.Contains = mergeStrAny(a.Contains, b.Contains)
	a.Matches = mergeStrAny(a.Matches, b.Matches)
	a.Glob = mergeStrAny(a.Glob, b.Glob)

	return a
}

// UnaryExpectation builds a unary mock expectation.
// Create via Server.ExpectUnary. Terminal: Return, ReturnProto, ReturnError, Run.
type UnaryExpectation struct {
	expectationBase

	delay   time.Duration
	session string

	chainIdx int

	kv    map[string]any
	err   *stuberError
	first uuid.UUID
}

func newUnaryExpectation(srv *Server, fullMethod string) *UnaryExpectation {
	e := &UnaryExpectation{}
	e.init(srv, fullMethod)

	return e
}

// Match accepts key-value pairs (shorthand for Equals on payload) or Matcher values.
// Multiple calls are AND-ed together.
//
//	Match("name", "Alex")           — Equals on payload
//	Match(sdk.Contains("name", …))  — Contains on payload
//
// For header matching use WithHeader(sdk.Contains("key", "val")).
func (e *UnaryExpectation) Match(matches ...any) *UnaryExpectation {
	e.matchers = append(e.matchers, compileMatchArgs(matches...)...)

	return e
}

func (e *UnaryExpectation) WithHeader(headers ...Matcher) *UnaryExpectation {
	for _, h := range headers {
		e.headers = mergeInputHeader(e.headers, h.compileHeader())
	}

	return e
}

// Return with optional Delay: Return(Delay(100*ms, "msg", "hello")).
func (e *UnaryExpectation) Return(kv ...any) *UnaryExpectation {
	e.delay, e.kv = extractDelay(kv, "sdk.Return")
	e.register()

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

	e.kv = m
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

// Run executes a custom handler for this expectation.
// Note: Run is currently a no-op — the handler is registered but never invoked.
func (e *UnaryExpectation) Run(fn UnaryHandler) *UnaryExpectation {
	_ = fn

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
	if existing := e.srv.budgerigar.FindByID(e.first); existing != nil && existing.Options.Times == 0 {
		existing.Options.Times = 1
		e.srv.budgerigar.PutMany(existing)
	}
}

func (e *UnaryExpectation) Once() *UnaryExpectation {
	e.times = 1

	return e
}

func (e *UnaryExpectation) Twice() *UnaryExpectation {
	e.times = 2

	return e
}

func (e *UnaryExpectation) Times(n int) *UnaryExpectation {
	e.times = n

	return e
}

func (e *UnaryExpectation) Priority(n int) *UnaryExpectation {
	e.priority = n

	return e
}

// Session isolates this stub to a specific session (X-Gripmock-Session header).
func (e *UnaryExpectation) Session(id string) *UnaryExpectation {
	e.session = id

	return e
}

func (e *UnaryExpectation) register() {
	e.committed = true
	output := e.buildOutput()
	e.first = e.registerOutput(output, e.priority)
	e.stubID = e.first
}

func (e *UnaryExpectation) registerOutput(output stuber.Output, priority int) uuid.UUID {
	matcher := mergeInputData(e.matchers...)

	times := e.times
	if e.chainIdx > 0 {
		times = 1
	}

	id := uuid.New()
	stub := &stuber.Stub{
		ID:       id,
		Service:  e.svc,
		Method:   e.method,
		Input:    matcher,
		Headers:  e.headers,
		Output:   output,
		Priority: priority,
		Session:  e.session,
		Options:  stuber.StubOptions{Times: times},
		Effects:  e.effects,
	}
	e.srv.trackExpectation(stub)

	return id
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

	out := stuber.Output{Data: e.kv}
	if e.delay > 0 {
		out.Delay = types.Duration(e.delay)
	}

	return out
}

// ServerStreamExpectation builds a server-stream mock expectation.
// Create via Server.ExpectServerStream. Terminal: SendStream.
type ServerStreamExpectation struct {
	expectationBase
}

func newServerStreamExpectation(srv *Server, fullMethod string) *ServerStreamExpectation {
	e := &ServerStreamExpectation{}
	e.init(srv, fullMethod)

	return e
}

// Match accepts key-value pairs (shorthand for Equals on payload) or Matcher values.
func (e *ServerStreamExpectation) Match(matches ...any) *ServerStreamExpectation {
	e.matchers = append(e.matchers, compileMatchArgs(matches...)...)

	return e
}

func (e *ServerStreamExpectation) WithHeader(headers ...Matcher) *ServerStreamExpectation {
	for _, h := range headers {
		e.headers = mergeInputHeader(e.headers, h.compileHeader())
	}

	return e
}

// SendStream sets the stream response messages and returns a builder for chaining.
// Accepts maps and DelayItem: SendStream(Delay(100*ms, "msg", "hello"), map[string]any{...}).
func (e *ServerStreamExpectation) SendStream(items ...any) *ServerStreamBuilder {
	e.committed = true
	matcher := mergeInputData(e.matchers...)
	id := uuid.New()

	stream := make([]any, 0, len(items))
	for _, item := range items {
		stream = append(stream, injectStreamDelay(item))
	}

	output := stuber.Output{Stream: stream}
	stub := &stuber.Stub{
		ID:       id,
		Service:  e.svc,
		Method:   e.method,
		Input:    matcher,
		Headers:  e.headers,
		Output:   output,
		Priority: e.priority,
		Options:  stuber.StubOptions{Times: e.times},
		Effects:  e.effects,
	}
	e.stubID = id
	e.srv.trackExpectation(stub)

	return &ServerStreamBuilder{
		srv:     e.srv,
		stubID:  id,
		msgs:    stream,
		svc:     e.svc,
		method:  e.method,
		matcher: matcher,
		headers: e.headers,
		pri:     e.priority,
		times:   e.times,
	}
}

func (e *ServerStreamExpectation) Times(n int) *ServerStreamExpectation {
	e.times = n

	return e
}

func (e *ServerStreamExpectation) Priority(n int) *ServerStreamExpectation {
	e.priority = n

	return e
}

func (e *ServerStreamExpectation) Once() *ServerStreamExpectation { return e.Times(1) }

func (e *ServerStreamExpectation) Twice() *ServerStreamExpectation { return e.Times(2) } //nolint:mnd

// ServerStreamBuilder extends ServerStreamExpectation for chaining additional stream messages.
// Returned by ServerStreamExpectation.SendStream(). Chain Send/NextWillReturn.
type ServerStreamBuilder struct {
	srv      *Server
	stubID   uuid.UUID
	msgs     []any
	svc      string
	method   string
	matcher  stuber.InputData
	headers  stuber.InputHeader
	pri      int
	times    int
	chainIdx int
}

// Send accepts KV pairs or DelayItem: Send(Delay(100*ms, "msg", "hello")).
func (b *ServerStreamBuilder) Send(kv ...any) *ServerStreamBuilder {
	if len(kv) == 1 {
		if d, ok := kv[0].(DelayItem); ok {
			b.msgs = append(b.msgs, injectStreamDelay(d))
			b.upsert()

			return b
		}
	}

	b.msgs = append(b.msgs, parseKVPairs(kv, "sdk.Send"))
	b.upsert()

	return b
}

// NextWillReturn registers a new stub with lower priority.
func (b *ServerStreamBuilder) NextWillReturn(kv ...any) *ServerStreamBuilder {
	b.chainIdx++
	b.fixFirstUnlimited()

	// Use the first stub's fields for matching, but with new stream content
	matcher := b.matcher
	headers := b.headers

	streamMsg := injectStreamDelay(extractDelayItem(kv))
	output := stuber.Output{Stream: []any{streamMsg}}

	stub := &stuber.Stub{
		ID:       uuid.New(),
		Service:  b.svc,
		Method:   b.method,
		Input:    matcher,
		Headers:  headers,
		Output:   output,
		Priority: b.pri - b.chainIdx,
		Options:  stuber.StubOptions{Times: 1},
	}
	b.srv.trackExpectation(stub)

	return b
}

func (b *ServerStreamBuilder) fixFirstUnlimited() {
	if existing := b.srv.budgerigar.FindByID(b.stubID); existing != nil && existing.Options.Times == 0 {
		existing.Options.Times = 1
		b.srv.budgerigar.PutMany(existing)
	}
}

func (b *ServerStreamBuilder) upsert() {
	existing := b.srv.budgerigar.FindByID(b.stubID)
	if existing != nil {
		existing.Output = stuber.Output{Stream: b.msgs}
		b.srv.budgerigar.PutMany(existing)
	} else {
		b.srv.budgerigar.PutMany(&stuber.Stub{
			ID:       b.stubID,
			Service:  b.svc,
			Method:   b.method,
			Input:    b.matcher,
			Headers:  b.headers,
			Output:   stuber.Output{Stream: b.msgs},
			Priority: b.pri,
			Options:  stuber.StubOptions{Times: b.times},
		})
	}
}

// ClientStreamExpectation builds a client-stream mock expectation.
// Create via Server.ExpectClientStream. Terminal: Return, ReturnError.
type ClientStreamExpectation struct {
	expectationBase

	kv           map[string]any
	err          *stuberError
	matchOnFirst bool
}

func newClientStreamExpectation(srv *Server, fullMethod string) *ClientStreamExpectation {
	e := &ClientStreamExpectation{}
	e.init(srv, fullMethod)

	return e
}

// Match accepts key-value pairs (shorthand for Equals on payload) or Matcher values.
func (e *ClientStreamExpectation) Match(matches ...any) *ClientStreamExpectation {
	e.matchers = append(e.matchers, compileMatchArgs(matches...)...)

	return e
}

func (e *ClientStreamExpectation) WithHeader(headers ...Matcher) *ClientStreamExpectation {
	for _, h := range headers {
		e.headers = mergeInputHeader(e.headers, h.compileHeader())
	}

	return e
}

// WithFirstPayload configures the stub to match on the first message only.
//
// Deprecated: use Match(sdk.Contains(...)) instead.
func (e *ClientStreamExpectation) WithFirstPayload(inputs ...Matcher) *ClientStreamExpectation {
	e.matchOnFirst = true
	for _, m := range inputs {
		e.matchers = append(e.matchers, m.compilePayload())
	}

	return e
}

func (e *ClientStreamExpectation) Return(kv ...any) *ClientStreamExpectation {
	e.kv = parseKVPairs(kv, "sdk.ClientStream.Return")
	e.register()

	return e
}

func (e *ClientStreamExpectation) ReturnError(code codes.Code, msg string) *ClientStreamExpectation {
	c := code
	e.err = &stuberError{code: c, msg: msg}
	e.register()

	return e
}

func (e *ClientStreamExpectation) Times(n int) *ClientStreamExpectation {
	e.times = n

	return e
}

func (e *ClientStreamExpectation) Priority(n int) *ClientStreamExpectation {
	e.priority = n

	return e
}

func (e *ClientStreamExpectation) register() {
	e.committed = true

	var output stuber.Output

	if e.err != nil {
		c := e.err.code
		output = stuber.Output{Code: &c, Error: e.err.msg, Details: e.err.details}
	} else {
		output = stuber.Output{Data: e.kv}
	}

	matcher := mergeInputData(e.matchers...)
	id := uuid.New()
	e.stubID = id
	stub := &stuber.Stub{
		ID:                  id,
		Service:             e.svc,
		Method:              e.method,
		Inputs:              []stuber.InputData{matcher},
		Headers:             e.headers,
		Output:              output,
		Priority:            e.priority,
		Options:             stuber.StubOptions{Times: e.times},
		MatchOnFirstMessage: e.matchOnFirst,
		Effects:             e.effects,
	}
	e.srv.trackExpectation(stub)
}

// BidirectionalExpectation builds a bidi-stream mock expectation.
// Create via Server.ExpectBidirectionalStream. Terminal: Run.
type BidirectionalExpectation struct {
	expectationBase
}

func newBidiExpectation(srv *Server, fullMethod string) *BidirectionalExpectation {
	e := &BidirectionalExpectation{}
	e.init(srv, fullMethod)

	return e
}

func (e *BidirectionalExpectation) WithHeader(headers ...Matcher) *BidirectionalExpectation {
	for _, h := range headers {
		e.headers = mergeInputHeader(e.headers, h.compileHeader())
	}

	return e
}

// Run executes a custom handler for this bidi expectation.
func (e *BidirectionalExpectation) Run(fn BidirectionalHandler) *BidirectionalExpectation {
	e.committed = true
	id := uuid.New()
	e.stubID = id
	stub := &stuber.Stub{
		ID:       id,
		Service:  e.svc,
		Method:   e.method,
		Inputs:   []stuber.InputData{{}},
		Headers:  e.headers,
		Output:   stuber.Output{},
		Priority: e.priority,
		Options:  stuber.StubOptions{Times: e.times},
		Handler:  stuber.StreamHandler(fn),
		Effects:  e.effects,
	}
	e.srv.trackExpectation(stub)

	return e
}

func (e *BidirectionalExpectation) Times(n int) *BidirectionalExpectation {
	e.times = n

	return e
}

func (e *BidirectionalExpectation) Priority(n int) *BidirectionalExpectation {
	e.priority = n

	return e
}

// compileMatchArgs processes variadic Match arguments.
// Single Matcher → compile as payload.
// (string, any) pairs → compile each as Equals on payload.
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
