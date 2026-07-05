package sdk

import (
	"context"
	"fmt"

	"github.com/cockroachdb/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// resolveDescriptorsFromReflection fetches FileDescriptorSet from a gRPC server via reflection.
func resolveDescriptorsFromReflection(ctx context.Context, addr string) (*descriptorpb.FileDescriptorSet, error) {
	conn, err := grpc.NewClient("passthrough:///"+addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to %s", addr)
	}
	defer conn.Close()

	client := reflectionpb.NewServerReflectionClient(conn)
	stream, err := client.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get reflection stream")
	}

	// ListServices
	if err := stream.Send(&reflectionpb.ServerReflectionRequest{
		MessageRequest: &reflectionpb.ServerReflectionRequest_ListServices{},
	}); err != nil {
		return nil, errors.Wrap(err, "failed to send ListServices")
	}

	resp, err := stream.Recv()
	if err != nil {
		return nil, errors.Wrap(err, "failed to recv ListServices response")
	}

	listResp := resp.GetListServicesResponse()
	if listResp == nil {
		return nil, ErrUnexpectedResponse
	}

	seen := make(map[string]*descriptorpb.FileDescriptorProto)
	for _, svc := range listResp.GetService() {
		name := svc.GetName()
		if name == "" {
			continue
		}
		// Skip reflection and health services
		if name == "grpc.reflection.v1.ServerReflection" || name == "grpc.reflection.v1alpha.ServerReflection" ||
			name == "grpc.health.v1.Health" {
			continue
		}

		if err := stream.Send(&reflectionpb.ServerReflectionRequest{
			MessageRequest: &reflectionpb.ServerReflectionRequest_FileContainingSymbol{
				FileContainingSymbol: name,
			},
		}); err != nil {
			return nil, errors.Wrapf(err, "failed to send FileContainingSymbol for %s", name)
		}

		fdResp, err := stream.Recv()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to recv FileContainingSymbol for %s", name)
		}

		fd := fdResp.GetFileDescriptorResponse()
		if fd == nil {
			if errResp := fdResp.GetErrorResponse(); errResp != nil {
				return nil, errors.Errorf("reflection error for %s: %s", name, errResp.GetErrorMessage())
			}
			return nil, errors.Errorf("unexpected response for %s: not FileDescriptorResponse", name)
		}

		for _, raw := range fd.GetFileDescriptorProto() {
			var fdp descriptorpb.FileDescriptorProto
			if err := proto.Unmarshal(raw, &fdp); err != nil {
				return nil, errors.Wrapf(err, "failed to unmarshal FileDescriptorProto for %s", name)
			}
			key := fdp.GetName()
			if key == "" {
				key = fmt.Sprintf("%s.%s", fdp.GetPackage(), "unknown")
			}
			if _, exists := seen[key]; !exists {
				seen[key] = &fdp
			}
		}
	}

	if len(seen) == 0 {
		return nil, ErrNoUsableServicesFoundViaReflection
	}

	fds := &descriptorpb.FileDescriptorSet{}
	for _, fdp := range seen {
		fds.File = append(fds.File, fdp)
	}
	return fds, nil
}
