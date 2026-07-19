package sdk

import (
	"google.golang.org/grpc/codes"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// Effect is a side effect that executes after a stub is matched.
// Use Effect() on a unary expectation to attach effects.
// Created via Upsert(service, method).Match(...).Return(...).Build() or DeleteStub(id).
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
// Created via Upsert(service, method). Chain Match/Return/ReturnError, then call Build().
type EffectBuilder struct {
	stub    stuber.Stub
	matcher stuber.InputData
}

func (b *EffectBuilder) Match(key string, value any) *EffectBuilder {
	b.matcher = mergeInputData(b.matcher, Equals(key, value).compilePayload())

	return b
}

func (b *EffectBuilder) Return(kv ...any) *EffectBuilder {
	b.stub.Output = stuber.Output{Data: parseKVPairs(kv, "sdk.Effect.Return")}

	return b
}

func (b *EffectBuilder) ReturnError(code codes.Code, msg string) *EffectBuilder {
	c := code
	b.stub.Output = stuber.Output{Code: &c, Error: msg}

	return b
}

func (b *EffectBuilder) Build() *Effect {
	b.stub.Input = b.matcher

	stubData := map[string]any{
		"service": b.stub.Service,
		"method":  b.stub.Method,
	}
	if data, ok := b.stub.Output.Data.(map[string]any); ok && len(data) > 0 {
		stubData["output"] = map[string]any{"data": b.stub.Output.Data}
	}

	if len(b.matcher.Equals) > 0 {
		stubData["input"] = map[string]any{"equals": b.matcher.Equals}
	}

	return &Effect{
		effect: stuber.Effect{
			Action: stuber.EffectActionUpsert,
			Stub:   stubData,
		},
	}
}

// Effect must be called before or after Return/Run — both ways work via re-registration.
func (e *UnaryExpectation) Effect(effects ...*Effect) *UnaryExpectation {
	for _, ef := range effects {
		e.effects = append(e.effects, ef.effect)
	}
	// If already committed, rebuild the stub with effects and re-register
	if e.committed {
		output := e.buildOutput()
		stub := &stuber.Stub{
			ID:       e.first,
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
		e.srv.budgerigar.PutMany(stub)
	}

	return e
}

// Effect attaches side effects to a server-stream expectation.
// Call before or after the terminal SendStream — both work via re-registration.
func (e *ServerStreamExpectation) Effect(effects ...*Effect) *ServerStreamExpectation {
	e.appendEffects(effects...)

	return e
}

// Effect attaches side effects to a client-stream expectation.
// Call before or after the terminal Return/ReturnError — both work via re-registration.
func (e *ClientStreamExpectation) Effect(effects ...*Effect) *ClientStreamExpectation {
	e.appendEffects(effects...)

	return e
}

// Effect attaches side effects to a bidirectional-stream expectation.
// Call before or after the terminal Run — both work via re-registration.
func (e *BidirectionalExpectation) Effect(effects ...*Effect) *BidirectionalExpectation {
	e.appendEffects(effects...)

	return e
}

// appendEffects records effects on the shared base and, when the stub has
// already been committed, re-registers it so the effects take hold.
func (b *expectationBase) appendEffects(effects ...*Effect) {
	for _, ef := range effects {
		b.effects = append(b.effects, ef.effect)
	}

	if !b.committed {
		return
	}

	if existing := b.srv.budgerigar.FindByID(b.stubID); existing != nil {
		existing.Effects = b.effects
		b.srv.budgerigar.PutMany(existing)
	}
}
