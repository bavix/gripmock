package sdk

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Section 1: Glob matching

func TestRunGlobExactMatch(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	err := mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Glob("name", "Alex")).
		Reply(Data("message", "globbed Alex")).
		Commit()
	require.NoError(t, err)

	msg := invokeGreeterSayHello(t, mock.Conn(), reg, "Alex")
	require.Equal(t, "globbed Alex", getMessageField(t, msg))
}

func TestRunGlobWildcardMatch(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	err := mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Glob("name", "A*")).
		Reply(Data("message", "starts with A")).
		Commit()
	require.NoError(t, err)

	msg := invokeGreeterSayHello(t, mock.Conn(), reg, "Alex")
	require.Equal(t, "starts with A", getMessageField(t, msg))
}

func TestRunGlobNoMatch(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	err := mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Glob("name", "Z*")).
		Reply(Data("message", "z-match")).
		Commit()
	require.NoError(t, err)

	outDesc, _ := reg.FindDescriptorByName("helloworld.HelloReply")
	err = mock.Conn().Invoke(t.Context(),
		"/helloworld.Greeter/SayHello",
		createGreeterRequest(t, reg, "Alex"),
		dynamicpb.NewMessage(outDesc.(protoreflect.MessageDescriptor)))
	require.Error(t, err)
}

func TestRunGlobPriorityOverEquals(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	err := mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Glob("name", "Al*")).
		Priority(5).
		Reply(Data("message", "glob-5")).
		Commit()
	require.NoError(t, err)

	err = mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "Alex")).
		Priority(10).
		Reply(Data("message", "equals-10")).
		Commit()
	require.NoError(t, err)

	msg := invokeGreeterSayHello(t, mock.Conn(), reg, "Alex")
	require.Equal(t, "equals-10", getMessageField(t, msg))
}

// Section 2: AnyOf matching

func TestRunAnyOfFirstMatch(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	err := mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(AnyOf(Equals("name", "Alex"), Equals("name", "Bob"))).
		Reply(Data("message", "anyof match")).
		Commit()
	require.NoError(t, err)

	msg := invokeGreeterSayHello(t, mock.Conn(), reg, "Bob")
	require.Equal(t, "anyof match", getMessageField(t, msg))
}

func TestRunAnyOfWithMerge(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	err := mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Merge(Equals("name", "Alex"), AnyOf(Glob("name", "A*"), Equals("name", "Bob")))).
		Reply(Data("message", "merge anyof")).
		Commit()
	require.NoError(t, err)

	msg := invokeGreeterSayHello(t, mock.Conn(), reg, "Alex")
	require.Equal(t, "merge anyof", getMessageField(t, msg))
}

func TestRunAnyOfNoMatch(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	outDesc, _ := reg.FindDescriptorByName("helloworld.HelloReply")

	err := mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(AnyOf(Equals("name", "Alex"), Equals("name", "Bob"))).
		Reply(Data("message", "anyof")).
		Commit()
	require.NoError(t, err)

	err = mock.Conn().Invoke(t.Context(),
		"/helloworld.Greeter/SayHello",
		createGreeterRequest(t, reg, "Charlie"),
		dynamicpb.NewMessage(outDesc.(protoreflect.MessageDescriptor)))
	require.Error(t, err)
}

func TestRunHeaderAnyOfMatch(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	outDesc, _ := reg.FindDescriptorByName("helloworld.HelloReply")

	err := mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "Alex")).
		WhenHeaders(HeaderAnyOf(
			HeaderEquals("x-source", "web"),
			HeaderEquals("x-source", "mobile"),
		)).
		Reply(Data("message", "Hi Alex from any source")).
		Commit()
	require.NoError(t, err)

	ctx := metadata.NewOutgoingContext(t.Context(),
		metadata.Pairs("x-source", "mobile"))

	out := dynamicpb.NewMessage(outDesc.(protoreflect.MessageDescriptor))
	err = mock.Conn().Invoke(ctx, "/helloworld.Greeter/SayHello",
		createGreeterRequest(t, reg, "Alex"), out)
	require.NoError(t, err)
	require.Equal(t, "Hi Alex from any source", getMessageField(t, out))
}

