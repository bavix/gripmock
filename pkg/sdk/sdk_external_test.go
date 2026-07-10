package sdk_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bufbuild/protocompile"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	sdk "github.com/bavix/gripmock/v3/pkg/sdk"
)

const testProto = `
syntax = "proto3";
package test;
service Greeter {
  rpc SayHello (HelloRequest) returns (HelloReply);
}
message HelloRequest { string name = 1; }
message HelloReply { string message = 1; }
`

const searchProto = `
syntax = "proto3";
package search;
service SearchService {
  rpc Search (SearchRequest) returns (stream SearchResult);
}
message SearchRequest { string query = 1; }
message SearchResult { string id = 1; string title = 2; }
`

const calcProto = `
syntax = "proto3";
package calc;
service Calculator {
  rpc SumNumbers (stream NumberRequest) returns (SumResponse);
}
message NumberRequest { double value = 1; }
message SumResponse { double result = 1; int32 count = 2; }
`

const chatProto = `
syntax = "proto3";
package chat;
service ChatService {
  rpc Chat (stream ChatMessage) returns (stream ChatMessage);
}
message ChatMessage { string text = 1; }
`

const nestedProto = `
syntax = "proto3";
package nested;
service ConfigService {
  rpc GetConfig (ConfigRequest) returns (ConfigResponse);
}
message ConfigRequest { string env = 1; }
message ConfigResponse {
  message Settings {
    string host = 1;
    int32 port = 2;
  }
  Settings settings = 1;
  string status = 2;
}
`

const wktProto = `
syntax = "proto3";
package wkt;
import "google/protobuf/timestamp.proto";
import "google/protobuf/duration.proto";
service TypeService {
  rpc GetTypes (TypeRequest) returns (TypeResponse);
}
message TypeRequest { string name = 1; }
message TypeResponse {
  google.protobuf.Timestamp ts = 1;
  google.protobuf.Duration dur = 2;
  string message = 3;
}
`

const workflowProto = `
syntax = "proto3";
package workflow;
service Workflow {
  rpc Start (StartRequest) returns (StartResponse);
  rpc Next (NextRequest) returns (NextResponse);
}
message StartRequest { string step = 1; }
message StartResponse { string status = 1; }
message NextRequest { string step = 1; }
message NextResponse { string status = 1; }
`

var errNotServerStream = errors.New("stream is not a ServerStream")

func compileInline(t *testing.T, source, name string) *descriptorpb.FileDescriptorSet {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(source), 0o600))

	compiler := protocompile.Compiler{
		Resolver: protocompile.CompositeResolver{
			&protocompile.SourceResolver{ImportPaths: []string{dir}},
			protocompile.WithStandardImports(&protocompile.SourceResolver{}),
		},
	}
	files, err := compiler.Compile(context.Background(), path)
	require.NoError(t, err)

	fds := &descriptorpb.FileDescriptorSet{}
	seen := map[string]bool{}

	for _, f := range files {
		fdp := protodesc.ToFileDescriptorProto(f)
		seen[fdp.GetName()] = true
		fds.File = append(fds.File, fdp)
	}
	// Add imported files from GlobalFiles (for WKT support)
	for _, f := range files {
		fdp := protodesc.ToFileDescriptorProto(f)
		for _, dep := range fdp.GetDependency() {
			if seen[dep] {
				continue
			}

			if fd, err := protoregistry.GlobalFiles.FindFileByPath(dep); err == nil {
				depFdp := protodesc.ToFileDescriptorProto(fd)
				seen[dep] = true

				fds.File = append(fds.File, depFdp)
			}
		}
	}

	return fds
}

type msgDesc struct {
	in  protoreflect.MessageDescriptor
	out protoreflect.MessageDescriptor
}

