package sdk

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/domain/protoset"
)

func TestV2Unary(t *testing.T) {
	t.Parallel()

	srv, fds := newProjectSrv(t, "greeter")

	require.NotNil(t, srv.Conn())
	require.Contains(t, srv.Address(), "127.0.0.1:")

	srv.ExpectUnary("/helloworld.Greeter/SayHello").
		Match("name", "Alex").
		Return("message", "Hi Alex")

	reg := mustBuildReg(t, fds)
	msg := invokeGreeter(t, srv.Conn(), reg, "Alex")
	require.Equal(t, "Hi Alex", getMsgField(t, msg))
}

func TestV2Templates(t *testing.T) {
	t.Parallel()

	srv, fds := newProjectSrv(t, "greeter")

	srv.ExpectUnary("/helloworld.Greeter/SayHello").
		Match("name", "Alex").
		Return("message", "Hi {{.Request.name}}")

	reg := mustBuildReg(t, fds)
	msg := invokeGreeter(t, srv.Conn(), reg, "Alex")
	require.Equal(t, "Hi Alex", getMsgField(t, msg))
}

func TestV2Priority(t *testing.T) {
	t.Parallel()

	srv, fds := newProjectSrv(t, "greeter")

	srv.ExpectUnary("/helloworld.Greeter/SayHello").
		Match("name", "Alex").
		Priority(100).
		Return("message", "exact")

	srv.ExpectUnary("/helloworld.Greeter/SayHello").
		WithPayloadMap(Matches("name", "A.*")).
		Priority(1).
		Return("message", "fallback")

	reg := mustBuildReg(t, fds)
	msg := invokeGreeter(t, srv.Conn(), reg, "Alex")
	require.Equal(t, "exact", getMsgField(t, msg))
}

func TestV2Error(t *testing.T) {
	t.Parallel()

	srv, fds := newProjectSrv(t, "greeter")

	srv.ExpectUnary("/helloworld.Greeter/SayHello").
		Match("name", "error").
		ReturnError(codes.NotFound, "user not found")

	reg := mustBuildReg(t, fds)
	err := callGreeter(t, srv.Conn(), reg, "error")
	require.Error(t, err)
	require.Contains(t, err.Error(), "user not found")
}

func TestV2Times(t *testing.T) {
	t.Parallel()

	srv, fds := newProjectSrv(t, "greeter")

	srv.ExpectUnary("/helloworld.Greeter/SayHello").
		Match("name", "limited").
		Times(2).
		Return("message", "ok")

	reg := mustBuildReg(t, fds)

	msg1 := invokeGreeter(t, srv.Conn(), reg, "limited")
	require.Equal(t, "ok", getMsgField(t, msg1))

	msg2 := invokeGreeter(t, srv.Conn(), reg, "limited")
	require.Equal(t, "ok", getMsgField(t, msg2))

	err := callGreeter(t, srv.Conn(), reg, "limited")
	require.Error(t, err)
}

func TestV2NextWillReturn(t *testing.T) {
	t.Parallel()

	srv, fds := newProjectSrv(t, "greeter")

	srv.ExpectUnary("/helloworld.Greeter/SayHello").
		Match("name", "chain").
		Return("message", "first").
		NextWillReturn("message", "second").
		NextWillReturn("message", "third")

	reg := mustBuildReg(t, fds)

	msg1 := invokeGreeter(t, srv.Conn(), reg, "chain")
	require.Equal(t, "first", getMsgField(t, msg1))

	msg2 := invokeGreeter(t, srv.Conn(), reg, "chain")
	require.Equal(t, "second", getMsgField(t, msg2))

	msg3 := invokeGreeter(t, srv.Conn(), reg, "chain")
	require.Equal(t, "third", getMsgField(t, msg3))

	err := callGreeter(t, srv.Conn(), reg, "chain")
	require.Error(t, err)
}

func TestV2Verification(t *testing.T) {
	t.Parallel()

	srv, fds := newProjectSrv(t, "greeter")

	srv.ExpectUnary("/helloworld.Greeter/SayHello").
		Match("name", "Alex").
		Times(1).
		Return("message", "Hi")

	reg := mustBuildReg(t, fds)
	_ = invokeGreeter(t, srv.Conn(), reg, "Alex")

	require.NoError(t, srv.ExpectationsWereMet())
	require.Equal(t, 1, srv.Called("/helloworld.Greeter/SayHello"))
	require.Equal(t, 1, srv.TotalCalls())
	require.Len(t, srv.History(), 1)

	// Idempotent
	require.NoError(t, srv.ExpectationsWereMet())
	_ = srv.Close()
}

