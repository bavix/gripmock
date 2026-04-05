package sdk

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func TestRunEmbeddedBufconn(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"))

	require.NotNil(t, mock.Conn())
	require.Equal(t, "bufnet", mock.Addr())

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "Alex")).
		Reply(Data("message", "Hi Alex")).
		Commit()
}

func TestRunDescriptorsAppend(t *testing.T) {
	t.Parallel()

	fdsGreeter := mustBuildFDS(t, sdkProtoPath("greeter"))
	fdsEcho := mustBuildFDS(t, filepath.Join("..", "..", "examples", "projects", "echo", "service_v1.proto"))

	mock, err := Run(t, WithDescriptors(fdsGreeter), WithDescriptors(fdsEcho))
	require.NoError(t, err)
	require.NotNil(t, mock)

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "x")).
		Reply(Data("message", "hi")).
		Commit()
	require.NotNil(t, mock.Conn())
}

func TestRunDescriptorsAppendDedup(t *testing.T) {
	t.Parallel()

	fds := mustBuildFDS(t, sdkProtoPath("greeter"))

	mock, err := Run(t, WithDescriptors(fds), WithDescriptors(fds))
	require.NoError(t, err)
	require.NotNil(t, mock)

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "x")).
		Reply(Data("message", "hi")).
		Commit()
}

func TestRunWhenStreamReplyStream(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("calculator"))

	// B7: WhenStream + Reply (client stream)
	mock.Stub(By("/calculator.CalculatorService/SumNumbers")).
		WhenStream(Matches("value", `\d+`), Matches("value", `\d+`)).
		Reply(Data("result", 42.0, "count", 2)).
		Commit()
}

func TestRunRealPort(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"),
		WithListenAddr("tcp", ":0"),
		WithHealthCheckTimeout(5*time.Second),
	)

	require.NotNil(t, mock.Conn())
	require.Regexp(t, `^127\.0\.0\.1:\d+$`, mock.Addr())
}

func TestRunRealPortDefaultNetwork(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"),
		WithListenAddr("", ":0"),
		WithHealthCheckTimeout(5*time.Second),
	)

	require.Regexp(t, `^127\.0\.0\.1:\d+$`, mock.Addr())
}

func TestRunDefaultHealthyTimeout(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"), WithHealthCheckTimeout(0))
	require.NotNil(t, mock.Conn())
}

func TestRunHealthCheckMockedViaSDK(t *testing.T) {
	t.Parallel()

	// Arrange
	mock := mustRunWithProto(t, sdkProtoPath("greeter"))
	mock.Stub(By(healthgrpc.Health_Check_FullMethodName)).
		When(Equals("service", "examples.health.backend")).
		Reply(Data("status", "NOT_SERVING")).
		Commit()

	client := healthgrpc.NewHealthClient(mock.Conn())

	// Act
	resp, err := client.Check(t.Context(), &healthgrpc.HealthCheckRequest{Service: "examples.health.backend"})

	// Assert
	require.NoError(t, err)
	require.Equal(t, healthgrpc.HealthCheckResponse_NOT_SERVING, resp.GetStatus())
}

func TestRunHealthCheckGripmockProtectedViaSDK(t *testing.T) {
	t.Parallel()

	// Arrange
	mock := mustRunWithProto(t, sdkProtoPath("greeter"))
	// Attempt to mock protected internal key.
	// Expected runtime behavior: this stub is stored but ignored.
	mock.Stub(By(healthgrpc.Health_Check_FullMethodName)).
		When(Equals("service", "gripmock")).
		Reply(Data("status", "NOT_SERVING")).
		Commit()

	client := healthgrpc.NewHealthClient(mock.Conn())

	// Act
	resp, err := client.Check(t.Context(), &healthgrpc.HealthCheckRequest{Service: "gripmock"})

	// Assert
	require.NoError(t, err)
	require.Equal(t, healthgrpc.HealthCheckResponse_SERVING, resp.GetStatus())
}

func TestRunHealthCheckUnknownServiceFallbackViaSDK(t *testing.T) {
	t.Parallel()

	// Arrange
	mock := mustRunWithProto(t, sdkProtoPath("greeter"))
	client := healthgrpc.NewHealthClient(mock.Conn())

	// Act
	resp, err := client.Check(t.Context(), &healthgrpc.HealthCheckRequest{Service: "examples.health.unknown"})

	// Assert
	require.Nil(t, resp)
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
	require.Contains(t, err.Error(), "unknown service")
}