func resolveDesc(t *testing.T, fds *descriptorpb.FileDescriptorSet, inName, outName protoreflect.FullName) msgDesc {
	t.Helper()

	reg, err := protodesc.NewFiles(fds)
	require.NoError(t, err)
	in, err := reg.FindDescriptorByName(inName)
	require.NoError(t, err)
	out, err := reg.FindDescriptorByName(outName)
	require.NoError(t, err)

	inDesc, ok := in.(protoreflect.MessageDescriptor)
	require.True(t, ok, "in is not a MessageDescriptor")
	outDesc, ok := out.(protoreflect.MessageDescriptor)
	require.True(t, ok, "out is not a MessageDescriptor")

	return msgDesc{in: inDesc, out: outDesc}
}

func newServer(t *testing.T, opts ...sdk.Option) (*sdk.Server, *descriptorpb.FileDescriptorSet) {
	t.Helper()
	fds := compileInline(t, testProto, "test.proto")
	all := append([]sdk.Option{sdk.WithDescriptors(fds)}, opts...)

	return sdk.NewServer(t, all...), fds
}

func newServerSearch(t *testing.T) (*sdk.Server, *descriptorpb.FileDescriptorSet) {
	t.Helper()
	t.Helper()
	fds := compileInline(t, searchProto, "search.proto")

	return sdk.NewServer(t, sdk.WithDescriptors(fds)), fds
}

func newServerCalc(t *testing.T) (*sdk.Server, *descriptorpb.FileDescriptorSet) {
	t.Helper()
	t.Helper()
	fds := compileInline(t, calcProto, "calc.proto")

	return sdk.NewServer(t, sdk.WithDescriptors(fds)), fds
}

func newServerChat(t *testing.T) (*sdk.Server, *descriptorpb.FileDescriptorSet) {
	t.Helper()
	t.Helper()
	fds := compileInline(t, chatProto, "chat.proto")

	return sdk.NewServer(t, sdk.WithDescriptors(fds)), fds
}

func newServerWorkflow(t *testing.T) (*sdk.Server, *descriptorpb.FileDescriptorSet) {
	t.Helper()
	fds := compileInline(t, workflowProto, "workflow.proto")

	return sdk.NewServer(t, sdk.WithDescriptors(fds)), fds
}

func newServerWKT(t *testing.T) (*sdk.Server, *descriptorpb.FileDescriptorSet) {
	t.Helper()
	fds := compileInline(t, wktProto, "wkt.proto")

	return sdk.NewServer(t, sdk.WithDescriptors(fds)), fds
}

func newServerNested(t *testing.T) (*sdk.Server, *descriptorpb.FileDescriptorSet) {
	t.Helper()
	fds := compileInline(t, nestedProto, "nested.proto")

	return sdk.NewServer(t, sdk.WithDescriptors(fds)), fds
}

func sayHello(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet, name string) *dynamicpb.Message {
	t.Helper()
	d := resolveDesc(t, fds, "test.HelloRequest", "test.HelloReply")
	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("name"), protoreflect.ValueOfString(name))
	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, srv.Conn().Invoke(t.Context(), "/test.Greeter/SayHello", in, out))

	return out
}

func sayHelloErr(t *testing.T, srv *sdk.Server, fds *descriptorpb.FileDescriptorSet, name string) error {
	t.Helper()
	d := resolveDesc(t, fds, "test.HelloRequest", "test.HelloReply")
	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("name"), protoreflect.ValueOfString(name))
	out := dynamicpb.NewMessage(d.out)

	return srv.Conn().Invoke(t.Context(), "/test.Greeter/SayHello", in, out)
}

func getMsg(t *testing.T, msg *dynamicpb.Message) string {
	t.Helper()

	return msg.Get(msg.Descriptor().Fields().ByName("message")).String()
}

func mustBuildRegFromFDS(t *testing.T, fds *descriptorpb.FileDescriptorSet) *protoregistry.Files {
	t.Helper()

	reg, err := protodesc.NewFiles(fds)
	require.NoError(t, err)

	return reg
}

func TestNewServer(t *testing.T) {
	t.Parallel()

	srv, _ := newServer(t)
	defer func() { _ = srv.Close() }()

	require.NotNil(t, srv.Conn())
	require.Contains(t, srv.Address(), "127.0.0.1:")
}

