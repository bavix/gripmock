package sdk_test

import (
	"context"
	"errors"
	"io"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	sdk "github.com/bavix/gripmock/v3/pkg/sdk"
)

var errHandlerBoom = errors.New("boom")

func TestUnaryReturnHeadersAndTrailers(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "meta").
		ReturnHeaders(map[string]string{"x-trace": "abc"}).
		ReturnTrailers(map[string]string{"x-cost": "7"}).
		Return("message", "hi")

	d := resolveDesc(t, fds, "test.HelloRequest", "test.HelloReply")
	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("name"), protoreflect.ValueOfString("meta"))
	out := dynamicpb.NewMessage(d.out)

	var header, trailer metadata.MD

	require.NoError(t, srv.Conn().Invoke(t.Context(), "/test.Greeter/SayHello", in, out,
		grpc.Header(&header), grpc.Trailer(&trailer)))

	require.Equal(t, []string{"abc"}, header.Get("x-trace"))
	require.Equal(t, []string{"7"}, trailer.Get("x-cost"))
}

func TestServerStreamReturnHeaders(t *testing.T) {
	t.Parallel()

	srv, fds := newServerSearch(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "hdr").
		ReturnHeaders(map[string]string{"x-stream": "on"}).
		SendStream(map[string]any{"id": "1", "title": "first"})

	stream := openSearchStream(t, srv, fds, "hdr")

	header, err := stream.Header()
	require.NoError(t, err)
	require.Equal(t, []string{"on"}, header.Get("x-stream"))
}

func TestWithIDPinsStubIdentifier(t *testing.T) {
	t.Parallel()

	srv, _ := newServer(t)
	defer func() { _ = srv.Close() }()

	id := uuid.New()
	e := srv.ExpectUnary("/test.Greeter/SayHello").
		WithID(id).
		Match("name", "pinned").
		Return("message", "hi")

	require.Equal(t, id.String(), e.StubID())
}

func TestUnaryReturnValueScalar(t *testing.T) {
	t.Parallel()

	srv, fds := newServerWKT(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/wkt.TypeService/GetTypes").
		Match("name", "scalar").
		ReturnValue(map[string]any{"message": "plain"})

	d := resolveDesc(t, fds, "wkt.TypeRequest", "wkt.TypeResponse")
	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("name"), protoreflect.ValueOfString("scalar"))
	out := dynamicpb.NewMessage(d.out)

	require.NoError(t, srv.Conn().Invoke(t.Context(), "/wkt.TypeService/GetTypes", in, out))
	require.Equal(t, "plain", out.Get(d.out.Fields().ByName("message")).String())
}

func TestUnaryReturnStatus(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "gone").
		ReturnStatus(codes.Unavailable)

	err := sayHelloErr(t, srv, fds, "gone")
	require.Error(t, err)
	require.Equal(t, codes.Unavailable, status.Code(err))
}

func TestServerStreamReturnErrorWithDetails(t *testing.T) {
	t.Parallel()

	srv, fds := newServerSearch(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "boom").
		ReturnErrorWithDetails(codes.ResourceExhausted, "quota exceeded")

	stream := openSearchStream(t, srv, fds, "boom")
	d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")

	err := stream.RecvMsg(dynamicpb.NewMessage(d.out))
	require.Error(t, err)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
}

func TestServerStreamStreamErrorElement(t *testing.T) {
	t.Parallel()

	srv, fds := newServerSearch(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "half").
		SendStream(
			map[string]any{"id": "1", "title": "first"},
			sdk.StreamError(codes.Aborted, "cut short"),
		)

	stream := openSearchStream(t, srv, fds, "half")
	d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")

	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, stream.RecvMsg(out))
	require.Equal(t, "first", out.Get(d.out.Fields().ByName("title")).String())

	err := stream.RecvMsg(dynamicpb.NewMessage(d.out))
	require.Error(t, err)
	require.Equal(t, codes.Aborted, status.Code(err))
}

func TestServerStreamUniformDelay(t *testing.T) {
	t.Parallel()

	srv, fds := newServerSearch(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "slow").
		Delay(80 * time.Millisecond).
		SendStream(map[string]any{"id": "1", "title": "late"})

	started := time.Now()
	stream := openSearchStream(t, srv, fds, "slow")
	d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")

	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, stream.RecvMsg(out))
	require.GreaterOrEqual(t, time.Since(started), 80*time.Millisecond)
}

