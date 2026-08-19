package sdk_test

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/app"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/domain/rest"
	"github.com/bavix/gripmock/v3/internal/infra/httputil"
	"github.com/bavix/gripmock/v3/internal/infra/muxmiddleware"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	sdk "github.com/bavix/gripmock/v3/pkg/sdk"
)

const (
	greeterFullMethod = "/test.Greeter/SayHello"
	sessionID         = "parity-suite"
)

const streamsProto = `
syntax = "proto3";
package test;
service Feed {
  rpc Watch (WatchRequest) returns (stream WatchEvent);
}
service Collector {
  rpc Collect (stream Chunk) returns (CollectSummary);
}
service Duplex {
  rpc Exchange (stream Ping) returns (stream Pong);
}
message WatchRequest { string topic = 1; }
message WatchEvent { string id = 1; string payload = 2; }
message Chunk { string part = 1; }
message CollectSummary { string digest = 1; int32 parts = 2; }
message Ping { string text = 1; }
message Pong { string text = 1; }
`

func compileParity(t *testing.T) *descriptorpb.FileDescriptorSet {
	t.Helper()

	unary := compileInline(t, testProto, "test.proto")
	streams := compileInline(t, streamsProto, "streams.proto")

	return &descriptorpb.FileDescriptorSet{File: append(unary.GetFile(), streams.GetFile()...)}
}

func startGripmock(t *testing.T, fds *descriptorpb.FileDescriptorSet) (string, string) {
	t.Helper()

	ctx := t.Context()
	budgerigar := stuber.NewBudgerigar()
	recorder := &history.MemoryStore{}
	extender := app.NewInstantExtender()

	grpcServer, err := app.BuildFromDescriptorSet(ctx, fds, budgerigar, extender, recorder)
	require.NoError(t, err)

	lis, err := net.Listen("tcp", "127.0.0.1:0") //nolint:noctx
	require.NoError(t, err)

	go func() { _ = grpcServer.Serve(lis) }()

	restServer, err := app.NewRestServer(ctx, budgerigar, extender, recorder, nil, nil, nil)
	require.NoError(t, err)

	router := mux.NewRouter()
	rest.HandlerWithOptions(restServer, rest.GorillaServerOptions{
		BaseURL:    "/api",
		BaseRouter: router,
		Middlewares: []rest.MiddlewareFunc{
			muxmiddleware.PanicRecoveryMiddleware,
			muxmiddleware.TransportSession,
			muxmiddleware.ContentType,
		},
	})

	httpSrv := httptest.NewServer(httputil.GzipRequestMiddleware(router))

	t.Cleanup(func() {
		httpSrv.Close()
		grpcServer.GracefulStop()
	})

	return lis.Addr().String(), httpSrv.URL
}

func TestRemoteHistoryIsReadableFromCleanup(t *testing.T) { //nolint:paralleltest // boots its own server
	grpcAddr, restURL := startGripmock(t, compileParity(t))
	fds := compileParity(t)

	srv := sdk.NewTestServer(t,
		sdk.WithDescriptors(fds),
		sdk.WithRemote(grpcAddr, restURL),
	)

	srv.ExpectUnary(greeterFullMethod).Match("name", "Cleanup").Return("message", "ok")
	require.Equal(t, "ok", getMsg(t, sayHello(t, srv, fds, "Cleanup")))

	t.Cleanup(func() {
		require.Len(t, srv.History(), 1)
		require.Equal(t, 1, srv.Called(greeterFullMethod))
		require.Equal(t, 1, srv.TotalCalls())
	})
}

func TestRemoteResetWithoutSessionClearsEverything(t *testing.T) { //nolint:paralleltest // boots its own server
	grpcAddr, restURL := startGripmock(t, compileParity(t))

	scoped := sdk.NewTestServer(t,
		sdk.WithDescriptors(compileParity(t)),
		sdk.WithRemote(grpcAddr, restURL),
		sdk.WithSession("keeps-its-calls"),
	)

	global, fds := sdk.NewTestServer(t,
		sdk.WithDescriptors(compileParity(t)),
		sdk.WithRemote(grpcAddr, restURL),
	), compileParity(t)

	for _, srv := range []*sdk.Server{scoped, global} {
		srv.ExpectUnary(greeterFullMethod).Match("name", "shared").Return("message", "hi")
		require.Equal(t, "hi", getMsg(t, sayHello(t, srv, fds, "shared")))
	}

	global.Reset()

	require.Equal(t, 0, global.TotalCalls())
	require.Equal(t, 0, scoped.TotalCalls(), "an unscoped purge takes the session's calls with it")
}