func TestNewServer_Panics(t *testing.T) {
	t.Parallel()
	require.Panics(t, func() { sdk.NewServer(nil) })
	require.Panics(t, func() { sdk.NewServer(t) })
}

func TestNewServer_WithHealthTimeout(t *testing.T) {
	t.Parallel()

	srv, _ := newServer(t, sdk.WithHealthCheckTimeout(5*time.Second))
	defer func() { _ = srv.Close() }()

	require.NotNil(t, srv.Conn())
}

func TestUnaryMatchReturn(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "Alex").
		Return("message", "Hi Alex")

	msg := sayHello(t, srv, fds, "Alex")
	require.Equal(t, "Hi Alex", getMsg(t, msg))
}

func TestUnaryTemplates(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "Alex").
		Return("message", "Hi {{.Request.name}}")

	msg := sayHello(t, srv, fds, "Alex")
	require.Equal(t, "Hi Alex", getMsg(t, msg))
}

func TestUnaryError(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "error").
		ReturnError(codes.NotFound, "user not found")

	err := sayHelloErr(t, srv, fds, "error")
	require.Error(t, err)
	require.Contains(t, err.Error(), "user not found")
}

func TestUnaryTimes(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "limited").
		Times(2).
		Return("message", "ok")

	require.Equal(t, "ok", getMsg(t, sayHello(t, srv, fds, "limited")))
	require.Equal(t, "ok", getMsg(t, sayHello(t, srv, fds, "limited")))
	require.Error(t, sayHelloErr(t, srv, fds, "limited"))
}

func TestUnaryPriority(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "Alex").
		Priority(100).
		Return("message", "exact")

	srv.ExpectUnary("/test.Greeter/SayHello").
		WithPayloadMap(sdk.Matches("name", "A.*")).
		Priority(1).
		Return("message", "fallback")

	msg := sayHello(t, srv, fds, "Alex")
	require.Equal(t, "exact", getMsg(t, msg))
}

func TestUnaryNextWillReturn(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "chain").
		Return("message", "first").
		NextWillReturn("message", "second").
		NextWillReturn("message", "third")

	require.Equal(t, "first", getMsg(t, sayHello(t, srv, fds, "chain")))
	require.Equal(t, "second", getMsg(t, sayHello(t, srv, fds, "chain")))
	require.Equal(t, "third", getMsg(t, sayHello(t, srv, fds, "chain")))
	require.Error(t, sayHelloErr(t, srv, fds, "chain"))
}

func TestVerification(t *testing.T) {
	t.Parallel()
	srv, fds := newServer(t)

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "Alex").
		Times(1).
		Return("message", "Hi")

	_ = sayHello(t, srv, fds, "Alex")
	require.NoError(t, srv.ExpectationsWereMet())
	require.NoError(t, srv.ExpectationsWereMet())
	_ = srv.Close()
}

func TestVerificationFailed(t *testing.T) {
	t.Parallel()
	srv, fds := newServer(t)

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "Alex").
		Times(2).
		Return("message", "Hi")

	_ = sayHello(t, srv, fds, "Alex")

	err := srv.ExpectationsWereMet()
	require.Error(t, err)
	require.ErrorIs(t, err, sdk.ErrVerificationFailed)

	var notMet *sdk.ExpectationNotMetError
	require.ErrorAs(t, err, &notMet)
	require.Equal(t, "test.Greeter", notMet.Service)
	require.Equal(t, "SayHello", notMet.Method)
	require.Equal(t, 2, notMet.Expected)
	require.Equal(t, 1, notMet.Actual)

	_ = srv.Close()
}

func TestServerCalled(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "Alex").
		Return("message", "Hi")

	require.Equal(t, 0, srv.TotalCalls())
	_ = sayHello(t, srv, fds, "Alex")
	require.Equal(t, 1, srv.Called("/test.Greeter/SayHello"))
	require.Equal(t, 1, srv.TotalCalls())
	require.Len(t, srv.History(), 1)
}

