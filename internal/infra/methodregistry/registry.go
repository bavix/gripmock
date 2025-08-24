package methodregistry

import (
	"sync"

	"github.com/bavix/gripmock/v3/internal/domain/types"
)

// Registry stores information about all registered gRPC methods.
type Registry struct {
	methods map[string]types.MethodInfo // key: "service/method"
	mu      sync.RWMutex
}

// New creates a new method registry.
func New() *Registry {
	return &Registry{
		methods: make(map[string]types.MethodInfo),
	}
}

// RegisterMethod adds a method to the registry.
func (r *Registry) RegisterMethod(service, method string, isClientStream, isServerStream bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := service + "/" + method
	r.methods[key] = types.MethodInfo{
		Service:        service,
		Method:         method,
		IsUnary:        !isClientStream && !isServerStream,
		IsClientStream: isClientStream,
		IsServerStream: isServerStream,
		IsBidiStream:   isClientStream && isServerStream,
	}
}

// GetMethodInfo retrieves method information by service and method name.
func (r *Registry) GetMethodInfo(service, method string) (types.MethodInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := service + "/" + method
	info, exists := r.methods[key]

	return info, exists
}

// IsUnary returns true if the method is unary.
func (r *Registry) IsUnary(service, method string) bool {
	info, exists := r.GetMethodInfo(service, method)

	return exists && info.IsUnary
}

// IsClientStream returns true if the method is client streaming.
func (r *Registry) IsClientStream(service, method string) bool {
	info, exists := r.GetMethodInfo(service, method)

	return exists && info.IsClientStream
}

// IsServerStream returns true if the method is server streaming.
func (r *Registry) IsServerStream(service, method string) bool {
	info, exists := r.GetMethodInfo(service, method)

	return exists && info.IsServerStream
}

// IsBidiStream returns true if the method is bidirectional streaming.
func (r *Registry) IsBidiStream(service, method string) bool {
	info, exists := r.GetMethodInfo(service, method)

	return exists && info.IsBidiStream
}

// GetAllMethods returns all registered methods.
func (r *Registry) GetAllMethods() []types.MethodInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	methods := make([]types.MethodInfo, 0, len(r.methods))
	for _, method := range r.methods {
		methods = append(methods, method)
	}

	return methods
}

// GetMethodsByService returns all methods for a specific service.
func (r *Registry) GetMethodsByService(service string) []types.MethodInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var methods []types.MethodInfo
	for _, method := range r.methods {
		if method.Service == service {
			methods = append(methods, method)
		}
	}

	return methods
}

// Clear removes all registered methods.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.methods = make(map[string]types.MethodInfo)
}

// Count returns the number of registered methods.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.methods)
}