func TestRunHealthWatchMockedStreamViaSDK(t *testing.T) {
	t.Parallel()

	// Arrange
	mock := mustRunWithProto(t, sdkProtoPath("greeter"))
	mock.Stub(By(healthgrpc.Health_Watch_FullMethodName)).
		When(Equals("service", "examples.health.watch")).
		ReplyStream(
			Data("status", "NOT_SERVING"),
			Data("status", "SERVING"),
		).
		Commit()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	client := healthgrpc.NewHealthClient(mock.Conn())
	stream, err := client.Watch(ctx, &healthgrpc.HealthCheckRequest{Service: "examples.health.watch"})
	require.NoError(t, err)

	// Act
	first, err := stream.Recv()
	require.NoError(t, err)

	second, err := stream.Recv()
	require.NoError(t, err)

	cancel()

	// Assert
	require.Equal(t, healthgrpc.HealthCheckResponse_NOT_SERVING, first.GetStatus())
	require.Equal(t, healthgrpc.HealthCheckResponse_SERVING, second.GetStatus())
}

func TestRunHealthWatchGripmockProtectedViaSDK(t *testing.T) {
	t.Parallel()

	// Arrange
	mock := mustRunWithProto(t, sdkProtoPath("greeter"))
	mock.Stub(By(healthgrpc.Health_Watch_FullMethodName)).
		When(Equals("service", "gripmock")).
		ReplyStream(Data("status", "NOT_SERVING")).
		Commit()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	client := healthgrpc.NewHealthClient(mock.Conn())
	stream, err := client.Watch(ctx, &healthgrpc.HealthCheckRequest{Service: "gripmock"})
	require.NoError(t, err)

	// Act
	first, err := stream.Recv()

	// Assert
	require.NoError(t, err)
	require.Equal(t, healthgrpc.HealthCheckResponse_SERVING, first.GetStatus())
}

func TestRunContextFromT(t *testing.T) {
	t.Parallel()

	// Run(t, opts) resolves context from t.Context() when t is *testing.T
	mock, err := Run(t, WithDescriptors(mustBuildFDS(t, sdkProtoPath("greeter"))))
	require.NoError(t, err)
	require.NotNil(t, mock)
	mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "x")).
		Reply(Data("message", "ok")).
		Commit()
}

func TestRunValidation(t *testing.T) {
	t.Parallel()

	_, err := Run(t)
	require.Error(t, err)
	require.Contains(t, err.Error(), "descriptors required")
}

func TestRunIgnoresNilOptions(t *testing.T) {
	t.Parallel()

	mock, err := Run(t,
		nil,
		WithDescriptors(mustBuildFDS(t, sdkProtoPath("greeter"))),
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, mock)
}

func TestRunInvalidDescriptors(t *testing.T) {
	t.Parallel()

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

	_, err := Run(t, WithDescriptors(fds))
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create files registry")
}

func TestRunListenError(t *testing.T) {
	t.Parallel()

	_, err := Run(t,
		WithDescriptors(mustBuildFDS(t, sdkProtoPath("greeter"))),
		WithListenAddr("tcp", ":99999"),
		WithHealthCheckTimeout(5*time.Second),
	)
	require.Error(t, err)
	errStr := err.Error()
	require.True(t,
		strings.Contains(errStr, "invalid") ||
			strings.Contains(errStr, "address") ||
			strings.Contains(errStr, "listen"),
		"err=%v", err)
}

func TestRunListenAddrStringUnixFallback(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("Unix sockets not supported on Windows")
	}

	sockPath := "/tmp/gripmock_" + uuid.New().String()[:8] + ".sock"
	mock, err := Run(t,
		WithDescriptors(mustBuildFDS(t, sdkProtoPath("greeter"))),
		WithListenAddr("unix", sockPath),
		WithHealthCheckTimeout(2*time.Second),
	)
	if err != nil {
		t.Logf("Run failed for unix (may hit listenAddrString before client dial): %v", err)

		return
	}
	require.Contains(t, mock.Addr(), ".sock")
}

func TestRunReplyStreamSkipsNilData(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("track-streaming"))

	// ReplyStream with nil Data entries - they are skipped (stuber.Output{Data: nil})
	mock.Stub(By("/TrackService/StreamTrack")).
		When(Equals("stn", "MS#00002")).
		ReplyStream(
			Data("stn", "MS#00002", "identity", "00"),
			stuber.Output{Data: nil}, // skipped
			Data("stn", "MS#00002", "identity", "01"),
		).
		Commit()
}

func TestRunReplyStream(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("track-streaming"))

	// Server stream: When + ReplyStream
	mock.Stub(By("/TrackService/StreamTrack")).
		When(Equals("stn", "MS#00001")).
		ReplyStream(
			Data("stn", "MS#00001", "identity", "00", "latitude", 0.08),
			Data("stn", "MS#00001", "identity", "01", "latitude", 0.09),
		).
		Commit()
}

