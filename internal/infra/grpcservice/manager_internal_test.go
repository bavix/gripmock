package grpcservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestManager_RegisterService(t *testing.T) {
	t.Parallel()

	manager := NewManager()

	// Register a test service
	methods := []MethodInfo{
		{
			Name:           "UnaryMethod",
			InputType:      "test.UnaryRequest",
			OutputType:     "test.UnaryResponse",
			IsClientStream: false,
			IsServerStream: false,
		},
		{
			Name:           "StreamMethod",
			InputType:      "test.StreamRequest",
			OutputType:     "test.StreamResponse",
			IsClientStream: true,
			IsServerStream: true,
		},
	}

	manager.RegisterService("test", "TestService", methods)

	// Check if service is registered
	service, exists := manager.GetService("test", "TestService")
	require.True(t, exists)
	assert.Equal(t, "TestService", service.Name)
	assert.Equal(t, "test", service.Package)
	assert.Len(t, service.Methods, 2)

	// Check if methods are registered in method registry
	methodRegistry := manager.GetMethodRegistry()
	assert.True(t, methodRegistry.IsUnary("test.TestService", "UnaryMethod"))
	assert.True(t, methodRegistry.IsBidiStream("test.TestService", "StreamMethod"))
}

func TestManager_GetAllServices(t *testing.T) {
	t.Parallel()

	manager := NewManager()

	// Register multiple services
	manager.RegisterService("test1", "Service1", []MethodInfo{{Name: "Method1"}})
	manager.RegisterService("test2", "Service2", []MethodInfo{{Name: "Method2"}})

	services := manager.GetAllServices()
	assert.Len(t, services, 2)

	// Check service names
	serviceNames := make(map[string]bool)
	for _, service := range services {
		serviceNames[service.Name] = true
	}

	assert.True(t, serviceNames["Service1"])
	assert.True(t, serviceNames["Service2"])
}

func TestManager_GetServicesByPackage(t *testing.T) {
	t.Parallel()

	manager := NewManager()

	// Register services in different packages
	manager.RegisterService("test1", "Service1", []MethodInfo{{Name: "Method1"}})
	manager.RegisterService("test1", "Service2", []MethodInfo{{Name: "Method2"}})
	manager.RegisterService("test2", "Service3", []MethodInfo{{Name: "Method3"}})

	// Get services from test1 package
	test1Services := manager.GetServicesByPackage("test1")
	assert.Len(t, test1Services, 2)

	// Get services from test2 package
	test2Services := manager.GetServicesByPackage("test2")
	assert.Len(t, test2Services, 1)

	// Get services from non-existent package
	nonExistentServices := manager.GetServicesByPackage("test3")
	assert.Empty(t, nonExistentServices)
}

func TestManager_IsServiceRegistered(t *testing.T) {
	t.Parallel()

	manager := NewManager()

	// Register a service
	manager.RegisterService("test", "TestService", []MethodInfo{{Name: "Method1"}})

	// Check if service is registered
	assert.True(t, manager.IsServiceRegistered("test", "TestService"))
	assert.False(t, manager.IsServiceRegistered("test", "NonExistentService"))
	assert.False(t, manager.IsServiceRegistered("nonExistent", "TestService"))
}

func TestManager_IsMethodRegistered(t *testing.T) {
	t.Parallel()

	manager := NewManager()

	// Register a service with methods
	manager.RegisterService("test", "TestService", []MethodInfo{
		{Name: "Method1", IsClientStream: false, IsServerStream: false},
		{Name: "Method2", IsClientStream: true, IsServerStream: true},
	})

	// Check if methods are registered
	assert.True(t, manager.IsMethodRegistered("test.TestService", "Method1"))
	assert.True(t, manager.IsMethodRegistered("test.TestService", "Method2"))
	assert.False(t, manager.IsMethodRegistered("test.TestService", "NonExistentMethod"))
}

func TestManager_Count(t *testing.T) {
	t.Parallel()

	manager := NewManager()

	assert.Equal(t, 0, manager.Count())

	// Register services
	manager.RegisterService("test1", "Service1", []MethodInfo{{Name: "Method1"}})
	assert.Equal(t, 1, manager.Count())

	manager.RegisterService("test2", "Service2", []MethodInfo{{Name: "Method2"}})
	assert.Equal(t, 2, manager.Count())
}

func TestManager_Clear(t *testing.T) {
	t.Parallel()

	manager := NewManager()

	// Register some services
	manager.RegisterService("test", "Service1", []MethodInfo{{Name: "Method1"}})
	manager.RegisterService("test", "Service2", []MethodInfo{{Name: "Method2"}})

	assert.Equal(t, 2, manager.Count())

	// Clear the manager
	manager.Clear()

	assert.Equal(t, 0, manager.Count())
	assert.False(t, manager.IsServiceRegistered("test", "Service1"))
	assert.False(t, manager.IsServiceRegistered("test", "Service2"))
}

func TestManager_RegisterFromDescriptor(t *testing.T) {
	t.Parallel()

	manager := NewManager()

	// Create a mock descriptor
	descriptor := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			{
				Package: stringPtr("test"),
				Service: []*descriptorpb.ServiceDescriptorProto{
					{
						Name: stringPtr("TestService"),
						Method: []*descriptorpb.MethodDescriptorProto{
							{
								Name:            stringPtr("UnaryMethod"),
								InputType:       stringPtr(".test.UnaryRequest"),
								OutputType:      stringPtr(".test.UnaryResponse"),
								ClientStreaming: boolPtr(false),
								ServerStreaming: boolPtr(false),
							},
							{
								Name:            stringPtr("StreamMethod"),
								InputType:       stringPtr(".test.StreamRequest"),
								OutputType:      stringPtr(".test.StreamResponse"),
								ClientStreaming: boolPtr(true),
								ServerStreaming: boolPtr(true),
							},
						},
					},
				},
			},
		},
	}

	manager.RegisterFromDescriptor([]*descriptorpb.FileDescriptorSet{descriptor})

	// Check if service is registered
	assert.True(t, manager.IsServiceRegistered("test", "TestService"))
	assert.True(t, manager.IsMethodRegistered("test.TestService", "UnaryMethod"))
	assert.True(t, manager.IsMethodRegistered("test.TestService", "StreamMethod"))

	// Check method registry
	methodRegistry := manager.GetMethodRegistry()
	assert.True(t, methodRegistry.IsUnary("test.TestService", "UnaryMethod"))
	assert.True(t, methodRegistry.IsBidiStream("test.TestService", "StreamMethod"))
}

// Helper functions for creating protobuf values.
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