func TestServerReset(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "Alex").
		Return("message", "Hi")

	_ = sayHello(t, srv, fds, "Alex")
	require.Equal(t, 1, srv.TotalCalls())

	srv.Reset()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "Bob").
		Return("message", "Hello Bob")

	msg := sayHello(t, srv, fds, "Bob")
	require.Equal(t, "Hello Bob", getMsg(t, msg))
}

func TestServerStream(t *testing.T) {
	t.Parallel()

	srv, fds := newServerSearch(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "test").
		SendStream(map[string]any{"id": "1", "title": "result 1"})

	d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")
	stream, err := srv.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "Search", ServerStreams: true, ClientStreams: false},
		"/search.SearchService/Search")
	require.NoError(t, err)

	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("query"), protoreflect.ValueOfString("test"))
	require.NoError(t, stream.SendMsg(in))

	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, stream.RecvMsg(out))
	require.Equal(t, "result 1", out.Get(d.out.Fields().ByName("title")).String())
}

func TestServerStreamEmpty(t *testing.T) {
	t.Parallel()

	srv, fds := newServerSearch(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "empty").
		SendStream()

	d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")
	stream, err := srv.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "Search", ServerStreams: true, ClientStreams: false},
		"/search.SearchService/Search")
	require.NoError(t, err)

	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("query"), protoreflect.ValueOfString("empty"))
	require.NoError(t, stream.SendMsg(in))

	out := dynamicpb.NewMessage(d.out)
	err = stream.RecvMsg(out)
	require.ErrorIs(t, err, io.EOF)
}

func TestClientStreamFirstPayload(t *testing.T) {
	t.Parallel()

	srv, fds := newServerCalc(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectClientStream("/calc.Calculator/SumNumbers").
		WithFirstPayload(sdk.Matches("value", "\\d+")).
		Return("result", 99.0, "count", 1)

	d := resolveDesc(t, fds, "calc.NumberRequest", "calc.SumResponse")
	stream, err := srv.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "SumNumbers", ServerStreams: false, ClientStreams: true},
		"/calc.Calculator/SumNumbers")
	require.NoError(t, err)

	valFd := d.in.Fields().ByName("value")
	msg := dynamicpb.NewMessage(d.in)
	msg.Set(valFd, protoreflect.ValueOfFloat64(7.0))
	require.NoError(t, stream.SendMsg(msg))
	require.NoError(t, stream.CloseSend())

	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, stream.RecvMsg(out))
	require.InDelta(t, 99.0, out.Get(d.out.Fields().ByName("result")).Float(), 0.001)
	require.Equal(t, int64(1), out.Get(d.out.Fields().ByName("count")).Int())
}

func TestClientStream(t *testing.T) {
	t.Parallel()

	srv, fds := newServerCalc(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectClientStream("/calc.Calculator/SumNumbers").
		WithPayloadMap(sdk.Matches("value", "\\d+")).
		Return("result", 42.0, "count", 2)

	d := resolveDesc(t, fds, "calc.NumberRequest", "calc.SumResponse")
	stream, err := srv.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "SumNumbers", ServerStreams: false, ClientStreams: true},
		"/calc.Calculator/SumNumbers")
	require.NoError(t, err)

	valFd := d.in.Fields().ByName("value")
	for _, v := range []float64{1.0, 2.0} {
		msg := dynamicpb.NewMessage(d.in)
		msg.Set(valFd, protoreflect.ValueOfFloat64(v))
		require.NoError(t, stream.SendMsg(msg))
	}

	require.NoError(t, stream.CloseSend())

	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, stream.RecvMsg(out))
	require.InDelta(t, 42.0, out.Get(d.out.Fields().ByName("result")).Float(), 0.001)
}

