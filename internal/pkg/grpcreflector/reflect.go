package grpcreflector

import (
	"context"
	"slices"
	"strings"

	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
)

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

	service := splits[len(splits)-1]
	slices.Delete(splits, len(splits)-1, 1)

	return Service{
		ID:      serviceID,
		Package: strings.Join(splits, sep),
		Name:    service,
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
		results[i] = g.makeService(service)
	}

	return results, nil
}

func (g *GReflector) Methods(ctx context.Context, serviceID string) ([]Method, error) {
	dest, err := g.client(ctx).ResolveService(serviceID)
	if err != nil {
		return nil, err
	}

	results := make([]Method, len(dest.GetMethods()))

	for i, method := range dest.GetMethods() {
		results[i] = g.makeMethod(serviceID, method.GetName())
	}

	return results, err
}
