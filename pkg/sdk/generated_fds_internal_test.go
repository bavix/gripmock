package sdk

import (
	"context"
	"io"
	"testing"

	chatpb "github.com/bavix/gripmock/v3/pkg/sdk/internal/examplefds/gen/examples/projects/chat"
	multiversepb "github.com/bavix/gripmock/v3/pkg/sdk/internal/examplefds/gen/examples/projects/multiverse"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestGeneratedDescriptorsCoverageScenarios(t *testing.T) {
	t.Parallel()

	// Arrange
	chat := descriptorSetFromFiles(chatpb.File_examples_projects_chat_service_proto)
	multiverse := descriptorSetFromFiles(multiversepb.File_examples_projects_multiverse_service_proto)

	// Assert: dedicated streaming fixture
	require.True(t, hasMethod(chat, "chat.ChatService", "SendMessage", true, false))
	require.True(t, hasMethod(chat, "chat.ChatService", "ReceiveMessages", false, true))
	require.True(t, hasMethod(chat, "chat.ChatService", "Chat", true, true))

	// Assert: unary + all streaming modes in one project
	require.True(t, hasMethod(multiverse, "multiverse.v1.MultiverseService", "Ping", false, false))
	require.True(t, hasMethod(multiverse, "multiverse.v1.MultiverseService", "UploadData", true, false))
	require.True(t, hasMethod(multiverse, "multiverse.v1.MultiverseService", "StreamData", false, true))
	require.True(t, hasMethod(multiverse, "multiverse.v1.MultiverseService", "Chat", true, true))

	// Assert: WKT import presence (google/protobuf/timestamp.proto)
	require.True(t, hasFile(multiverse, "google/protobuf/timestamp.proto"))
}

func callUnary(t *testing.T, ctx context.Context, conn *grpc.ClientConn, reg *protoregistry.Files, service, method string, req map[string]any) *dynamicpb.Message {
	t.Helper()

	md := methodDescriptor(t, reg, service, method)
	in := dynamicpb.NewMessage(md.Input())
	setFields(t, in, req)
	out := dynamicpb.NewMessage(md.Output())
	require.NoError(t, conn.Invoke(ctx, "/"+service+"/"+method, in, out))

	return out
}

func callUnaryWithHeader(t *testing.T, ctx context.Context, conn *grpc.ClientConn, reg *protoregistry.Files, service, method string, req map[string]any, head *metadata.MD) *dynamicpb.Message {
	t.Helper()

	md := methodDescriptor(t, reg, service, method)
	in := dynamicpb.NewMessage(md.Input())
	setFields(t, in, req)
	out := dynamicpb.NewMessage(md.Output())
	require.NoError(t, conn.Invoke(ctx, "/"+service+"/"+method, in, out, grpc.Header(head)))

	return out
}

func callServerStream(t *testing.T, ctx context.Context, conn *grpc.ClientConn, reg *protoregistry.Files, service, method string, req map[string]any) []*dynamicpb.Message {
	t.Helper()

	md := methodDescriptor(t, reg, service, method)
	in := dynamicpb.NewMessage(md.Input())
	setFields(t, in, req)

	stream, err := conn.NewStream(ctx, &grpc.StreamDesc{StreamName: method, ServerStreams: true, ClientStreams: false}, "/"+service+"/"+method)
	require.NoError(t, err)
	require.NoError(t, stream.SendMsg(in))
	require.NoError(t, stream.CloseSend())

	out := make([]*dynamicpb.Message, 0, 2)
	for {
		msg := dynamicpb.NewMessage(md.Output())
		err = stream.RecvMsg(msg)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		out = append(out, msg)
	}

	return out
}

func callClientStream(t *testing.T, ctx context.Context, conn *grpc.ClientConn, reg *protoregistry.Files, service, method string, reqs []map[string]any) *dynamicpb.Message {
	t.Helper()

	md := methodDescriptor(t, reg, service, method)
	stream, err := conn.NewStream(ctx, &grpc.StreamDesc{StreamName: method, ServerStreams: false, ClientStreams: true}, "/"+service+"/"+method)
	require.NoError(t, err)

	for _, req := range reqs {
		msg := dynamicpb.NewMessage(md.Input())
		setFields(t, msg, req)
		require.NoError(t, stream.SendMsg(msg))
	}
	require.NoError(t, stream.CloseSend())

	out := dynamicpb.NewMessage(md.Output())
	require.NoError(t, stream.RecvMsg(out))

	return out
}

func callBidiStream(t *testing.T, ctx context.Context, conn *grpc.ClientConn, reg *protoregistry.Files, service, method string, reqs []map[string]any) []*dynamicpb.Message {
	t.Helper()

	md := methodDescriptor(t, reg, service, method)
	stream, err := conn.NewStream(ctx, &grpc.StreamDesc{StreamName: method, ServerStreams: true, ClientStreams: true}, "/"+service+"/"+method)
	require.NoError(t, err)

	for _, req := range reqs {
		msg := dynamicpb.NewMessage(md.Input())
		setFields(t, msg, req)
		require.NoError(t, stream.SendMsg(msg))
	}
	require.NoError(t, stream.CloseSend())

	out := make([]*dynamicpb.Message, 0, 1)
	for {
		msg := dynamicpb.NewMessage(md.Output())
		err = stream.RecvMsg(msg)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		out = append(out, msg)
	}

	return out
}

func methodDescriptor(t *testing.T, reg *protoregistry.Files, service, method string) protoreflect.MethodDescriptor {
	t.Helper()

	svcDesc, err := reg.FindDescriptorByName(protoreflect.FullName(service))
	require.NoError(t, err)
	svc := svcDesc.(protoreflect.ServiceDescriptor)
	m := svc.Methods().ByName(protoreflect.Name(method))
	require.NotNil(t, m)

	return m
}

func setFields(t *testing.T, msg *dynamicpb.Message, values map[string]any) {
	t.Helper()

	fields := msg.Descriptor().Fields()
	for k, v := range values {
		fd := fields.ByName(protoreflect.Name(k))
		require.NotNilf(t, fd, "field %q not found in %s", k, msg.Descriptor().FullName())

		switch fd.Kind() {
		case protoreflect.StringKind:
			s, ok := v.(string)
			require.True(t, ok)
			msg.Set(fd, protoreflect.ValueOfString(s))
		case protoreflect.BoolKind:
			b, ok := v.(bool)
			require.True(t, ok)
			msg.Set(fd, protoreflect.ValueOfBool(b))
		case protoreflect.Int32Kind:
			switch n := v.(type) {
			case int:
				msg.Set(fd, protoreflect.ValueOfInt32(int32(n)))
			case int32:
				msg.Set(fd, protoreflect.ValueOfInt32(n))
			default:
				t.Fatalf("unsupported int32 value type %T", v)
			}
		case protoreflect.Int64Kind:
			switch n := v.(type) {
			case int:
				msg.Set(fd, protoreflect.ValueOfInt64(int64(n)))
			case int64:
				msg.Set(fd, protoreflect.ValueOfInt64(n))
			default:
				t.Fatalf("unsupported int64 value type %T", v)
			}
		case protoreflect.DoubleKind:
			switch n := v.(type) {
			case float64:
				msg.Set(fd, protoreflect.ValueOfFloat64(n))
			case float32:
				msg.Set(fd, protoreflect.ValueOfFloat64(float64(n)))
			case int:
				msg.Set(fd, protoreflect.ValueOfFloat64(float64(n)))
			default:
				t.Fatalf("unsupported double value type %T", v)
			}
		case protoreflect.BytesKind:
			b, ok := v.([]byte)
			require.True(t, ok)
			msg.Set(fd, protoreflect.ValueOfBytes(b))
		default:
			t.Fatalf("unsupported field kind %s for %q", fd.Kind(), k)
		}
	}
}

func getStringField(t *testing.T, msg *dynamicpb.Message, field string) string {
	t.Helper()
	fd := msg.Descriptor().Fields().ByName(protoreflect.Name(field))
	require.NotNil(t, fd)

	return msg.Get(fd).String()
}

func getBoolField(t *testing.T, msg *dynamicpb.Message, field string) bool {
	t.Helper()
	fd := msg.Descriptor().Fields().ByName(protoreflect.Name(field))
	require.NotNil(t, fd)

	return msg.Get(fd).Bool()
}

func descriptorSetFromFiles(root protoreflect.FileDescriptor) *descriptorpb.FileDescriptorSet {
	fds := &descriptorpb.FileDescriptorSet{}
	seen := map[string]struct{}{}
	appendFileRecursive(fds, seen, root)

	return fds
}

func appendFileRecursive(fds *descriptorpb.FileDescriptorSet, seen map[string]struct{}, fd protoreflect.FileDescriptor) {
	name := fd.Path()
	if _, ok := seen[name]; ok {
		return
	}
	seen[name] = struct{}{}

	imports := fd.Imports()
	for i := 0; i < imports.Len(); i++ {
		appendFileRecursive(fds, seen, imports.Get(i).FileDescriptor)
	}

	fds.File = append(fds.File, protodesc.ToFileDescriptorProto(fd))
}

func hasFile(fds *descriptorpb.FileDescriptorSet, name string) bool {
	for _, file := range fds.GetFile() {
		if file.GetName() == name {
			return true
		}
	}

	return false
}

func hasMethod(fds *descriptorpb.FileDescriptorSet, service, method string, clientStreaming, serverStreaming bool) bool {
	for _, file := range fds.GetFile() {
		pkg := file.GetPackage()
		for _, svc := range file.GetService() {
			svcName := svc.GetName()
			if pkg != "" {
				svcName = pkg + "." + svcName
			}
			if svcName != service {
				continue
			}

			for _, m := range svc.GetMethod() {
				if m.GetName() == method && m.GetClientStreaming() == clientStreaming && m.GetServerStreaming() == serverStreaming {
					return true
				}
			}
		}
	}

	return false
}
