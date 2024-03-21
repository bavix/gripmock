package grpcreflector

import (
	"context"
	"strings"

	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
)

const prefix = "grpc.reflection.v1"

type Service struct {
	ID      string
	Package string
	Name    string
}

type Method struct {
	Service Service
	Name    string
}

type GReflector struct {
	conn *grpc.ClientConn
}

func New(conn *grpc.ClientConn) *GReflector {
	return &GReflector{conn: conn}
}

func (g *GReflector) client(ctx context.Context) *grpcreflect.Client {
	return grpcreflect.NewClientAuto(ctx, g.conn)
}

func (g *GReflector) makeService(serviceID string) Service {
	const sep = "."

	splits := strings.Split(serviceID, sep)

	return Service{
		ID:      serviceID,
		Package: strings.Join(splits[:len(splits)-1], sep),
		Name:    splits[len(splits)-1],
	}
}

func (g *GReflector) makeMethod(service, method string) Method {
	return Method{
		Service: g.makeService(service),
		Name:    method,
	}
}

func (g *GReflector) Services(ctx context.Context) ([]Service, error) {
	services, err := g.client(ctx).ListServices()
	if err != nil {
		return nil, err
	}

	results := make([]Service, len(services))

	for i, service := range services {
		if !strings.HasPrefix(service, prefix) {
			results[i] = g.makeService(service)
		}
	}

	return results, nil
}

func (g *GReflector) Methods(ctx context.Context, serviceID string) ([]Method, error) {
	dest, err := g.client(ctx).ResolveService(serviceID)
	if err != nil {
		return nil, err
	}

	results := make([]Method, len(dest.GetMethods()))

	if !strings.HasPrefix(serviceID, prefix) {
		for i, method := range dest.GetMethods() {
			results[i] = g.makeMethod(serviceID, method.GetName())
		}
	}

	return results, err
}
