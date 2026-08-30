package app

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bavix/gripmock/v3/internal/domain/history"
	protosetdom "github.com/bavix/gripmock/v3/internal/domain/protoset"
	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

const bidiProxyTestMethod = "/proxytest.Chat/Talk"

var errUnexpectedRawCodecValue = errors.New("raw codec expects *[]byte")

type rawBytesCodec struct{}

func (rawBytesCodec) Marshal(v any) ([]byte, error) {
	payload, ok := v.(*[]byte)
	if !ok {
		return nil, errUnexpectedRawCodecValue
	}

	return *payload, nil
}

func (rawBytesCodec) Unmarshal(data []byte, v any) error {
	payload, ok := v.(*[]byte)
	if !ok {
		return errUnexpectedRawCodecValue
	}

	*payload = append((*payload)[:0], data...)

	return nil
}

func (rawBytesCodec) Name() string { return "proto" }

type halfClosedDownstream struct {
	grpc.ServerStream

	ctx context.Context //nolint:containedctx

	mu   sync.Mutex
	sent []*dynamicpb.Message
}

func (d *halfClosedDownstream) Context() context.Context { return d.ctx }

func (d *halfClosedDownstream) RecvMsg(any) error { return io.EOF }

func (d *halfClosedDownstream) SendMsg(msg any) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if message, ok := msg.(*dynamicpb.Message); ok {
		d.sent = append(d.sent, message)
	}

	return nil
}

func (d *halfClosedDownstream) SetHeader(metadata.MD) error  { return nil }
func (d *halfClosedDownstream) SendHeader(metadata.MD) error { return nil }
func (d *halfClosedDownstream) SetTrailer(metadata.MD)       {}

func (d *halfClosedDownstream) received() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	return len(d.sent)
}

func startSlowBidiUpstream(t *testing.T, replyAfter time.Duration) *grpc.ClientConn {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)

	handler := func(_ any, stream grpc.ServerStream) error {
		for {
			var in []byte
			if err := stream.RecvMsg(&in); err != nil {
				break
			}
		}

		select {
		case <-time.After(replyAfter):
		case <-stream.Context().Done():
			return stream.Context().Err()
		}

		out, err := proto.Marshal(wrapperspb.String("late"))
		if err != nil {
			return err
		}

		return stream.SendMsg(&out)
	}

	server := grpc.NewServer(
		grpc.UnknownServiceHandler(handler),
		grpc.ForceServerCodec(rawBytesCodec{}),
	)

	go func() { _ = server.Serve(listener) }()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = conn.Close()

		server.Stop()

		_ = listener.Close()
	})

	return conn
}

func newBidiProxyMocker() *grpcMocker {
	desc := (&wrapperspb.StringValue{}).ProtoReflect().Descriptor()

	return &grpcMocker{
		inputDesc:       desc,
		outputDesc:      desc,
		fullMethod:      bidiProxyTestMethod,
		fullServiceName: "proxytest.Chat",
		methodName:      "Talk",
		serverStream:    true,
		clientStream:    true,
		maxNestingDepth: defaultConvertDepth,
	}
}

func TestProxyBidiWaitsForUpstreamAfterClientHalfClose(t *testing.T) {
	t.Parallel()

	const (
		replyAfter  = 300 * time.Millisecond
		drainWindow = 50 * time.Millisecond
	)

	conn := startSlowBidiUpstream(t, replyAfter)

	downstream := &halfClosedDownstream{ctx: t.Context()}
	route := &proxyroutes.Route{
		Conn:   conn,
		Source: &protosetdom.Source{ReflectTimeout: drainWindow},
	}

	err := newBidiProxyMocker().proxyBidiStreamWithRequests(downstream, route, nil, false)
	require.NoError(t, err)

	require.Equal(t, 1, downstream.received(),
		"upstream response after half-close was dropped: the drain guard fired on a healthy stream")
}

var errBidiSide = errors.New("bidi side failed")

func requireCancelled(t *testing.T, cancelled <-chan struct{}) {
	t.Helper()

	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("guard window expired without cancelling the stuck side")
	}
}

func TestAwaitBidiCompletionDrainPolicy(t *testing.T) {
	t.Parallel()

	const guard = 20 * time.Millisecond

	t.Run("clean request finish waits out the response side", func(t *testing.T) {
		t.Parallel()

		reqDone := make(chan error, 1)
		respDone := make(chan error, 1)

		reqDone <- nil

		go func() {
			time.Sleep(4 * guard)

			respDone <- nil
		}()

		cancelled := false
		reqErr, respErr := awaitBidiCompletion(reqDone, respDone, guard, func() { cancelled = true })

		require.NoError(t, reqErr)
		require.NoError(t, respErr)
		require.False(t, cancelled, "a healthy stream must not be cancelled by the guard")
	})

	t.Run("request failure does not wait forever on a stuck response side", func(t *testing.T) {
		t.Parallel()

		reqDone := make(chan error, 1)
		reqDone <- errBidiSide

		cancelled := make(chan struct{})

		reqErr, respErr := awaitBidiCompletion(reqDone, make(chan error), guard,
			func() { close(cancelled) })

		require.ErrorIs(t, reqErr, errBidiSide)
		require.NoError(t, respErr)
		requireCancelled(t, cancelled)
	})

	t.Run("upstream finishing first does not wait forever on the client", func(t *testing.T) {
		t.Parallel()

		respDone := make(chan error, 1)
		respDone <- errBidiSide

		cancelled := make(chan struct{})

		reqErr, respErr := awaitBidiCompletion(make(chan error), respDone, guard,
			func() { close(cancelled) })

		require.NoError(t, reqErr)
		require.ErrorIs(t, respErr, errBidiSide)
		requireCancelled(t, cancelled)
	})
}

