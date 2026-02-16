package sdk

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Test_resolveDescriptorsFromReflection_ConnectionClosed verifies error when server
// closes connection before/during reflection (covers Recv/Send error paths).
func Test_resolveDescriptorsFromReflection_InvalidAddress(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Invalid address - may fail at NewClient or at first RPC (lazy connect)
	_, err := resolveDescriptorsFromReflection(ctx, "invalid-address-with-no-port")
	require.Error(t, err)
	errStr := err.Error()
	require.True(t,
		strings.Contains(errStr, "failed to connect") ||
			strings.Contains(errStr, "failed to get reflection stream") ||
			strings.Contains(errStr, "missing port"))
}

// fakeReflectionAbortImmediately returns error from ServerReflectionInfo before Recv,
// causing client's stream.Send to fail (stream is aborted).
type fakeReflectionAbortImmediately struct {
	reflectionpb.UnimplementedServerReflectionServer
}

func (f *fakeReflectionAbortImmediately) ServerReflectionInfo(stream reflectionpb.ServerReflection_ServerReflectionInfoServer) error {
	return status.Error(codes.Internal, "server abort")
}

func Test_resolveDescriptorsFromReflection_StreamAborted(t *testing.T) {
	t.Parallel()

	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer lis.Close()

	_, port, _ := net.SplitHostPort(lis.Addr().String())
	addr := "127.0.0.1:" + port

	server := grpc.NewServer()
	reflectionpb.RegisterServerReflectionServer(server, &fakeReflectionAbortImmediately{})
	go func() { _ = server.Serve(lis) }()
	defer server.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = resolveDescriptorsFromReflection(ctx, addr)
	require.Error(t, err)
	// Server aborts immediately; client Send or stream setup fails
	errStr := err.Error()
	require.True(t,
		strings.Contains(errStr, "failed to send ListServices") ||
			strings.Contains(errStr, "failed to get reflection stream") ||
			strings.Contains(errStr, "Internal"))
}

