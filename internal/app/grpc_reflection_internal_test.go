package app

import (
	"context"
	"net"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	reflectiongrpc "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/features"
	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/protoset"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func TestReflectionIncludesDynamicService(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	registry := descriptors.NewRegistry()
	grpcServer := NewGRPCServer("tcp", "127.0.0.1:0", nil, stuber.NewBudgerigar(features.New()), nil, nil, registry)

	server, err := grpcServer.Build(ctx)
	require.NoError(t, err)

	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	serveErr := make(chan error, 1)

	go func() {
		serveErr <- server.Serve(listener)
	}()

	t.Cleanup(func() {
		server.Stop()

		_ = listener.Close()

		select {
		case <-serveErr:
		default:
		}
	})

	registerGreeterDescriptors(t, ctx, registry)

	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = conn.Close()
	})

	services, err := reflectionListServices(ctx, conn)
	require.NoError(t, err)
	require.Contains(t, services, "helloworld.Greeter")

	symbolResp, err := reflectionFileContainingSymbol(ctx, conn, "helloworld.Greeter")
	require.NoError(t, err)
	require.NotNil(t, symbolResp.GetFileDescriptorResponse())
	require.NotEmpty(t, symbolResp.GetFileDescriptorResponse().GetFileDescriptorProto())
}

func registerGreeterDescriptors(t *testing.T, ctx context.Context, registry *descriptors.Registry) {
	t.Helper()

	protoPath := filepath.Join("..", "..", "examples", "projects", "greeter", "service.proto")
	fdsList, err := protoset.Build(ctx, nil, []string{protoPath})
	require.NoError(t, err)
	require.NotEmpty(t, fdsList)

	var merged descriptorpb.FileDescriptorSet
	for _, set := range fdsList {
		merged.File = append(merged.File, set.GetFile()...)
	}

	files, err := decodeDescriptorFiles(&merged)
	require.NoError(t, err)

	for _, fd := range files {
		registry.Register(fd)
	}
}

func reflectionListServices(ctx context.Context, conn *grpc.ClientConn) ([]string, error) {
	client := reflectiongrpc.NewServerReflectionClient(conn)

	stream, err := client.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, err
	}

	req := &reflectiongrpc.ServerReflectionRequest{
		MessageRequest: &reflectiongrpc.ServerReflectionRequest_ListServices{ListServices: "*"},
	}
	if err := stream.Send(req); err != nil {
		return nil, err
	}

	resp, err := stream.Recv()
	if err != nil {
		return nil, err
	}

	serviceList := resp.GetListServicesResponse().GetService()
	services := make([]string, 0, len(serviceList))

	for _, service := range serviceList {
		services = append(services, service.GetName())
	}

	sort.Strings(services)

	return services, nil
}

func reflectionFileContainingSymbol(
	ctx context.Context,
	conn *grpc.ClientConn,
	symbol string,
) (*reflectiongrpc.ServerReflectionResponse, error) {
	client := reflectiongrpc.NewServerReflectionClient(conn)

	stream, err := client.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, err
	}

	req := &reflectiongrpc.ServerReflectionRequest{
		MessageRequest: &reflectiongrpc.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: string(protoreflect.FullName(symbol))},
	}
	if err := stream.Send(req); err != nil {
		return nil, err
	}

	return stream.Recv()
}