func TestProxyBidiCaptureStillRecordsStub(t *testing.T) {
	t.Parallel()

	conn := startSlowBidiUpstream(t, 10*time.Millisecond)

	mocker := newBidiProxyMocker()
	mocker.budgerigar = stuber.NewBudgerigar()

	downstream := &halfClosedDownstream{ctx: t.Context()}
	route := &proxyroutes.Route{Conn: conn, Source: &protosetdom.Source{}}

	require.NoError(t, mocker.proxyBidiStreamWithRequests(downstream, route, nil, true))

	stubs := mocker.budgerigar.All()
	require.Len(t, stubs, 1)
	require.Equal(t, "proxytest.Chat", stubs[0].Service)
	require.Len(t, stubs[0].Output.Stream, 1, "the captured stub must carry the upstream response")
}

func TestStreamCaptureStateNilIsInert(t *testing.T) {
	t.Parallel()

	var state *StreamCaptureState

	state.AppendRequest(map[string]any{"name": "a"})
	state.AppendResponseWithTiming(map[string]any{"message": "b"}, time.Now())

	requests, responses := state.Snapshot()

	require.Nil(t, requests)
	require.Nil(t, responses)
	require.False(t, state.HasTimedResponses())
}

type trailerExtraRecorder struct{ lines []string }

func (r *trailerExtraRecorder) setTrailerExtra(lines ...string) {
	r.lines = append(r.lines, lines...)
}

func TestForwardTrailerExtrasFiltersHopByHop(t *testing.T) {
	t.Parallel()

	recorder := &trailerExtraRecorder{}

	forwardTrailerExtras(recorder,
		metadata.MD{"x-upstream": []string{"a", "b"}, "content-type": []string{"application/grpc"}},
		metadata.MD{"x-tail": []string{"z"}, "grpc-status": []string{"0"}},
	)

	require.ElementsMatch(t, []string{"x-upstream: a,b", "x-tail: z"}, recorder.lines)
}

func startScriptedUpstream(t *testing.T, values ...string) *grpc.ClientConn {
	t.Helper()

	payloads := make([]string, 0, len(values))

	for _, value := range values {
		encoded, err := proto.Marshal(wrapperspb.String(value))
		require.NoError(t, err)

		payloads = append(payloads, string(encoded))
	}

	return startRawUpstream(t, payloads...)
}

func startRawUpstream(t *testing.T, payloads ...string) *grpc.ClientConn {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)

	handler := func(_ any, stream grpc.ServerStream) error {
		for {
			var in []byte
			if err := stream.RecvMsg(&in); err != nil {
				break
			}
		}

		for _, payload := range payloads {
			out := []byte(payload)
			if err := stream.SendMsg(&out); err != nil {
				return err
			}
		}

		return nil
	}

	server := grpc.NewServer(
		grpc.UnknownServiceHandler(handler),
		grpc.ForceServerCodec(rawBytesCodec{}),
	)

	go func() { _ = server.Serve(listener) }()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = conn.Close()

		server.Stop()

		_ = listener.Close()
	})

	return conn
}

func TestProxyStreamCaptureKeepsScalarResponses(t *testing.T) {
	t.Parallel()

	for name, run := range map[string]func(*grpcMocker, grpc.ServerStream, *proxyroutes.Route) error{
		"server stream": func(m *grpcMocker, s grpc.ServerStream, r *proxyroutes.Route) error {
			return m.proxyServerStreamWithRequest(s, r, dynamicpb.NewMessage(m.inputDesc), true)
		},
		"bidi": func(m *grpcMocker, s grpc.ServerStream, r *proxyroutes.Route) error {
			return m.proxyBidiStreamWithRequests(s, r, nil, true)
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mocker := newBidiProxyMocker()
			mocker.budgerigar = stuber.NewBudgerigar()

			route := &proxyroutes.Route{
				Conn:   startScriptedUpstream(t, "one", "two"),
				Source: &protosetdom.Source{},
			}

			require.NoError(t, run(mocker, &halfClosedDownstream{ctx: t.Context()}, route))

			stubs := mocker.budgerigar.All()
			require.Len(t, stubs, 1)
			require.Equal(t, []any{"one", "two"}, stubs[0].Output.Stream)
		})
	}
}