func TestBidiStream(t *testing.T) {
	t.Parallel()

	srv, fds := newServerChat(t)
	defer func() { _ = srv.Close() }()

	d := resolveDesc(t, fds, "chat.ChatMessage", "chat.ChatMessage")

	srv.ExpectBidirectionalStream("/chat.ChatService/Chat").
		Run(func(ctx context.Context, stream any) error {
			srvStream, ok := stream.(grpc.ServerStream)
			if !ok {
				return errNotServerStream
			}

			msg := dynamicpb.NewMessage(d.in)
			if err := srvStream.RecvMsg(msg); err != nil {
				return err
			}

			return nil
		})

	stream, err := srv.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "Chat", ServerStreams: true, ClientStreams: true},
		"/chat.ChatService/Chat")
	require.NoError(t, err)

	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("text"), protoreflect.ValueOfString("hello"))
	require.NoError(t, stream.SendMsg(in))
	require.NoError(t, stream.CloseSend())
}

func TestBidiStreamWithHandler(t *testing.T) {
	t.Parallel()

	srv, fds := newServerChat(t)
	defer func() { _ = srv.Close() }()

	d := resolveDesc(t, fds, "chat.ChatMessage", "chat.ChatMessage")
	received := make(chan string, 1)

	srv.ExpectBidirectionalStream("/chat.ChatService/Chat").
		Run(func(ctx context.Context, stream any) error {
			s, ok := stream.(grpc.ServerStream)
			if !ok {
				return errNotServerStream
			}

			msg := dynamicpb.NewMessage(d.in)
			if err := s.RecvMsg(msg); err != nil {
				received <- ""

				return nil //nolint:nilerr
			}

			fd := msg.Descriptor().Fields().ByName("text")
			received <- msg.Get(fd).String()

			return nil
		})

	stream, err := srv.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "Chat", ServerStreams: true, ClientStreams: true},
		"/chat.ChatService/Chat")
	require.NoError(t, err)

	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("text"), protoreflect.ValueOfString("handler-works"))
	require.NoError(t, stream.SendMsg(in))
	require.NoError(t, stream.CloseSend())

	select {
	case msg := <-received:
		require.Equal(t, "handler-works", msg)
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not receive message")
	}
}

func TestWorkflowUnary(t *testing.T) {
	t.Parallel()

	srv, fds := newServerWorkflow(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/workflow.Workflow/Start").
		Match("step", "begin").
		Return("status", "started")

	d := resolveDesc(t, fds, "workflow.StartRequest", "workflow.StartResponse")
	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("step"), protoreflect.ValueOfString("begin"))
	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, srv.Conn().Invoke(t.Context(), "/workflow.Workflow/Start", in, out))
	require.Equal(t, "started", out.Get(d.out.Fields().ByName("status")).String())
}

func TestStreamNextWillReturn(t *testing.T) {
	t.Parallel()

	srv, fds := newServerSearch(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "chain").
		SendStream(map[string]any{"id": "1", "title": "first"}).
		NextWillReturn("id", "2", "title", "second")

	d := resolveDesc(t, fds, "search.SearchRequest", "search.SearchResult")

	stream, err := srv.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "Search", ServerStreams: true, ClientStreams: false},
		"/search.SearchService/Search")
	require.NoError(t, err)

	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("query"), protoreflect.ValueOfString("chain"))
	require.NoError(t, stream.SendMsg(in))

	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, stream.RecvMsg(out))
	require.Equal(t, "first", out.Get(d.out.Fields().ByName("title")).String())

	stream2, err := srv.Conn().NewStream(t.Context(),
		&grpc.StreamDesc{StreamName: "Search", ServerStreams: true, ClientStreams: false},
		"/search.SearchService/Search")
	require.NoError(t, err)

	in2 := dynamicpb.NewMessage(d.in)
	in2.Set(d.in.Fields().ByName("query"), protoreflect.ValueOfString("chain"))
	require.NoError(t, stream2.SendMsg(in2))

	out2 := dynamicpb.NewMessage(d.out)
	require.NoError(t, stream2.RecvMsg(out2))
	require.Equal(t, "second", out2.Get(d.out.Fields().ByName("title")).String())
}

