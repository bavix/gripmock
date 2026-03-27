package proxyroutes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"

	protosetdom "github.com/bavix/gripmock/v3/internal/domain/protoset"
)

type fakeRemoteClient struct {
	sets map[string]*descriptorpb.FileDescriptorSet
}

func (f fakeRemoteClient) FetchDescriptorSet(_ context.Context, source *protosetdom.Source) (*descriptorpb.FileDescriptorSet, error) {
	return f.sets[source.ReflectAddress], nil
}

func TestRegistryRouteByMethodNoFallback(t *testing.T) {
	t.Parallel()

	route := &Route{Mode: ModeProxy}
	r := &Registry{
		routes: []*Route{route},
		index: map[string]*Route{
			"/svc/Method": route,
		},
	}

	require.Same(t, route, r.RouteByMethod("/svc/Method"))
	require.Nil(t, r.RouteByMethod("/svc/Unknown"))
}

func TestNewFirstSourceWinsPerService(t *testing.T) {
	t.Parallel()

	client := fakeRemoteClient{sets: map[string]*descriptorpb.FileDescriptorSet{
		"proxy:123": buildDescriptorSet(map[string][]string{
			"greeter":  {"Ping"},
			"greeter1": {"Ping"},
		}),
		"proxy1:321": buildDescriptorSet(map[string][]string{
			"greeter1": {"Ping"},
			"greeter2": {"Ping"},
		}),
		"proxy2:444": buildDescriptorSet(map[string][]string{
			"greeter2": {"Ping"},
			"greeter3": {"Ping"},
		}),
	}}

	r, err := New(context.Background(), []string{
		"grpc+proxy://proxy:123",
		"grpc+replay://proxy1:321",
		"grpc+capture://proxy2:444",
	}, client)
	require.NoError(t, err)
	t.Cleanup(r.Close)

	require.Equal(t, ModeProxy, r.RouteByMethod("/greeter/Ping").Mode)
	require.Equal(t, ModeProxy, r.RouteByMethod("/greeter1/Ping").Mode)
	require.Equal(t, ModeReplay, r.RouteByMethod("/greeter2/Ping").Mode)
	require.Equal(t, ModeCapture, r.RouteByMethod("/greeter3/Ping").Mode)
	require.Nil(t, r.RouteByMethod("/unknown/Method"))
}

func buildDescriptorSet(services map[string][]string) *descriptorpb.FileDescriptorSet {
	fileName := new(string)
	*fileName = "test.proto"
	file := &descriptorpb.FileDescriptorProto{Name: fileName}

	for serviceName, methods := range services {
		svcName := new(string)
		*svcName = serviceName
		svc := &descriptorpb.ServiceDescriptorProto{Name: svcName}

		for _, method := range methods {
			methodName := new(string)
			*methodName = method
			svc.Method = append(svc.Method, &descriptorpb.MethodDescriptorProto{Name: methodName})
		}

		file.Service = append(file.Service, svc)
	}

	return &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{file}}
}
