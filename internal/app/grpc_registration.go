package app

//nolint:revive
import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
	reflectiongrpc "google.golang.org/grpc/reflection/grpc_reflection_v1"
	reflectiongrpcv1alpha "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/infra/grpccontext"
	protosetinfra "github.com/bavix/gripmock/v3/internal/infra/protoset"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

func (s *GRPCServer) createServer(ctx context.Context) *grpc.Server {
	logger := zerolog.Ctx(ctx)

	opts := []grpc.ServerOption{
		grpc.NumStreamWorkers(uint32(runtimeNumStreamWorkers)), //nolint:gosec
		grpc.MaxConcurrentStreams(maxConcurrentStreams),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     keepaliveMaxIdle,
			MaxConnectionAge:      keepaliveMaxAge,
			MaxConnectionAgeGrace: keepaliveMaxAgeGrace,
			Time:                  keepaliveTime,
			Timeout:               keepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             keepaliveMinTime,
			PermitWithoutStream: true,
		}),
		grpc.ChainUnaryInterceptor(
			grpccontext.PanicRecoveryUnaryInterceptor,
			grpccontext.UnaryInterceptor(logger),
			LogUnaryInterceptor,
		),
		grpc.ChainStreamInterceptor(
			grpccontext.PanicRecoveryStreamInterceptor,
			grpccontext.StreamInterceptor(logger),
			LogStreamInterceptor,
		),
		grpc.UnknownServiceHandler(s.handleUnknownService),
	}

	if s.otelEnabled {
		opts = append(opts, grpc.StatsHandler(otelgrpc.NewServerHandler()))
	}

	if s.tlsConfig != nil {
		opts = append(opts, grpc.Creds(credentials.NewTLS(s.tlsConfig)))

		logger.Info().Msg("gRPC server configured with TLS")
	}

	return grpc.NewServer(opts...)
}

func (s *GRPCServer) handleUnknownService(_ any, stream grpc.ServerStream) error {
	fullMethod, ok := grpc.MethodFromServerStream(stream)
	if !ok {
		return status.Error(codes.Unimplemented, "method not found")
	}

	serviceName, methodName := splitMethodName(fullMethod)
	if serviceName == unknownValue || methodName == unknownValue {
		return status.Error(codes.Unimplemented, "method not found")
	}

	methodDesc, err := s.findMethodDescriptor(serviceName, methodName)
	if err != nil {
		return status.Error(codes.Unimplemented, err.Error())
	}

	templateEngine := template.New(stream.Context(), nil)
	mocker := &grpcMocker{
		budgerigar:         s.budgerigar,
		templateEngine:     templateEngine,
		errorFormatter:     s.errorFormatter,
		recorder:           s.recorder,
		descriptorResolver: &dynamicDescriptorResolver{static: protoregistry.GlobalFiles, dynamic: s.descriptors},
		proxies:            s.proxies,
		validator:          s.validator,
		maxNestingDepth:    s.maxNestingDepth,
		inputDesc:          methodDesc.Input(),
		outputDesc:         methodDesc.Output(),
		fullServiceName:    serviceName,
		serviceName:        serviceName,
		methodName:         methodName,
		fullMethod:         fullMethod,
		serverStream:       methodDesc.IsStreamingServer(),
		clientStream:       methodDesc.IsStreamingClient(),
		strictServiceMatch: s.proxies != nil && s.proxies.RouteByMethod(fullMethod) != nil,
	}

	if methodDesc.IsStreamingServer() || methodDesc.IsStreamingClient() {
		return mocker.streamHandler(nil, stream)
	}

	req := dynamicpb.NewMessage(methodDesc.Input())
	if err := stream.RecvMsg(req); err != nil {
		return err
	}

	resp, err := mocker.handleUnary(stream.Context(), stream, req)
	if err != nil {
		return err
	}

	return stream.SendMsg(resp)
}

func (s *GRPCServer) findMethodDescriptor(serviceName, methodName string) (protoreflect.MethodDescriptor, error) { //nolint:ireturn
	if method := findMethodInGlobalFiles(serviceName, methodName); method != nil {
		return method, nil
	}

	var found protoreflect.MethodDescriptor

	s.descriptors.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		services := file.Services()
		for i := range services.Len() {
			service := services.Get(i)
			if string(service.FullName()) != serviceName {
				continue
			}

			methods := service.Methods()
			for j := range methods.Len() {
				method := methods.Get(j)
				if string(method.Name()) != methodName {
					continue
				}

				found = method

				return false
			}
		}

		return true
	})

	if found == nil {
		return nil, errors.Errorf("unknown service/method: %s/%s", serviceName, methodName)
	}

	return found, nil
}

func findMethodInGlobalFiles(serviceName, methodName string) protoreflect.MethodDescriptor { //nolint:ireturn
	return findMethodInFiles(protoregistry.GlobalFiles, serviceName, methodName)
}

