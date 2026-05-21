package proxyroutes

import (
	"context"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"

	protosetdom "github.com/bavix/gripmock/v3/internal/domain/protoset"
)

type fakeRemoteClient struct {
	sets    map[string]*descriptorpb.FileDescriptorSet
	calls   int
	failAll bool
}

func (f *fakeRemoteClient) FetchDescriptorSet(_ context.Context, source *protosetdom.Source) (*descriptorpb.FileDescriptorSet, error) {
	f.calls++

	if f.failAll {
		return nil, errors.New("reflection unavailable")
	}

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

	client := &fakeRemoteClient{sets: map[string]*descriptorpb.FileDescriptorSet{
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

	r, err := New(t.Context(), []string{
		"grpc+proxy://proxy:123",
		"grpc+replay://proxy1:321",
		"grpc+capture://proxy2:444",
	}, client, nil)
	require.NoError(t, err)
	t.Cleanup(r.Close)

	require.Equal(t, ModeProxy, r.RouteByMethod("/greeter/Ping").Mode)
	require.Equal(t, ModeProxy, r.RouteByMethod("/greeter1/Ping").Mode)
	require.Equal(t, ModeReplay, r.RouteByMethod("/greeter2/Ping").Mode)
	require.Equal(t, ModeCapture, r.RouteByMethod("/greeter3/Ping").Mode)
	require.Nil(t, r.RouteByMethod("/unknown/Method"))
}

func TestNewSkipsReflectionWhenLocalDescriptorsPresent(t *testing.T) {
	t.Parallel()

	client := &fakeRemoteClient{failAll: true}

	local := []*descriptorpb.FileDescriptorSet{
		buildDescriptorSet(map[string][]string{
			"orders.OrderService": {"Create", "Get"},
		}),
	}

	r, err := New(
		t.Context(),
		[]string{"grpc+capture://orders.internal:443"},
		client,
		local,
	)
	require.NoError(t, err)
	t.Cleanup(r.Close)

	require.Equal(t, 0, client.calls)
	require.Equal(t, ModeCapture, r.RouteByMethod("/orders.OrderService/Create").Mode)
	require.Equal(t, ModeCapture, r.RouteByMethod("/orders.OrderService/Get").Mode)
}

func TestNewUsesReflectionWhenLocalDescriptorsAbsent(t *testing.T) {
	t.Parallel()

	client := &fakeRemoteClient{sets: map[string]*descriptorpb.FileDescriptorSet{
		"orders.internal:443": buildDescriptorSet(map[string][]string{
			"orders.OrderService": {"Create"},
		}),
	}}

	r, err := New(
		t.Context(),
		[]string{"grpc+capture://orders.internal:443"},
		client,
		nil,
	)
	require.NoError(t, err)
	t.Cleanup(r.Close)

	require.Equal(t, 1, client.calls)
	require.Equal(t, ModeCapture, r.RouteByMethod("/orders.OrderService/Create").Mode)
}

func TestNewStoresReflectionDescriptors(t *testing.T) {
	t.Parallel()

	fds := buildDescriptorSet(map[string][]string{
		"orders.OrderService": {"Create"},
	})

	client := &fakeRemoteClient{sets: map[string]*descriptorpb.FileDescriptorSet{
		"orders.internal:443": fds,
	}}

	r, err := New(
		t.Context(),
		[]string{"grpc+capture://orders.internal:443"},
		client,
		nil,
	)
	require.NoError(t, err)
	t.Cleanup(r.Close)

	files := r.Files()
	require.Len(t, files, 1)
	require.Same(t, fds, files[0])
}

func TestNewDoesNotStoreLocalDescriptors(t *testing.T) {
	t.Parallel()

	client := &fakeRemoteClient{failAll: true}

	local := []*descriptorpb.FileDescriptorSet{
		buildDescriptorSet(map[string][]string{
			"orders.OrderService": {"Create"},
		}),
	}

	r, err := New(
		t.Context(),
		[]string{"grpc+capture://orders.internal:443"},
		client,
		local,
	)
	require.NoError(t, err)
	t.Cleanup(r.Close)

	files := r.Files()
	require.Empty(t, files)
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

func TestMultiProxyWithLocalDescriptorsSharedBehavior(t *testing.T) {
	t.Parallel()

	client := &fakeRemoteClient{failAll: true}

	local := []*descriptorpb.FileDescriptorSet{
		buildDescriptorSet(map[string][]string{
			"greeter": {"SayHello"},
			"orders":  {"CreateOrder"},
		}),
	}

	r, err := New(
		t.Context(),
		[]string{
			"grpc+proxy://upstream1:4111",
			"grpc+proxy://upstream2:4222",
		},
		client,
		local,
	)
	require.NoError(t, err)
	t.Cleanup(r.Close)

	require.Equal(t, 0, client.calls)

	greeterRoute := r.RouteByMethod("/greeter/SayHello")
	require.NotNil(t, greeterRoute)
	require.Equal(t, "upstream1:4111", greeterRoute.Source.ReflectAddress)

	ordersRoute := r.RouteByMethod("/orders/CreateOrder")
	require.NotNil(t, ordersRoute)
	require.Equal(t, "upstream1:4111", ordersRoute.Source.ReflectAddress)
}

func TestMultiProxyWithoutLocalDescriptorsUsesReflection(t *testing.T) {
	t.Parallel()

	client := &fakeRemoteClient{sets: map[string]*descriptorpb.FileDescriptorSet{
		"upstream1:4111": buildDescriptorSet(map[string][]string{
			"greeter": {"SayHello"},
		}),
		"upstream2:4222": buildDescriptorSet(map[string][]string{
			"orders": {"CreateOrder"},
		}),
	}}

	r, err := New(
		t.Context(),
		[]string{
			"grpc+proxy://upstream1:4111",
			"grpc+proxy://upstream2:4222",
		},
		client,
		nil,
	)
	require.NoError(t, err)
	t.Cleanup(r.Close)

	require.Equal(t, 2, client.calls)

	greeterRoute := r.RouteByMethod("/greeter/SayHello")
	require.NotNil(t, greeterRoute)
	require.Equal(t, "upstream1:4111", greeterRoute.Source.ReflectAddress)

	ordersRoute := r.RouteByMethod("/orders/CreateOrder")
	require.NotNil(t, ordersRoute)
	require.Equal(t, "upstream2:4222", ordersRoute.Source.ReflectAddress)
}

func TestLocalDescriptorsPriorityOverReflection(t *testing.T) {
	t.Parallel()

	client := &fakeRemoteClient{
		sets: map[string]*descriptorpb.FileDescriptorSet{
			"upstream:4111": buildDescriptorSet(map[string][]string{
				"greeter": {"ShouldNotAppear"},
			}),
		},
	}

	local := []*descriptorpb.FileDescriptorSet{
		buildDescriptorSet(map[string][]string{
			"greeter": {"SayHello"},
		}),
	}

	r, err := New(
		t.Context(),
		[]string{"grpc+proxy://upstream:4111"},
		client,
		local,
	)
	require.NoError(t, err)
	t.Cleanup(r.Close)

	require.Equal(t, 0, client.calls)

	route := r.RouteByMethod("/greeter/SayHello")
	require.NotNil(t, route)

	route = r.RouteByMethod("/greeter/ShouldNotAppear")
	require.Nil(t, route)
}

func TestEmptyLocalDescriptorsFallbackToReflection(t *testing.T) {
	t.Parallel()

	client := &fakeRemoteClient{sets: map[string]*descriptorpb.FileDescriptorSet{
		"upstream:4111": buildDescriptorSet(map[string][]string{
			"greeter": {"SayHello"},
		}),
	}}

	r, err := New(
		t.Context(),
		[]string{"grpc+proxy://upstream:4111"},
		client,
		[]*descriptorpb.FileDescriptorSet{},
	)
	require.NoError(t, err)
	t.Cleanup(r.Close)

	require.Equal(t, 1, client.calls)

	route := r.RouteByMethod("/greeter/SayHello")
	require.NotNil(t, route)
}

func TestMultiProxyDifferentModes(t *testing.T) {
	t.Parallel()

	client := &fakeRemoteClient{sets: map[string]*descriptorpb.FileDescriptorSet{
		"upstream1:4111": buildDescriptorSet(map[string][]string{
			"greeter": {"SayHello"},
		}),
		"upstream2:4222": buildDescriptorSet(map[string][]string{
			"orders": {"CreateOrder"},
		}),
	}}

	r, err := New(
		t.Context(),
		[]string{
			"grpc+proxy://upstream1:4111",
			"grpc+capture://upstream2:4222",
		},
		client,
		nil,
	)
	require.NoError(t, err)
	t.Cleanup(r.Close)

	require.Equal(t, 2, client.calls)

	greeterRoute := r.RouteByMethod("/greeter/SayHello")
	require.NotNil(t, greeterRoute)
	require.Equal(t, ModeProxy, greeterRoute.Mode)

	ordersRoute := r.RouteByMethod("/orders/CreateOrder")
	require.NotNil(t, ordersRoute)
	require.Equal(t, ModeCapture, ordersRoute.Mode)
}

func TestNewWithPerProxyDescriptors_EmptyBindings(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := &fakeRemoteClient{}

	r, err := NewWithPerProxyDescriptors(ctx, nil, client)
	require.NoError(t, err)
	require.NotNil(t, r)
	require.Empty(t, r.routes)
	require.Empty(t, r.index)
	require.Empty(t, r.files)
}

func TestNewWithPerProxyDescriptors_NonProxyURL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := &fakeRemoteClient{}

	// Regular grpc URL without mode should be skipped
	// Invalid URLs are also treated as non-proxy and skipped
	bindings := []ProxyDescriptorBinding{
		{
			ProxyURL:    "grpc://upstream:4111",
			Descriptors: nil,
		},
		{
			ProxyURL:    "not-a-valid-url",
			Descriptors: nil,
		},
	}

	r, err := NewWithPerProxyDescriptors(ctx, bindings, client)
	require.NoError(t, err)
	require.NotNil(t, r)
	require.Empty(t, r.routes, "non-proxy URLs should be skipped")
}

func TestNewWithPerProxyDescriptors_ReflectionFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := &fakeRemoteClient{failAll: true}

	bindings := []ProxyDescriptorBinding{
		{
			ProxyURL:    "grpc+proxy://upstream:4111",
			Descriptors: nil, // No local descriptors, should try reflection
		},
	}

	_, err := NewWithPerProxyDescriptors(ctx, bindings, client)
	require.Error(t, err)
	require.Contains(t, err.Error(), "reflection unavailable")
}

func TestNewWithPerProxyDescriptors_MixedSuccessAndSkip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := &fakeRemoteClient{sets: map[string]*descriptorpb.FileDescriptorSet{
		"upstream:4111": buildDescriptorSet(map[string][]string{
			"greeter": {"SayHello"},
		}),
	}}

	bindings := []ProxyDescriptorBinding{
		{
			ProxyURL:    "grpc://not-proxy:1111", // Should be skipped
			Descriptors: nil,
		},
		{
			ProxyURL:    "grpc+proxy://upstream:4111", // Valid proxy
			Descriptors: nil,
		},
		{
			ProxyURL:    "grpc://another-not-proxy:2222", // Should be skipped
			Descriptors: nil,
		},
	}

	r, err := NewWithPerProxyDescriptors(ctx, bindings, client)
	require.NoError(t, err)
	t.Cleanup(r.Close)

	require.Len(t, r.routes, 1, "only valid proxy URLs should create routes")
	require.Equal(t, 1, client.calls, "reflection should only be called once")
}

