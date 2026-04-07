package proxyroutes

import (
	"context"

	"github.com/cockroachdb/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/descriptorpb"

	protosetdom "github.com/bavix/gripmock/v3/internal/domain/protoset"
	grpcclient "github.com/bavix/gripmock/v3/internal/infra/grpcclient"
)

var errRemoteClientNil = errors.New("remote client is not configured")

const (
	descriptorMethodsInitCap = 16
)

type Mode uint8

const (
	ModeProxy Mode = iota + 1
	ModeReplay
	ModeCapture
)

type Route struct {
	Mode   Mode
	Source *protosetdom.Source
	Conn   *grpc.ClientConn
}

type Registry struct {
	routes []*Route
	index  map[string]*Route
}

//nolint:cyclop,funlen
func New(ctx context.Context, paths []string, remoteClient protosetdom.RemoteClient) (*Registry, error) {
	sources := make([]*protosetdom.Source, 0, len(paths))

	for _, path := range paths {
		source, err := protosetdom.ParseSource(path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse source: %s", path)
		}

		if source.ProxyMode == "" {
			continue
		}

		sources = append(sources, source)
	}

	if len(sources) == 0 {
		return &Registry{}, nil
	}

	if remoteClient == nil {
		return nil, errRemoteClientNil
	}

	routes := make([]*Route, 0, len(sources))
	index := make(map[string]*Route)
	assignedServices := make(map[string]struct{})

	for _, source := range sources {
		fds, err := remoteClient.FetchDescriptorSet(ctx, source)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to fetch proxy descriptors: %s", source.Raw)
		}

		conn, err := grpc.NewClient("passthrough:///"+source.ReflectAddress, grpcclient.DialOptions(
			source.ReflectTimeout,
			source.ReflectTLS,
			source.ReflectServerName,
			source.ReflectBearer,
			source.ReflectInsecure,
		)...)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to connect proxy upstream: %s", source.ReflectAddress)
		}

		route := &Route{
			Mode:   mapMode(source.ProxyMode),
			Source: source,
			Conn:   conn,
		}

		for service, methods := range collectServiceMethods(fds) {
			if _, exists := assignedServices[service]; exists {
				continue
			}

			assignedServices[service] = struct{}{}

			for _, method := range methods {
				if _, exists := index[method]; !exists {
					index[method] = route
				}
			}
		}

		routes = append(routes, route)
	}

	return &Registry{routes: routes, index: index}, nil
}

func (r *Registry) RouteByMethod(fullMethod string) *Route {
	if r == nil {
		return nil
	}

	if route, ok := r.index[fullMethod]; ok {
		return route
	}

	return nil
}

func (r *Registry) Routes() []*Route {
	if r == nil {
		return nil
	}

	return r.routes
}

func (r *Registry) Close() {
	if r == nil {
		return
	}

	for _, route := range r.routes {
		if route == nil || route.Conn == nil {
			continue
		}

		_ = route.Conn.Close()
	}
}

func (r *Route) WithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if r == nil || r.Source == nil {
		return ctx, func() {}
	}

	if _, hasDeadline := ctx.Deadline(); hasDeadline || r.Source.ReflectTimeout <= 0 {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, r.Source.ReflectTimeout)
}

func ForwardIncomingMetadata(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || len(md) == 0 {
		return ctx
	}

	return metadata.NewOutgoingContext(ctx, md.Copy())
}

func mapMode(mode string) Mode {
	switch mode {
	case "proxy":
		return ModeProxy
	case "capture":
		return ModeCapture
	case "replay":
		return ModeReplay
	default:
		return ModeProxy
	}
}

func collectServiceMethods(fds *descriptorpb.FileDescriptorSet) map[string][]string {
	if fds == nil {
		return nil
	}

	serviceMethods := make(map[string][]string)

	for _, file := range fds.GetFile() {
		pkg := file.GetPackage()

		for _, service := range file.GetService() {
			serviceName := service.GetName()
			if pkg != "" {
				serviceName = pkg + "." + serviceName
			}

			methods := serviceMethods[serviceName]
			if methods == nil {
				methods = make([]string, 0, descriptorMethodsInitCap)
			}

			for _, method := range service.GetMethod() {
				methods = append(methods, "/"+serviceName+"/"+method.GetName())
			}

			serviceMethods[serviceName] = methods
		}
	}

	return serviceMethods
}
