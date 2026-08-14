package app

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	reflectionv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	reflectionv1alpha "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	protosetinfra "github.com/bavix/gripmock/v3/internal/infra/protoset"
)

const (
	reflectionServiceV1      = "grpc.reflection.v1.ServerReflection"
	reflectionServiceV1Alpha = "grpc.reflection.v1alpha.ServerReflection"
	reflectionMethodInfo     = "ServerReflectionInfo"
)

func isReflectionMethod(service, method string) bool {
	if method != reflectionMethodInfo {
		return false
	}

	return service == reflectionServiceV1 || service == reflectionServiceV1Alpha
}

// gatewayReflection answers gRPC server reflection over the HTTP gateways.
// A Connect or gRPC-Web client picks its content type and framing from whether
// the method streams, so without reflection on the same port a client that has
// no local descriptors has no way to get that right.
type gatewayReflection struct {
	v1       reflectionv1.ServerReflectionServer
	v1alpha  reflectionv1alpha.ServerReflectionServer
	resolver *protosetinfra.TypeResolver
}

func newGatewayReflection(registry *descriptors.Registry) *gatewayReflection {
	resolver := &dynamicDescriptorResolver{static: protoregistry.GlobalFiles, dynamic: registry}
	opts := reflection.ServerOptions{
		Services:           &gatewayServiceInfoProvider{registry: registry},
		DescriptorResolver: resolver,
	}

	return &gatewayReflection{
		v1:       reflection.NewServerV1(opts),
		v1alpha:  reflection.NewServer(opts),
		resolver: protosetinfra.NewTypeResolver(resolver),
	}
}

// gatewayServiceInfoProvider lists what the gateway can dispatch. The gRPC
// server answers ListServices from grpc.Server.GetServiceInfo, but the gateways
// route straight off the descriptors, so the service list has to be read from
// there instead: runtime-compiled protos land in GlobalFiles, proxy-fetched
// ones in the dynamic registry.
type gatewayServiceInfoProvider struct {
	registry *descriptors.Registry
}

func (p *gatewayServiceInfoProvider) GetServiceInfo() map[string]grpc.ServiceInfo {
	result := make(map[string]grpc.ServiceInfo)

	collect := func(file protoreflect.FileDescriptor) bool {
		services := file.Services()
		for i := range services.Len() {
			result[string(services.Get(i).FullName())] = grpc.ServiceInfo{}
		}

		return true
	}

	protoregistry.GlobalFiles.RangeFiles(collect)

	if p.registry != nil {
		p.registry.RangeFiles(collect)
	}

	return result
}

func (g *gatewayReflection) serve(service string, stream grpc.ServerStream) error {
	if service == reflectionServiceV1Alpha {
		return g.v1alpha.ServerReflectionInfo(&reflectionStreamV1Alpha{ServerStream: stream})
	}

	return g.v1.ServerReflectionInfo(&reflectionStreamV1{ServerStream: stream})
}

type reflectionStreamV1 struct {
	grpc.ServerStream
}

func (s *reflectionStreamV1) Send(m *reflectionv1.ServerReflectionResponse) error {
	return s.SendMsg(m)
}

func (s *reflectionStreamV1) Recv() (*reflectionv1.ServerReflectionRequest, error) {
	m := new(reflectionv1.ServerReflectionRequest)
	if err := s.RecvMsg(m); err != nil {
		return nil, err
	}

	return m, nil
}

// The v1alpha service is deprecated upstream but still the only reflection API
// some clients speak, so gripmock keeps answering it.
type reflectionStreamV1Alpha struct {
	grpc.ServerStream
}

//nolint:staticcheck
func (s *reflectionStreamV1Alpha) Send(m *reflectionv1alpha.ServerReflectionResponse) error {
	return s.SendMsg(m)
}

//nolint:staticcheck
func (s *reflectionStreamV1Alpha) Recv() (*reflectionv1alpha.ServerReflectionRequest, error) {
	m := new(reflectionv1alpha.ServerReflectionRequest)
	if err := s.RecvMsg(m); err != nil {
		return nil, err
	}

	return m, nil
}
