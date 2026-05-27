package proto_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/domain/proto"
)

// TestIntegration_PerProxyBindingFlow tests the complete flow of per-proxy binding
// from argument parsing to final Arguments structure.
func TestIntegration_PerProxyBindingFlow(t *testing.T) {
	t.Parallel()

	t.Run("first proxy with sources, second uses reflection", testFirstProxyWithSources)
	t.Run("different sources per proxy", testDifferentSourcesPerProxy)
	t.Run("mixed proto paths and proxy URLs", testMixedProtoAndProxy)
	t.Run("three proxies with different configurations", testThreeProxies)
	t.Run("source flag formats", testSourceFlagFormats)
	t.Run("proxy with query parameters", testProxyWithQueryParams)
	t.Run("no proxies uses all sources", testNoProxiesUsesAllSources)
	t.Run("empty sources list", testEmptySourcesList)
}

func testFirstProxyWithSources(t *testing.T) {
	t.Parallel()

	args := []string{
		"-S", "service_a.proto",
		"-S", "service_b.proto",
		"grpc+proxy://upstream1:4111",
		"grpc+proxy://upstream2:4222",
	}

	result := proto.ParseArgumentsWithBindings(args, []string{"./imports"}, nil)

	require.True(t, result.HasProxyBindings())
	require.Equal(t, []string{"./imports"}, result.Imports())

	bindings := result.ProxyBindings()
	require.Len(t, bindings, 2)

	require.Equal(t, "grpc+proxy://upstream1:4111", bindings[0].ProxyURL)
	require.Equal(t, []string{"service_a.proto", "service_b.proto"}, bindings[0].Sources)

	require.Equal(t, "grpc+proxy://upstream2:4222", bindings[1].ProxyURL)
	require.Empty(t, bindings[1].Sources)
}

func testDifferentSourcesPerProxy(t *testing.T) {
	t.Parallel()

	args := []string{
		"-S", "service_a.proto",
		"grpc+proxy://upstream1:4111",
		"-S", "service_b.proto",
		"grpcs+capture://upstream2:4222",
	}

	result := proto.ParseArgumentsWithBindings(args, nil, nil)

	require.True(t, result.HasProxyBindings())

	bindings := result.ProxyBindings()
	require.Len(t, bindings, 2)

	require.Equal(t, "grpc+proxy://upstream1:4111", bindings[0].ProxyURL)
	require.Equal(t, []string{"service_a.proto"}, bindings[0].Sources)

	require.Equal(t, "grpcs+capture://upstream2:4222", bindings[1].ProxyURL)
	require.Equal(t, []string{"service_b.proto"}, bindings[1].Sources)
}

func testMixedProtoAndProxy(t *testing.T) {
	t.Parallel()

	args := []string{
		"common.proto",
		"-S", "service_a.proto",
		"grpc+proxy://upstream1:4111",
		"types.proto",
		"-S", "service_b.proto",
		"grpc+replay://upstream2:4222",
	}

	result := proto.ParseArgumentsWithBindings(args, nil, nil)

	require.Equal(t, []string{"common.proto", "types.proto"}, result.ProtoPath())
	require.True(t, result.HasProxyBindings())

	bindings := result.ProxyBindings()
	require.Len(t, bindings, 2)

	require.Equal(t, "grpc+proxy://upstream1:4111", bindings[0].ProxyURL)
	require.Equal(t, []string{"service_a.proto"}, bindings[0].Sources)

	require.Equal(t, "grpc+replay://upstream2:4222", bindings[1].ProxyURL)
	require.Equal(t, []string{"service_b.proto"}, bindings[1].Sources)
}

func testThreeProxies(t *testing.T) {
	t.Parallel()

	args := []string{
		"-S", "service_a.proto",
		"grpc+proxy://upstream1:4111",
		"grpc+capture://upstream2:4222",
		"-S", "service_c.proto",
		"-S", "service_d.proto",
		"grpc+replay://upstream3:4333",
	}

	result := proto.ParseArgumentsWithBindings(args, nil, nil)

	require.True(t, result.HasProxyBindings())

	bindings := result.ProxyBindings()
	require.Len(t, bindings, 3)

	require.Equal(t, "grpc+proxy://upstream1:4111", bindings[0].ProxyURL)
	require.Equal(t, []string{"service_a.proto"}, bindings[0].Sources)

	require.Equal(t, "grpc+capture://upstream2:4222", bindings[1].ProxyURL)
	require.Empty(t, bindings[1].Sources)

	require.Equal(t, "grpc+replay://upstream3:4333", bindings[2].ProxyURL)
	require.Equal(t, []string{"service_c.proto", "service_d.proto"}, bindings[2].Sources)
}