func TestMatchersExist(t *testing.T) {
	t.Parallel()

	require.NotNil(t, sdk.Equals("k", "v"))
	require.NotNil(t, sdk.Contains("k", "v"))
	require.NotNil(t, sdk.Matches("k", "v.*"))
	require.NotNil(t, sdk.Merge(sdk.Equals("a", 1), sdk.Contains("b", "2")))
	require.NotNil(t, sdk.IgnoreArrayOrder(sdk.Equals("a", []int{1, 2})))
}

func TestWKTWithTimestamp(t *testing.T) {
	t.Parallel()

	srv, fds := newServerWKT(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/wkt.TypeService/GetTypes").
		Match("name", "wkt-test").
		Return("message", "WKT works")

	d := resolveDesc(t, fds, "wkt.TypeRequest", "wkt.TypeResponse")
	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("name"), protoreflect.ValueOfString("wkt-test"))
	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, srv.Conn().Invoke(t.Context(), "/wkt.TypeService/GetTypes", in, out))
	require.Equal(t, "WKT works", out.Get(d.out.Fields().ByName("message")).String())
}

func TestNestedMessages(t *testing.T) {
	t.Parallel()

	srv, fds := newServerNested(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/nested.ConfigService/GetConfig").
		Match("env", "prod").
		Return("status", "configured")

	d := resolveDesc(t, fds, "nested.ConfigRequest", "nested.ConfigResponse")
	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("env"), protoreflect.ValueOfString("prod"))
	out := dynamicpb.NewMessage(d.out)
	require.NoError(t, srv.Conn().Invoke(t.Context(), "/nested.ConfigService/GetConfig", in, out))
	require.Equal(t, "configured", out.Get(d.out.Fields().ByName("status")).String())
}

func TestHealthCheckStubbable(t *testing.T) {
	t.Parallel()

	fds := compileInline(t, testProto, "test.proto")

	srv := sdk.NewServer(t, sdk.WithDescriptors(fds))
	defer func() { _ = srv.Close() }()

	// Health is built-in. Just verify we can register expectation for it.
	srv.ExpectUnary("/grpc.health.v1.Health/Check").
		Match("service", "my-svc").
		Return("status", "SERVING")
}

const extendedProto = `
syntax = "proto3";
package extended;
service FinanceService {
  rpc GetBalance (BalanceRequest) returns (BalanceResponse);
}
message BalanceRequest { string account_id = 1; }
message BalanceResponse {
  message Money {
    string currency_code = 1;
    int64 units = 2;
    int32 nanos = 3;
  }
  Money balance = 1;
  string status = 2;
}
`

func TestExtendedTypes(t *testing.T) {
	t.Parallel()

	fds := compileInline(t, extendedProto, "extended.proto")

	srv := sdk.NewServer(t, sdk.WithDescriptors(fds))
	defer func() { _ = srv.Close() }()

	// Match on account_id, return nested Money
	srv.ExpectUnary("/extended.FinanceService/GetBalance").
		Match("account_id", "acc-42").
		Return("balance", map[string]any{
			"currency_code": "USD",
			"units":         float64(100),
			"nanos":         float64(50),
		}, "status", "active")

	reg := mustBuildRegFromFDS(t, fds)
	inDesc, _ := reg.FindDescriptorByName("extended.BalanceRequest")
	outDesc, _ := reg.FindDescriptorByName("extended.BalanceResponse")

	require.NotNil(t, inDesc)
	require.NotNil(t, outDesc)

	inMsgDesc, ok := inDesc.(protoreflect.MessageDescriptor)
	require.True(t, ok)
	outMsgDesc, ok := outDesc.(protoreflect.MessageDescriptor)
	require.True(t, ok)

	in := dynamicpb.NewMessage(inMsgDesc)
	accFd := inMsgDesc.Fields().ByName("account_id")
	in.Set(accFd, protoreflect.ValueOfString("acc-42"))

	out := dynamicpb.NewMessage(outMsgDesc)
	err := srv.Conn().Invoke(t.Context(), "/extended.FinanceService/GetBalance", in, out)
	require.NoError(t, err)

	// Check balance fields
	balFd := outMsgDesc.Fields().ByName("balance")
	balMsg := out.Get(balFd).Message()
	require.NotNil(t, balMsg)

	moneyDesc := balMsg.Descriptor()
	ccFd := moneyDesc.Fields().ByName("currency_code")
	require.Equal(t, "USD", balMsg.Get(ccFd).String())

	unitsFd := moneyDesc.Fields().ByName("units")
	require.Equal(t, int64(100), balMsg.Get(unitsFd).Int())

	statusFd := outMsgDesc.Fields().ByName("status")
	require.Equal(t, "active", out.Get(statusFd).String())
}