func TestClientStreamMatchSequence(t *testing.T) {
	t.Parallel()

	srv, fds := newServerCalc(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectClientStream("/calc.Calculator/SumNumbers").
		MatchSequence(sdk.Equals("value", 1.0), sdk.Equals("value", 2.0)).
		Return("result", 3.0, "count", 2)

	result := sumNumbers(t, srv, fds, 1.0, 2.0)
	require.InDelta(t, 3.0, result, 0.0001)
}

func TestClientStreamReturnErrorWithDetails(t *testing.T) {
	t.Parallel()

	srv, fds := newServerCalc(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectClientStream("/calc.Calculator/SumNumbers").
		Match(sdk.Matches("value", ".*")).
		ReturnErrorWithDetails(codes.InvalidArgument, "bad input")

	_, err := sumNumbersErr(t, srv, fds, 5.0)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestClientStreamReturnHonoursDelay(t *testing.T) {
	t.Parallel()

	srv, fds := newServerCalc(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectClientStream("/calc.Calculator/SumNumbers").
		Match(sdk.Matches("value", ".*")).
		Return(sdk.Delay(80*time.Millisecond, "result", 5.0, "count", 1))

	started := time.Now()
	result := sumNumbers(t, srv, fds, 5.0)
	require.InDelta(t, 5.0, result, 0.0001)
	require.GreaterOrEqual(t, time.Since(started), 80*time.Millisecond)
}

func TestClientStreamOnceTwice(t *testing.T) {
	t.Parallel()

	srv, fds := newServerCalc(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectClientStream("/calc.Calculator/SumNumbers").
		Match(sdk.Matches("value", ".*")).
		Once().
		Return("result", 1.0, "count", 1)

	require.InDelta(t, 1.0, sumNumbers(t, srv, fds, 1.0), 0.0001)
	require.NoError(t, srv.ExpectationsWereMetContext(t.Context()))
}

func TestBidirectionalStaticSendStream(t *testing.T) {
	t.Parallel()

	srv, fds := newServerChat(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectBidirectionalStream("/chat.ChatService/Chat").
		Match("text", "ping").
		SendStream(map[string]any{"text": "pong"})

	reply, err := chatExchange(t, srv, fds, "ping")
	require.NoError(t, err)
	require.Equal(t, "pong", reply)
}

func TestStreamExpectationsAcceptSession(t *testing.T) {
	t.Parallel()

	srv, _ := newServerSearch(t)
	defer func() { _ = srv.Close() }()

	e := srv.ExpectServerStream("/search.SearchService/Search").
		Session("team-a").
		Match("query", "scoped")
	e.SendStream(map[string]any{"id": "1", "title": "scoped"})

	require.NotEmpty(t, e.StubID())
}

func TestUnaryRunExecutesHandler(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "handler").
		Run(func(_ context.Context, in any) (any, error) {
			req, ok := in.(map[string]any)
			require.True(t, ok)

			name, ok := req["name"].(string)
			require.True(t, ok)

			return map[string]any{"message": "hello " + name}, nil
		})

	require.Equal(t, "hello handler", getMsg(t, sayHello(t, srv, fds, "handler")))
}

func TestUnaryRunPropagatesStatus(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "fail").
		Run(func(_ context.Context, _ any) (any, error) {
			return nil, status.Error(codes.FailedPrecondition, "nope")
		})

	err := sayHelloErr(t, srv, fds, "fail")
	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func TestUnaryRunWrapsPlainError(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "plain").
		Run(func(_ context.Context, _ any) (any, error) {
			return nil, errHandlerBoom
		})

	err := sayHelloErr(t, srv, fds, "plain")
	require.Error(t, err)
	require.Equal(t, codes.Internal, status.Code(err))
}

func TestBidiScriptAnswersEveryDeclaredTurn(t *testing.T) {
	t.Parallel()

	srv, fds := newServerChat(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectBidirectionalStream("/chat.ChatService/Chat").
		MatchSequence(sdk.Equals("text", "one"), sdk.Equals("text", "two")).
		SendStream(
			map[string]any{"text": "first"},
			map[string]any{"text": "second"},
		)

	d := resolveDesc(t, fds, "chat.ChatMessage", "chat.ChatMessage")
	stream := openChatStream(t, srv)

	for i, sent := range []string{"one", "two"} {
		in := dynamicpb.NewMessage(d.in)

		in.Set(d.in.Fields().ByName("text"), protoreflect.ValueOfString(sent))
		require.NoError(t, stream.SendMsg(in))

		out := dynamicpb.NewMessage(d.out)
		require.NoError(t, stream.RecvMsg(out))
		require.Equal(t, []string{"first", "second"}[i], out.Get(d.out.Fields().ByName("text")).String())
	}
}

func TestBidiScriptExhaustionFailsLoudly(t *testing.T) {
	t.Parallel()

	srv, fds := newServerChat(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectBidirectionalStream("/chat.ChatService/Chat").
		Match(sdk.Matches("text", "^turn-")).
		SendStream(map[string]any{"text": "reply"})

	d := resolveDesc(t, fds, "chat.ChatMessage", "chat.ChatMessage")
	stream := openChatStream(t, srv)

	first := dynamicpb.NewMessage(d.in)

	first.Set(d.in.Fields().ByName("text"), protoreflect.ValueOfString("turn-1"))
	require.NoError(t, stream.SendMsg(first))

	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, stream.RecvMsg(out))
	require.Equal(t, "reply", out.Get(d.out.Fields().ByName("text")).String())

	second := dynamicpb.NewMessage(d.in)

	second.Set(d.in.Fields().ByName("text"), protoreflect.ValueOfString("turn-2"))
	require.NoError(t, stream.SendMsg(second))

	err := stream.RecvMsg(dynamicpb.NewMessage(d.out))
	require.Error(t, err, "an exhausted script must fail instead of leaving the client waiting")
	require.Equal(t, codes.NotFound, status.Code(err))
	require.Contains(t, status.Convert(err).Message(), "scripts 1 message(s)")
}

//nolint:ireturn // grpc.ClientStream is the only type NewStream can return.
func openChatStream(t *testing.T, srv *sdk.Server) grpc.ClientStream {
	t.Helper()

	stream, err := srv.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "Chat", ServerStreams: true, ClientStreams: true},
		"/chat.ChatService/Chat")
	require.NoError(t, err)

	return stream
}

func TestBidiMatchersBeyondEquals(t *testing.T) {
	t.Parallel()

	srv, fds := newServerChat(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectBidirectionalStream("/chat.ChatService/Chat").
		Match(sdk.Matches("text", "^pi")).
		SendStream(map[string]any{"text": "regex-pong"})

	reply, err := chatExchange(t, srv, fds, "ping")
	require.NoError(t, err)
	require.Equal(t, "regex-pong", reply)
}

func TestClientStreamSequenceMatchersBeyondEquals(t *testing.T) {
	t.Parallel()

	srv, fds := newServerCalc(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectClientStream("/calc.Calculator/SumNumbers").
		MatchSequence(sdk.Matches("value", "^1"), sdk.Contains("value", 2.0)).
		Return("result", 3.0, "count", 2)

	require.InDelta(t, 3.0, sumNumbers(t, srv, fds, 1.0, 2.0), 0.0001)
}

func TestWithIDDoesNotCollapseAChain(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		WithID(uuid.MustParse("22222222-2222-2222-2222-222222222222")).
		Match("name", "Chain").
		Return("message", "first").
		NextWillReturn("message", "second")

	require.Equal(t, "first", getMsg(t, sayHello(t, srv, fds, "Chain")))
	require.Equal(t, "second", getMsg(t, sayHello(t, srv, fds, "Chain")))
}

func TestServerStreamChainKeepsResponseHeaders(t *testing.T) {
	t.Parallel()

	srv, fds := newServerSearch(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "chain").
		ReturnHeaders(map[string]string{"x-chain": "yes"}).
		SendStream(map[string]any{"id": "1", "title": "first"}).
		NextWillReturn("id", "2", "title", "second")

	d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")

	for _, want := range []string{"first", "second"} {
		stream := openSearchStream(t, srv, fds, "chain")

		out := dynamicpb.NewMessage(d.out)
		require.NoError(t, stream.RecvMsg(out))
		require.Equal(t, want, out.Get(d.out.Fields().ByName("title")).String())

		header, err := stream.Header()
		require.NoError(t, err)
		require.Equal(t, []string{"yes"}, header.Get("x-chain"), "a chained stub keeps the declared metadata")
	}
}

func TestServerStreamRunKeepsMetadataEffectsAndHistory(t *testing.T) {
	t.Parallel()

	srv, fds := newServerSearch(t)
	defer func() { _ = srv.Close() }()

	unlock := sdk.Upsert("search.SearchService", "Search").
		Match("query", "unlocked").
		Return("id", "9", "title", "unlocked").
		Build()

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "handler").
		Once().
		ReturnHeaders(map[string]string{"x-handler": "yes"}).
		Effect(unlock).
		Run(func(_ context.Context, _ any, stream any) error {
			serverStream, ok := stream.(grpc.ServerStream)
			if !ok {
				return errNotServerStream
			}

			d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")
			out := dynamicpb.NewMessage(d.out)
			out.Set(d.out.Fields().ByName("title"), protoreflect.ValueOfString("from handler"))

			return serverStream.SendMsg(out)
		})

	stream := openSearchStream(t, srv, fds, "handler")
	d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")

	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, stream.RecvMsg(out))
	require.Equal(t, "from handler", out.Get(d.out.Fields().ByName("title")).String())
	require.Error(t, stream.RecvMsg(dynamicpb.NewMessage(d.out)))

	header, err := stream.Header()
	require.NoError(t, err)
	require.Equal(t, []string{"yes"}, header.Get("x-handler"))

	require.Equal(t, 1, srv.Called("/search.SearchService/Search"))
	require.NoError(t, srv.ExpectationsWereMetContext(t.Context()))

	unlocked := openSearchStream(t, srv, fds, "unlocked")
	unlockedOut := dynamicpb.NewMessage(d.out)
	require.NoError(t, unlocked.RecvMsg(unlockedOut))
	require.Equal(t, "unlocked", unlockedOut.Get(d.out.Fields().ByName("title")).String())
}

func TestServerStreamRunWritesStream(t *testing.T) {
	t.Parallel()

	srv, fds := newServerSearch(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "gen").
		Run(func(_ context.Context, _ any, stream any) error {
			ss, ok := stream.(grpc.ServerStream)
			if !ok {
				return errNotServerStream
			}

			d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")
			msg := dynamicpb.NewMessage(d.out)
			msg.Set(d.out.Fields().ByName("title"), protoreflect.ValueOfString("generated"))

			return ss.SendMsg(msg)
		})

	stream := openSearchStream(t, srv, fds, "gen")
	d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")

	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, stream.RecvMsg(out))
	require.Equal(t, "generated", out.Get(d.out.Fields().ByName("title")).String())
}

func TestClientStreamRunDrainsStream(t *testing.T) {
	t.Parallel()

	srv, fds := newServerCalc(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectClientStream("/calc.Calculator/SumNumbers").
		Match(sdk.Matches("value", ".*")).
		Run(func(_ context.Context, messages []any) (any, error) {
			total := 0.0

			for _, m := range messages {
				msg, ok := m.(map[string]any)
				if !ok {
					return nil, errNotServerStream
				}

				value, ok := msg["value"].(float64)
				if !ok {
					return nil, errHandlerBoom
				}

				total += value
			}

			return map[string]any{"result": total, "count": len(messages)}, nil
		})

	require.InDelta(t, 7.0, sumNumbers(t, srv, fds, 3.0, 4.0), 0.0001)
}

func TestEffectBuilderCarriesRicherStub(t *testing.T) {
	t.Parallel()

	srv, fds := newServerWorkflow(t)
	defer func() { _ = srv.Close() }()

	effect := sdk.Upsert("workflow.Workflow", "Next").
		MatchAny(sdk.Matches("step", "final.*")).
		Priority(50).
		ReturnHeaders(map[string]string{"x-effect": "yes"}).
		Return("status", "done").
		Build()

	srv.ExpectUnary("/workflow.Workflow/Start").
		Match("step", "begin").
		Effect(effect).
		Return("status", "started")

	d := resolveDesc(t, fds, "workflow.StartRequest", "workflow.StartResponse")
	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("step"), protoreflect.ValueOfString("begin"))
	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, srv.Conn().Invoke(t.Context(), "/workflow.Workflow/Start", in, out))

	nd := resolveDesc(t, fds, "workflow.NextRequest", "workflow.NextResponse")
	nin := dynamicpb.NewMessage(nd.in)
	nin.Set(nd.in.Fields().ByName("step"), protoreflect.ValueOfString("finalize"))
	nout := dynamicpb.NewMessage(nd.out)

	var header metadata.MD

	require.NoError(t, srv.Conn().Invoke(t.Context(), "/workflow.Workflow/Next", nin, nout,
		grpc.Header(&header)))
	require.Equal(t, "done", nout.Get(nd.out.Fields().ByName("status")).String())
	require.Equal(t, []string{"yes"}, header.Get("x-effect"))
}

func TestRunPanicsInRemoteMode(t *testing.T) {
	t.Parallel()

	fds := compileInline(t, testProto, "test.proto")
	grpcAddr, restURL := startGripmock(t, compileInline(t, testProto, "test.proto"))

	srv := sdk.NewTestServer(t, sdk.WithDescriptors(fds), sdk.WithRemote(grpcAddr, restURL))

	require.Panics(t, func() {
		srv.ExpectUnary("/test.Greeter/SayHello").
			Match("name", "x").
			Run(func(_ context.Context, _ any) (any, error) { return map[string]any{}, nil })
	})
}

//nolint:ireturn // grpc.ClientStream is the only type NewStream can return.
func openSearchStream(
	t *testing.T,
	srv *sdk.Server,
	fds *descriptorpb.FileDescriptorSet,
	query string,
) grpc.ClientStream {
	t.Helper()

	d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")
	stream, err := srv.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "Search", ServerStreams: true, ClientStreams: false},
		"/search.SearchService/Search")
	require.NoError(t, err)

	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("query"), protoreflect.ValueOfString(query))
	require.NoError(t, stream.SendMsg(in))

	return stream
}

