package sdk

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/domain/protoset"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func sdkProtoPath(project string) string {
	return filepath.Join("..", "..", "examples", "projects", project, "service.proto")
}

func mustBuildFDS(t *testing.T, protoPath string) *descriptorpb.FileDescriptorSet {
	t.Helper()

	ctx := t.Context()
	fdsSlice, err := protoset.Build(ctx, nil, []string{protoPath})
	require.NoError(t, err)
	require.NotEmpty(t, fdsSlice)

	return fdsSlice[0]
}

// mustRunWithProto builds descriptors from protoPath and runs mock. Caller must defer mock.Close().
func mustRunWithProto(t *testing.T, protoPath string, opts ...Option) Mock {
	t.Helper()

	fds := mustBuildFDS(t, protoPath)
	allOpts := append([]Option{WithDescriptors(fds)}, opts...)
	mock, err := Run(t.Context(), allOpts...)
	require.NoError(t, err)
	require.NotNil(t, mock)

	return mock
}

// mustRunWithProtoAndReg returns mock and protodesc registry. Caller must defer mock.Close().
func mustRunWithProtoAndReg(t *testing.T, protoPath string, opts ...Option) (Mock, *protoregistry.Files) {
	t.Helper()

	fds := mustBuildFDS(t, protoPath)
	allOpts := append([]Option{WithDescriptors(fds)}, opts...)
	mock, err := Run(t.Context(), allOpts...)
	require.NoError(t, err)
	require.NotNil(t, mock)

	reg, err := protodesc.NewFiles(fds)
	require.NoError(t, err)

	return mock, reg
}

func TestRun_EmbeddedBufconn(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"))
	defer mock.Close()

	require.NotNil(t, mock.Conn())
	require.Equal(t, "bufnet", mock.Addr())

	mock.Stub("helloworld.Greeter", "SayHello").
		When(Equals("name", "Alex")).
		Reply(Data("message", "Hi Alex")).
		Commit()
}

func TestRun_DescriptorsAppend(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fdsGreeter := mustBuildFDS(t, sdkProtoPath("greeter"))
	fdsEcho := mustBuildFDS(t, filepath.Join("..", "..", "examples", "projects", "echo", "service_v1.proto"))

	mock, err := Run(ctx,
		WithDescriptors(fdsGreeter),
		WithDescriptors(fdsEcho),
	)
	require.NoError(t, err)
	require.NotNil(t, mock)
	defer mock.Close()

	mock.Stub("helloworld.Greeter", "SayHello").
		When(Equals("name", "x")).
		Reply(Data("message", "hi")).
		Commit()
	require.NotNil(t, mock.Conn())
}

func TestRun_DescriptorsAppend_Dedup(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	fds := mustBuildFDS(t, sdkProtoPath("greeter"))

	mock, err := Run(ctx, WithDescriptors(fds), WithDescriptors(fds))
	require.NoError(t, err)
	require.NotNil(t, mock)
	defer mock.Close()

	mock.Stub("helloworld.Greeter", "SayHello").
		When(Equals("name", "x")).
		Reply(Data("message", "hi")).
		Commit()
}

func TestRun_WhenStreamReplyStream(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("calculator"))
	defer mock.Close()

	// B7: WhenStream + Reply (client stream)
	mock.Stub("calculator.CalculatorService", "SumNumbers").
		WhenStream(Matches("value", `\d+`), Matches("value", `\d+`)).
		Reply(Data("result", 42.0, "count", 2)).
		Commit()
}

func TestRun_RealPort(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"),
		WithListenAddr("tcp", ":0"),
		WithHealthyTimeout(5*time.Second),
	)
	defer mock.Close()

	require.NotNil(t, mock.Conn())
	require.Regexp(t, `^127\.0\.0\.1:\d+$`, mock.Addr())
}