func TestStubID(t *testing.T) {
	t.Parallel()

	t.Run("unary", func(t *testing.T) {
		t.Parallel()

		srv, _ := newServer(t)
		defer func() { _ = srv.Close() }()

		e := srv.ExpectUnary("/test.Greeter/SayHello").
			Match("name", "Alex").
			Return("message", "Hi")

		require.NotEmpty(t, e.StubID())
	})

	t.Run("server_stream", func(t *testing.T) {
		t.Parallel()

		srv, _ := newServerSearch(t)
		defer func() { _ = srv.Close() }()

		exp := srv.ExpectServerStream("/search.SearchService/Search").
			Match("query", "test")
		exp.SendStream(map[string]any{"id": "1"})

		require.NotEmpty(t, exp.StubID())
	})

	t.Run("client_stream", func(t *testing.T) {
		t.Parallel()

		srv, _ := newServerSearch(t)
		defer func() { _ = srv.Close() }()

		e := srv.ExpectClientStream("/search.SearchService/Search").
			Match(sdk.Matches("value", "\\d+")).
			Return("result", 42.0)

		require.NotEmpty(t, e.StubID())
	})

	t.Run("bidi", func(t *testing.T) {
		t.Parallel()

		srv, _ := newServerChat(t)
		defer func() { _ = srv.Close() }()

		e := srv.ExpectBidirectionalStream("/chat.ChatService/Chat").
			Run(func(ctx context.Context, stream any) error {
				return nil
			})

		require.NotEmpty(t, e.StubID())
	})
}

func TestOnceTwice(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "once").
		Once().
		Return("message", "once")

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "twice").
		Twice().
		Return("message", "twice")

	// Once: first call succeeds, second fails
	msg1 := sayHello(t, srv, fds, "once")
	require.Equal(t, "once", getMsg(t, msg1))
	require.Error(t, sayHelloErr(t, srv, fds, "once"))

	// Twice: two calls succeed, third fails
	msg2 := sayHello(t, srv, fds, "twice")
	require.Equal(t, "twice", getMsg(t, msg2))
	msg3 := sayHello(t, srv, fds, "twice")
	require.Equal(t, "twice", getMsg(t, msg3))
	require.Error(t, sayHelloErr(t, srv, fds, "twice"))
}

func TestMatchers(t *testing.T) {
	t.Parallel()

	t.Run("glob", func(t *testing.T) {
		t.Parallel()

		srv, fds := newServer(t)
		defer func() { _ = srv.Close() }()

		srv.ExpectUnary("/test.Greeter/SayHello").
			Match(sdk.Glob("name", "A*")).
			Return("message", "glob")

		msg := sayHello(t, srv, fds, "Alex")
		require.Equal(t, "glob", getMsg(t, msg))
	})

	t.Run("anyof", func(t *testing.T) {
		t.Parallel()

		srv, fds := newServer(t)
		defer func() { _ = srv.Close() }()

		srv.ExpectUnary("/test.Greeter/SayHello").
			Match(sdk.AnyOf(
				sdk.Equals("name", "Alex"),
				sdk.Equals("name", "Bob"),
			)).
			Return("message", "anyof")

		require.Equal(t, "anyof", getMsg(t, sayHello(t, srv, fds, "Alex")))
		require.Equal(t, "anyof", getMsg(t, sayHello(t, srv, fds, "Bob")))
	})
}