func startStructUpstream(t *testing.T, messages ...string) *grpc.ClientConn {
	t.Helper()

	payloads := make([]string, 0, len(messages))

	for _, message := range messages {
		value, err := structpb.NewStruct(map[string]any{"message": message})
		require.NoError(t, err)

		encoded, err := proto.Marshal(value)
		require.NoError(t, err)

		payloads = append(payloads, string(encoded))
	}

	return startRawUpstream(t, payloads...)
}

func TestProxyStreamRecordsHistory(t *testing.T) {
	t.Parallel()

	for name, run := range map[string]func(*grpcMocker, grpc.ServerStream, *proxyroutes.Route) error{
		"server stream": func(m *grpcMocker, s grpc.ServerStream, r *proxyroutes.Route) error {
			return m.proxyServerStreamWithRequest(s, r, dynamicpb.NewMessage(m.inputDesc), false)
		},
		"bidi": func(m *grpcMocker, s grpc.ServerStream, r *proxyroutes.Route) error {
			return m.proxyBidiStreamWithRequests(s, r, nil, false)
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			store := history.NewMemoryStore(1 << 20)

			structDesc := (&structpb.Struct{}).ProtoReflect().Descriptor()

			mocker := newBidiProxyMocker()
			mocker.recorder = store
			mocker.inputDesc = structDesc
			mocker.outputDesc = structDesc

			route := &proxyroutes.Route{
				Conn:   startStructUpstream(t, "one", "two"),
				Source: &protosetdom.Source{},
			}

			require.NoError(t, run(mocker, &halfClosedDownstream{ctx: t.Context()}, route))

			calls := store.All()
			require.Len(t, calls, 1)
			require.Equal(t, "proxytest.Chat", calls[0].Service)
			require.Equal(t, uuid.Nil, calls[0].StubID, "a proxied call is served by no stub")
			require.Equal(t, uint32(0), calls[0].Code)
			require.Equal(t,
				[]any{map[string]any{"message": "one"}, map[string]any{"message": "two"}},
				calls[0].Responses,
				"every upstream message belongs in the record, not just the first")
		})
	}
}

func TestCapturableResultRejectsUselessStubs(t *testing.T) {
	t.Parallel()

	cancelErr := status.Error(codes.Canceled, "context canceled")

	for name, testCase := range map[string]struct {
		clientGone bool
		requests   int
		responses  int
		callErr    error
		want       bool
	}{
		"nothing exchanged":                   {false, 0, 0, cancelErr, false},
		"client went away":                    {true, 1, 0, cancelErr, false},
		"upstream error on a real request":    {false, 1, 0, status.Error(codes.NotFound, "no stub"), true},
		"upstream pushed without being asked": {false, 0, 2, nil, true},
		"clean call":                          {false, 1, 1, nil, true},
		"upstream cancelled, client alive":    {false, 1, 0, cancelErr, true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)

			if testCase.clientGone {
				cancel()
			}

			require.Equal(t, testCase.want,
				capturableResult(ctx, testCase.requests, testCase.responses, testCase.callErr))
		})
	}
}

func TestProxyBidiAbortedClientCapturesNothing(t *testing.T) {
	t.Parallel()

	mocker := newBidiProxyMocker()
	mocker.budgerigar = stuber.NewBudgerigar()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	route := &proxyroutes.Route{
		Conn:   startScriptedUpstream(t),
		Source: &protosetdom.Source{},
	}

	_ = mocker.proxyBidiStreamWithRequests(&halfClosedDownstream{ctx: ctx}, route, nil, true)

	require.Empty(t, mocker.budgerigar.All(),
		"an aborted stream must not leave a catch-all error stub behind")
}

func TestCapturedHeadersDropSecretsAndVolatileIDs(t *testing.T) {
	t.Parallel()

	headers := requestHeadersFromMetadata(metadata.MD{
		"authorization": []string{"Bearer sk-live-SECRET"},
		"cookie":        []string{"session=abc"},
		"traceparent":   []string{"00-abc-def-01"},
		"x-request-id":  []string{"req-1"},
		"x-tenant":      []string{"acme"},
	})

	require.Equal(t, map[string]any{"x-tenant": "acme"}, headers)
}

func TestCapturedHeadersEmptyWhenAllDenied(t *testing.T) {
	t.Parallel()

	require.Nil(t, requestHeadersFromMetadata(metadata.MD{
		"authorization": []string{"Bearer x"},
		"x-request-id":  []string{"req-1"},
	}))
}

func TestCapturedHeadersDropSessionControlHeader(t *testing.T) {
	t.Parallel()

	require.Equal(t, map[string]any{"x-tenant": "acme"}, requestHeadersFromMetadata(metadata.MD{
		sessionHeaderKey: []string{"team-a"},
		"x-tenant":       []string{"acme"},
	}))
}