func TestUnaryDelayHoldsTheReply(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	const pause = 80 * time.Millisecond

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "slow").
		Delay(pause).
		Return("message", "late")

	started := time.Now()

	require.Equal(t, "late", getMsg(t, sayHello(t, srv, fds, "slow")))
	require.GreaterOrEqual(t, time.Since(started), pause)
}

func TestClientStreamReturnValueScalar(t *testing.T) {
	t.Parallel()

	srv, fds := newServerCalc(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectClientStream("/calc.Calculator/SumNumbers").
		Match(sdk.Matches("value", ".*")).
		ReturnValue(map[string]any{"result": 7.5, "count": 1})

	require.InDelta(t, 7.5, sumNumbers(t, srv, fds, 7.5), 0.0001)
}

func TestClientStreamReturnJSON(t *testing.T) {
	t.Parallel()

	srv, fds := newServerCalc(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectClientStream("/calc.Calculator/SumNumbers").
		Match(sdk.Matches("value", ".*")).
		ReturnJSON(struct {
			Result float64 `json:"result"`
			Count  int     `json:"count"`
		}{Result: 4.5, Count: 2})

	require.InDelta(t, 4.5, sumNumbers(t, srv, fds, 1.5, 3.0), 0.0001)
}

func TestClientStreamReturnProto(t *testing.T) {
	t.Parallel()

	srv, fds := newServerCalc(t)
	defer func() { _ = srv.Close() }()

	d := resolveDesc(t, fds, "calc.NumberRequest", "calc.SumResponse")
	reply := dynamicpb.NewMessage(d.out)
	reply.Set(d.out.Fields().ByName("result"), protoreflect.ValueOfFloat64(9.5))

	srv.ExpectClientStream("/calc.Calculator/SumNumbers").
		Match(sdk.Matches("value", ".*")).
		ReturnProto(reply)

	require.InDelta(t, 9.5, sumNumbers(t, srv, fds, 9.5), 0.0001)
}

func TestClientStreamReturnStatus(t *testing.T) {
	t.Parallel()

	srv, fds := newServerCalc(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectClientStream("/calc.Calculator/SumNumbers").
		Match(sdk.Matches("value", ".*")).
		ReturnStatus(codes.ResourceExhausted)

	_, err := sumNumbersErr(t, srv, fds, 1.0)
	require.Error(t, err)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
	require.Empty(t, status.Convert(err).Message())
}

func TestClientStreamDelayHoldsTheReply(t *testing.T) {
	t.Parallel()

	srv, fds := newServerCalc(t)
	defer func() { _ = srv.Close() }()

	const pause = 80 * time.Millisecond

	srv.ExpectClientStream("/calc.Calculator/SumNumbers").
		Match(sdk.Matches("value", ".*")).
		Delay(pause).
		Return("result", 2.0, "count", 1)

	started := time.Now()

	require.InDelta(t, 2.0, sumNumbers(t, srv, fds, 2.0), 0.0001)
	require.GreaterOrEqual(t, time.Since(started), pause)
}

func TestServerStreamReturnStatus(t *testing.T) {
	t.Parallel()

	srv, fds := newServerSearch(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "gone").
		ReturnStatus(codes.Unavailable)

	stream := openSearchStream(t, srv, fds, "gone")
	d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")

	err := stream.RecvMsg(dynamicpb.NewMessage(d.out))
	require.Error(t, err)
	require.Equal(t, codes.Unavailable, status.Code(err))
	require.Empty(t, status.Convert(err).Message())
}

func TestBidirectionalReturnErrorWithDetails(t *testing.T) {
	t.Parallel()

	srv, fds := newServerChat(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectBidirectionalStream("/chat.ChatService/Chat").
		Match("text", "ping").
		ReturnErrorWithDetails(codes.FailedPrecondition, "room closed")

	_, err := chatExchange(t, srv, fds, "ping")
	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	require.Equal(t, "room closed", status.Convert(err).Message())
}

func TestBidirectionalReturnStatus(t *testing.T) {
	t.Parallel()

	srv, fds := newServerChat(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectBidirectionalStream("/chat.ChatService/Chat").
		Match("text", "whoami").
		ReturnStatus(codes.Unauthenticated)

	_, err := chatExchange(t, srv, fds, "whoami")
	require.Error(t, err)
	require.Equal(t, codes.Unauthenticated, status.Code(err))
	require.Empty(t, status.Convert(err).Message())
}

func TestBidirectionalDelayHoldsEveryMessage(t *testing.T) {
	t.Parallel()

	srv, fds := newServerChat(t)
	defer func() { _ = srv.Close() }()

	const pause = 70 * time.Millisecond

	srv.ExpectBidirectionalStream("/chat.ChatService/Chat").
		Match("text", "ping").
		Delay(pause).
		SendStream(map[string]any{"text": "pong"})

	started := time.Now()

	reply, err := chatExchange(t, srv, fds, "ping")
	require.NoError(t, err)
	require.Equal(t, "pong", reply)
	require.GreaterOrEqual(t, time.Since(started), pause)
}

func TestBidirectionalResponseMetadata(t *testing.T) {
	t.Parallel()

	srv, fds := newServerChat(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectBidirectionalStream("/chat.ChatService/Chat").
		Match("text", "ping").
		ReturnHeaders(map[string]string{"x-room": "lobby"}).
		SendStream(map[string]any{"text": "pong"})

	stream := chatStream(t, srv, fds, "ping")

	header, err := stream.Header()
	require.NoError(t, err)
	require.Equal(t, []string{"lobby"}, header.Get("x-room"))
}

func drainToEnd(t *testing.T, stream grpc.ClientStream, d msgDesc) {
	t.Helper()

	for {
		err := stream.RecvMsg(dynamicpb.NewMessage(d.out))
		if err != nil {
			return
		}
	}
}

//nolint:ireturn // grpc.ClientStream is the only type NewStream can return.
func chatStream(
	t *testing.T,
	srv *sdk.Server,
	fds *descriptorpb.FileDescriptorSet,
	text string,
) grpc.ClientStream {
	t.Helper()

	d := resolveDesc(t, fds, "chat.ChatMessage", "chat.ChatMessage")

	stream, err := srv.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "Chat", ServerStreams: true, ClientStreams: true},
		"/chat.ChatService/Chat")
	require.NoError(t, err)

	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("text"), protoreflect.ValueOfString(text))
	require.NoError(t, stream.SendMsg(in))
	require.NoError(t, stream.CloseSend())

	return stream
}

func chatExchange(
	t *testing.T,
	srv *sdk.Server,
	fds *descriptorpb.FileDescriptorSet,
	text string,
) (string, error) {
	t.Helper()

	d := resolveDesc(t, fds, "chat.ChatMessage", "chat.ChatMessage")
	stream := chatStream(t, srv, fds, text)

	out := dynamicpb.NewMessage(d.out)

	err := stream.RecvMsg(out)
	if err != nil {
		return "", err
	}

	return out.Get(d.out.Fields().ByName("text")).String(), nil
}

func sumNumbers(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet, values ...float64) float64 {
	t.Helper()

	result, err := sumNumbersErr(t, srv, fds, values...)
	require.NoError(t, err)

	return result
}

func sumNumbersErr(
	t *testing.T,
	srv *sdk.Server,
	fds *descriptorpb.FileDescriptorSet,
	values ...float64,
) (float64, error) {
	t.Helper()

	d := resolveDesc(t, fds, "calc.NumberRequest", "calc.SumResponse")
	stream, err := srv.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "SumNumbers", ServerStreams: false, ClientStreams: true},
		"/calc.Calculator/SumNumbers")
	require.NoError(t, err)

	for _, v := range values {
		msg := dynamicpb.NewMessage(d.in)
		msg.Set(d.in.Fields().ByName("value"), protoreflect.ValueOfFloat64(v))
		require.NoError(t, stream.SendMsg(msg))
	}

	require.NoError(t, stream.CloseSend())

	out := dynamicpb.NewMessage(d.out)

	err = stream.RecvMsg(out)
	if err != nil {
		return 0, err
	}

	return out.Get(d.out.Fields().ByName("result")).Float(), nil
}

func TestServerStreamBuilderSurface(t *testing.T) {
	t.Parallel()

	srv, fds := newServerSearch(t)
	defer func() { _ = srv.Close() }()

	id := uuid.New()
	e := srv.ExpectServerStream("/search.SearchService/Search").
		WithID(id).
		WithHeader(sdk.Equals("x-tenant", "acme")).
		ReturnTrailers(map[string]string{"x-done": "1"}).
		Priority(5).
		Once()
	e.SendStream(map[string]any{"id": "1", "title": "one"})

	require.Equal(t, id.String(), e.StubID())

	ctx := metadata.AppendToOutgoingContext(t.Context(), "x-tenant", "acme")
	d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")
	stream, err := srv.Conn().NewStream(ctx,
		&grpc.StreamDesc{StreamName: "Search", ServerStreams: true, ClientStreams: false},
		"/search.SearchService/Search")
	require.NoError(t, err)

	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("query"), protoreflect.ValueOfString("anything"))
	require.NoError(t, stream.SendMsg(in))

	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, stream.RecvMsg(out))
	require.Equal(t, "one", out.Get(d.out.Fields().ByName("title")).String())

	require.ErrorIs(t, stream.RecvMsg(dynamicpb.NewMessage(d.out)), io.EOF)
	require.Equal(t, []string{"1"}, stream.Trailer().Get("x-done"))
}

