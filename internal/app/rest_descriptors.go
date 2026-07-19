package app

import (
	"net/http"
	"sort"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/domain/rest"
	"github.com/bavix/gripmock/v3/internal/infra/httputil"
	protosetinfra "github.com/bavix/gripmock/v3/internal/infra/protoset"
)

func (h *RestServer) ListDescriptors(w http.ResponseWriter, r *http.Request) {
	h.writeResponse(r.Context(), w, rest.DescriptorServiceIDs{ServiceIDs: h.restDescriptors.ServiceIDs()})
}

// AddDescriptors accepts binary FileDescriptorSet and registers it for discovery.
// Returns service IDs; use DELETE /services/{serviceID} to remove.
func (h *RestServer) AddDescriptors(w http.ResponseWriter, r *http.Request) {
	byt, err := httputil.RequestBody(r)
	if err != nil {
		h.responseError(r.Context(), w, err)

		return
	}

	if len(byt) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		h.writeResponseError(r.Context(), w, ErrEmptyBody)

		return
	}

	serviceIDs, err := registerDescriptorBytes(h, byt)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		h.writeResponseError(r.Context(), w, err)

		return
	}

	h.writeResponse(r.Context(), w, rest.AddDescriptorsResponse{
		Message:    "ok",
		Time:       time.Now(),
		ServiceIDs: serviceIDs,
	})
}

// DeleteService removes a service added via POST /descriptors.
// Services from startup (proto path) cannot be removed and return 404.
func (h *RestServer) DeleteService(w http.ResponseWriter, r *http.Request, serviceID string) {
	if unregisterService(h, serviceID) == 0 {
		w.WriteHeader(http.StatusNotFound)
		h.writeResponseError(r.Context(), w, serviceNotRemovable(serviceID))

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func unregisterService(h *RestServer, serviceID string) int {
	h.descriptorOpsMu.Lock()
	defer h.descriptorOpsMu.Unlock()

	return h.restDescriptors.UnregisterByService(serviceID)
}

func registerDescriptorBytes(h *RestServer, byt []byte) ([]string, error) {
	h.descriptorOpsMu.Lock()
	defer h.descriptorOpsMu.Unlock()

	var fds descriptorpb.FileDescriptorSet
	if err := proto.Unmarshal(byt, &fds); err != nil {
		return nil, invalidFileDescriptorSetError(err)
	}

	if len(fds.GetFile()) == 0 {
		return nil, ErrFileDescriptorSetNoFiles
	}

	files, err := decodeDescriptorFiles(&fds)
	if err != nil {
		return nil, err
	}

	serviceIDs := make([]string, 0)

	for _, fd := range files {
		h.restDescriptors.Register(fd)

		services := fd.Services()
		for i := range services.Len() {
			serviceIDs = append(serviceIDs, string(services.Get(i).FullName()))
		}
	}

	sort.Strings(serviceIDs)

	return serviceIDs, nil
}

func decodeDescriptorFiles(fds *descriptorpb.FileDescriptorSet) ([]protoreflect.FileDescriptor, error) {
	registry := new(protoregistry.Files)
	pending := make([]*descriptorpb.FileDescriptorProto, 0, len(fds.GetFile()))

	for _, fd := range fds.GetFile() {
		if fd != nil {
			pending = append(pending, fd)
		}
	}

	for len(pending) > 0 {
		progress := false
		nextPending := make([]*descriptorpb.FileDescriptorProto, 0, len(pending))

		resolver := &protosetinfra.Fallback{Primary: registry, Fallback: protoregistry.GlobalFiles}

		for _, fd := range pending {
			fileDesc, err := protodesc.NewFile(fd, resolver)
			if err != nil {
				nextPending = append(nextPending, fd)

				continue
			}

			if err := registry.RegisterFile(fileDesc); err != nil {
				return nil, registerDescriptorFileError(fd.GetName(), err)
			}

			progress = true
		}

		if !progress {
			return nil, ErrResolveDescriptorDeps
		}

		pending = nextPending
	}

	files := make([]protoreflect.FileDescriptor, 0, len(fds.GetFile()))

	registry.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		files = append(files, fd)

		return true
	})

	return files, nil
}

// DeleteStubByID removes a stub by ID.
