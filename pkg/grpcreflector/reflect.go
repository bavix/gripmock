package grpcreflector

import (
	"context"
	"strings"

	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
)

// GReflector is a client for the gRPC reflection API.
// It provides methods to list services and methods available on a gRPC server.
type GReflector struct {
	conn *grpc.ClientConn // grpc connection to the server
}

// Service represents a gRPC service.
type Service struct {
	ID      string // service ID
	Package string // service package
	Name    string // service name
}

// Method represents a gRPC method.
type Method struct {
	ID   string // method ID
	Name string // method name
}

const prefix = "grpc.reflection.v1"

// New creates a new GReflector with the given grpc connection.
func New(conn *grpc.ClientConn) *GReflector {
	return &GReflector{conn: conn}
}

// client returns a new gRPC reflection client.
// It uses the given context and the grpc connection of the GReflector.
func (g *GReflector) client(ctx context.Context) *grpcreflect.Client {
	return grpcreflect.NewClientAuto(ctx, g.conn)
}

// makeService creates a Service struct from a service ID.
// The service ID is split into its package and name parts.
func (g *GReflector) makeService(serviceID string) Service {
	const sep = "."

	splits := strings.Split(serviceID, sep)

	return Service{
		ID:      serviceID,
		Package: strings.Join(splits[:len(splits)-1], sep),
		Name:    splits[len(splits)-1],
	}
}

// makeMethod creates a Method struct from a service ID and method name.
// The method ID is created by concatenating the service ID and method name with a slash.
func (g *GReflector) makeMethod(serviceID, method string) Method {
	return Method{
		ID:   serviceID + "/" + method,
		Name: method,
	}
}

// Services lists all services available on the gRPC server.
// It uses the gRPC reflection client to get the list of services and filters out the reflection service.
func (g *GReflector) Services(ctx context.Context) ([]Service, error) {
	services, err := g.client(ctx).ListServices()
	if err != nil {
		return nil, err
	}

	results := make([]Service, 0, len(services))

	for _, service := range services {
		if !strings.HasPrefix(service, prefix) {
			results = append(results, g.makeService(service))
		}
	}

	return results, nil
}

// Methods lists all methods available on a service.
// It uses the gRPC reflection client to resolve the service and filter out the reflection methods.
func (g *GReflector) Methods(ctx context.Context, serviceID string) ([]Method, error) {
	dest, err := g.client(ctx).ResolveService(serviceID)
	if err != nil {
		return nil, err
	}

	results := make([]Method, 0, len(dest.GetMethods()))

	if !strings.HasPrefix(serviceID, prefix) {
		for _, method := range dest.GetMethods() {
			results = append(results, g.makeMethod(serviceID, method.GetName()))
		}
	}

	return results, err
}