func TestNewWithPerProxyDescriptors_IsolatedDescriptors(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := &fakeRemoteClient{}

	// Proxy 1 has local descriptor for greeter
	descriptors1 := []*descriptorpb.FileDescriptorSet{
		buildDescriptorSet(map[string][]string{
			"greeter": {"SayHello"},
		}),
	}

	// Proxy 2 has local descriptor for orders
	descriptors2 := []*descriptorpb.FileDescriptorSet{
		buildDescriptorSet(map[string][]string{
			"orders": {"CreateOrder"},
		}),
	}

	bindings := []ProxyDescriptorBinding{
		{
			ProxyURL:    "grpc+proxy://upstream1:4111",
			Descriptors: descriptors1,
		},
		{
			ProxyURL:    "grpc+proxy://upstream2:4222",
			Descriptors: descriptors2,
		},
	}

	r, err := NewWithPerProxyDescriptors(ctx, bindings, client)
	require.NoError(t, err)
	t.Cleanup(r.Close)

	// No reflection calls should happen since all services have local descriptors
	require.Equal(t, 0, client.calls)

	// Each service should route to its correct proxy
	greeterRoute := r.RouteByMethod("/greeter/SayHello")
	require.NotNil(t, greeterRoute)
	require.Equal(t, "upstream1:4111", greeterRoute.Source.ReflectAddress)

	ordersRoute := r.RouteByMethod("/orders/CreateOrder")
	require.NotNil(t, ordersRoute)
	require.Equal(t, "upstream2:4222", ordersRoute.Source.ReflectAddress)

	// Services should not cross-contaminate
	require.Nil(t, r.RouteByMethod("/greeter/CreateOrder"))
	require.Nil(t, r.RouteByMethod("/orders/SayHello"))
}