// Section 3: Dynamic templates

func TestRunDynamicTemplateNestedField(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("nested-messages"))

	err := mock.Stub(By("/nested.ConfigurationService/UpdateConfig")).
		Match("host", "localhost", "port", 8080).
		Return("success", true, "message", "config {{.Request.host}}:{{.Request.port}}").
		Commit()
	require.NoError(t, err)

	inDesc, _ := reg.FindDescriptorByName("nested.Config.Settings.NetworkSettings")
	outDesc, _ := reg.FindDescriptorByName("nested.Config.Response")
	in := dynamicpb.NewMessage(inDesc.(protoreflect.MessageDescriptor))
	out := dynamicpb.NewMessage(outDesc.(protoreflect.MessageDescriptor))

	in.Set(inDesc.(protoreflect.MessageDescriptor).Fields().ByName("host"), protoreflect.ValueOfString("localhost"))
	in.Set(inDesc.(protoreflect.MessageDescriptor).Fields().ByName("port"), protoreflect.ValueOfInt32(8080))

	err = mock.Conn().Invoke(t.Context(),
		"/nested.ConfigurationService/UpdateConfig", in, out)
	require.NoError(t, err)

	msgField := outDesc.(protoreflect.MessageDescriptor).Fields().ByName("message")
	require.Equal(t, "config localhost:8080", out.Get(msgField).String())
}

func TestRunDynamicTemplateWithMultipleTypes(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("types"))

	err := mock.Stub(By("/types.TypeService/GetAllTypes")).
		When(Equals("name", "test")).
		Reply(Data("string_field", "hello {{.Request.name}}", "int32_field", 42, "bool_field", true)).
		Commit()
	require.NoError(t, err)

	inDesc, _ := reg.FindDescriptorByName("types.AllTypesRequest")
	outDesc, _ := reg.FindDescriptorByName("types.AllTypesResponse")
	in := dynamicpb.NewMessage(inDesc.(protoreflect.MessageDescriptor))
	in.Set(inDesc.(protoreflect.MessageDescriptor).Fields().ByName("name"), protoreflect.ValueOfString("test"))
	out := dynamicpb.NewMessage(outDesc.(protoreflect.MessageDescriptor))

	err = mock.Conn().Invoke(t.Context(),
		"/types.TypeService/GetAllTypes", in, out)
	require.NoError(t, err)

	strField := outDesc.(protoreflect.MessageDescriptor).Fields().ByName("string_field")
	require.Equal(t, "hello test", out.Get(strField).String())

	intField := outDesc.(protoreflect.MessageDescriptor).Fields().ByName("int32_field")
	require.Equal(t, int64(42), out.Get(intField).Int())

	boolField := outDesc.(protoreflect.MessageDescriptor).Fields().ByName("bool_field")
	require.True(t, out.Get(boolField).Bool())
}

// Section 4: Session isolation

func TestRunSessionIsolation(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"), WithSession("session-A"))

	err := mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "Alex")).
		Reply(Data("message", "session A Alex")).
		Commit()
	require.NoError(t, err)

	msg := invokeGreeterSayHello(t, mock.Conn(), reg, "Alex")
	require.Equal(t, "session A Alex", getMessageField(t, msg))
}

func TestRunSessionIsolationSeparateContexts(t *testing.T) {
	t.Parallel()

	mockA, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"), WithSession("session-B"))

	err := mockA.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "Alex")).
		Reply(Data("message", "session B Alex")).
		Commit()
	require.NoError(t, err)

	mockB, reg2 := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	err = mockB.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "Alex")).
		Reply(Data("message", "global Alex")).
		Commit()
	require.NoError(t, err)

	msg := invokeGreeterSayHello(t, mockA.Conn(), reg, "Alex")
	require.Equal(t, "session B Alex", getMessageField(t, msg))

	msg2 := invokeGreeterSayHello(t, mockB.Conn(), reg2, "Alex")
	require.Equal(t, "global Alex", getMessageField(t, msg2))
}

// Section 5: Merge with Glob