func TestReturnFormats(t *testing.T) {
	t.Parallel()

	t.Run("proto", func(t *testing.T) {
		t.Parallel()

		srv, fds := newServer(t)
		defer func() { _ = srv.Close() }()

		d := resolveDesc(t, fds, "test.HelloRequest", "test.HelloReply")
		reply := dynamicpb.NewMessage(d.out)
		reply.Set(d.out.Fields().ByName("message"), protoreflect.ValueOfString("proto-reply"))

		srv.ExpectUnary("/test.Greeter/SayHello").
			Match("name", "proto").
			ReturnProto(reply)

		msg := sayHello(t, srv, fds, "proto")
		require.Equal(t, "proto-reply", getMsg(t, msg))
	})

	t.Run("json", func(t *testing.T) {
		t.Parallel()

		srv, fds := newServer(t)
		defer func() { _ = srv.Close() }()

		srv.ExpectUnary("/test.Greeter/SayHello").
			Match("name", "json").
			ReturnJSON(map[string]any{"message": "json-reply"})

		msg := sayHello(t, srv, fds, "json")
		require.Equal(t, "json-reply", getMsg(t, msg))
	})
}

func TestNextWillReturnSequences(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	// Test: first 2 calls error, 3rd succeeds
	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "retry").
		ReturnError(codes.Unavailable, "try again").
		NextWillReturnError(codes.Unavailable, "one more").
		NextWillReturn("message", "success")

	require.Error(t, sayHelloErr(t, srv, fds, "retry"))
	require.Error(t, sayHelloErr(t, srv, fds, "retry"))
	msg := sayHello(t, srv, fds, "retry")
	require.Equal(t, "success", getMsg(t, msg))
}

func TestWithHeaderMatch(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "auth").
		WithHeader(sdk.Equals("token", "secret")).
		Return("message", "authorized")

	d := resolveDesc(t, fds, "test.HelloRequest", "test.HelloReply")
	in := dynamicpb.NewMessage(d.in)
	in.Set(d.in.Fields().ByName("name"), protoreflect.ValueOfString("auth"))
	out := dynamicpb.NewMessage(d.out)

	ctx := metadata.NewOutgoingContext(t.Context(), metadata.Pairs("token", "secret"))
	require.NoError(t, srv.Conn().Invoke(ctx, "/test.Greeter/SayHello", in, out))
	require.Equal(t, "authorized", out.Get(d.out.Fields().ByName("message")).String())
}

func TestDeleteStubEffect(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	target := srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "target").
		Return("message", "target")

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "trigger").
		Effect(sdk.DeleteStub(target.StubID())).
		Return("message", "triggered")

	// Trigger the delete
	msg1 := sayHello(t, srv, fds, "trigger")
	require.Equal(t, "triggered", getMsg(t, msg1))

	// Target stub should now be deleted
	require.Error(t, sayHelloErr(t, srv, fds, "target"))
}

func TestHealthStubbingReal(t *testing.T) {
	t.Parallel()

	fds := compileInline(t, testProto, "test.proto")

	srv := sdk.NewServer(t, sdk.WithDescriptors(fds))
	defer func() { _ = srv.Close() }()

	// Stub health check
	srv.ExpectUnary("/grpc.health.v1.Health/Check").
		Match("service", "my-svc").
		Return("status", float64(grpc_health_v1.HealthCheckResponse_SERVING))

	client := grpc_health_v1.NewHealthClient(srv.Conn())
	resp, err := client.Check(t.Context(), &grpc_health_v1.HealthCheckRequest{Service: "my-svc"})
	require.NoError(t, err)
	require.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.GetStatus())
}