type methodFilesLister interface {
	RangeFiles(f func(protoreflect.FileDescriptor) bool)
}

func findMethodInFiles(files methodFilesLister, serviceName, methodName string) protoreflect.MethodDescriptor { //nolint:ireturn
	var found protoreflect.MethodDescriptor

	files.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		services := file.Services()
		for i := range services.Len() {
			service := services.Get(i)
			if string(service.FullName()) != serviceName {
				continue
			}

			methods := service.Methods()
			for j := range methods.Len() {
				method := methods.Get(j)
				if string(method.Name()) != methodName {
					continue
				}

				found = method

				return false
			}
		}

		return true
	})

	return found
}

func (s *GRPCServer) setupHealthCheck(server *grpc.Server, descResolver *protoregistry.Files) {
	healthServer := health.NewServer()
	healthgrpc.RegisterHealthServer(server, newMockableHealthServer(healthServer, s.budgerigar, descResolver, s.proxies))

	provider := &dynamicServiceInfoProvider{base: server, registry: s.descriptors}

	var staticResolver protodesc.Resolver = protoregistry.GlobalFiles
	if descResolver != nil {
		staticResolver = descResolver
	}

	resolver := &dynamicDescriptorResolver{
		static:  staticResolver,
		dynamic: s.descriptors,
	}

	reflectionSvr := reflection.NewServerV1(reflection.ServerOptions{
		Services:           provider,
		DescriptorResolver: resolver,
	})
	reflectiongrpc.RegisterServerReflectionServer(server, reflectionSvr)

	reflectiongrpcv1alpha.RegisterServerReflectionServer(server, reflection.NewServer(reflection.ServerOptions{
		Services:           provider,
		DescriptorResolver: resolver,
	}))
}

type dynamicServiceInfoProvider struct {
	base     reflection.ServiceInfoProvider
	registry *descriptors.Registry
}

func (p *dynamicServiceInfoProvider) GetServiceInfo() map[string]grpc.ServiceInfo {
	result := make(map[string]grpc.ServiceInfo)

	if p.base != nil {
		maps.Copy(result, p.base.GetServiceInfo())
	}

	if p.registry != nil {
		p.registry.RangeFiles(func(file protoreflect.FileDescriptor) bool {
			services := file.Services()
			for i := range services.Len() {
				serviceName := string(services.Get(i).FullName())
				if _, ok := result[serviceName]; !ok {
					result[serviceName] = grpc.ServiceInfo{}
				}
			}

			return true
		})
	}

	return result
}

type dynamicDescriptorResolver struct {
	static  protodesc.Resolver
	dynamic *descriptors.Registry
}

func (r *dynamicDescriptorResolver) FindFileByPath(path string) (protoreflect.FileDescriptor, error) { //nolint:ireturn
	return (&protosetinfra.Fallback{Primary: r.dynamicFiles(), Fallback: r.static}).FindFileByPath(path)
}

func (r *dynamicDescriptorResolver) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) { //nolint:ireturn
	return (&protosetinfra.Fallback{Primary: r.dynamicFiles(), Fallback: r.static}).FindDescriptorByName(name)
}

func (r *dynamicDescriptorResolver) dynamicFiles() *protoregistry.Files {
	if r.dynamic == nil {
		return nil
	}

	reg := new(protoregistry.Files)

	r.dynamic.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		_ = reg.RegisterFile(file)

		return true
	})

	return reg
}

func (s *GRPCServer) registerServices(
	ctx context.Context,
	server *grpc.Server,
	descriptors []*descriptorpb.FileDescriptorSet,
	reg *protoregistry.Files,
) {
	logger := zerolog.Ctx(ctx)
	registered := make(map[string]struct{})

	for serviceName := range server.GetServiceInfo() {
		registered[serviceName] = struct{}{}
	}

	for _, descriptor := range descriptors {
		for _, file := range descriptor.GetFile() {
			for _, svc := range file.GetService() {
				serviceDesc := s.createServiceDesc(file, svc)

				if _, exists := registered[serviceDesc.ServiceName]; exists {
					logger.Warn().Str("service", serviceDesc.ServiceName).Msg("Service already registered; skipping")

					continue
				}

				if err := s.registerServiceMethods(ctx, &serviceDesc, svc, reg); err != nil {
					logger.Warn().Err(err).Str("service", serviceDesc.ServiceName).Msg("Skipping service due to descriptor error")

					continue
				}

				server.RegisterService(&serviceDesc, nil)
				registered[serviceDesc.ServiceName] = struct{}{}
				logger.Info().Str("service", serviceDesc.ServiceName).Msg("Registered gRPC service")
			}
		}
	}
}