func TestServerStreamBuilderSendAndUpsert(t *testing.T) {
	t.Parallel()

	srv, fds := newServerSearch(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "grow").
		SendStream(map[string]any{"id": "1", "title": "first"}).
		Send("id", "2", "title", "second")

	stream := openSearchStream(t, srv, fds, "grow")
	d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")

	titles := make([]string, 0, 2)

	for range 2 {
		out := dynamicpb.NewMessage(d.out)
		require.NoError(t, stream.RecvMsg(out))
		titles = append(titles, out.Get(d.out.Fields().ByName("title")).String())
	}

	require.Equal(t, []string{"first", "second"}, titles)
}

func TestClientStreamBuilderSurface(t *testing.T) {
	t.Parallel()

	srv, fds := newServerCalc(t)
	defer func() { _ = srv.Close() }()

	id := uuid.New()
	e := srv.ExpectClientStream("/calc.Calculator/SumNumbers").
		WithID(id).
		WithHeader(sdk.Equals("x-tenant", "acme")).
		Match(sdk.Matches("value", ".*")).
		Session("").
		Priority(3).
		Once().
		ReturnHeaders(map[string]string{"x-kind": "sum"}).
		ReturnTrailers(map[string]string{"x-done": "1"}).
		Return("result", 9.0, "count", 1)

	require.Equal(t, id.String(), e.StubID())

	ctx := metadata.AppendToOutgoingContext(t.Context(), "x-tenant", "acme")
	d := resolveDesc(t, fds, "calc.NumberRequest", "calc.SumResponse")
	stream, err := srv.Conn().NewStream(ctx,
		&grpc.StreamDesc{StreamName: "SumNumbers", ServerStreams: false, ClientStreams: true},
		"/calc.Calculator/SumNumbers")
	require.NoError(t, err)

	msg := dynamicpb.NewMessage(d.in)
	msg.Set(d.in.Fields().ByName("value"), protoreflect.ValueOfFloat64(9.0))
	require.NoError(t, stream.SendMsg(msg))
	require.NoError(t, stream.CloseSend())

	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, stream.RecvMsg(out))
	require.InDelta(t, 9.0, out.Get(d.out.Fields().ByName("result")).Float(), 0.0001)

	header, err := stream.Header()
	require.NoError(t, err)
	require.Equal(t, []string{"sum"}, header.Get("x-kind"))
}

