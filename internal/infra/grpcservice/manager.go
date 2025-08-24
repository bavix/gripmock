package grpcservice

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/domain/protoset"
	"github.com/bavix/gripmock/v3/internal/infra/methodregistry"
)

// ServiceInfo represents information about a gRPC service.
type ServiceInfo struct {
	Name    string
	Package string
	Methods []MethodInfo
}

// MethodInfo represents information about a gRPC method.
type MethodInfo struct {
	Name           string
	InputType      string
	OutputType     string
	IsClientStream bool
	IsServerStream bool
}

// Manager manages gRPC services and their registration.
type Manager struct {
	services       map[string]*ServiceInfo // key: "package.Service"
	methodRegistry *methodregistry.Registry
	mu             sync.RWMutex
}

// NewManager creates a new gRPC service manager.
func NewManager() *Manager {
	return &Manager{
		services:       make(map[string]*ServiceInfo),
		methodRegistry: methodregistry.New(),
	}
}

// RegisterService registers a gRPC service with all its methods.
func (m *Manager) RegisterService(packageName, serviceName string, methods []MethodInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	serviceKey := fmt.Sprintf("%s.%s", packageName, serviceName)

	service := &ServiceInfo{
		Name:    serviceName,
		Package: packageName,
		Methods: make([]MethodInfo, len(methods)),
	}

	copy(service.Methods, methods)
	m.services[serviceKey] = service

	// Register all methods in the method registry
	for _, method := range methods {
		m.methodRegistry.RegisterMethod(
			serviceKey,
			method.Name,
			method.IsClientStream,
			method.IsServerStream,
		)
	}
}

// RegisterFromDescriptor registers services from protobuf descriptors.
func (m *Manager) RegisterFromDescriptor(descriptors []*descriptorpb.FileDescriptorSet) {
	for _, descriptor := range descriptors {
		for _, file := range descriptor.GetFile() {
			packageName := file.GetPackage()

			for _, svc := range file.GetService() {
				serviceName := svc.GetName()
				methods := make([]MethodInfo, 0, len(svc.GetMethod()))

				for _, method := range svc.GetMethod() {
					methods = append(methods, MethodInfo{
						Name:           method.GetName(),
						InputType:      method.GetInputType(),
						OutputType:     method.GetOutputType(),
						IsClientStream: method.GetClientStreaming(),
						IsServerStream: method.GetServerStreaming(),
					})
				}

				m.RegisterService(packageName, serviceName, methods)
			}
		}
	}
}

// RegisterFromProtoFile registers services from proto file path.
func (m *Manager) RegisterFromProtoFile(ctx context.Context, imports []string, protoPath string) error {
	// Use the existing protoset.Build function to compile proto files to descriptors
	descriptors, err := protoset.Build(ctx, imports, []string{protoPath})
	if err != nil {
		return fmt.Errorf("failed to build proto descriptors: %w", err)
	}

	// Register services from the compiled descriptors
	m.RegisterFromDescriptor(descriptors)

	return nil
}

// RegisterFromProtoFiles registers services from multiple proto file paths.
func (m *Manager) RegisterFromProtoFiles(ctx context.Context, imports []string, protoPaths []string) error {
	// Use the existing protoset.Build function to compile proto files to descriptors
	descriptors, err := protoset.Build(ctx, imports, protoPaths)
	if err != nil {
		return fmt.Errorf("failed to build proto descriptors: %w", err)
	}

	// Register services from the compiled descriptors
	m.RegisterFromDescriptor(descriptors)

	return nil
}

// GetService returns service information by package and service name.
func (m *Manager) GetService(packageName, serviceName string) (*ServiceInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	serviceKey := fmt.Sprintf("%s.%s", packageName, serviceName)
	service, exists := m.services[serviceKey]

	return service, exists
}

// GetAllServices returns all registered services.
func (m *Manager) GetAllServices() []*ServiceInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	services := make([]*ServiceInfo, 0, len(m.services))
	for _, service := range m.services {
		services = append(services, service)
	}

	return services
}

// GetServicesByPackage returns all services for a specific package.
func (m *Manager) GetServicesByPackage(packageName string) []*ServiceInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var services []*ServiceInfo
	for _, service := range m.services {
		if service.Package == packageName {
			services = append(services, service)
		}
	}

	return services
}

// GetMethodRegistry returns the method registry.
func (m *Manager) GetMethodRegistry() *methodregistry.Registry {
	return m.methodRegistry
}

// RegisterGRPCServices registers all services with a gRPC server.
func (m *Manager) RegisterGRPCServices(ctx context.Context, server *grpc.Server, descriptors []*descriptorpb.FileDescriptorSet) error {
	// First register services in our manager
	m.RegisterFromDescriptor(descriptors)

	// Then register with gRPC server
	// This ensures we have all service information before gRPC registration
	return m.registerWithGRPCServer(ctx, server, descriptors)
}

// IsServiceRegistered checks if a service is registered.
func (m *Manager) IsServiceRegistered(packageName, serviceName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	serviceKey := fmt.Sprintf("%s.%s", packageName, serviceName)
	_, exists := m.services[serviceKey]

	return exists
}

// IsMethodRegistered checks if a method is registered.
func (m *Manager) IsMethodRegistered(serviceName, methodName string) bool {
	_, exists := m.methodRegistry.GetMethodInfo(serviceName, methodName)

	return exists
}

// Count returns the number of registered services.
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.services)
}

// Clear removes all registered services.
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.services = make(map[string]*ServiceInfo)
	m.methodRegistry.Clear()
}

// registerWithGRPCServer registers services with the gRPC server.
// This is a placeholder - actual implementation would be similar to the current GRPCServer.registerServices.
func (m *Manager) registerWithGRPCServer(ctx context.Context, server *grpc.Server, descriptors []*descriptorpb.FileDescriptorSet) error {
	// This would contain the logic currently in GRPCServer.registerServices
	// but with access to the manager's service information
	return nil
}