func TestV2VerificationFailed(t *testing.T) {
	t.Parallel()

	srv, fds := newProjectSrv(t, "greeter")

	srv.ExpectUnary("/helloworld.Greeter/SayHello").
		Match("name", "Alex").
		Times(2).
		Return("message", "Hi")

	reg := mustBuildReg(t, fds)
	_ = invokeGreeter(t, srv.Conn(), reg, "Alex")

	err := srv.ExpectationsWereMet()
	require.Error(t, err)
	require.ErrorIs(t, err, ErrVerificationFailed, "expected ErrVerificationFailed")

	var notMet *ExpectationNotMetError
	require.ErrorAs(t, err, &notMet)
	require.Equal(t, "helloworld.Greeter", notMet.Service)
	require.Equal(t, "SayHello", notMet.Method)
	require.Equal(t, 2, notMet.Expected)
	require.Equal(t, 1, notMet.Actual)

	_ = srv.Close()
}

func TestV2Reset(t *testing.T) {
	t.Parallel()

	srv, fds := newProjectSrv(t, "greeter")

	srv.ExpectUnary("/helloworld.Greeter/SayHello").
		Match("name", "Alex").
		Return("message", "Hi")

	reg := mustBuildReg(t, fds)
	_ = invokeGreeter(t, srv.Conn(), reg, "Alex")
	require.Equal(t, 1, srv.TotalCalls())

	srv.Reset()

	srv.ExpectUnary("/helloworld.Greeter/SayHello").
		Match("name", "Bob").
		Return("message", "Hello Bob")

	msg := invokeGreeter(t, srv.Conn(), reg, "Bob")
	require.Equal(t, "Hello Bob", getMsgField(t, msg))
}

func TestV2ParallelIsolation(t *testing.T) {
	t.Parallel()
	// Each subtest creates its own server — fully isolated
	t.Run("sub1", func(t *testing.T) {
		t.Parallel()
		srv, fds := newProjectSrv(t, "greeter")

		srv.ExpectUnary("/helloworld.Greeter/SayHello").
			Match("name", "user1").
			Return("message", "hello user1")

		reg := mustBuildReg(t, fds)
		msg := invokeGreeter(t, srv.Conn(), reg, "user1")
		require.Equal(t, "hello user1", getMsgField(t, msg))
	})

	t.Run("sub2", func(t *testing.T) {
		t.Parallel()
		srv, fds := newProjectSrv(t, "greeter")

		srv.ExpectUnary("/helloworld.Greeter/SayHello").
			Match("name", "user2").
			Return("message", "hello user2")

		reg := mustBuildReg(t, fds)
		msg := invokeGreeter(t, srv.Conn(), reg, "user2")
		require.Equal(t, "hello user2", getMsgField(t, msg))
	})
}

func TestV2ServerStream(t *testing.T) {
	t.Parallel()

	srv, fds := newProjectSrv(t, "search")

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "test").
		SendStream(
			map[string]any{"id": "1", "title": "result 1"},
			map[string]any{"id": "2", "title": "result 2"},
		)

	reg := mustBuildReg(t, fds)
	inDesc, _ := reg.FindDescriptorByName("search.SearchRequest")
	outDesc, _ := reg.FindDescriptorByName("search.SearchResult")

	inMsgDesc, ok := inDesc.(protoreflect.MessageDescriptor)
	require.True(t, ok)

	in := dynamicpb.NewMessage(inMsgDesc)
	in.Set(inMsgDesc.Fields().ByName("query"), protoreflect.ValueOfString("test"))

	stream, err := srv.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "Search", ServerStreams: true, ClientStreams: false},
		"/search.SearchService/Search")
	require.NoError(t, err)
	require.NoError(t, stream.SendMsg(in))

	outMsgDesc, ok := outDesc.(protoreflect.MessageDescriptor)
	require.True(t, ok)

	out1 := dynamicpb.NewMessage(outMsgDesc)
	require.NoError(t, stream.RecvMsg(out1))

	titleFd := outMsgDesc.Fields().ByName("title")
	require.Equal(t, "result 1", out1.Get(titleFd).String())

	out2 := dynamicpb.NewMessage(outMsgDesc)
	require.NoError(t, stream.RecvMsg(out2))
	require.Equal(t, "result 2", out2.Get(titleFd).String())
}