func TestRemoteResetKeepsOtherSessions(t *testing.T) { //nolint:paralleltest // boots its own server
	grpcAddr, restURL := startGripmock(t, compileParity(t))

	newRemote := func(session string) (*sdk.Server, *descriptorpb.FileDescriptorSet) {
		fds := compileParity(t)

		return sdk.NewTestServer(t,
			sdk.WithDescriptors(fds),
			sdk.WithRemote(grpcAddr, restURL),
			sdk.WithSession(session),
		), fds
	}

	mine, mineFDS := newRemote("reset-mine")
	theirs, theirsFDS := newRemote("reset-theirs")

	for _, pair := range []struct {
		srv *sdk.Server
		fds *descriptorpb.FileDescriptorSet
	}{{mine, mineFDS}, {theirs, theirsFDS}} {
		pair.srv.ExpectUnary(greeterFullMethod).
			Match("name", "shared").
			Return("message", "hi")

		require.Equal(t, "hi", getMsg(t, sayHello(t, pair.srv, pair.fds, "shared")))
	}

	mine.Reset()

	require.Equal(t, 0, mine.TotalCalls())
	require.Equal(t, 1, theirs.TotalCalls(), "a reset must not purge another session")
}

func TestRemoteDescriptorUploadsDoNotAccumulate(t *testing.T) { //nolint:paralleltest // boots its own server
	grpcAddr, restURL := startGripmock(t, compileParity(t))

	fds := compileParity(t)

	services := func() []string {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, restURL+"/api/descriptors", nil)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		defer func() { _ = resp.Body.Close() }()

		require.Equal(t, http.StatusOK, resp.StatusCode)

		var payload struct {
			ServiceIDs []string `json:"serviceIDs"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))

		return payload.ServiceIDs
	}

	newRemote := func(session string) *sdk.Server {
		return sdk.NewTestServer(t,
			sdk.WithDescriptors(fds),
			sdk.WithRemote(grpcAddr, restURL),
			sdk.WithSession(session),
		)
	}

	first := newRemote("descriptors-first")
	afterFirst := services()
	require.NotEmpty(t, afterFirst)
	require.NoError(t, first.Close())

	second := newRemote("descriptors-second")

	require.ElementsMatch(t, afterFirst, services(), "re-uploading the same files must not add services")

	second.ExpectUnary(greeterFullMethod).Match("name", "still-here").Return("message", "yes")
	require.Equal(t, "yes", getMsg(t, sayHello(t, second, fds, "still-here")))
}

type scenario struct {
	name string
	run  func(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet)
}

func parityScenarios() []scenario {
	return []scenario{
		{name: "unary next will return", run: scenarioNextWillReturn},
		{name: "unary error", run: scenarioUnaryError},
		{name: "history keeps all fields", run: scenarioHistoryFields},
		{name: "effect after terminal", run: scenarioEffectAfterTerminal},
		{name: "expectations were met", run: scenarioExpectationsWereMet},
		{name: "reset removes stubs", run: scenarioResetRemovesStubs},
		{name: "reset clears history", run: scenarioResetClearsHistory},
		{name: "session scoped stub", run: scenarioSessionScopedStub},
		{name: "server stream", run: scenarioServerStream},
		{name: "server stream aborts", run: scenarioServerStreamAborts},
		{name: "client stream sequence", run: scenarioClientStreamSequence},
		{name: "client stream status", run: scenarioClientStreamStatus},
		{name: "static bidi", run: scenarioStaticBidi},
		{name: "bidi header gate", run: scenarioBidiHeaderGate},
		{name: "bidi error", run: scenarioBidiError},
		{name: "stream call budget", run: scenarioStreamCallBudget},
		{name: "stream effect", run: scenarioStreamEffect},
	}
}

//nolint:paralleltest // concurrency starves the health check under -race
func TestEmbeddedRemoteParity(t *testing.T) {
	for _, sc := range parityScenarios() {
		t.Run(sc.name+"/embedded", func(t *testing.T) {
			fds := compileParity(t)
			sc.run(t, sdk.NewTestServer(t,
				sdk.WithDescriptors(fds),
				sdk.WithSession(sessionID),
			), fds)
		})

		t.Run(sc.name+"/remote", func(t *testing.T) {
			grpcAddr, restURL := startGripmock(t, compileParity(t))
			fds := compileParity(t)
			sc.run(t, sdk.NewTestServer(t,
				sdk.WithDescriptors(fds),
				sdk.WithRemote(grpcAddr, restURL),
				sdk.WithSession(sessionID),
			), fds)
		})
	}
}

func scenarioNextWillReturn(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet) {
	t.Helper()

	srv.ExpectUnary(greeterFullMethod).
		Match("name", "Alex").
		Return("message", "first").
		NextWillReturn("message", "second")

	require.Equal(t, "first", getMsg(t, sayHello(t, srv, fds, "Alex")))
	require.Equal(t, "second", getMsg(t, sayHello(t, srv, fds, "Alex")))

	require.Equal(t, 2, srv.Called(greeterFullMethod))
	require.Equal(t, 2, srv.TotalCalls())
}

func scenarioUnaryError(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet) {
	t.Helper()

	srv.ExpectUnary(greeterFullMethod).
		Match("name", "Bob").
		Once().
		ReturnError(codes.NotFound, "no such greeting")

	err := sayHelloErr(t, srv, fds, "Bob")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no such greeting")
}

func scenarioHistoryFields(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet) {
	t.Helper()

	srv.ExpectUnary(greeterFullMethod).
		Match("name", "Ann").
		Once().
		ReturnError(codes.PermissionDenied, "denied")

	require.Error(t, sayHelloErr(t, srv, fds, "Ann"))

	records := srv.History()
	require.Len(t, records, 1)

	require.Equal(t, uint32(codes.PermissionDenied), records[0].Code)
	require.Equal(t, "test.Greeter", records[0].Service)
	require.Equal(t, "SayHello", records[0].Method)
	require.NotEmpty(t, records[0].Requests[0])
}

func scenarioEffectAfterTerminal(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet) {
	t.Helper()

	effect := sdk.Upsert("test.Greeter", "SayHello").
		Match("name", "step2").
		Return("message", "unlocked").
		Build()

	srv.ExpectUnary(greeterFullMethod).
		Match("name", "step1").
		Return("message", "started").
		Effect(effect)

	require.Equal(t, "started", getMsg(t, sayHello(t, srv, fds, "step1")))
	require.Equal(t, "unlocked", getMsg(t, sayHello(t, srv, fds, "step2")))
}

func scenarioExpectationsWereMet(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet) {
	t.Helper()

	srv.ExpectUnary(greeterFullMethod).
		Match("name", "Zoe").
		Twice().
		Return("message", "hi")

	require.Equal(t, "hi", getMsg(t, sayHello(t, srv, fds, "Zoe")))
	require.Error(t, srv.ExpectationsWereMetContext(t.Context()), "one call short must fail")

	require.Equal(t, "hi", getMsg(t, sayHello(t, srv, fds, "Zoe")))
	require.Equal(t, 2, srv.Called(greeterFullMethod))
}

func scenarioResetRemovesStubs(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet) {
	t.Helper()

	srv.ExpectUnary(greeterFullMethod).
		Match("name", "before").
		Return("message", "old")

	require.Equal(t, "old", getMsg(t, sayHello(t, srv, fds, "before")))

	srv.Reset()

	require.Error(t, sayHelloErr(t, srv, fds, "before"), "stub must be gone after Reset")

	srv.ExpectUnary(greeterFullMethod).
		Match("name", "after").
		Return("message", "new")

	require.Equal(t, "new", getMsg(t, sayHello(t, srv, fds, "after")))
}

func scenarioResetClearsHistory(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet) {
	t.Helper()

	srv.ExpectUnary(greeterFullMethod).
		Match("name", "counted").
		Return("message", "one")

	require.Equal(t, "one", getMsg(t, sayHello(t, srv, fds, "counted")))
	require.Equal(t, 1, srv.TotalCalls())

	srv.Reset()

	require.Empty(t, srv.History())
	require.Equal(t, 0, srv.TotalCalls())
	require.Equal(t, 0, srv.Called(greeterFullMethod))
}

func scenarioServerStream(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet) {
	t.Helper()

	srv.ExpectServerStream("/test.Feed/Watch").
		Match("topic", "orders").
		ReturnHeaders(map[string]string{"x-feed": "live"}).
		SendStream(
			map[string]any{"id": "1", "payload": "first"},
			map[string]any{"id": "2", "payload": "second"},
		)

	stream, d := watch(t, srv, fds, "orders")

	header, err := stream.Header()
	require.NoError(t, err)
	require.Equal(t, []string{"live"}, header.Get("x-feed"))

	for _, want := range []string{"first", "second"} {
		out := dynamicpb.NewMessage(d.out)
		require.NoError(t, stream.RecvMsg(out))
		require.Equal(t, want, out.Get(d.out.Fields().ByName("payload")).String())
	}

	require.Error(t, stream.RecvMsg(dynamicpb.NewMessage(d.out)), "stream must end after the declared items")
}

func scenarioServerStreamAborts(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet) {
	t.Helper()

	srv.ExpectServerStream("/test.Feed/Watch").
		Match("topic", "flaky").
		SendStream(
			map[string]any{"id": "1", "payload": "partial"},
			sdk.StreamError(codes.Aborted, "producer died"),
		)

	stream, d := watch(t, srv, fds, "flaky")

	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, stream.RecvMsg(out))
	require.Equal(t, "partial", out.Get(d.out.Fields().ByName("payload")).String())

	err := stream.RecvMsg(dynamicpb.NewMessage(d.out))
	require.Error(t, err)
	require.Equal(t, codes.Aborted, status.Code(err))
}

func scenarioClientStreamSequence(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet) {
	t.Helper()

	srv.ExpectClientStream("/test.Collector/Collect").
		MatchSequence(sdk.Equals("part", "a"), sdk.Equals("part", "b")).
		Return("digest", "ab", "parts", 2)

	out, d, err := collect(t, srv, fds, "a", "b")
	require.NoError(t, err)
	require.Equal(t, "ab", out.Get(d.out.Fields().ByName("digest")).String())

	_, _, err = collect(t, srv, fds, "b", "a")
	require.Error(t, err, "the reversed order must not match a positional stub")
	require.Equal(t, codes.NotFound, status.Code(err))
}

func scenarioClientStreamStatus(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet) {
	t.Helper()

	srv.ExpectClientStream("/test.Collector/Collect").
		Match(sdk.Matches("part", ".*")).
		ReturnStatus(codes.ResourceExhausted)

	_, _, err := collect(t, srv, fds, "x")
	require.Error(t, err)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
	require.Empty(t, status.Convert(err).Message())
}

func scenarioStaticBidi(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet) {
	t.Helper()

	srv.ExpectBidirectionalStream("/test.Duplex/Exchange").
		Match("text", "ping").
		SendStream(map[string]any{"text": "pong"})

	reply, err := exchange(t, srv, fds, "ping")
	require.NoError(t, err)
	require.Equal(t, "pong", reply)

	_, err = exchange(t, srv, fds, "nope")
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func scenarioBidiHeaderGate(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet) {
	t.Helper()

	srv.ExpectBidirectionalStream("/test.Duplex/Exchange").
		WithHeader(sdk.Equals("x-tier", "gold")).
		Match("text", "ping").
		SendStream(map[string]any{"text": "gold-pong"})

	_, err := exchange(t, srv, fds, "ping")
	require.Error(t, err, "a gated stub must not answer a call without the header")
	require.Equal(t, codes.NotFound, status.Code(err))
}

func scenarioBidiError(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet) {
	t.Helper()

	srv.ExpectBidirectionalStream("/test.Duplex/Exchange").
		Match("text", "boom").
		ReturnError(codes.FailedPrecondition, "channel closed")

	_, err := exchange(t, srv, fds, "boom")
	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	require.Equal(t, "channel closed", status.Convert(err).Message())
}

func scenarioStreamCallBudget(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet) {
	t.Helper()

	srv.ExpectServerStream("/test.Feed/Watch").
		Match("topic", "once").
		Once().
		SendStream(map[string]any{"id": "1", "payload": "only"})

	stream, d := watch(t, srv, fds, "once")

	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, stream.RecvMsg(out))
	require.Equal(t, "only", out.Get(d.out.Fields().ByName("payload")).String())

	require.Error(t, stream.RecvMsg(dynamicpb.NewMessage(d.out)))
	require.NoError(t, srv.ExpectationsWereMetContext(t.Context()))

	exhausted, d2 := watch(t, srv, fds, "once")
	err := exhausted.RecvMsg(dynamicpb.NewMessage(d2.out))
	require.Error(t, err, "a spent Once stub must stop matching")
	require.Equal(t, codes.NotFound, status.Code(err))
}

func scenarioStreamEffect(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet) {
	t.Helper()

	unlock := sdk.Upsert("test.Collector", "Collect").
		Match("part", "unlocked").
		Return("digest", "granted", "parts", 1).
		Build()

	srv.ExpectServerStream("/test.Feed/Watch").
		Match("topic", "unlock").
		Effect(unlock).
		SendStream(map[string]any{"id": "1", "payload": "ok"})

	_, _, err := collect(t, srv, fds, "unlocked")
	require.Error(t, err, "the effect stub must not exist before the stream runs")

	stream, d := watch(t, srv, fds, "unlock")
	require.NoError(t, stream.RecvMsg(dynamicpb.NewMessage(d.out)))

	out, summary, err := collect(t, srv, fds, "unlocked")
	require.NoError(t, err)
	require.Equal(t, "granted", out.Get(summary.out.Fields().ByName("digest")).String())
}

func scenarioSessionScopedStub(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet) {
	t.Helper()

	srv.ExpectUnary(greeterFullMethod).
		Session(sessionID).
		Match("name", "scoped").
		Return("message", "scoped-reply")

	srv.ExpectUnary(greeterFullMethod).
		Session("another-suite").
		Match("name", "foreign").
		Return("message", "foreign-reply")

	require.Equal(t, "scoped-reply", getMsg(t, sayHello(t, srv, fds, "scoped")))
	require.Equal(t, 1, srv.Called(greeterFullMethod))

	err := sayHelloErr(t, srv, fds, "foreign")
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

//nolint:ireturn // grpc.ClientStream is the only type NewStream can return.
func watch(
	t *testing.T,
	srv *sdk.Server,
	fds *descriptorpb.FileDescriptorSet,
	topic string,
) (grpc.ClientStream, msgDesc) {
	t.Helper()

	d := resolveDesc(t, fds, "test.WatchRequest", "test.WatchEvent")

	stream, err := srv.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "Watch", ServerStreams: true},
		"/test.Feed/Watch")
	require.NoError(t, err)

	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("topic"), protoreflect.ValueOfString(topic))
	require.NoError(t, stream.SendMsg(in))
	require.NoError(t, stream.CloseSend())

	return stream, d
}

func collect(
	t *testing.T,
	srv *sdk.Server,
	fds *descriptorpb.FileDescriptorSet,
	parts ...string,
) (*dynamicpb.Message, msgDesc, error) {
	t.Helper()

	d := resolveDesc(t, fds, "test.Chunk", "test.CollectSummary")

	stream, err := srv.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "Collect", ClientStreams: true},
		"/test.Collector/Collect")
	require.NoError(t, err)

	for _, part := range parts {
		msg := dynamicpb.NewMessage(d.in)
		msg.Set(d.in.Fields().ByName("part"), protoreflect.ValueOfString(part))
		require.NoError(t, stream.SendMsg(msg))
	}

	require.NoError(t, stream.CloseSend())

	out := dynamicpb.NewMessage(d.out)
	if err := stream.RecvMsg(out); err != nil {
		return nil, d, err
	}

	return out, d, nil
}

func exchange(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet, text string) (string, error) {
	t.Helper()

	d := resolveDesc(t, fds, "test.Ping", "test.Pong")

	stream, err := srv.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "Exchange", ServerStreams: true, ClientStreams: true},
		"/test.Duplex/Exchange")
	require.NoError(t, err)

	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("text"), protoreflect.ValueOfString(text))
	require.NoError(t, stream.SendMsg(in))
	require.NoError(t, stream.CloseSend())

	out := dynamicpb.NewMessage(d.out)
	if err := stream.RecvMsg(out); err != nil {
		return "", err
	}

	return out.Get(d.out.Fields().ByName("text")).String(), nil
}