func TestNewWithPerProxyDescriptors_AllThreeModes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := &fakeRemoteClient{sets: map[string]*descriptorpb.FileDescriptorSet{
		"upstream1:4111": buildDescriptorSet(map[string][]string{
			"greeter": {"SayHello"},
		}),
		"upstream2:4222": buildDescriptorSet(map[string][]string{
			"orders": {"CreateOrder"},
		}),
		"upstream3:4333": buildDescriptorSet(map[string][]string{
			"payments": {"ProcessPayment"},
		}),
	}}

	bindings := []ProxyDescriptorBinding{
		{ProxyURL: "grpc+proxy://upstream1:4111"},
		{ProxyURL: "grpc+capture://upstream2:4222"},
		{ProxyURL: "grpc+replay://upstream3:4333"},
	}

	r, err := NewWithPerProxyDescriptors(ctx, bindings, client)
	require.NoError(t, err)
	t.Cleanup(r.Close)

	greeterRoute := r.RouteByMethod("/greeter/SayHello")
	require.NotNil(t, greeterRoute)
	require.Equal(t, ModeProxy, greeterRoute.Mode)

	ordersRoute := r.RouteByMethod("/orders/CreateOrder")
	require.NotNil(t, ordersRoute)
	require.Equal(t, ModeCapture, ordersRoute.Mode)

	paymentsRoute := r.RouteByMethod("/payments/ProcessPayment")
	require.NotNil(t, paymentsRoute)
	require.Equal(t, ModeReplay, paymentsRoute.Mode)
}