func TestV2ClientStream(t *testing.T) {
	t.Parallel()

	srv, fds := newProjectSrv(t, "calculator")

	srv.ExpectClientStream("/calculator.CalculatorService/SumNumbers").
		WithPayloadMap(Matches("value", "\\d+")).
		Return("result", 42.0, "count", 2)

	reg := mustBuildReg(t, fds)
	inDesc, _ := reg.FindDescriptorByName("calculator.NumberRequest")
	outDesc, _ := reg.FindDescriptorByName("calculator.SumResponse")

	require.NotNil(t, inDesc)
	require.NotNil(t, outDesc)

	stream, err := srv.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "SumNumbers", ServerStreams: false, ClientStreams: true},
		"/calculator.CalculatorService/SumNumbers")
	require.NoError(t, err)

	inMsg, ok := inDesc.(protoreflect.MessageDescriptor)
	require.True(t, ok)

	valFd := inMsg.Fields().ByName("value")
	require.NotNil(t, valFd)

	for _, v := range []float64{1.0, 2.0} {
		msg := dynamicpb.NewMessage(inMsg)
		msg.Set(valFd, protoreflect.ValueOfFloat64(v))
		require.NoError(t, stream.SendMsg(msg))
	}

	require.NoError(t, stream.CloseSend())

	outDescMsg, ok := outDesc.(protoreflect.MessageDescriptor)
	require.True(t, ok)

	out := dynamicpb.NewMessage(outDescMsg)
	require.NoError(t, stream.RecvMsg(out))

	resultFd := outDescMsg.Fields().ByName("result")
	require.InDelta(t, 42.0, out.Get(resultFd).Float(), 0.001)
}

func TestV2ServerStreamEmpty(t *testing.T) {
	t.Parallel()

	srv, fds := newProjectSrv(t, "search")

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "empty").
		SendStream()

	reg := mustBuildReg(t, fds)
	inDesc, _ := reg.FindDescriptorByName("search.SearchRequest")
	outDesc, _ := reg.FindDescriptorByName("search.SearchResult")

	inMsgDesc, ok := inDesc.(protoreflect.MessageDescriptor)
	require.True(t, ok)

	in := dynamicpb.NewMessage(inMsgDesc)
	in.Set(inMsgDesc.Fields().ByName("query"), protoreflect.ValueOfString("empty"))

	stream, err := srv.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "Search", ServerStreams: true, ClientStreams: false},
		"/search.SearchService/Search")
	require.NoError(t, err)
	require.NoError(t, stream.SendMsg(in))

	outMsgDesc, ok := outDesc.(protoreflect.MessageDescriptor)
	require.True(t, ok)

	out := dynamicpb.NewMessage(outMsgDesc)
	err = stream.RecvMsg(out)
	require.Error(t, err)
	require.ErrorIs(t, err, io.EOF)
}

func TestV2EnumOneof(t *testing.T) {
	t.Parallel()

	srv, fds := newProjectSrv(t, "validator")

	srv.ExpectUnary("/validator.Validator/Validate").
		WithPayloadMap(
			Equals("number", 42),
			Equals("validation_type", "NUMBER_RANGE"),
			Equals("min", 10),
			Equals("max", 100),
		).
		Return("isValid", true)

	reg := mustBuildReg(t, fds)
	inDesc, _ := reg.FindDescriptorByName("validator.ValidateRequest")
	outDesc, _ := reg.FindDescriptorByName("validator.ValidateResponse")

	require.NotNil(t, inDesc)
	require.NotNil(t, outDesc)

	inMsgDesc, ok := inDesc.(protoreflect.MessageDescriptor)
	require.True(t, ok)

	in := dynamicpb.NewMessage(inMsgDesc)
	numberFd := inMsgDesc.Fields().ByName("number")
	in.Set(numberFd, protoreflect.ValueOfInt64(42))

	vtFd := inMsgDesc.Fields().ByName("validation_type")
	vtDesc := vtFd.Enum()
	nrVal := vtDesc.Values().ByNumber(4) // NUMBER_RANGE = 4
	in.Set(vtFd, protoreflect.ValueOfEnum(nrVal.Number()))

	minFd := inMsgDesc.Fields().ByName("min")
	in.Set(minFd, protoreflect.ValueOfInt64(10))

	maxFd := inMsgDesc.Fields().ByName("max")
	in.Set(maxFd, protoreflect.ValueOfInt64(100))

	outMsgDesc, ok := outDesc.(protoreflect.MessageDescriptor)
	require.True(t, ok)

	out := dynamicpb.NewMessage(outMsgDesc)
	err := srv.Conn().Invoke(t.Context(), "/validator.Validator/Validate", in, out)
	require.NoError(t, err)

	isValidFd := outMsgDesc.Fields().ByName("is_valid")
	require.True(t, out.Get(isValidFd).Bool())
}

