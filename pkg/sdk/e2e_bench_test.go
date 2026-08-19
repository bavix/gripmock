package sdk_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/bufbuild/protocompile"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	sdk "github.com/bavix/gripmock/v3/pkg/sdk"
)

func benchServer(b *testing.B, stubs int) (*sdk.Server, msgDesc) {
	b.Helper()

	fds := compileInlineBench(b, testProto, "test.proto")
	srv := sdk.NewServer(benchT{b}, sdk.WithDescriptors(fds))

	for i := range stubs {
		srv.ExpectUnary("/test.Greeter/SayHello").
			Match("name", "filler-"+itoa(i)).
			Return("message", "filler")
	}

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "Alex").
		Return("message", "Hello Alex")

	return srv, resolveDescBench(b, fds, "test.HelloRequest", "test.HelloReply")
}

func BenchmarkUnaryCallEndToEnd(b *testing.B) {
	for _, stubs := range []int{0, 100, 1000} {
		b.Run("stubs="+itoa(stubs), func(b *testing.B) {
			srv, d := benchServer(b, stubs)
			defer func() { _ = srv.Close() }()

			conn := srv.Conn()
			request := dynamicpb.NewMessage(d.in)
			request.Set(d.in.Fields().ByName("name"), protoreflect.ValueOfString("Alex"))

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				reply := dynamicpb.NewMessage(d.out)
				if err := conn.Invoke(b.Context(), "/test.Greeter/SayHello", request, reply); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkUnaryCallWithTemplate(b *testing.B) {
	fds := compileInlineBench(b, testProto, "test.proto")
	srv := sdk.NewServer(benchT{b}, sdk.WithDescriptors(fds))

	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match(sdk.Matches("name", ".+")).
		Return("message", "Hello {{.Request.name}}")

	d := resolveDescBench(b, fds, "test.HelloRequest", "test.HelloReply")
	conn := srv.Conn()
	request := dynamicpb.NewMessage(d.in)
	request.Set(d.in.Fields().ByName("name"), protoreflect.ValueOfString("Alex"))

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		reply := dynamicpb.NewMessage(d.out)
		if err := conn.Invoke(b.Context(), "/test.Greeter/SayHello", request, reply); err != nil {
			b.Fatal(err)
		}
	}
}

type benchT struct{ b *testing.B }

func (t benchT) Error(args ...any)        { t.b.Error(args...) }
func (t benchT) Fail()                    { t.b.Fail() }
func (t benchT) Context() context.Context { return t.b.Context() }
func (t benchT) Cleanup(f func())         { t.b.Cleanup(f) }

func itoa(i int) string { return strconv.Itoa(i) }

func compileInlineBench(b *testing.B, source, name string) *descriptorpb.FileDescriptorSet {
	b.Helper()

	dir := b.TempDir()
	path := filepath.Join(dir, name)

	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		b.Fatal(err)
	}

	compiler := protocompile.Compiler{
		Resolver: protocompile.CompositeResolver{
			&protocompile.SourceResolver{ImportPaths: []string{dir}},
			protocompile.WithStandardImports(&protocompile.SourceResolver{}),
		},
	}

	files, err := compiler.Compile(b.Context(), path)
	if err != nil {
		b.Fatal(err)
	}

	fds := &descriptorpb.FileDescriptorSet{}
	for _, f := range files {
		fds.File = append(fds.File, protodesc.ToFileDescriptorProto(f))
	}

	return fds
}

func resolveDescBench(b *testing.B, fds *descriptorpb.FileDescriptorSet, inName, outName protoreflect.FullName) msgDesc {
	b.Helper()

	files, err := protodesc.NewFiles(fds)
	if err != nil {
		b.Fatal(err)
	}

	in, err := files.FindDescriptorByName(inName)
	if err != nil {
		b.Fatal(err)
	}

	out, err := files.FindDescriptorByName(outName)
	if err != nil {
		b.Fatal(err)
	}

	inMsg, ok := in.(protoreflect.MessageDescriptor)
	if !ok {
		b.Fatal("not a message")
	}

	outMsg, ok := out.(protoreflect.MessageDescriptor)
	if !ok {
		b.Fatal("not a message")
	}

	return msgDesc{in: inMsg, out: outMsg}
}

func BenchmarkServerStreamEndToEnd(b *testing.B) {
	fds := compileInlineBench(b, searchProto, "search.proto")
	srv := sdk.NewServer(benchT{b}, sdk.WithDescriptors(fds))

	defer func() { _ = srv.Close() }()

	srv.ExpectServerStream("/search.SearchService/Search").
		Match("query", "bench").
		SendStream(
			map[string]any{"id": "1", "title": "first"},
			map[string]any{"id": "2", "title": "second"},
			map[string]any{"id": "3", "title": "third"},
		)

	d := resolveDescBench(b, fds, "search.SearchRequest", "search.SearchResult")
	conn := srv.Conn()
	request := dynamicpb.NewMessage(d.in)
	request.Set(d.in.Fields().ByName("query"), protoreflect.ValueOfString("bench"))

	desc := &grpc.StreamDesc{StreamName: "Search", ServerStreams: true}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		stream, err := conn.NewStream(b.Context(), desc, "/search.SearchService/Search")
		if err != nil {
			b.Fatal(err)
		}

		if err := stream.SendMsg(request); err != nil {
			b.Fatal(err)
		}

		if err := stream.CloseSend(); err != nil {
			b.Fatal(err)
		}

		for {
			out := dynamicpb.NewMessage(d.out)
			if err := stream.RecvMsg(out); err != nil {
				break
			}
		}
	}
}

func BenchmarkClientStreamEndToEnd(b *testing.B) {
	fds := compileInlineBench(b, calcProto, "calc.proto")
	srv := sdk.NewServer(benchT{b}, sdk.WithDescriptors(fds))

	defer func() { _ = srv.Close() }()

	srv.ExpectClientStream("/calc.Calculator/SumNumbers").
		Match(sdk.Matches("value", ".*")).
		Return("result", 6.0, "count", 3)

	d := resolveDescBench(b, fds, "calc.NumberRequest", "calc.SumResponse")
	conn := srv.Conn()
	desc := &grpc.StreamDesc{StreamName: "SumNumbers", ClientStreams: true}

	msgs := make([]*dynamicpb.Message, 3)
	for i := range msgs {
		msgs[i] = dynamicpb.NewMessage(d.in)
		msgs[i].Set(d.in.Fields().ByName("value"), protoreflect.ValueOfFloat64(float64(i+1)))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		stream, err := conn.NewStream(b.Context(), desc, "/calc.Calculator/SumNumbers")
		if err != nil {
			b.Fatal(err)
		}

		for _, msg := range msgs {
			if err := stream.SendMsg(msg); err != nil {
				b.Fatal(err)
			}
		}

		if err := stream.CloseSend(); err != nil {
			b.Fatal(err)
		}

		out := dynamicpb.NewMessage(d.out)
		if err := stream.RecvMsg(out); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBidiStreamEndToEnd(b *testing.B) {
	fds := compileInlineBench(b, chatProto, "chat.proto")
	srv := sdk.NewServer(benchT{b}, sdk.WithDescriptors(fds))

	defer func() { _ = srv.Close() }()

	srv.ExpectBidirectionalStream("/chat.ChatService/Chat").
		Match("text", "ping").
		SendStream(map[string]any{"text": "pong"})

	d := resolveDescBench(b, fds, "chat.ChatMessage", "chat.ChatMessage")
	conn := srv.Conn()
	desc := &grpc.StreamDesc{StreamName: "Chat", ServerStreams: true, ClientStreams: true}

	request := dynamicpb.NewMessage(d.in)
	request.Set(d.in.Fields().ByName("text"), protoreflect.ValueOfString("ping"))

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		stream, err := conn.NewStream(b.Context(), desc, "/chat.ChatService/Chat")
		if err != nil {
			b.Fatal(err)
		}

		if err := stream.SendMsg(request); err != nil {
			b.Fatal(err)
		}

		if err := stream.CloseSend(); err != nil {
			b.Fatal(err)
		}

		for {
			out := dynamicpb.NewMessage(d.out)
			if err := stream.RecvMsg(out); err != nil {
				break
			}
		}
	}
}

func BenchmarkBidiMultiTurnEndToEnd(b *testing.B) {
	fds := compileInlineBench(b, chatProto, "chat.proto")
	srv := sdk.NewServer(benchT{b}, sdk.WithDescriptors(fds))

	defer func() { _ = srv.Close() }()

	const turns = 5

	matchers := make([]sdk.Matcher, turns)
	replies := make([]any, turns)

	for i := range turns {
		matchers[i] = sdk.Equals("text", "turn-"+itoa(i))
		replies[i] = map[string]any{"text": "reply-" + itoa(i)}
	}

	srv.ExpectBidirectionalStream("/chat.ChatService/Chat").
		MatchSequence(matchers...).
		SendStream(replies...)

	d := resolveDescBench(b, fds, "chat.ChatMessage", "chat.ChatMessage")
	conn := srv.Conn()
	desc := &grpc.StreamDesc{StreamName: "Chat", ServerStreams: true, ClientStreams: true}

	msgs := make([]*dynamicpb.Message, turns)
	for i := range msgs {
		msgs[i] = dynamicpb.NewMessage(d.in)
		msgs[i].Set(d.in.Fields().ByName("text"), protoreflect.ValueOfString("turn-"+itoa(i)))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		stream, err := conn.NewStream(b.Context(), desc, "/chat.ChatService/Chat")
		if err != nil {
			b.Fatal(err)
		}

		for _, msg := range msgs {
			if err := stream.SendMsg(msg); err != nil {
				b.Fatal(err)
			}

			if err := stream.RecvMsg(dynamicpb.NewMessage(d.out)); err != nil {
				b.Fatal(err)
			}
		}

		if err := stream.CloseSend(); err != nil {
			b.Fatal(err)
		}
	}
}