func TestBidirectionalBuilderSurface(t *testing.T) {
	t.Parallel()

	srv, fds := newServerChat(t)
	defer func() { _ = srv.Close() }()

	id := uuid.New()
	e := srv.ExpectBidirectionalStream("/chat.ChatService/Chat").
		WithID(id).
		WithHeader(sdk.Equals("x-tenant", "acme")).
		Session("").
		Priority(2).
		Once().
		ReturnHeaders(map[string]string{"x-chat": "on"}).
		ReturnTrailers(map[string]string{"x-done": "1"}).
		MatchSequence(sdk.Equals("text", "ping"))
	e.SendStream(map[string]any{"text": "pong"})

	require.Equal(t, id.String(), e.StubID())

	ctx := metadata.AppendToOutgoingContext(t.Context(), "x-tenant", "acme")
	d := resolveDesc(t, fds, "chat.ChatMessage", "chat.ChatMessage")
	stream, err := srv.Conn().NewStream(ctx,
		&grpc.StreamDesc{StreamName: "Chat", ServerStreams: true, ClientStreams: true},
		"/chat.ChatService/Chat")
	require.NoError(t, err)

	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("text"), protoreflect.ValueOfString("ping"))
	require.NoError(t, stream.SendMsg(in))
	require.NoError(t, stream.CloseSend())

	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, stream.RecvMsg(out))
	require.Equal(t, "pong", out.Get(d.out.Fields().ByName("text")).String())
}