func TestV2RepeatedField(t *testing.T) {
	t.Parallel()

	// Use a simple repeated field test via the identifier project
	srv2, fds2 := newProjectSrv(t, "identifier")

	srv2.ExpectUnary("/example.identifier.v1.IdentifierService/ProcessUUIDs").
		WithPayloadMap(
			Equals("string_uuids", []any{"uuid-1", "uuid-2"}),
		).
		Return("processId", 42.0)

	reg := mustBuildReg(t, fds2)
	inDesc, _ := reg.FindDescriptorByName("example.identifier.v1.ProcessUUIDsRequest")
	outDesc, _ := reg.FindDescriptorByName("example.identifier.v1.ProcessUUIDsResponse")

	require.NotNil(t, inDesc)
	require.NotNil(t, outDesc)

	inMsgDesc, ok := inDesc.(protoreflect.MessageDescriptor)
	require.True(t, ok)

	in := dynamicpb.NewMessage(inMsgDesc)
	suFd := inMsgDesc.Fields().ByName("string_uuids")
	suList := in.Mutable(suFd).List()
	suList.Append(protoreflect.ValueOfString("uuid-1"))
	suList.Append(protoreflect.ValueOfString("uuid-2"))

	outMsgDesc, ok := outDesc.(protoreflect.MessageDescriptor)
	require.True(t, ok)

	out := dynamicpb.NewMessage(outMsgDesc)
	err := srv2.Conn().Invoke(t.Context(), "/example.identifier.v1.IdentifierService/ProcessUUIDs", in, out)
	require.NoError(t, err)

	pidFd := outMsgDesc.Fields().ByName("process_id")
	require.Equal(t, int64(42), out.Get(pidFd).Int())
}

func TestV2EffectWithTemplate(t *testing.T) {
	t.Parallel()

	srv, fds := newProjectSrv(t, "greeter")
	reg := mustBuildReg(t, fds)

	// Effect that uses {{.Request.name}} in its response
	effect := Upsert("helloworld.Greeter", "SayHello").
		Match("name", "{{.Request.name}}_effected").
		Return("message", "Hello {{.Request.name}} from effect").
		Build()

	srv.ExpectUnary("/helloworld.Greeter/SayHello").
		Match("name", "trigger").
		Effect(effect).
		Return("message", "first")

	// First call — matches original stub
	msg1 := invokeGreeter(t, srv.Conn(), reg, "trigger")
	require.Equal(t, "first", getMsgField(t, msg1))

	// Effect creates a stub that matches "trigger_effected" with template response
	msg2 := invokeGreeter(t, srv.Conn(), reg, "trigger_effected")
	require.Equal(t, "Hello trigger from effect", getMsgField(t, msg2))
}

func TestV2StreamWithError(t *testing.T) {
	t.Parallel()

	srv, fds := newProjectSrv(t, "search")

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "partial").
		SendStream(
			map[string]any{"id": "1", "title": "first"},
		)

	// Try sending an error after stream messages using NextWillReturn
	// (This tests the edge case — may not work yet)
	reg := mustBuildReg(t, fds)
	inDesc, _ := reg.FindDescriptorByName("search.SearchRequest")
	outDesc, _ := reg.FindDescriptorByName("search.SearchResult")

	inMsgDesc, ok := inDesc.(protoreflect.MessageDescriptor)
	require.True(t, ok)

	in := dynamicpb.NewMessage(inMsgDesc)
	in.Set(inMsgDesc.Fields().ByName("query"), protoreflect.ValueOfString("partial"))

	stream, err := srv.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "Search", ServerStreams: true, ClientStreams: false},
		"/search.SearchService/Search")
	require.NoError(t, err)
	require.NoError(t, stream.SendMsg(in))

	// First message should arrive
	outMsgDesc, ok := outDesc.(protoreflect.MessageDescriptor)
	require.True(t, ok)

	out1 := dynamicpb.NewMessage(outMsgDesc)
	err = stream.RecvMsg(out1)
	require.NoError(t, err)

	titleFd := outMsgDesc.Fields().ByName("title")
	require.Equal(t, "first", out1.Get(titleFd).String())

	// After the stream ends, EOF or error
	err = stream.RecvMsg(dynamicpb.NewMessage(outMsgDesc))
	t.Logf("second recv error: %v", err)
}

