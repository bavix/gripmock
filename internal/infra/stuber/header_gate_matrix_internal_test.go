package stuber

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type rpcShape struct {
	name        string
	build       func(headers InputHeader) *Stub
	call        func(t *testing.T, budgerigar *Budgerigar, headers map[string]any) bool
	callSession func(t *testing.T, budgerigar *Budgerigar, session string) bool
}

func unaryLikeCall(t *testing.T, budgerigar *Budgerigar, headers map[string]any) bool {
	t.Helper()

	result, err := budgerigar.FindByQuery(Query{
		Service: "svc",
		Method:  "M",
		Headers: headers,
		Input:   []map[string]any{{"deal": "D1"}},
	})

	return err == nil && result != nil && result.Found() != nil
}

func bidiCall(t *testing.T, budgerigar *Budgerigar, headers map[string]any) bool {
	t.Helper()

	return bidiQuery(budgerigar, QueryBidi{Service: "svc", Method: "M", Headers: headers})
}

func bidiQuery(budgerigar *Budgerigar, query QueryBidi) bool {
	result, err := budgerigar.FindByQueryBidi(query)
	if err != nil {
		return false
	}

	stub, err := result.Next(map[string]any{"deal": "D1"})

	return err == nil && stub != nil
}

func unaryLikeSession(t *testing.T, budgerigar *Budgerigar, session string) bool {
	t.Helper()

	result, err := budgerigar.FindByQuery(Query{
		Service: "svc",
		Method:  "M",
		Session: session,
		Input:   []map[string]any{{"deal": "D1"}},
	})

	return err == nil && result != nil && result.Found() != nil
}

func bidiSession(_ *testing.T, budgerigar *Budgerigar, session string) bool {
	return bidiQuery(budgerigar, QueryBidi{Service: "svc", Method: "M", Session: session})
}

func rpcShapes() []rpcShape {
	body := InputData{Equals: map[string]any{"deal": "D1"}}

	return []rpcShape{
		{
			name: "unary",
			build: func(h InputHeader) *Stub {
				return &Stub{Headers: h, Input: body, Output: Output{Data: map[string]any{"ok": true}}}
			},
			call:        unaryLikeCall,
			callSession: unaryLikeSession,
		},
		{
			name: "server stream",
			build: func(h InputHeader) *Stub {
				return &Stub{Headers: h, Input: body, Output: Output{Stream: []any{map[string]any{"ok": true}}}}
			},
			call:        unaryLikeCall,
			callSession: unaryLikeSession,
		},
		{
			name: "client stream",
			build: func(h InputHeader) *Stub {
				return &Stub{Headers: h, Inputs: []InputData{body}, Output: Output{Data: map[string]any{"ok": true}}}
			},
			call:        unaryLikeCall,
			callSession: unaryLikeSession,
		},
		{
			name: "bidi",
			build: func(h InputHeader) *Stub {
				return &Stub{Headers: h, Inputs: []InputData{body}, Output: Output{Stream: []any{map[string]any{"ok": true}}}}
			},
			call:        bidiCall,
			callSession: bidiSession,
		},
	}
}

func storeWith(t *testing.T, stub *Stub) *Budgerigar {
	t.Helper()

	stub.ID = uuid.New()
	stub.Service = "svc"
	stub.Method = "M"

	budgerigar := NewBudgerigar()
	budgerigar.PutMany(stub)

	return budgerigar
}