func TestBidirectionalReturnErrorViaEffectBuilder(t *testing.T) {
	t.Parallel()

	srv, fds := newServerWorkflow(t)
	defer func() { _ = srv.Close() }()

	effectID := uuid.New()
	effect := sdk.Upsert("workflow.Workflow", "Next").
		WithID(effectID).
		Session("").
		Times(1).
		WithHeader(sdk.Contains("x-tenant", "acme")).
		Match("step", "boom").
		ReturnTrailers(map[string]string{"x-fail": "1"}).
		ReturnErrorWithDetails(codes.Aborted, "effect failed").
		Build()

	srv.ExpectUnary("/workflow.Workflow/Start").
		Match("step", "arm").
		Effect(effect).
		Return("status", "armed")

	d := resolveDesc(t, fds, "workflow.StartRequest", "workflow.StartResponse")
	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("step"), protoreflect.ValueOfString("arm"))
	require.NoError(t, srv.Conn().Invoke(t.Context(), "/workflow.Workflow/Start", in, dynamicpb.NewMessage(d.out)))

	ctx := metadata.AppendToOutgoingContext(t.Context(), "x-tenant", "acme")
	nd := resolveDesc(t, fds, "workflow.NextRequest", "workflow.NextResponse")
	nin := dynamicpb.NewMessage(nd.in)
	nin.Set(nd.in.Fields().ByName("step"), protoreflect.ValueOfString("boom"))

	err := srv.Conn().Invoke(ctx, "/workflow.Workflow/Next", nin, dynamicpb.NewMessage(nd.out))
	require.Error(t, err)
	require.Equal(t, codes.Aborted, status.Code(err))
}