func TestRunMergeGlobAndEquals(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	err := mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Merge(Glob("name", "A*"), Equals("name", "Alex"))).
		Reply(Data("message", "merge glob+equals")).
		Commit()
	require.NoError(t, err)

	msg := invokeGreeterSayHello(t, mock.Conn(), reg, "Alex")
	require.Equal(t, "merge glob+equals", getMessageField(t, msg))
}

func TestRunMergeHeadersGlob(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	err := mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "Alex")).
		WhenHeaders(Merge(
			HeaderEquals("x-version", "v2"),
			HeaderGlob("x-request-id", "req-*"),
		)).
		Reply(Data("message", "header glob matched")).
		Commit()
	require.NoError(t, err)

	ctx := metadata.NewOutgoingContext(t.Context(),
		metadata.Pairs("x-version", "v2", "x-request-id", "req-abc-123"))

	outDesc, _ := reg.FindDescriptorByName("helloworld.HelloReply")
	out := dynamicpb.NewMessage(outDesc.(protoreflect.MessageDescriptor))
	err = mock.Conn().Invoke(ctx, "/helloworld.Greeter/SayHello",
		createGreeterRequest(t, reg, "Alex"), out)
	require.NoError(t, err)
	require.Equal(t, "header glob matched", getMessageField(t, out))
}

func TestRunMergeHeadersGlobNoMatch(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	err := mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "Alex")).
		WhenHeaders(Merge(
			HeaderEquals("x-version", "v2"),
			HeaderGlob("x-request-id", "req-*"),
		)).
		Reply(Data("message", "header glob matched")).
		Commit()
	require.NoError(t, err)

	ctx := metadata.NewOutgoingContext(t.Context(),
		metadata.Pairs("x-version", "v2", "x-request-id", "other-xyz"))
	outDesc, _ := reg.FindDescriptorByName("helloworld.HelloReply")
	out := dynamicpb.NewMessage(outDesc.(protoreflect.MessageDescriptor))
	err = mock.Conn().Invoke(ctx, "/helloworld.Greeter/SayHello",
		createGreeterRequest(t, reg, "Alex"), out)
	require.Error(t, err)
}

func TestRunMergeHeadersContains(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	err := mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "Alex")).
		WhenHeaders(Merge(
			HeaderEquals("x-version", "v1"),
			HeaderEquals("x-trace", "trace-abc"),
		)).
		Reply(Data("message", "Hi {{.Request.name}} with v1")).
		Commit()
	require.NoError(t, err)

	ctx := metadata.NewOutgoingContext(t.Context(),
		metadata.Pairs("x-version", "v1", "x-trace", "trace-abc"))

	outDesc, _ := reg.FindDescriptorByName("helloworld.HelloReply")
	out := dynamicpb.NewMessage(outDesc.(protoreflect.MessageDescriptor))
	err = mock.Conn().Invoke(ctx, "/helloworld.Greeter/SayHello",
		createGreeterRequest(t, reg, "Alex"), out)
	require.NoError(t, err)
	require.Equal(t, "Hi Alex with v1", getMessageField(t, out))
}

// Section 6: Priority with fallback

func TestRunPriorityFallbackToGlob(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	err := mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "Alex")).
		Priority(100).
		Reply(Data("message", "exact Alex")).
		Commit()
	require.NoError(t, err)

	err = mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Glob("name", "A*")).
		Priority(1).
		Reply(Data("message", "fallback glob")).
		Commit()
	require.NoError(t, err)

	msg := invokeGreeterSayHello(t, mock.Conn(), reg, "Alice")
	require.Equal(t, "fallback glob", getMessageField(t, msg))
}

// Section 7: Edge cases

func TestRunEqualsAndGlobOnSameField(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	err := mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Merge(Equals("name", "Alex"), Glob("name", "Al*"))).
		Reply(Data("message", "both match")).
		Commit()
	require.NoError(t, err)

	msg := invokeGreeterSayHello(t, mock.Conn(), reg, "Alex")
	require.Equal(t, "both match", getMessageField(t, msg))
}