func TestNewWithPerProxyDescriptors_EmptyDescriptorsUsesReflection(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := &fakeRemoteClient{sets: map[string]*descriptorpb.FileDescriptorSet{
		"upstream:4111": buildDescriptorSet(map[string][]string{
			"greeter": {"SayHello"},
		}),
	}}

	bindings := []ProxyDescriptorBinding{
		{
			ProxyURL:    "grpc+proxy://upstream:4111",
			Descriptors: []*descriptorpb.FileDescriptorSet{}, // Empty slice - should use reflection
		},
	}

	r, err := NewWithPerProxyDescriptors(ctx, bindings, client)
	require.NoError(t, err)
	t.Cleanup(r.Close)

	require.Equal(t, 1, client.calls, "should have used reflection")

	route := r.RouteByMethod("/greeter/SayHello")
	require.NotNil(t, route)
}

func TestNewWithPerProxyDescriptors_NilDescriptorsUsesReflection(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := &fakeRemoteClient{sets: map[string]*descriptorpb.FileDescriptorSet{
		"upstream:4111": buildDescriptorSet(map[string][]string{
			"greeter": {"SayHello"},
		}),
	}}

	bindings := []ProxyDescriptorBinding{
		{
			ProxyURL:    "grpc+proxy://upstream:4111",
			Descriptors: nil, // Nil - should use reflection
		},
	}

	r, err := NewWithPerProxyDescriptors(ctx, bindings, client)
	require.NoError(t, err)
	t.Cleanup(r.Close)

	require.Equal(t, 1, client.calls, "should have used reflection")

	route := r.RouteByMethod("/greeter/SayHello")
	require.NotNil(t, route)
}