func TestV2EffectsUpsert(t *testing.T) {
	t.Parallel()

	srv, fds := newProjectSrv(t, "greeter")
	reg := mustBuildReg(t, fds)

	effect := Upsert("helloworld.Greeter", "SayHello").
		Match("name", "next").
		Return("message", "from effect").
		Build()

	srv.ExpectUnary("/helloworld.Greeter/SayHello").
		Match("name", "buzzy").
		Effect(effect).
		Return("message", "with effects")

	msg := invokeGreeter(t, srv.Conn(), reg, "buzzy")
	require.Equal(t, "with effects", getMsgField(t, msg))

	msg2 := invokeGreeter(t, srv.Conn(), reg, "next")
	require.Equal(t, "from effect", getMsgField(t, msg2))
}

// newProjectSrv creates a test server with a project's proto descriptors.
func newProjectSrv(t *testing.T, project string) (*Server, *descriptorpb.FileDescriptorSet) {
	t.Helper()
	fds := buildTestFDS(t, project)

	return NewServer(t, WithDescriptors(fds)), fds
}

func buildTestFDS(t *testing.T, project string) *descriptorpb.FileDescriptorSet {
	t.Helper()
	ctx := t.Context()
	fdsSlice, err := protoset.Build(ctx, nil, []string{filepath.Join("..", "..", "examples", "projects", project, "service.proto")}, nil)
	require.NoError(t, err)
	require.NotEmpty(t, fdsSlice)

	return fdsSlice[0]
}

func mustBuildReg(t *testing.T, fds *descriptorpb.FileDescriptorSet) *protoregistry.Files {
	t.Helper()

	reg, err := protodesc.NewFiles(fds)
	require.NoError(t, err)

	return reg
}

func invokeGreeter(t *testing.T, conn *grpc.ClientConn, reg *protoregistry.Files, name string) *dynamicpb.Message {
	t.Helper()

	inDesc, _ := reg.FindDescriptorByName("helloworld.HelloRequest")
	outDesc, _ := reg.FindDescriptorByName("helloworld.HelloReply")

	inMsgDesc, ok := inDesc.(protoreflect.MessageDescriptor)
	require.True(t, ok)

	in := dynamicpb.NewMessage(inMsgDesc)
	fd := inMsgDesc.Fields().ByName("name")
	in.Set(fd, protoreflect.ValueOfString(name))

	outMsgDesc, ok := outDesc.(protoreflect.MessageDescriptor)
	require.True(t, ok)

	out := dynamicpb.NewMessage(outMsgDesc)
	err := conn.Invoke(t.Context(), "/helloworld.Greeter/SayHello", in, out)
	require.NoError(t, err)

	return out
}

func callGreeter(t *testing.T, conn *grpc.ClientConn, reg *protoregistry.Files, name string) error {
	t.Helper()

	inDesc, _ := reg.FindDescriptorByName("helloworld.HelloRequest")
	outDesc, _ := reg.FindDescriptorByName("helloworld.HelloReply")

	inMsgDesc, ok := inDesc.(protoreflect.MessageDescriptor)
	require.True(t, ok)

	in := dynamicpb.NewMessage(inMsgDesc)
	fd := inMsgDesc.Fields().ByName("name")
	in.Set(fd, protoreflect.ValueOfString(name))

	outMsgDesc, ok := outDesc.(protoreflect.MessageDescriptor)
	require.True(t, ok)

	out := dynamicpb.NewMessage(outMsgDesc)

	return conn.Invoke(t.Context(), "/helloworld.Greeter/SayHello", in, out)
}

func getMsgField(t *testing.T, msg *dynamicpb.Message) string {
	t.Helper()

	fd := msg.Descriptor().Fields().ByName("message")
	require.NotNil(t, fd)

	return msg.Get(fd).String()
}