func TestRunReplyError(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "error")).
		ReplyError(codes.NotFound, "user not found").
		Commit()

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "Alex")).
		Reply(Data("message", "Hi")).
		Commit()
}

func TestRunReplyErrorWithDetails(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "error-details")).
		ReplyErrorWithDetails(codes.InvalidArgument, "validation failed", map[string]any{
			"type":  "type.googleapis.com/google.protobuf.StringValue",
			"value": "invalid name value",
		}).
		Commit()

	err := mock.Conn().Invoke(ctx, "/helloworld.Greeter/SayHello", createGreeterRequest(t, reg, "error-details"), &dynamicpb.Message{})
	require.Error(t, err)

	st := status.Convert(err)
	require.Equal(t, codes.InvalidArgument, st.Code())
	require.Equal(t, "validation failed", st.Message())

	details := st.Details()
	require.Len(t, details, 1)

	msg, ok := details[0].(*wrapperspb.StringValue)
	require.True(t, ok)
	require.Equal(t, "invalid name value", msg.GetValue())
}

func TestRunPriority(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "priority")).
		Reply(Data("message", "low")).
		Priority(10).
		Commit()

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "priority")).
		Reply(Data("message", "high")).
		Priority(100).
		Commit()
}

func TestRunContains(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Contains("name", "Alice")).
		Reply(Data("message", "Hello Alice")).
		Commit()
}

func TestRunMap(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Map("name", "Bob", "extra", "value")).
		Reply(Data("message", "Hi Bob")).
		Commit()
}

func TestRunHealthyTimeout(t *testing.T) {
	t.Parallel()

	_, err := Run(t, WithDescriptors(mustBuildFDS(t, sdkProtoPath("greeter"))), WithHealthCheckTimeout(1))
	require.Error(t, err)
	errStr := err.Error()
	require.True(t,
		errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) ||
			strings.Contains(errStr, "deadline exceeded") || strings.Contains(errStr, "context canceled"),
		"err=%v", err)
}

func TestRunMergeIntegration(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Merge(Equals("name", "Alex"))).
		Reply(MergeOutput(Data("message", "Hi from Merge"))).
		Commit()

	msg := invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "Alex")
	require.Equal(t, "Hi from Merge", getMessageField(t, msg, "message"))
}

func TestRunSugarMatchReturn(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		Match("name", "Alex").
		Return("message", "Hi sugar").
		Commit()

	msg := invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "Alex")
	require.Equal(t, "Hi sugar", getMessageField(t, msg, "message"))
}

func TestRunSugarUnary(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		Unary("name", "Bob", "message", "Hello Bob").
		Commit()

	msg := invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "Bob")
	require.Equal(t, "Hello Bob", getMessageField(t, msg, "message"))
}

func TestRunDynamicTemplateMatchReturn(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		Match("name", "Alex").
		Return("message", "Hi {{.Request.name}}").
		Commit()

	msg := invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "Alex")
	require.Equal(t, "Hi Alex", getMessageField(t, msg, "message"))
}

func TestRunDynamicTemplateWhenReply(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Map("name", "Charlie")).
		Reply(Data("message", "Greetings {{.Request.name}}!")).
		Commit()

	msg := invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "Charlie")
	require.Equal(t, "Greetings Charlie!", getMessageField(t, msg, "message"))
}

func TestRunDynamicTemplateUnary(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		Unary("name", "Diana", "message", "Dear {{.Request.name}}").
		Commit()

	msg := invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "Diana")
	require.Equal(t, "Dear Diana", getMessageField(t, msg, "message"))
}

func TestRunDynamicTemplateMergeOutput(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "Eve")).
		Reply(MergeOutput(Data("message", "Hi {{.Request.name}} from Merge"))).
		Commit()

	msg := invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "Eve")
	require.Equal(t, "Hi Eve from Merge", getMessageField(t, msg, "message"))
}

func TestRunSugarMatchPanicOddArgs(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"))

	require.PanicsWithValue(t, "sdk.Match: need pairs (key, value), got 1 args", func() {
		mock.Stub(By("/helloworld.Greeter/SayHello")).Match("name").Commit()
	})
}

func TestRunSugarReturnPanicOddArgs(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"))

	require.PanicsWithValue(t, "sdk.Return: need pairs (key, value), got 1 args", func() {
		mock.Stub(By("/helloworld.Greeter/SayHello")).When(Equals("name", "x")).Return("message").Commit()
	})
}