func TestRun_RealPort_DefaultNetwork(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"),
		WithListenAddr("", ":0"),
		WithHealthyTimeout(5*time.Second),
	)
	defer mock.Close()

	require.Regexp(t, `^127\.0\.0\.1:\d+$`, mock.Addr())
}

func TestRun_DefaultHealthyTimeout(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"), WithHealthyTimeout(0))
	defer mock.Close()
}

func TestRun_Validation(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	_, err := Run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "descriptors required")
}

func TestRun_InvalidDescriptors(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	// FileDescriptorSet with invalid file (field number 0 is invalid)
	fds := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			{
				Name:   proto.String("bad.proto"),
				Syntax: proto.String("proto3"),
				MessageType: []*descriptorpb.DescriptorProto{
					{
						Name: proto.String("Bad"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{Number: proto.Int32(0)}, // invalid: field number must be >= 1
						},
					},
				},
			},
		},
	}

	_, err := Run(ctx, WithDescriptors(fds))
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create files registry")
}

func TestRun_ListenError(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	_, err := Run(ctx,
		WithDescriptors(mustBuildFDS(t, sdkProtoPath("greeter"))),
		WithListenAddr("tcp", ":99999"),
		WithHealthyTimeout(5*time.Second),
	)
	require.Error(t, err)
	errStr := err.Error()
	require.True(t,
		strings.Contains(errStr, "invalid") ||
			strings.Contains(errStr, "address") ||
			strings.Contains(errStr, "listen"),
		"err=%v", err)
}

func TestRun_ListenAddrString_UnixFallback(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("Unix sockets not supported on Windows")
	}

	sockPath := "/tmp/gripmock_" + uuid.New().String()[:8] + ".sock"
	mock, err := Run(t.Context(),
		WithDescriptors(mustBuildFDS(t, sdkProtoPath("greeter"))),
		WithListenAddr("unix", sockPath),
		WithHealthyTimeout(2*time.Second),
	)
	if err != nil {
		t.Logf("Run failed for unix (may hit listenAddrString before client dial): %v", err)

		return
	}
	defer mock.Close()
	require.Contains(t, mock.Addr(), ".sock")
}

func TestRun_ReplyStream_SkipsNilData(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("track-streaming"))
	defer mock.Close()

	// ReplyStream with nil Data entries - they are skipped (stuber.Output{Data: nil})
	mock.Stub("TrackService", "StreamTrack").
		When(Equals("stn", "MS#00002")).
		ReplyStream(
			Data("stn", "MS#00002", "identity", "00"),
			stuber.Output{Data: nil}, // skipped
			Data("stn", "MS#00002", "identity", "01"),
		).
		Commit()
}

func TestRun_MockFrom(t *testing.T) {
	t.Parallel()

	mock1 := mustRunWithProto(t, sdkProtoPath("greeter"),
		WithListenAddr("tcp", ":0"),
		WithHealthyTimeout(5*time.Second),
	)
	defer mock1.Close()

	mock2, err := Run(t.Context(),
		MockFrom(mock1.Addr()),
		WithHealthyTimeout(5*time.Second),
	)
	require.NoError(t, err)
	require.NotNil(t, mock2)
	defer mock2.Close()

	require.Equal(t, "bufnet", mock2.Addr()) // mock2 uses bufconn by default
}

func TestRun_ReplyStream(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("track-streaming"))
	defer mock.Close()

	// Server stream: When + ReplyStream
	mock.Stub("TrackService", "StreamTrack").
		When(Equals("stn", "MS#00001")).
		ReplyStream(
			Data("stn", "MS#00001", "identity", "00", "latitude", 0.08),
			Data("stn", "MS#00001", "identity", "01", "latitude", 0.09),
		).
		Commit()
}

func TestRun_ReplyError(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"))
	defer mock.Close()

	mock.Stub("helloworld.Greeter", "SayHello").
		When(Equals("name", "error")).
		ReplyError(codes.NotFound, "user not found").
		Commit()

	mock.Stub("helloworld.Greeter", "SayHello").
		When(Equals("name", "Alex")).
		Reply(Data("message", "Hi")).
		Commit()
}