func TestEffectBuilderSendStream(t *testing.T) {
	t.Parallel()

	srv, fds := newServerSearch(t)
	defer func() { _ = srv.Close() }()

	effect := sdk.Upsert("search.SearchService", "Search").
		Match("query", "armed").
		SendStream(map[string]any{"id": "1", "title": "from effect"}).
		Build()

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "arm").
		Effect(effect).
		SendStream(map[string]any{"id": "0", "title": "arming"})

	first := openSearchStream(t, srv, fds, "arm")
	d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")
	require.NoError(t, first.RecvMsg(dynamicpb.NewMessage(d.out)))

	second := openSearchStream(t, srv, fds, "armed")
	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, second.RecvMsg(out))
	require.Equal(t, "from effect", out.Get(d.out.Fields().ByName("title")).String())
}

func TestStreamErrorCarriesDetails(t *testing.T) {
	t.Parallel()

	srv, fds := newServerSearch(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "detailed").
		SendStream(sdk.StreamError(codes.DataLoss, "gone", map[string]any{
			"type":   "type.googleapis.com/google.rpc.ErrorInfo",
			"reason": "GONE",
			"domain": "sdk.test",
		}))

	stream := openSearchStream(t, srv, fds, "detailed")
	d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")

	err := stream.RecvMsg(dynamicpb.NewMessage(d.out))
	require.Error(t, err)
	require.Equal(t, codes.DataLoss, status.Code(err))
	require.NotEmpty(t, status.Convert(err).Details())
}

func TestWithBatchFlushesOnClose(t *testing.T) {
	t.Parallel()

	fds := compileInline(t, testProto, "test.proto")
	grpcAddr, restURL := startGripmock(t, compileInline(t, testProto, "test.proto"))

	srv := sdk.NewTestServer(t,
		sdk.WithDescriptors(fds),
		sdk.WithRemote(grpcAddr, restURL),
		sdk.WithBatch(),
	)

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "batched").
		Return("message", "queued")

	require.NoError(t, srv.Flush())
	require.Equal(t, "queued", getMsg(t, sayHello(t, srv, fds, "batched")))
}

func TestUnarySessionAndErrorDetails(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Session("").
		Match("name", "detailed").
		ReturnErrorWithDetails(codes.FailedPrecondition, "nope", map[string]any{
			"type":   "type.googleapis.com/google.rpc.ErrorInfo",
			"reason": "NOPE",
			"domain": "sdk.test",
		})

	err := sayHelloErr(t, srv, fds, "detailed")
	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	require.NotEmpty(t, status.Convert(err).Details())
}

func TestServerStreamReturnErrorDelegates(t *testing.T) {
	t.Parallel()

	srv, fds := newServerSearch(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "plain-err").
		ReturnError(codes.Unimplemented, "not yet")

	stream := openSearchStream(t, srv, fds, "plain-err")
	d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")

	err := stream.RecvMsg(dynamicpb.NewMessage(d.out))
	require.Error(t, err)
	require.Equal(t, codes.Unimplemented, status.Code(err))
}