func testSourceFlagFormats(t *testing.T) {
	t.Parallel()

	args := []string{
		"-S", "a.proto",
		"-S=b.proto",
		"--source", "c.proto",
		"--source=d.proto",
		"grpc+proxy://upstream:4111",
	}

	result := proto.ParseArgumentsWithBindings(args, nil, nil)

	require.True(t, result.HasProxyBindings())

	bindings := result.ProxyBindings()
	require.Len(t, bindings, 1)
	require.Equal(t, []string{"a.proto", "b.proto", "c.proto", "d.proto"}, bindings[0].Sources)
}

func testProxyWithQueryParams(t *testing.T) {
	t.Parallel()

	args := []string{
		"-S", "service.proto",
		"grpc+proxy://upstream:4111?timeout=30s&insecureSkipVerify=true",
	}

	result := proto.ParseArgumentsWithBindings(args, nil, nil)

	require.True(t, result.HasProxyBindings())

	bindings := result.ProxyBindings()
	require.Len(t, bindings, 1)
	require.Equal(t, "grpc+proxy://upstream:4111?timeout=30s&insecureSkipVerify=true", bindings[0].ProxyURL)
	require.Equal(t, []string{"service.proto"}, bindings[0].Sources)
}

func testNoProxiesUsesAllSources(t *testing.T) {
	t.Parallel()

	args := []string{
		"-S", "service.proto",
		"common.proto",
		"grpc://upstream:4111", // No +mode suffix
	}

	result := proto.ParseArgumentsWithBindings(args, nil, nil)

	require.False(t, result.HasProxyBindings())
	require.Nil(t, result.ProxyBindings())
	require.Equal(t, []string{"common.proto", "grpc://upstream:4111"}, result.ProtoPath())
	require.Equal(t, []string{"service.proto"}, result.Sources())
}

func testEmptySourcesList(t *testing.T) {
	t.Parallel()

	args := []string{
		"grpc+proxy://upstream1:4111",
		"grpc+proxy://upstream2:4222",
		"grpc+proxy://upstream3:4333",
	}

	result := proto.ParseArgumentsWithBindings(args, nil, nil)

	require.True(t, result.HasProxyBindings())

	bindings := result.ProxyBindings()
	require.Len(t, bindings, 3)

	for i, binding := range bindings {
		require.Empty(t, binding.Sources, "proxy %d should have empty sources", i)
	}
}

// TestIntegration_RealWorldScenarios tests common real-world usage patterns.
func TestIntegration_RealWorldScenarios(t *testing.T) {
	t.Parallel()

	t.Run("microservices: local proto + two upstream proxies", testMicroservicesScenario)
	t.Run("testing: capture mode for recording", testCaptureScenario)
	t.Run("mixed environment: local + proxy + replay", testMixedEnvironment)
}

func testMicroservicesScenario(t *testing.T) {
	t.Parallel()

	args := []string{
		"examples/local_service.proto",
		"-S", "examples/auth_service.proto",
		"grpc+proxy://auth-service:50051",
		"-S", "examples/payment_service.proto",
		"grpc+proxy://payment-service:50052",
	}

	result := proto.ParseArgumentsWithBindings(args, []string{"./examples"}, nil)

	require.Equal(t, []string{"examples/local_service.proto"}, result.ProtoPath())
	require.True(t, result.HasProxyBindings())

	bindings := result.ProxyBindings()
	require.Len(t, bindings, 2)

	require.Equal(t, "grpc+proxy://auth-service:50051", bindings[0].ProxyURL)
	require.Equal(t, []string{"examples/auth_service.proto"}, bindings[0].Sources)

	require.Equal(t, "grpc+proxy://payment-service:50052", bindings[1].ProxyURL)
	require.Equal(t, []string{"examples/payment_service.proto"}, bindings[1].Sources)
}

func testCaptureScenario(t *testing.T) {
	t.Parallel()

	args := []string{
		"-S", "examples/user_service.proto",
		"grpc+capture://staging-server:443?recordDelay=true",
	}

	result := proto.ParseArgumentsWithBindings(args, nil, nil)

	require.True(t, result.HasProxyBindings())

	bindings := result.ProxyBindings()
	require.Len(t, bindings, 1)
	require.Equal(t, "grpc+capture://staging-server:443?recordDelay=true", bindings[0].ProxyURL)
	require.Equal(t, []string{"examples/user_service.proto"}, bindings[0].Sources)
}

func testMixedEnvironment(t *testing.T) {
	t.Parallel()

	args := []string{
		"local/greeter.proto",
		"-S", "upstream/auth.proto",
		"grpcs+proxy://prod-auth:443?serverName=auth.example.com",
		"-S", "upstream/payment.proto",
		"grpc+replay://localhost:50052",
	}

	result := proto.ParseArgumentsWithBindings(args, nil, nil)

	require.Equal(t, []string{"local/greeter.proto"}, result.ProtoPath())

	bindings := result.ProxyBindings()
	require.Len(t, bindings, 2)

	require.Equal(t, "grpcs+proxy://prod-auth:443?serverName=auth.example.com", bindings[0].ProxyURL)
	require.Equal(t, "grpc+replay://localhost:50052", bindings[1].ProxyURL)
}