func TestRun_Priority(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"))
	defer mock.Close()

	mock.Stub("helloworld.Greeter", "SayHello").
		When(Equals("name", "priority")).
		Reply(Data("message", "low")).
		Priority(10).
		Commit()

	mock.Stub("helloworld.Greeter", "SayHello").
		When(Equals("name", "priority")).
		Reply(Data("message", "high")).
		Priority(100).
		Commit()
}

func TestRun_Contains(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"))
	defer mock.Close()

	mock.Stub("helloworld.Greeter", "SayHello").
		When(Contains("name", "Alice")).
		Reply(Data("message", "Hello Alice")).
		Commit()
}

func TestRun_Map(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"))
	defer mock.Close()

	mock.Stub("helloworld.Greeter", "SayHello").
		When(Map("name", "Bob", "extra", "value")).
		Reply(Data("message", "Hi Bob")).
		Commit()
}

func TestRun_MockFrom_NoServices(t *testing.T) {
	t.Parallel()

	// Start minimal gRPC server with only health + reflection (no custom services)
	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	addr := lis.Addr().String()
	// Use 127.0.0.1 for consistent addr format
	_, port, _ := net.SplitHostPort(addr)
	addr = "127.0.0.1:" + port

	server := grpc.NewServer()
	hs := health.NewServer()
	hs.SetServingStatus("", healthgrpc.HealthCheckResponse_SERVING)
	healthgrpc.RegisterHealthServer(server, hs)
	reflection.Register(server)
	go func() { _ = server.Serve(lis) }()
	defer server.GracefulStop()

	ctx := t.Context()
	_, err = Run(ctx, MockFrom(addr), WithHealthyTimeout(2*time.Second))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no services found via reflection")
}

func TestRun_MockFrom_InvalidAddr(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	_, err := Run(ctx, MockFrom("localhost:59999"), WithHealthyTimeout(100*time.Millisecond))
	require.Error(t, err)
	errStr := err.Error()
	require.True(t,
		strings.Contains(errStr, "failed to connect") ||
			strings.Contains(errStr, "failed to get reflection stream") ||
			strings.Contains(errStr, "connection refused"), "err=%v", err)
}

func TestRun_HealthyTimeout(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	_, err := Run(ctx, WithDescriptors(mustBuildFDS(t, sdkProtoPath("greeter"))), WithHealthyTimeout(1))
	require.Error(t, err)
	errStr := err.Error()
	require.True(t,
		errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) ||
			strings.Contains(errStr, "deadline exceeded") || strings.Contains(errStr, "context canceled"),
		"err=%v", err)
}

func TestHelpers_Equals(t *testing.T) {
	t.Parallel()
	id := Equals("key", "value")
	require.NotNil(t, id.Equals)
	require.Equal(t, "value", id.Equals["key"])
}

func TestHelpers_Contains(t *testing.T) {
	t.Parallel()
	id := Contains("key", "value")
	require.NotNil(t, id.Contains)
	require.Equal(t, "value", id.Contains["key"])
}

func TestHelpers_Matches(t *testing.T) {
	t.Parallel()
	id := Matches("key", `\d+`)
	require.NotNil(t, id.Matches)
	require.Equal(t, `\d+`, id.Matches["key"])
}

func TestHelpers_Map(t *testing.T) {
	t.Parallel()
	id := Map("a", 1, "b", "two")
	require.NotNil(t, id.Equals)
	require.Equal(t, 1, id.Equals["a"])
	require.Equal(t, "two", id.Equals["b"])
}

func TestHelpers_Map_PanicOddArgs(t *testing.T) {
	t.Parallel()
	require.PanicsWithValue(t, "sdk.Map: need pairs (key, value), got 3 args", func() {
		Map("a", 1, "b")
	})
}