func TestClientStreamReturnErrorDelegates(t *testing.T) {
	t.Parallel()

	srv, fds := newServerCalc(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectClientStream("/calc.Calculator/SumNumbers").
		Match(sdk.Matches("value", ".*")).
		ReturnError(codes.Unimplemented, "not yet")

	_, err := sumNumbersErr(t, srv, fds, 1.0)
	require.Error(t, err)
	require.Equal(t, codes.Unimplemented, status.Code(err))
}

func TestServerStreamTwiceBudget(t *testing.T) {
	t.Parallel()

	srv, fds := newServerSearch(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "twice").
		Twice().
		SendStream(map[string]any{"id": "1", "title": "x"})

	d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")
	for range 2 {
		require.NoError(t, openSearchStream(t, srv, fds, "twice").RecvMsg(dynamicpb.NewMessage(d.out)))
	}

	require.NoError(t, srv.ExpectationsWereMetContext(t.Context()))
}

func TestClientStreamTwiceBudget(t *testing.T) {
	t.Parallel()

	srv, fds := newServerCalc(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectClientStream("/calc.Calculator/SumNumbers").
		Match(sdk.Matches("value", ".*")).
		Twice().
		Return("result", 1.0, "count", 1)

	for range 2 {
		require.InDelta(t, 1.0, sumNumbers(t, srv, fds, 1.0), 0.0001)
	}

	require.NoError(t, srv.ExpectationsWereMetContext(t.Context()))
}

func TestBidirectionalTwiceBudget(t *testing.T) {
	t.Parallel()

	srv, fds := newServerChat(t)
	defer func() { _ = srv.Close() }()

	e := srv.ExpectBidirectionalStream("/chat.ChatService/Chat").
		Match("text", "ping").
		Twice()
	e.SendStream(map[string]any{"text": "pong"})

	d := resolveDesc(t, fds, "chat.ChatMessage", "chat.ChatMessage")
	for range 2 {
		stream, err := srv.Conn().NewStream(t.Context(),
			&grpc.StreamDesc{StreamName: "Chat", ServerStreams: true, ClientStreams: true},
			"/chat.ChatService/Chat")
		require.NoError(t, err)

		in := dynamicpb.NewMessage(d.in)
		in.Set(d.in.Fields().ByName("text"), protoreflect.ValueOfString("ping"))
		require.NoError(t, stream.SendMsg(in))
		require.NoError(t, stream.CloseSend())
		require.NoError(t, stream.RecvMsg(dynamicpb.NewMessage(d.out)))

		drainToEnd(t, stream, d)
	}

	require.NoError(t, srv.ExpectationsWereMetContext(t.Context()))
}

func TestSendScalarElement(t *testing.T) {
	t.Parallel()

	srv, fds := newServerSearch(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "scalar-send").
		SendStream(map[string]any{"id": "1", "title": "first"}).
		Send(map[string]any{"id": "2", "title": "second"})

	stream := openSearchStream(t, srv, fds, "scalar-send")
	d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")

	for _, want := range []string{"first", "second"} {
		out := dynamicpb.NewMessage(d.out)
		require.NoError(t, stream.RecvMsg(out))
		require.Equal(t, want, out.Get(d.out.Fields().ByName("title")).String())
	}
}

func TestExpectationNotMetErrorMessage(t *testing.T) {
	t.Parallel()

	srv, _ := newServer(t)

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "never").
		Twice().
		Return("message", "hi")

	err := srv.ExpectationsWereMetContext(t.Context())
	require.Error(t, err)
	require.ErrorIs(t, err, sdk.ErrVerificationFailed)
	require.Contains(t, err.Error(), "expected 2 call(s), got 0")

	srv.Reset()
}

func TestConfigAfterTerminalPanicsOnEveryShape(t *testing.T) {
	t.Parallel()

	srv, _ := newServer(t)
	defer func() { _ = srv.Close() }()

	id := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	headers := map[string]string{"x": "y"}

	unary := func() *sdk.UnaryExpectation {
		return srv.ExpectUnary("/test.Greeter/SayHello").Match("name", "x").Return("message", "x")
	}

	stream := func() *sdk.ServerStreamExpectation {
		e := srv.ExpectServerStream("/test.Greeter/SayHello").Match("name", "x")
		e.SendStream(map[string]any{"message": "x"})

		return e
	}

	client := func() *sdk.ClientStreamExpectation {
		return srv.ExpectClientStream("/test.Greeter/SayHello").Match("name", "x").Return("message", "x")
	}

	bidi := func() *sdk.BidirectionalExpectation {
		return srv.ExpectBidirectionalStream("/test.Greeter/SayHello").
			Match("name", "x").
			SendStream(map[string]any{"message": "x"})
	}

	cases := map[string]func(){
		"unary Times":           func() { unary().Times(2) },
		"unary Session":         func() { unary().Session("s") },
		"unary WithID":          func() { unary().WithID(id) },
		"unary ReturnHeaders":   func() { unary().ReturnHeaders(headers) },
		"unary Delay":           func() { unary().Delay(time.Millisecond) },
		"stream Priority":       func() { stream().Priority(2) },
		"stream Session":        func() { stream().Session("s") },
		"stream ReturnTrailers": func() { stream().ReturnTrailers(headers) },
		"client Match":          func() { client().Match("name", "y") },
		"client Delay":          func() { client().Delay(time.Millisecond) },
		"client WithID":         func() { client().WithID(id) },
		"bidi Times":            func() { bidi().Times(2) },
		"bidi MatchSequence":    func() { bidi().MatchSequence(sdk.Equals("name", "y")) },
		"bidi ReturnHeaders":    func() { bidi().ReturnHeaders(headers) },
	}

	for name, mutate := range cases {
		require.Panicsf(t, mutate, "%s must panic after the terminal method", name)
	}

	srv.Reset()
}

func TestStaticStubDataStaysImmutableAcrossCalls(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "static").
		Return("message", "same every time")

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match(sdk.Matches("name", "^user-")).
		Return("message", "Hello {{.Request.name}}")

	for _, name := range []string{"user-1", "user-2", "user-3"} {
		require.Equal(t, "same every time", getMsg(t, sayHello(t, srv, fds, "static")))
		require.Equal(t, "Hello "+name, getMsg(t, sayHello(t, srv, fds, name)),
			"each call must render from its own request, not from a mutated stub")
	}

	history := srv.History()
	require.Len(t, history, 6)

	for _, record := range history {
		response, _ := record.Responses[0].(map[string]any)
		msg, _ := response["message"].(string)
		require.NotContains(t, msg, "{{", "history must hold the rendered response")
	}
}

func TestConcurrentCallsAndHistoryReads(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "static").
		Return("message", "same every time")

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match(sdk.Matches("name", "^worker-")).
		Return("message", "Hello {{.Request.name}}")

	d := resolveDesc(t, fds, "test.HelloRequest", "test.HelloReply")
	conn := srv.Conn()

	var wg sync.WaitGroup

	results := make([]string, 8)

	wg.Add(len(results))

	for i := range results {
		go func() {
			defer wg.Done()

			results[i] = runCallerLoop(t.Context(), conn, d, "worker-"+strconv.Itoa(i))
		}()
	}

	readers := make(chan struct{})

	go func() {
		defer close(readers)

		for range 50 {
			for _, record := range srv.History() {
				_ = record.Responses
			}

			_ = srv.TotalCalls()
		}
	}()

	wg.Wait()
	<-readers

	for i, result := range results {
		require.Equalf(t, "ok", result, "worker %d", i)
	}

	require.Equal(t, 320, srv.TotalCalls())
}

func runCallerLoop(ctx context.Context, conn *grpc.ClientConn, d msgDesc, name string) string {
	for range 20 {
		reply, err := invokeHello(ctx, conn, d, name)
		if err != nil {
			return err.Error()
		}

		if reply != "Hello "+name {
			return "got " + reply
		}

		_, err = invokeHello(ctx, conn, d, "static")
		if err != nil {
			return err.Error()
		}
	}

	return "ok"
}

func invokeHello(ctx context.Context, conn *grpc.ClientConn, d msgDesc, name string) (string, error) {
	in := dynamicpb.NewMessage(d.in)

	in.Set(d.in.Fields().ByName("name"), protoreflect.ValueOfString(name))

	out := dynamicpb.NewMessage(d.out)

	err := conn.Invoke(ctx, "/test.Greeter/SayHello", in, out)
	if err != nil {
		return "", err
	}

	return out.Get(d.out.Fields().ByName("message")).String(), nil
}