func TestHeaderGateIsConsistentAcrossRPCShapes(t *testing.T) {
	t.Parallel()

	gated := InputHeader{Equals: map[string]any{"x-tier": "gold"}}

	cases := []struct {
		name    string
		headers map[string]any
		want    bool
	}{
		{name: "matching header", headers: map[string]any{"x-tier": "gold"}, want: true},
		{name: "wrong value", headers: map[string]any{"x-tier": "bronze"}, want: false},
		{name: "absent header", headers: nil, want: false},
		{name: "other header only", headers: map[string]any{"x-trace": "abc"}, want: false},
		{name: "matching plus extra", headers: map[string]any{"x-tier": "gold", "x-trace": "abc"}, want: true},
	}

	for _, shape := range rpcShapes() {
		for _, tc := range cases {
			t.Run(shape.name+"/"+tc.name, func(t *testing.T) {
				t.Parallel()

				budgerigar := storeWith(t, shape.build(gated))
				require.Equal(t, tc.want, shape.call(t, budgerigar, tc.headers))
			})
		}
	}
}

func TestUngatedStubsAcceptAnyHeadersAcrossRPCShapes(t *testing.T) {
	t.Parallel()

	for _, shape := range rpcShapes() {
		t.Run(shape.name, func(t *testing.T) {
			t.Parallel()

			budgerigar := storeWith(t, shape.build(InputHeader{}))

			require.True(t, shape.call(t, budgerigar, nil))
			require.True(t, shape.call(t, budgerigar, map[string]any{"x-trace": "abc"}))
		})
	}
}

func TestEveryHeaderMatcherKindGatesBidi(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		headers InputHeader
		accept  map[string]any
		reject  map[string]any
	}{
		{
			name:    "equals",
			headers: InputHeader{Equals: map[string]any{"x-tier": "gold"}},
			accept:  map[string]any{"x-tier": "gold"},
			reject:  map[string]any{"x-tier": "bronze"},
		},
		{
			name:    "contains",
			headers: InputHeader{Contains: map[string]any{"x-tier": "gold"}},
			accept:  map[string]any{"x-tier": "gold", "x-trace": "abc"},
			reject:  map[string]any{"x-trace": "abc"},
		},
		{
			name:    "matches",
			headers: InputHeader{Matches: map[string]any{"authorization": "^Bearer .+"}},
			accept:  map[string]any{"authorization": "Bearer token-123"},
			reject:  map[string]any{"authorization": "Basic dXNlcg=="},
		},
		{
			name:    "glob",
			headers: InputHeader{Glob: map[string]any{"x-region": "eu-*"}},
			accept:  map[string]any{"x-region": "eu-west-1"},
			reject:  map[string]any{"x-region": "us-east-1"},
		},
		{
			name: "anyOf",
			headers: InputHeader{AnyOf: []AnyOfHeaderElement{
				{Equals: map[string]any{"x-tier": "gold"}},
				{Equals: map[string]any{"x-tier": "platinum"}},
			}},
			accept: map[string]any{"x-tier": "platinum"},
			reject: map[string]any{"x-tier": "bronze"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			budgerigar := storeWith(t, &Stub{
				Headers: tc.headers,
				Inputs:  []InputData{{Equals: map[string]any{"deal": "D1"}}},
				Output:  Output{Stream: []any{map[string]any{"ok": true}}},
			})

			require.True(t, bidiCall(t, budgerigar, tc.accept))
			require.False(t, bidiCall(t, budgerigar, tc.reject))
		})
	}
}

func TestGatedBidiStubWinsDeterministically(t *testing.T) {
	t.Parallel()

	for range 32 {
		require.Equal(t, "gold", servedBidiTier(t, map[string]any{"x-tier": "gold"}))
	}
}

func TestGatedBidiStubDoesNotShadowTheUngatedOne(t *testing.T) {
	t.Parallel()

	require.Equal(t, "gold", servedBidiTier(t, map[string]any{"x-tier": "gold"}))
	require.Equal(t, "any", servedBidiTier(t, nil))
	require.Equal(t, "any", servedBidiTier(t, map[string]any{"x-tier": "bronze"}))
}

