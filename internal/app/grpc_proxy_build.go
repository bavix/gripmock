package app

//nolint:revive
import (
	"context"
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	protosetdom "github.com/bavix/gripmock/v3/internal/domain/protoset"
	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func (s *GRPCServer) buildProxiesWithBindings(ctx context.Context, imports []string) (
	[]*descriptorpb.FileDescriptorSet,
	*proxyroutes.Registry,
	error,
) {
	var err error

	bindings := s.params.ProxyBindings()
	proxyBindings := make([]proxyroutes.ProxyDescriptorBinding, 0, len(bindings))
	logger := zerolog.Ctx(ctx)

	for _, binding := range bindings {
		logger.Info().
			Str("proxy", binding.ProxyURL).
			Strs("sources", binding.Sources).
			Msg("processing proxy binding")

		var bindingDescriptors []*descriptorpb.FileDescriptorSet

		if len(binding.Sources) > 0 {
			bindingDescriptors, err = protosetdom.Build(ctx, imports, binding.Sources, s.remoteClient)
			if err != nil {
				return nil, nil, errors.Wrapf(err, "failed to build descriptors for proxy %s", binding.ProxyURL)
			}

			logger.Info().
				Str("proxy", binding.ProxyURL).
				Int("num_descriptors", len(bindingDescriptors)).
				Msg("built descriptors for proxy")
		} else {
			logger.Info().
				Str("proxy", binding.ProxyURL).
				Msg("no sources for proxy, will use reflection")
		}

		proxyBindings = append(proxyBindings, proxyroutes.ProxyDescriptorBinding{
			ProxyURL:    binding.ProxyURL,
			Descriptors: bindingDescriptors,
		})
	}

	proxies, err := proxyroutes.NewWithPerProxyDescriptors(ctx, proxyBindings, s.remoteClient)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to initialize proxy routes")
	}

	proxyFiles := proxies.Files()
	if len(proxyFiles) > 0 {
		descriptors := make([]*descriptorpb.FileDescriptorSet, 0, len(proxyFiles))

		return append(descriptors, proxyFiles...), proxies, nil
	}

	return nil, proxies, nil
}

func (s *GRPCServer) buildProxiesFromSources(ctx context.Context, imports []string, protoPaths []string, sources []string) (
	[]*descriptorpb.FileDescriptorSet,
	*proxyroutes.Registry,
	error,
) {
	allPaths := make([]string, 0, len(protoPaths)+len(sources))
	allPaths = append(allPaths, protoPaths...)
	allPaths = append(allPaths, sources...)

	var descriptors []*descriptorpb.FileDescriptorSet

	if len(allPaths) > 0 {
		var err error

		descriptors, err = protosetdom.Build(ctx, imports, allPaths, s.remoteClient)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to build descriptors")
		}
	}

	proxies, err := proxyroutes.New(ctx, allPaths, s.remoteClient, descriptors)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to initialize proxy routes")
	}

	return descriptors, proxies, nil
}

func (s *GRPCServer) startProxyCleanup(ctx context.Context) {
	go func() {
		<-ctx.Done()
		s.proxies.Close()
	}()
}

func (s *GRPCServer) registerProxyDescriptors(ctx context.Context) {
	proxyFiles := s.proxies.Files()
	if len(proxyFiles) == 0 {
		return
	}

	for i, fds := range proxyFiles {
		source := fmt.Sprintf("proxy-descriptor-set-%d", i)
		if err := protosetdom.RegisterDescriptorSetFiles(ctx, source, fds); err != nil {
			zerolog.Ctx(ctx).Err(err).Int("index", i).Msg("failed to register proxy descriptor set")
		}
	}
}

func BuildFromDescriptorSet(
	ctx context.Context,
	fds *descriptorpb.FileDescriptorSet,
	budgerigar *stuber.Budgerigar,
	waiter Extender,
	recorder history.Recorder,
) (*grpc.Server, error) {
	reg, err := protodesc.NewFiles(fds)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create files registry")
	}

	var healthState stuber.Aliveness
	if budgerigar != nil {
		healthState = budgerigar
	}

	s := &GRPCServer{
		budgerigar:     budgerigar,
		healthState:    healthState,
		waiter:         waiter,
		recorder:       recorder,
		descriptors:    descriptors.NewRegistry(),
		validator:      mustNewStubValidator(),
		errorFormatter: NewErrorFormatter(),
	}
	server := s.createServer(ctx)
	s.setupHealthCheck(server, reg)
	s.registerServices(ctx, server, []*descriptorpb.FileDescriptorSet{fds}, reg)

	// Mark server as ready synchronously after all descriptors and stubs are loaded.
	s.markServerReady(ctx)

	return server, nil
}