func (s *GRPCServer) createServiceDesc(file *descriptorpb.FileDescriptorProto, svc *descriptorpb.ServiceDescriptorProto) grpc.ServiceDesc {
	return grpc.ServiceDesc{
		ServiceName: getServiceName(file, svc),
		HandlerType: (*any)(nil),
	}
}

func (s *GRPCServer) registerServiceMethods(
	ctx context.Context,
	serviceDesc *grpc.ServiceDesc,
	svc *descriptorpb.ServiceDescriptorProto,
	reg *protoregistry.Files,
) error {
	for _, method := range svc.GetMethod() {
		inputDesc, outputDesc, err := s.resolveMethodMessageDescriptors(serviceDesc.ServiceName, method, reg)
		if err != nil {
			return err
		}

		m := s.createGrpcMocker(ctx, serviceDesc, svc, method, inputDesc, outputDesc, reg)

		if method.GetServerStreaming() || method.GetClientStreaming() {
			serviceDesc.Streams = append(serviceDesc.Streams, grpc.StreamDesc{
				StreamName:    method.GetName(),
				Handler:       m.streamHandler,
				ServerStreams: m.serverStream,
				ClientStreams: m.clientStream,
			})
		} else {
			serviceDesc.Methods = append(serviceDesc.Methods, grpc.MethodDesc{
				MethodName: method.GetName(),
				Handler:    m.unaryHandler(),
			})
		}
	}

	return nil
}

//nolint:ireturn
func (s *GRPCServer) resolveMethodMessageDescriptors(
	serviceName string,
	method *descriptorpb.MethodDescriptorProto,
	reg *protoregistry.Files,
) (protoreflect.MessageDescriptor, protoreflect.MessageDescriptor, error) {
	if reg != nil {
		inputDesc, err := getMessageDescriptor(reg, method.GetInputType())
		if err == nil {
			outputDesc, outErr := getMessageDescriptor(reg, method.GetOutputType())
			if outErr == nil {
				return inputDesc, outputDesc, nil
			}
		}
	}

	methodDesc, err := s.findMethodDescriptor(serviceName, method.GetName())
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to resolve method descriptor %s/%s", serviceName, method.GetName())
	}

	return methodDesc.Input(), methodDesc.Output(), nil
}

func (s *GRPCServer) createGrpcMocker(
	ctx context.Context,
	serviceDesc *grpc.ServiceDesc,
	svc *descriptorpb.ServiceDescriptorProto,
	method *descriptorpb.MethodDescriptorProto,
	inputDesc, outputDesc protoreflect.MessageDescriptor,
	reg *protoregistry.Files,
) *grpcMocker {
	templateEngine := template.New(ctx, nil)

	var resolver protodesc.Resolver = protoregistry.GlobalFiles
	if reg != nil {
		resolver = reg
	}

	return &grpcMocker{
		budgerigar:         s.budgerigar,
		templateEngine:     templateEngine,
		errorFormatter:     s.errorFormatter,
		recorder:           s.recorder,
		descriptorResolver: resolver,
		proxies:            s.proxies,
		validator:          s.validator,
		maxNestingDepth:    s.maxNestingDepth,

		inputDesc:  inputDesc,
		outputDesc: outputDesc,

		fullServiceName: serviceDesc.ServiceName,
		serviceName:     svc.GetName(),
		methodName:      method.GetName(),
		fullMethod:      fmt.Sprintf("/%s/%s", serviceDesc.ServiceName, method.GetName()),

		serverStream: method.GetServerStreaming(),
		clientStream: method.GetClientStreaming(),

		strictServiceMatch: s.proxies != nil && s.proxies.RouteByMethod(fmt.Sprintf("/%s/%s", serviceDesc.ServiceName, method.GetName())) != nil,
	}
}

func (s *GRPCServer) markServerReady(ctx context.Context) {
	logger := zerolog.Ctx(ctx)

	logger.Info().Msg("gRPC server is ready to accept requests")

	if s.healthState != nil {
		s.healthState.SetAlive()
	}
}

func getServiceName(file *descriptorpb.FileDescriptorProto, svc *descriptorpb.ServiceDescriptorProto) string {
	if file.GetPackage() != "" {
		return fmt.Sprintf("%s.%s", file.GetPackage(), svc.GetName())
	}

	return svc.GetName()
}

func getMessageDescriptor(reg *protoregistry.Files, messageType string) (protoreflect.MessageDescriptor, error) { //nolint:ireturn
	if reg == nil {
		reg = protoregistry.GlobalFiles
	}

	msgName := protoreflect.FullName(strings.TrimPrefix(messageType, "."))

	desc, err := reg.FindDescriptorByName(msgName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Message descriptor not found: %v", err)
	}

	msgDesc, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, status.Errorf(codes.Internal, "Not a message descriptor: %s", msgName)
	}

	return msgDesc, nil
}
