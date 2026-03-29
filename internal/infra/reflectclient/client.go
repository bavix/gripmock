package reflectclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"sort"
	"time"

	"github.com/cockroachdb/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	reflectionpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/domain/protoset"
)

const (
	serviceReflectionV1      = "grpc.reflection.v1.ServerReflection"
	serviceReflectionV1Alpha = "grpc.reflection.v1alpha.ServerReflection"
	serviceHealth            = "grpc.health.v1.Health"
	defaultTimeout           = 5 * time.Second
	dialOptionCapacity       = 3
)

var (
	errSourceNil                      = errors.New("source is nil")
	errReflectAddressEmpty            = errors.New("reflect address is empty")
	errUnexpectedListServicesResponse = errors.New("unexpected response: not ListServicesResponse")
	errNoUsableServices               = errors.New("no services found via reflection")
	errReflectionError                = errors.New("reflection error")
	errUnexpectedFileResponse         = errors.New("unexpected response: not FileDescriptorResponse")
)

type Client struct{}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) FetchDescriptorSet(ctx context.Context, source *protoset.Source) (*descriptorpb.FileDescriptorSet, error) {
	if source == nil {
		return nil, errSourceNil
	}

	if source.ReflectAddress == "" {
		return nil, errReflectAddressEmpty
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		timeout := source.ReflectTimeout
		if timeout <= 0 {
			timeout = defaultTimeout
		}

		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	conn, err := grpc.NewClient(
		"passthrough:///"+source.ReflectAddress,
		buildDialOptions(source)...,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to %s", source.ReflectAddress)
	}

	defer func() {
		_ = conn.Close()
	}()

	return fetchDescriptorSet(ctx, conn)
}

func buildDialOptions(source *protoset.Source) []grpc.DialOption {
	options := make([]grpc.DialOption, 0, dialOptionCapacity)

	if source.ReflectTLS {
		tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
		if source.ReflectServerName != "" {
			tlsConfig.ServerName = source.ReflectServerName
		}

		tlsConfig.InsecureSkipVerify = source.ReflectInsecure

		options = append(options, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		options = append(options, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if source.ReflectBearer == "" {
		return options
	}

	token := source.ReflectBearer

	options = append(options,
		grpc.WithUnaryInterceptor(func(
			ctx context.Context,
			method string,
			req, reply any,
			cc *grpc.ClientConn,
			invoker grpc.UnaryInvoker,
			opts ...grpc.CallOption,
		) error {
			return invoker(withBearer(ctx, token), method, req, reply, cc, opts...)
		}),
		grpc.WithStreamInterceptor(func(
			ctx context.Context,
			desc *grpc.StreamDesc,
			cc *grpc.ClientConn,
			method string,
			streamer grpc.Streamer,
			opts ...grpc.CallOption,
		) (grpc.ClientStream, error) {
			return streamer(withBearer(ctx, token), desc, cc, method, opts...)
		}),
	)

	return options
}

func withBearer(ctx context.Context, token string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
}

func fetchDescriptorSet(ctx context.Context, conn *grpc.ClientConn) (*descriptorpb.FileDescriptorSet, error) {
	stream, err := reflectionpb.NewServerReflectionClient(conn).ServerReflectionInfo(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get reflection stream")
	}

	services, err := listServices(stream)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]*descriptorpb.FileDescriptorProto)

	for _, svc := range services.GetService() {
		name := svc.GetName()
		if shouldSkipService(name) {
			continue
		}

		if err := fetchServiceDescriptors(stream, seen, name); err != nil {
			return nil, err
		}
	}

	if len(seen) == 0 {
		return nil, errNoUsableServices
	}

	return buildResult(seen), nil
}

func listServices(stream reflectionpb.ServerReflection_ServerReflectionInfoClient) (*reflectionpb.ListServiceResponse, error) {
	if err := stream.Send(&reflectionpb.ServerReflectionRequest{
		MessageRequest: &reflectionpb.ServerReflectionRequest_ListServices{},
	}); err != nil {
		return nil, errors.Wrap(err, "failed to send ListServices")
	}

	listResp, err := stream.Recv()
	if err != nil {
		return nil, errors.Wrap(err, "failed to recv ListServices response")
	}

	services := listResp.GetListServicesResponse()
	if services == nil {
		return nil, errUnexpectedListServicesResponse
	}

	return services, nil
}

func fetchServiceDescriptors(
	stream reflectionpb.ServerReflection_ServerReflectionInfoClient,
	seen map[string]*descriptorpb.FileDescriptorProto,
	serviceName string,
) error {
	if err := stream.Send(&reflectionpb.ServerReflectionRequest{
		MessageRequest: &reflectionpb.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: serviceName},
	}); err != nil {
		return errors.Wrapf(err, "failed to send FileContainingSymbol for %s", serviceName)
	}

	fdResp, err := stream.Recv()
	if err != nil {
		return errors.Wrapf(err, "failed to recv FileContainingSymbol for %s", serviceName)
	}

	return collectDescriptors(seen, fdResp, serviceName)
}

func shouldSkipService(name string) bool {
	return name == "" || name == serviceReflectionV1 || name == serviceReflectionV1Alpha || name == serviceHealth
}

func collectDescriptors(
	seen map[string]*descriptorpb.FileDescriptorProto,
	resp *reflectionpb.ServerReflectionResponse,
	serviceName string,
) error {
	fd := resp.GetFileDescriptorResponse()
	if fd == nil {
		if errResp := resp.GetErrorResponse(); errResp != nil {
			return errors.Wrapf(errReflectionError, "for %s: %s", serviceName, errResp.GetErrorMessage())
		}

		return errors.Wrapf(errUnexpectedFileResponse, "for %s", serviceName)
	}

	for _, raw := range fd.GetFileDescriptorProto() {
		var fdp descriptorpb.FileDescriptorProto
		if err := proto.Unmarshal(raw, &fdp); err != nil {
			return errors.Wrapf(err, "failed to unmarshal FileDescriptorProto for %s", serviceName)
		}

		key := fdp.GetName()
		if key == "" {
			key = fmt.Sprintf("%s.%s", fdp.GetPackage(), "unknown")
		}

		if _, ok := seen[key]; !ok {
			seen[key] = &fdp
		}
	}

	return nil
}

func buildResult(seen map[string]*descriptorpb.FileDescriptorProto) *descriptorpb.FileDescriptorSet {
	keys := make([]string, 0, len(seen))
	for name := range seen {
		keys = append(keys, name)
	}

	sort.Strings(keys)

	out := &descriptorpb.FileDescriptorSet{File: make([]*descriptorpb.FileDescriptorProto, 0, len(keys))}
	for _, name := range keys {
		out.File = append(out.File, seen[name])
	}

	return out
}