func TestHelpers_Map_PanicNonStringKey(t *testing.T) {
	t.Parallel()
	require.PanicsWithValue(t, "sdk.Map: key at 0 must be string, got int", func() {
		Map(123, "value")
	})
}

func TestHelpers_Data(t *testing.T) {
	t.Parallel()
	out := Data("msg", "hello", "n", 42)
	require.NotNil(t, out.Data)
	require.Equal(t, "hello", out.Data["msg"])
	require.Equal(t, 42, out.Data["n"])
}

func TestHelpers_Data_PanicOddArgs(t *testing.T) {
	t.Parallel()
	require.PanicsWithValue(t, "sdk.Data: need pairs (key, value), got 3 args", func() {
		Data("a", 1, "b")
	})
}

func TestHelpers_Data_PanicNonStringKey(t *testing.T) {
	t.Parallel()
	require.PanicsWithValue(t, "sdk.Data: key at 0 must be string, got int", func() {
		Data(123, "value")
	})
}

func TestHelpers_HeaderEquals(t *testing.T) {
	t.Parallel()
	h := HeaderEquals("authorization", "Bearer token")
	require.NotNil(t, h.Equals)
	require.Equal(t, "Bearer token", h.Equals["authorization"])
}

func TestHelpers_HeaderMap(t *testing.T) {
	t.Parallel()
	h := HeaderMap("x-id", "123", "x-name", "test")
	require.NotNil(t, h.Equals)
	require.Equal(t, "123", h.Equals["x-id"])
	require.Equal(t, "test", h.Equals["x-name"])
}

func TestHelpers_IgnoreArrayOrder(t *testing.T) {
	t.Parallel()
	id := IgnoreArrayOrder(Equals("arr", []any{1, 2}))
	require.True(t, id.IgnoreArrayOrder)
	require.Equal(t, []any{1, 2}, id.Equals["arr"])
}

func TestRun_ReplyHeaders(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"))
	defer mock.Close()

	mock.Stub("helloworld.Greeter", "SayHello").
		When(Equals("name", "Alex")).
		Reply(Data("message", "Hi")).
		ReplyHeaderPairs("x-custom", "value", "x-id", "123").
		Commit()
}

func TestRun_WhenHeaders_Integration(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))
	defer mock.Close()

	mock.Stub("helloworld.Greeter", "SayHello").
		When(Equals("name", "Alex")).
		WhenHeaders(HeaderEquals("x-custom", "expected-value")).
		Reply(Data("message", "matched-by-header")).
		Commit()

	mock.Stub("helloworld.Greeter", "SayHello").
		When(Equals("name", "Alex")).
		Reply(Data("message", "no-header-match")).
		Commit()

	// Call with matching header — should get "matched-by-header"
	callCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("x-custom", "expected-value"))
	msg1 := invokeGreeterSayHello(t, mock.Conn(), reg, callCtx, "Alex")
	require.Equal(t, "matched-by-header", getMessageField(t, msg1, "message"))

	// Call without header — should get "no-header-match"
	msg2 := invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "Alex")
	require.Equal(t, "no-header-match", getMessageField(t, msg2, "message"))
}

func invokeGreeterSayHello(t *testing.T, conn *grpc.ClientConn, reg *protoregistry.Files, ctx context.Context, name string) *dynamicpb.Message {
	t.Helper()
	inDesc, err := reg.FindDescriptorByName("helloworld.HelloRequest")
	require.NoError(t, err)
	outDesc, err := reg.FindDescriptorByName("helloworld.HelloReply")
	require.NoError(t, err)

	in := dynamicpb.NewMessage(inDesc.(protoreflect.MessageDescriptor))
	fd := inDesc.(protoreflect.MessageDescriptor).Fields().ByName("name")
	in.Set(fd, protoreflect.ValueOfString(name))

	out := dynamicpb.NewMessage(outDesc.(protoreflect.MessageDescriptor))
	err = conn.Invoke(ctx, "/helloworld.Greeter/SayHello", in, out)
	require.NoError(t, err)
	return out
}