func TestRunMergeIgnoreArrayOrderWithGlob(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	err := mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(IgnoreArrayOrder(Glob("name", "A*"))).
		Reply(Data("message", "ignore order glob")).
		Commit()
	require.NoError(t, err)

	msg := invokeGreeterSayHello(t, mock.Conn(), reg, "Alex")
	require.Equal(t, "ignore order glob", getMessageField(t, msg))
}

func TestRunDynamicTemplateWithReplyHeaders(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	err := mock.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "Alex")).
		Reply(Merge(
			Data("message", "Hello with headers"),
			ReplyHeader("x-greeting", "custom"),
		)).
		Commit()
	require.NoError(t, err)

	ctx := t.Context()
	outDesc, _ := reg.FindDescriptorByName("helloworld.HelloReply")
	msg := dynamicpb.NewMessage(outDesc.(protoreflect.MessageDescriptor))
	header := metadata.MD{}
	err = mock.Conn().Invoke(ctx, "/helloworld.Greeter/SayHello",
		createGreeterRequest(t, reg, "Alex"), msg, grpc.Header(&header))
	require.NoError(t, err)
	require.Equal(t, "Hello with headers", getMessageField(t, msg))
	require.Equal(t, []string{"custom"}, header["x-greeting"])
}

func TestRunDynamicTemplateWithInvalidField(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("types"))

	err := mock.Stub(By("/types.TypeService/GetAllTypes")).
		When(Equals("name", "tpl")).
		Reply(Data("string_field", "value: {{.Request.missing_field}}")).
		Commit()
	require.NoError(t, err)

	inDesc, _ := reg.FindDescriptorByName("types.AllTypesRequest")
	outDesc, _ := reg.FindDescriptorByName("types.AllTypesResponse")
	in := dynamicpb.NewMessage(inDesc.(protoreflect.MessageDescriptor))
	in.Set(inDesc.(protoreflect.MessageDescriptor).Fields().ByName("name"), protoreflect.ValueOfString("tpl"))
	out := dynamicpb.NewMessage(outDesc.(protoreflect.MessageDescriptor))

	err = mock.Conn().Invoke(t.Context(),
		"/types.TypeService/GetAllTypes", in, out)
	require.NoError(t, err)

	strField := outDesc.(protoreflect.MessageDescriptor).Fields().ByName("string_field")
	val := out.Get(strField).String()
	require.Contains(t, val, "no value",
		"expected template to render missing field as '<no value>', got: %s", val)
}

// TestRunStreamWithReg tests streaming using mustRunWithProtoAndReg + dynamicpb.
func TestRunStreamWithReg(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("search"))

	err := mock.Stub(By("/search.SearchService/Search")).
		When(Equals("query", "test-stream")).
		ReplyStream(
			Data("id", "1", "title", "result 1", "relevance_score", 0.95),
			Data("id", "2", "title", "result 2", "relevance_score", 0.85),
		).
		Commit()
	require.NoError(t, err)

	inDesc, _ := reg.FindDescriptorByName("search.SearchRequest")
	outDesc, _ := reg.FindDescriptorByName("search.SearchResult")
	in := dynamicpb.NewMessage(inDesc.(protoreflect.MessageDescriptor))
	queryFd := inDesc.(protoreflect.MessageDescriptor).Fields().ByName("query")
	in.Set(queryFd, protoreflect.ValueOfString("test-stream"))

	stream, err := mock.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{
			StreamName:    "Search",
			ServerStreams: true,
			ClientStreams: false,
		}, "/search.SearchService/Search")
	require.NoError(t, err)

	err = stream.SendMsg(in)
	require.NoError(t, err)

	out1 := dynamicpb.NewMessage(outDesc.(protoreflect.MessageDescriptor))
	err = stream.RecvMsg(out1)
	require.NoError(t, err)
	titleFd := outDesc.(protoreflect.MessageDescriptor).Fields().ByName("title")
	require.Equal(t, "result 1", out1.Get(titleFd).String())

	out2 := dynamicpb.NewMessage(outDesc.(protoreflect.MessageDescriptor))
	err = stream.RecvMsg(out2)
	require.NoError(t, err)
	require.Equal(t, "result 2", out2.Get(titleFd).String())

	err = stream.RecvMsg(&dynamicpb.Message{})
	require.Equal(t, io.EOF, err)
}
