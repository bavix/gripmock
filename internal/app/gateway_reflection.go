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
		//nolint:staticcheck // deprecated upstream, still the only reflection API some clients speak.
		return g.v1alpha.ServerReflectionInfo(
			&reflectionStream[reflectionv1alpha.ServerReflectionRequest, reflectionv1alpha.ServerReflectionResponse]{
				ServerStream: stream,
			})
	}

	return g.v1.ServerReflectionInfo(
		&reflectionStream[reflectionv1.ServerReflectionRequest, reflectionv1.ServerReflectionResponse]{
			ServerStream: stream,
		})
}

// reflectionStream adapts a gateway stream adapter to the generated
// ServerReflectionInfo server interface. The v1 and v1alpha services differ
// only in their message types, so one generic wrapper serves both.
type reflectionStream[Req, Resp any] struct {
	grpc.ServerStream
}

func (s *reflectionStream[Req, Resp]) Send(m *Resp) error {
	return s.SendMsg(m)
}

func (s *reflectionStream[Req, Resp]) Recv() (*Req, error) {
	m := new(Req)

	err := s.RecvMsg(m)
	if err != nil {
		return nil, err
	}

	return m, nil
}