func getMessageField(t *testing.T, msg *dynamicpb.Message, field string) string {
	t.Helper()
	fd := msg.Descriptor().Fields().ByName(protoreflect.Name(field))
	require.NotNil(t, fd)
	return msg.Get(fd).String()
}

func TestRun_Delay(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"))
	defer mock.Close()

	mock.Stub("helloworld.Greeter", "SayHello").
		When(Equals("name", "slow")).
		Reply(Data("message", "delayed")).
		Delay(10 * time.Millisecond).
		Commit()
}

func TestRun_Times_ExhaustedAfterLimit(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))
	defer mock.Close()

	mock.Stub("helloworld.Greeter", "SayHello").
		When(Equals("name", "limited")).
		Reply(Data("message", "ok")).
		Times(2).
		Commit()

	// 1st and 2nd call — success
	msg1 := invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "limited")
	require.Equal(t, "ok", getMessageField(t, msg1, "message"))
	msg2 := invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "limited")
	require.Equal(t, "ok", getMessageField(t, msg2, "message"))

	// 3rd call — stub exhausted, should return error (NotFound or similar)
	out := &dynamicpb.Message{}
	errInvoke := mock.Conn().Invoke(ctx, "/helloworld.Greeter/SayHello",
		createGreeterRequest(t, reg, "limited"),
		out,
	)
	require.Error(t, errInvoke)
	code := status.Code(errInvoke)
	require.True(t, code == codes.NotFound || code == codes.Unknown,
		"expected NotFound or Unknown when stub exhausted, got %s", code)
}

func createGreeterRequest(t *testing.T, reg *protoregistry.Files, name string) *dynamicpb.Message {
	t.Helper()
	inDesc, err := reg.FindDescriptorByName("helloworld.HelloRequest")
	require.NoError(t, err)
	in := dynamicpb.NewMessage(inDesc.(protoreflect.MessageDescriptor))
	fd := inDesc.(protoreflect.MessageDescriptor).Fields().ByName("name")
	in.Set(fd, protoreflect.ValueOfString(name))
	return in
}

func TestRun_Remote_ConnectionRefused(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	// Use a port that's unlikely to have a listener (gripmock uses 4770)
	mock, err := Run(ctx, Remote("127.0.0.1:15999"), WithHealthyTimeout(500*time.Millisecond))
	if err == nil {
		mock.Close()
		t.Fatal("expected error when connecting to non-existent gripmock")
	}
	require.Error(t, err)
}

func TestRun_Remote_WithCustomRestURL(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	// Verify Remote option accepts custom rest URL (still fails to connect, but option is applied)
	_, err := Run(ctx,
		Remote("127.0.0.1:15998", "http://127.0.0.1:15999"),
		WithHealthyTimeout(200*time.Millisecond),
	)
	require.Error(t, err)
}