func Test_resolveDescriptorsFromReflection_ConnectionClosed(t *testing.T) {
	t.Parallel()

	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer lis.Close()

	_, port, _ := net.SplitHostPort(lis.Addr().String())
	addr := "127.0.0.1:" + port

	go func() {
		conn, _ := lis.Accept()
		if conn != nil {
			time.Sleep(100 * time.Millisecond)
			_ = conn.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = resolveDescriptorsFromReflection(ctx, addr)
	require.Error(t, err)
	errStr := err.Error()
	require.True(t,
		strings.Contains(errStr, "failed to get reflection stream") ||
			strings.Contains(errStr, "failed to recv ListServices") ||
			strings.Contains(errStr, "failed to send ListServices") ||
			strings.Contains(errStr, "failed to connect"),
		"err=%v", err)
}

// fakeReflectionServer returns unexpected response to trigger listResp == nil path.
type fakeReflectionServer struct {
	reflectionpb.UnimplementedServerReflectionServer
	response *reflectionpb.ServerReflectionResponse
}

func (f *fakeReflectionServer) ServerReflectionInfo(stream reflectionpb.ServerReflection_ServerReflectionInfoServer) error {
	_, err := stream.Recv()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	// Send wrong response type - client expects ListServicesResponse, we send FileDescriptorResponse
	return stream.Send(f.response)
}

func Test_resolveDescriptorsFromReflection_UnexpectedResponse(t *testing.T) {
	t.Parallel()

	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer lis.Close()

	_, port, _ := net.SplitHostPort(lis.Addr().String())
	addr := "127.0.0.1:" + port

	server := grpc.NewServer()
	fake := &fakeReflectionServer{
		response: &reflectionpb.ServerReflectionResponse{
			MessageResponse: &reflectionpb.ServerReflectionResponse_FileDescriptorResponse{
				FileDescriptorResponse: &reflectionpb.FileDescriptorResponse{}, // wrong type for ListServices
			},
		},
	}
	reflectionpb.RegisterServerReflectionServer(server, fake)
	go func() { _ = server.Serve(lis) }()
	defer server.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = resolveDescriptorsFromReflection(ctx, addr)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected response: not ListServicesResponse")
}

// fakeReflectionErrorResponse returns ListServices with one service, then ErrorResponse
// for FileContainingSymbol to trigger fd == nil with GetErrorResponse branch.
type fakeReflectionErrorResponse struct {
	reflectionpb.UnimplementedServerReflectionServer
}

func (f *fakeReflectionErrorResponse) ServerReflectionInfo(stream reflectionpb.ServerReflection_ServerReflectionInfoServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if _, ok := req.MessageRequest.(*reflectionpb.ServerReflectionRequest_ListServices); ok {
			_ = stream.Send(&reflectionpb.ServerReflectionResponse{
				MessageResponse: &reflectionpb.ServerReflectionResponse_ListServicesResponse{
					ListServicesResponse: &reflectionpb.ListServiceResponse{
						Service: []*reflectionpb.ServiceResponse{
							{Name: "test.Echo"},
						},
					},
				},
			})
		} else if req.GetFileContainingSymbol() != "" {
			_ = stream.Send(&reflectionpb.ServerReflectionResponse{
				MessageResponse: &reflectionpb.ServerReflectionResponse_ErrorResponse{
					ErrorResponse: &reflectionpb.ErrorResponse{
						ErrorMessage: "symbol not found",
					},
				},
			})
		}
	}
}

func Test_resolveDescriptorsFromReflection_ErrorResponse(t *testing.T) {
	t.Parallel()

	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer lis.Close()

	_, port, _ := net.SplitHostPort(lis.Addr().String())
	addr := "127.0.0.1:" + port

	server := grpc.NewServer()
	reflectionpb.RegisterServerReflectionServer(server, &fakeReflectionErrorResponse{})
	go func() { _ = server.Serve(lis) }()
	defer server.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = resolveDescriptorsFromReflection(ctx, addr)
	require.Error(t, err)
	require.Contains(t, err.Error(), "reflection error for")
	require.Contains(t, err.Error(), "symbol not found")
}

// fakeReflectionUnexpectedFileResp returns ListServices, then wrong type for FileContainingSymbol
// (not FileDescriptorResponse, not ErrorResponse) to trigger fd == nil without GetErrorResponse.
type fakeReflectionUnexpectedFileResp struct {
	reflectionpb.UnimplementedServerReflectionServer
}

func (f *fakeReflectionUnexpectedFileResp) ServerReflectionInfo(stream reflectionpb.ServerReflection_ServerReflectionInfoServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if _, ok := req.MessageRequest.(*reflectionpb.ServerReflectionRequest_ListServices); ok {
			_ = stream.Send(&reflectionpb.ServerReflectionResponse{
				MessageResponse: &reflectionpb.ServerReflectionResponse_ListServicesResponse{
					ListServicesResponse: &reflectionpb.ListServiceResponse{
						Service: []*reflectionpb.ServiceResponse{{Name: "test.Svc"}},
					},
				},
			})
		} else if req.GetFileContainingSymbol() != "" {
			// Send ListServicesResponse again (wrong type) - fdResp.GetFileDescriptorResponse() is nil
			// and GetErrorResponse() is also nil
			_ = stream.Send(&reflectionpb.ServerReflectionResponse{
				MessageResponse: &reflectionpb.ServerReflectionResponse_ListServicesResponse{
					ListServicesResponse: &reflectionpb.ListServiceResponse{},
				},
			})
		}
	}
}

func Test_resolveDescriptorsFromReflection_UnexpectedFileResponse(t *testing.T) {
	t.Parallel()

	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer lis.Close()

	_, port, _ := net.SplitHostPort(lis.Addr().String())
	addr := "127.0.0.1:" + port

	server := grpc.NewServer()
	reflectionpb.RegisterServerReflectionServer(server, &fakeReflectionUnexpectedFileResp{})
	go func() { _ = server.Serve(lis) }()
	defer server.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = resolveDescriptorsFromReflection(ctx, addr)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected response for")
	require.Contains(t, err.Error(), "not FileDescriptorResponse")
}