func TestRunReplyHeaders(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "Alex")).
		Reply(Data("message", "Hi")).
		ReplyHeaderPairs("x-custom", "value", "x-id", "123").
		Commit()
}

func TestRunWhenHeadersIntegration(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "Alex")).
		WhenHeaders(HeaderEquals("x-custom", "expected-value")).
		Reply(Data("message", "matched-by-header")).
		Commit()

	mock.Stub(By("/helloworld.Greeter/SayHello")).
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

func TestRunDelay(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "slow")).
		Reply(Data("message", "delayed")).
		Delay(10 * time.Millisecond).
		Commit()
}

func TestRunTimesExhaustedAfterLimit(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
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

func TestRunHistoryAndVerify(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
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
	mock.Verify().Method(By("/helloworld.Greeter/SayHello")).Called(t, 2)
	mock.Verify().Total(t, 2)
}

func TestRunVerifyStubTimesFromStubTimes(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	// Ben: 1 call, Alice: 2 calls — SDK tracks sum = 3
	mock.Stub(By("/helloworld.Greeter/SayHello")).When(Equals("name", "Ben")).Reply(Data("message", "Hi Ben")).Times(1).Commit()
	mock.Stub(By("/helloworld.Greeter/SayHello")).When(Equals("name", "Alice")).Reply(Data("message", "Hi Alice")).Times(2).Commit()

	_ = invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "Ben")
	_ = invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "Alice")
	_ = invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "Alice")
	// Close() runs VerifyStubTimes — passes (3 calls, expected 3)
}

func TestRunCloseVerifiesStubTimes(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).When(Equals("name", "x")).Reply(Data("message", "ok")).Times(1).Commit()
	_ = invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "x")
	// Close() runs VerifyStubTimes — passes (1 call, expected 1)
}

func TestRunVerifyStubTimesErrNoErrorWhenMatch(t *testing.T) {
	t.Parallel()

	// Test VerifyStubTimesErr returns no error when expected and actual calls match
	// Since we now require non-nil TestingT and cleanup always runs, we ensure
	// the cleanup verification passes by making expected and actual calls match.

	fds := mustBuildFDS(t, sdkProtoPath("greeter"))
	mock, err := Run(t, WithDescriptors(fds))
	require.NoError(t, err)

	// Setup a stub with Times(1) and make exactly 1 call so cleanup verification passes
	mock.Stub(By("/helloworld.Greeter/SayHello")).When(Equals("name", "x")).Reply(Data("message", "ok")).Times(1).Commit()

	// Make the expected call
	ctx := t.Context()
	reg := mustBuildRegistryFromProto(t, sdkProtoPath("greeter"))
	invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "x")

	// Verify that VerifyStubTimesErr returns nil when counts match
	err = mock.Verify().VerifyStubTimesErr()
	require.NoError(t, err) // Should be no error since calls match expected times
}

func TestRunWithSessionEmbeddedNop(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"), WithSession("test-session"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "x")).
		Reply(Data("message", "ok")).
		Commit()
	msg := invokeGreeterSayHello(t, mock.Conn(), reg, ctx, "x")
	require.Equal(t, "ok", getMessageField(t, msg, "message"))
}

func TestMockCLoseIdempotent(t *testing.T) {
	t.Parallel()

	mock := mustRunWithProto(t, sdkProtoPath("greeter"))

	err := mock.Close()
	require.NoError(t, err)
	err = mock.Close()
	require.NoError(t, err) // second Close is no-op
}

func TestRunReplyStreamEmptyStream(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("search"))

	mock.Stub(By("/search.SearchService/Search")).
		When(Equals("query", "empty")).
		ReplyStream().
		Commit()

	inDesc, err := reg.FindDescriptorByName("search.SearchRequest")
	require.NoError(t, err)

	outDesc, err := reg.FindDescriptorByName("search.SearchResult")
	require.NoError(t, err)

	in := dynamicpb.NewMessage(inDesc.(protoreflect.MessageDescriptor))
	queryFd := inDesc.(protoreflect.MessageDescriptor).Fields().ByName("query")
	in.Set(queryFd, protoreflect.ValueOfString("empty"))

	stream, err := mock.Conn().NewStream(ctx, &grpc.StreamDesc{
		StreamName:    "Search",
		ServerStreams: true,
		ClientStreams: false,
	}, "/search.SearchService/Search")
	require.NoError(t, err)

	err = stream.SendMsg(in)
	require.NoError(t, err)

	out := dynamicpb.NewMessage(outDesc.(protoreflect.MessageDescriptor))
	err = stream.RecvMsg(out)
	require.Error(t, err)
	require.Equal(t, io.EOF, err)
}
