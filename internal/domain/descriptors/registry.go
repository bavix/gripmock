package descriptors

import (
	"sort"
	"sync"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// Registry holds descriptors added via REST API. Supports add and remove.
// GlobalFiles (startup descriptors) are separate; list operations merge both.
type Registry struct {
	mu    sync.RWMutex
	files map[string]protoreflect.FileDescriptor // path -> file
	gen   uint64                                 // bumped on every mutation; lets consumers cache derived views
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{files: make(map[string]protoreflect.FileDescriptor)}
}

// Register adds a file descriptor. Replaces if path exists.
func (r *Registry) Register(fd protoreflect.FileDescriptor) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.files[fd.Path()] = fd
	r.gen++
}

// FileByPath returns the file registered under path, if any.
//
//nolint:ireturn // protoreflect models descriptors as interfaces.
func (r *Registry) FileByPath(path string) (protoreflect.FileDescriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	file, ok := r.files[path]

	return file, ok
}

// UnregisterByService removes file(s) that contain the given service.
func (r *Registry) UnregisterByService(serviceID string) int {
	r.mu.Lock()
	defer r.mu.Unlock()

	var removed int

	for path, fd := range r.files {
		services := fd.Services()

		for i := range services.Len() {
			if string(services.Get(i).FullName()) == serviceID {
				delete(r.files, path)
				r.gen++

				removed++

				break
			}
		}
	}

	return removed
}

// Generation returns a counter that changes on every mutation.
func (r *Registry) Generation() uint64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.gen
}

// RangeFiles calls f for each registered file.
func (r *Registry) RangeFiles(f func(protoreflect.FileDescriptor) bool) {
	r.mu.RLock()

	files := make([]protoreflect.FileDescriptor, 0, len(r.files))

	for _, fd := range r.files {
		files = append(files, fd)
	}

	r.mu.RUnlock()

	for _, fd := range files {
		if !f(fd) {
			return
		}
	}
}

// Paths returns all registered file paths.
func (r *Registry) Paths() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]string, 0, len(r.files))
	for p := range r.files {
		out = append(out, p)
	}

	sort.Strings(out)

	return out
}

// ServiceIDs returns all service IDs (e.g. helloworld.Greeter) from registered files.
func (r *Registry) ServiceIDs() []string {
	r.mu.RLock()

	ids := make([]string, 0)

	for _, fd := range r.files {
		services := fd.Services()

		for i := range services.Len() {
			ids = append(ids, string(services.Get(i).FullName()))
		}
	}

	r.mu.RUnlock()

	sort.Strings(ids)

	return ids
}