// fakeReflectionCorruptProto returns ListServices, then FileDescriptorResponse with
// corrupt bytes to trigger proto.Unmarshal error.
type fakeReflectionCorruptProto struct {
	reflectionpb.UnimplementedServerReflectionServer
}

func (f *fakeReflectionCorruptProto) ServerReflectionInfo(stream reflectionpb.ServerReflection_ServerReflectionInfoServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if _, ok := req.MessageRequest.(*reflectionpb.ServerReflectionRequest_ListServices); ok {
			_ = stream.Send(&reflectionpb.ServerReflectionResponse{
				MessageResponse: &reflectionpb.ServerReflectionResponse_ListServicesResponse{
					ListServicesResponse: &reflectionpb.ListServiceResponse{
						Service: []*reflectionpb.ServiceResponse{{Name: "test.Corrupt"}},
					},
				},
			})
		} else if req.GetFileContainingSymbol() != "" {
			// Corrupt proto bytes - unmarshal will fail
			_ = stream.Send(&reflectionpb.ServerReflectionResponse{
				MessageResponse: &reflectionpb.ServerReflectionResponse_FileDescriptorResponse{
					FileDescriptorResponse: &reflectionpb.FileDescriptorResponse{
						FileDescriptorProto: [][]byte{{0xff, 0xfe}}, // invalid proto
					},
				},
			})
		}
	}
}

func Test_resolveDescriptorsFromReflection_CorruptProto(t *testing.T) {
	t.Parallel()

	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer lis.Close()

	_, port, _ := net.SplitHostPort(lis.Addr().String())
	addr := "127.0.0.1:" + port

	server := grpc.NewServer()
	reflectionpb.RegisterServerReflectionServer(server, &fakeReflectionCorruptProto{})
	go func() { _ = server.Serve(lis) }()
	defer server.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = resolveDescriptorsFromReflection(ctx, addr)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal FileDescriptorProto")
}

// fakeReflectionEmptyName returns FileDescriptorProto with empty Name to trigger key=="" branch.
type fakeReflectionEmptyName struct {
	reflectionpb.UnimplementedServerReflectionServer
}

func (f *fakeReflectionEmptyName) ServerReflectionInfo(stream reflectionpb.ServerReflection_ServerReflectionInfoServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if _, ok := req.MessageRequest.(*reflectionpb.ServerReflectionRequest_ListServices); ok {
			// Include empty name to trigger name=="" continue branch
			_ = stream.Send(&reflectionpb.ServerReflectionResponse{
				MessageResponse: &reflectionpb.ServerReflectionResponse_ListServicesResponse{
					ListServicesResponse: &reflectionpb.ListServiceResponse{
						Service: []*reflectionpb.ServiceResponse{
							{Name: ""}, // skipped: name == ""
							{Name: "test.EmptyName"},
						},
					},
				},
			})
		} else if req.GetFileContainingSymbol() != "" {
			fdp := &descriptorpb.FileDescriptorProto{
				Name:    proto.String(""), // empty -> key becomes "test.unknown"
				Package: proto.String("test"),
			}
			raw, _ := proto.Marshal(fdp)
			_ = stream.Send(&reflectionpb.ServerReflectionResponse{
				MessageResponse: &reflectionpb.ServerReflectionResponse_FileDescriptorResponse{
					FileDescriptorResponse: &reflectionpb.FileDescriptorResponse{
						FileDescriptorProto: [][]byte{raw},
					},
				},
			})
		}
	}
}

func Test_resolveDescriptorsFromReflection_EmptyNameKey(t *testing.T) {
	t.Parallel()

	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer lis.Close()

	_, port, _ := net.SplitHostPort(lis.Addr().String())
	addr := "127.0.0.1:" + port

	server := grpc.NewServer()
	reflectionpb.RegisterServerReflectionServer(server, &fakeReflectionEmptyName{})
	go func() { _ = server.Serve(lis) }()
	defer server.GracefulStop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	fds, err := resolveDescriptorsFromReflection(ctx, addr)
	require.NoError(t, err)
	require.NotNil(t, fds)
	require.Len(t, fds.File, 1)
	// key was "test.unknown" from empty name + package
	require.Equal(t, "test", fds.File[0].GetPackage())
}
