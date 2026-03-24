package reflectclient

import (
	stderrors "errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/domain/protoset"
)

type fakeReflectionServer struct {
	reflectionpb.UnimplementedServerReflectionServer

	requireAuth string
	seenAuth    string
	rawFile     []byte
}

func (f *fakeReflectionServer) ServerReflectionInfo(stream reflectionpb.ServerReflection_ServerReflectionInfoServer) error {
	md, _ := metadata.FromIncomingContext(stream.Context())
	f.seenAuth = first(md.Get("authorization"))

	if f.requireAuth != "" && f.seenAuth != f.requireAuth {
		return status.Error(codes.Unauthenticated, "missing or invalid token")
	}

	for {
		req, err := stream.Recv()
		if stderrors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return err
		}

		switch req.GetMessageRequest().(type) {
		case *reflectionpb.ServerReflectionRequest_ListServices:
			if err := stream.Send(&reflectionpb.ServerReflectionResponse{
				MessageResponse: &reflectionpb.ServerReflectionResponse_ListServicesResponse{
					ListServicesResponse: &reflectionpb.ListServiceResponse{
						Service: []*reflectionpb.ServiceResponse{{Name: "grpc.health.v1.Health"}, {Name: "test.Echo"}},
					},
				},
			}); err != nil {
				return err
			}
		case *reflectionpb.ServerReflectionRequest_FileContainingSymbol:
			if err := stream.Send(&reflectionpb.ServerReflectionResponse{
				MessageResponse: &reflectionpb.ServerReflectionResponse_FileDescriptorResponse{
					FileDescriptorResponse: &reflectionpb.FileDescriptorResponse{FileDescriptorProto: [][]byte{f.rawFile}},
				},
			}); err != nil {
				return err
			}
		}
	}
}

func TestClientFetchDescriptorSet(t *testing.T) {
	t.Parallel()

	raw := newRawDescriptor(t, "test.proto")
	addr, _ := startFakeReflectionServer(t, raw, "")

	client := NewClient()
	source := &protoset.Source{ReflectAddress: addr, ReflectTimeout: time.Second}

	fds, err := client.FetchDescriptorSet(t.Context(), source)
	require.NoError(t, err)
	require.Len(t, fds.GetFile(), 1)
	require.Equal(t, "test.proto", fds.GetFile()[0].GetName())
}

func TestClientFetchDescriptorSetWithBearer(t *testing.T) {
	t.Parallel()

	raw := newRawDescriptor(t, "auth.proto")
	addr, fake := startFakeReflectionServer(t, raw, "Bearer secret")

	client := NewClient()
	source := &protoset.Source{ReflectAddress: addr, ReflectTimeout: time.Second, ReflectBearer: "secret"}

	fds, err := client.FetchDescriptorSet(t.Context(), source)
	require.NoError(t, err)
	require.Len(t, fds.GetFile(), 1)
	require.Equal(t, "Bearer secret", fake.seenAuth)
}

func newRawDescriptor(t *testing.T, fileName string) []byte {
	t.Helper()

	pkg := "test"
	fdp := &descriptorpb.FileDescriptorProto{Name: &fileName, Package: &pkg}

	raw, err := proto.Marshal(fdp)
	require.NoError(t, err)

	return raw
}

func startFakeReflectionServer(t *testing.T, raw []byte, requiredAuth string) (string, *fakeReflectionServer) {
	t.Helper()

	lis, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = lis.Close() })

	_, port, err := net.SplitHostPort(lis.Addr().String())
	require.NoError(t, err)

	addr := net.JoinHostPort("127.0.0.1", port)

	fake := &fakeReflectionServer{rawFile: raw, requireAuth: requiredAuth}
	server := grpc.NewServer()
	reflectionpb.RegisterServerReflectionServer(server, fake)

	go func() {
		_ = server.Serve(lis)
	}()

	t.Cleanup(server.GracefulStop)

	return addr, fake
}

func first(items []string) string {
	if len(items) == 0 {
		return ""
	}

	return items[0]
}