func servedBidiTier(t *testing.T, headers map[string]any) string {
	t.Helper()

	body := []InputData{{Equals: map[string]any{"deal": "D1"}}}

	budgerigar := NewBudgerigar()
	budgerigar.PutMany(
		&Stub{
			ID: uuid.New(), Service: "svc", Method: "M",
			Headers: InputHeader{Equals: map[string]any{"x-tier": "gold"}},
			Inputs:  body,
			Output:  Output{Stream: []any{map[string]any{"tier": "gold"}}},
		},
		&Stub{
			ID: uuid.New(), Service: "svc", Method: "M",
			Inputs: body,
			Output: Output{Stream: []any{map[string]any{"tier": "any"}}},
		},
	)

	result, err := budgerigar.FindByQueryBidi(QueryBidi{Service: "svc", Method: "M", Headers: headers})
	require.NoError(t, err)

	stub, err := result.Next(map[string]any{"deal": "D1"})
	require.NoError(t, err)
	require.NotNil(t, stub)

	message, ok := stub.Output.Messages()[0].(map[string]any)
	require.True(t, ok)

	tier, ok := message["tier"].(string)
	require.True(t, ok)

	return tier
}

func handlerStub(headers InputHeader, session string) *Stub {
	return &Stub{
		ID:      uuid.New(),
		Service: "svc",
		Method:  "M",
		Session: session,
		Headers: headers,
		Inputs:  []InputData{{}},
		Handler: func(context.Context, any) error { return nil },
	}
}

func TestHandlerCandidateRespectsHeaderMatchers(t *testing.T) {
	t.Parallel()

	stub := handlerStub(InputHeader{Equals: map[string]any{"x-tier": "gold"}}, "")

	require.True(t, HandlerCandidate(stub, QueryBidi{Headers: map[string]any{"x-tier": "gold"}}))
	require.False(t, HandlerCandidate(stub, QueryBidi{Headers: map[string]any{"x-tier": "bronze"}}))
	require.False(t, HandlerCandidate(stub, QueryBidi{}))
}

func TestHandlerCandidateRespectsSessions(t *testing.T) {
	t.Parallel()

	scoped := handlerStub(InputHeader{}, "team-a")

	require.True(t, HandlerCandidate(scoped, QueryBidi{Session: "team-a"}))
	require.False(t, HandlerCandidate(scoped, QueryBidi{Session: "team-b"}))
	require.False(t, HandlerCandidate(scoped, QueryBidi{}))

	global := handlerStub(InputHeader{}, "")

	require.True(t, HandlerCandidate(global, QueryBidi{}))
	require.True(t, HandlerCandidate(global, QueryBidi{Session: "team-a"}))
}

func TestHandlerCandidateIgnoresStubsWithoutHandler(t *testing.T) {
	t.Parallel()

	static := &Stub{
		ID: uuid.New(), Service: "svc", Method: "M",
		Inputs: []InputData{{}},
		Output: Output{Stream: []any{map[string]any{"ok": true}}},
	}

	require.False(t, HandlerCandidate(static, QueryBidi{}))
}

func TestSessionScopingIsConsistentAcrossRPCShapes(t *testing.T) {
	t.Parallel()

	for _, shape := range rpcShapes() {
		t.Run(shape.name, func(t *testing.T) {
			t.Parallel()

			scoped := shape.build(InputHeader{})
			scoped.Session = "team-a"
			budgerigar := storeWith(t, scoped)

			require.True(t, shape.callSession(t, budgerigar, "team-a"))
			require.False(t, shape.callSession(t, budgerigar, "team-b"))
			require.False(t, shape.callSession(t, budgerigar, ""))
		})
	}
}

func TestCallBudgetIsConsistentAcrossRPCShapes(t *testing.T) {
	t.Parallel()

	for _, shape := range rpcShapes() {
		t.Run(shape.name, func(t *testing.T) {
			t.Parallel()

			limited := shape.build(InputHeader{})
			limited.Options = StubOptions{Times: 1}
			budgerigar := storeWith(t, limited)

			require.True(t, shape.call(t, budgerigar, nil))
			require.False(t, shape.call(t, budgerigar, nil))
		})
	}
}