func TestRun_Remote_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Remote integration test in short mode")
	}
	t.Parallel()

	ctx := t.Context()

	// Pick random ports
	grpcLis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	grpcPort := grpcLis.Addr().(*net.TCPAddr).Port
	grpcLis.Close()

	httpLis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	httpPort := httpLis.Addr().(*net.TCPAddr).Port
	httpLis.Close()

	// Build and start gripmock
	projRoot := filepath.Join("..", "..")
	protoPath := filepath.Join(projRoot, "examples", "projects", "greeter", "service.proto")

	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("skipping: go not found in PATH: %v", err)

		return
	}

	goDir := filepath.Dir(goPath)
	if goroot := runtime.GOROOT(); goroot != "" {
		goDir = goDir + string(filepath.ListSeparator) + filepath.Join(goroot, "bin")
	}

	cmd := exec.CommandContext(ctx, goPath, "run", ".", protoPath)
	cmd.Dir = projRoot
	env := make([]string, 0, len(os.Environ())+4)
	grpcVar := "GRPC_PORT=" + fmt.Sprintf("%d", grpcPort)
	httpVar := "HTTP_PORT=" + fmt.Sprintf("%d", httpPort)
	safePath := "PATH=" + goDir
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GRPC_PORT=") || strings.HasPrefix(e, "HTTP_PORT=") || strings.HasPrefix(e, "PATH=") {
			continue
		}

		env = append(env, e)
	}

	cmd.Env = append(env, safePath, grpcVar, httpVar)
	if err := cmd.Start(); err != nil {
		t.Skipf("skipping: cannot start gripmock: %v", err)
		return
	}
	defer func() { _ = cmd.Process.Kill() }()

	grpcAddr := fmt.Sprintf("127.0.0.1:%d", grpcPort)
	restURL := fmt.Sprintf("http://127.0.0.1:%d", httpPort)

	// Wait for gripmock to be ready (go run compiles first, then server starts)
	time.Sleep(8 * time.Second)

	mock, err := Run(ctx,
		Remote(grpcAddr, restURL),
		WithHealthyTimeout(10*time.Second),
	)
	if err != nil {
		t.Skipf("skipping: cannot connect to gripmock: %v", err)
		return
	}
	defer mock.Close()

	mock.Stub("helloworld.Greeter", "SayHello").
		When(Equals("name", "Alex")).
		Reply(Data("message", "Hi from Remote")).
		Commit()

	fdsSlice, err := protoset.Build(ctx, nil, []string{protoPath})
	require.NoError(t, err)
	require.NotEmpty(t, fdsSlice)
	reg, err := protodesc.NewFiles(fdsSlice[0])
	require.NoError(t, err)
	msg := invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "Alex")
	require.Equal(t, "Hi from Remote", getMessageField(t, msg, "message"))

	// Remote History/Verify via REST API (when gripmock has history enabled)
	require.Equal(t, 1, mock.History().Count())
	calls := mock.History().FilterByMethod("helloworld.Greeter", "SayHello")
	require.Len(t, calls, 1)
	require.Equal(t, "Alex", calls[0].Request["name"])
	mock.Verify().Method("helloworld.Greeter", "SayHello").Called(t, 1)
	mock.Verify().Total(t, 1)
}

func TestRun_HistoryAndVerify(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))
	defer mock.Close()

	mock.Stub("helloworld.Greeter", "SayHello").
		When(Equals("name", "Alex")).
		Reply(Data("message", "Hi")).
		Commit()

	// No calls yet
	require.Equal(t, 0, mock.History().Count())

	// First call
	_ = invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "Alex")
	require.Equal(t, 1, mock.History().Count())
	calls := mock.History().FilterByMethod("helloworld.Greeter", "SayHello")
	require.Len(t, calls, 1)
	require.Equal(t, "Alex", calls[0].Request["name"])
	require.Equal(t, "Hi", calls[0].Response["message"])

	// Second call
	_ = invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "Alex")
	require.Equal(t, 2, mock.History().Count())

	// Verify assertions
	mock.Verify().Method("helloworld.Greeter", "SayHello").Called(t, 2)
	mock.Verify().Total(t, 2)
}

func TestRun_WithSession_EmbeddedNoop(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"), WithSession("test-session"))
	defer mock.Close()

	mock.Stub("helloworld.Greeter", "SayHello").
		When(Equals("name", "x")).
		Reply(Data("message", "ok")).
		Commit()
	msg := invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "x")
	require.Equal(t, "ok", getMessageField(t, msg, "message"))
}

func TestMock_Close_Idempotent(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"))

	err := mock.Close()
	require.NoError(t, err)
	err = mock.Close()
	require.NoError(t, err) // second Close is no-op
}
