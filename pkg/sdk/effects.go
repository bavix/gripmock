package sdk

import (
	"maps"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

// Effect is a side effect that executes after a stub is matched.
type Effect struct {
	effect stuber.Effect
}

// Upsert creates an effect that registers another stub on match.
func Upsert(service, method string) *EffectBuilder {
	return &EffectBuilder{
		stub: stuber.Stub{Service: service, Method: method},
	}
}

// DeleteStub creates an effect that deletes a stub by ID on match.
func DeleteStub(id string) *Effect {
	return &Effect{
		effect: stuber.Effect{
			Action: stuber.EffectActionDelete,
			ID:     id,
		},
	}
}

// EffectBuilder builds a stub that is registered when the triggering stub is matched.
type EffectBuilder struct {
	stub    stuber.Stub
	matcher stuber.InputData
	headers stuber.InputHeader
	id      *uuid.UUID

	respHeaders  map[string]string
	respTrailers map[string]string
}

// Match adds an exact-equality matcher. Use MatchAny for contains/matches/glob.
func (b *EffectBuilder) Match(key string, value any) *EffectBuilder {
	return b.MatchAny(Equals(key, value))
}

func (b *EffectBuilder) MatchAny(matchers ...Matcher) *EffectBuilder {
	for _, m := range matchers {
		b.matcher = mergeInputData(b.matcher, m.compilePayload())
	}

	return b
}

// WithHeader matches request metadata on the effect stub.
func (b *EffectBuilder) WithHeader(headers ...Matcher) *EffectBuilder {
	for _, h := range headers {
		b.headers = mergeInputHeader(b.headers, h.compileHeader())
	}

	return b
}

// WithID pins the effect stub's identifier so a later DeleteStub can target it.
func (b *EffectBuilder) WithID(id uuid.UUID) *EffectBuilder {
	b.id = &id

	return b
}

// Session scopes the effect stub to one session.
func (b *EffectBuilder) Session(id string) *EffectBuilder {
	b.stub.Session = id

	return b
}

// Priority ranks the effect stub against others for the same method.
func (b *EffectBuilder) Priority(n int) *EffectBuilder {
	b.stub.Priority = n

	return b
}

// Times limits how many calls the effect stub may answer.
func (b *EffectBuilder) Times(n int) *EffectBuilder {
	b.stub.Options.Times = n

	return b
}

func (b *EffectBuilder) Return(kv ...any) *EffectBuilder {
	delay, data := extractDelay(kv, "sdk.Effect.Return")
	b.stub.Output = stuber.Output{Data: data}

	if delay > 0 {
		b.stub.Output.Delay = types.NewDelay(delay)
	}

	return b
}

// ReturnTemplate makes the effect stub answer with a Go template, rendered per
// request against the child's own request rather than the parent's.
func (b *EffectBuilder) ReturnTemplate(document string) *EffectBuilder {
	b.stub.Output = stuber.Output{Template: true, Data: document}

	return b
}

// SendStreamTemplate makes the effect stub answer with a stream computed by a Go
// template.
func (b *EffectBuilder) SendStreamTemplate(document string) *EffectBuilder {
	b.stub.Output = stuber.Output{Template: true, Stream: document}

	return b
}

// SendStream makes the effect stub answer with a server stream.
func (b *EffectBuilder) SendStream(items ...any) *EffectBuilder {
	stream := make([]any, 0, len(items))
	for _, item := range items {
		stream = append(stream, injectStreamDelay(item))
	}

	b.stub.Output = stuber.Output{Stream: stream}

	return b
}

// ReturnHeaders sets response metadata on the effect stub.
func (b *EffectBuilder) ReturnHeaders(headers map[string]string) *EffectBuilder {
	if b.respHeaders == nil {
		b.respHeaders = make(map[string]string, len(headers))
	}

	maps.Copy(b.respHeaders, headers)

	return b
}

// ReturnTrailers sets trailing metadata on the effect stub.
func (b *EffectBuilder) ReturnTrailers(trailers map[string]string) *EffectBuilder {
	if b.respTrailers == nil {
		b.respTrailers = make(map[string]string, len(trailers))
	}

	maps.Copy(b.respTrailers, trailers)

	return b
}

func (b *EffectBuilder) ReturnError(code codes.Code, msg string) *EffectBuilder {
	return b.ReturnErrorWithDetails(code, msg)
}

// ReturnErrorWithDetails fails the effect stub with google.rpc.* details.
func (b *EffectBuilder) ReturnErrorWithDetails(code codes.Code, msg string, details ...map[string]any) *EffectBuilder {
	c := code
	b.stub.Output = stuber.Output{Code: &c, Error: msg, Details: details}

	return b
}

func (b *EffectBuilder) Build() *Effect {
	b.stub.Input = b.matcher
	b.stub.Headers = b.headers

	if len(b.respHeaders) > 0 {
		b.stub.Output.Headers = b.respHeaders
	}

	if len(b.respTrailers) > 0 {
		b.stub.Output.Trailers = b.respTrailers
	}

	stubData := map[string]any{
		"service": b.stub.Service,
		"method":  b.stub.Method,
	}

	if b.id != nil {
		stubData["id"] = b.id.String()
	}

	if b.stub.Session != "" {
		stubData["session"] = b.stub.Session
	}

	if b.stub.Priority != 0 {
		stubData["priority"] = b.stub.Priority
	}

	if b.stub.Options.Times != 0 {
		stubData["options"] = map[string]any{"times": b.stub.Options.Times}
	}

	if input := inputDataToMap(b.matcher); len(input) > 0 {
		stubData["input"] = input
	}

	if headers := inputHeaderToMap(b.headers); len(headers) > 0 {
		stubData["headers"] = headers
	}

	if output := outputToMap(b.stub.Output); len(output) > 0 {
		stubData["output"] = output
	}

	return &Effect{
		effect: stuber.Effect{
			Action: stuber.EffectActionUpsert,
			Stub:   stubData,
		},
	}
}

func inputDataToMap(in stuber.InputData) map[string]any {
	out := map[string]any{}
	putIfAny(out, "equals", in.Equals)
	putIfAny(out, "contains", in.Contains)
	putIfAny(out, "matches", in.Matches)
	putIfAny(out, "glob", in.Glob)

	if len(in.AnyOf) > 0 {
		out["anyOf"] = in.AnyOf
	}

	if in.IgnoreArrayOrder {
		out["ignoreArrayOrder"] = true
	}

	return out
}

func inputHeaderToMap(in stuber.InputHeader) map[string]any {
	out := map[string]any{}
	putIfAny(out, "equals", in.Equals)
	putIfAny(out, "contains", in.Contains)
	putIfAny(out, "matches", in.Matches)
	putIfAny(out, "glob", in.Glob)

	if len(in.AnyOf) > 0 {
		out["anyOf"] = in.AnyOf
	}

	return out
}

func outputToMap(o stuber.Output) map[string]any {
	out := map[string]any{}

	if o.Data != nil {
		out["data"] = o.Data
	}

	if o.Stream != nil {
		out["stream"] = o.Stream
	}

	if o.HasTemplate() {
		out["template"] = true
	}

	if len(o.Headers) > 0 {
		out["headers"] = o.Headers
	}

	if len(o.Trailers) > 0 {
		out["trailers"] = o.Trailers
	}

	if o.Code != nil {
		out["code"] = uint32(*o.Code)
		out["error"] = o.Error
	}

	if len(o.Details) > 0 {
		out["details"] = o.Details
	}

	if o.Delay != "" {
		out["delay"] = string(o.Delay)
	}

	return out
}

func putIfAny(dst map[string]any, key string, src map[string]any) {
	if len(src) > 0 {
		dst[key] = src
	}
}

// Effect attaches side effects to a unary expectation.
func (e *UnaryExpectation) Effect(effects ...*Effect) *UnaryExpectation {
	e.appendEffects(effects...)

	return e
}

// Effect attaches side effects to a server-stream expectation.
func (e *ServerStreamExpectation) Effect(effects ...*Effect) *ServerStreamExpectation {
	e.appendEffects(effects...)

	return e
}

// Effect attaches side effects to a client-stream expectation.
func (e *ClientStreamExpectation) Effect(effects ...*Effect) *ClientStreamExpectation {
	e.appendEffects(effects...)

	return e
}

// Effect attaches side effects to a bidirectional-stream expectation.
func (e *BidirectionalExpectation) Effect(effects ...*Effect) *BidirectionalExpectation {
	e.appendEffects(effects...)

	return e
}

func (b *expectationBase) appendEffects(effects ...*Effect) {
	for _, ef := range effects {
		b.effects = append(b.effects, ef.effect)
	}

	if !b.committed || b.stub == nil {
		return
	}

	b.stub.Effects = b.effects
	b.srv.upsertStub(b.stub)
}
