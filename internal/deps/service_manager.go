package deps

import (
	"github.com/bavix/gripmock/v3/internal/infra/grpcservice"
)

// ServiceManager returns a singleton instance of the gRPC service manager.
func (b *Builder) ServiceManager() *grpcservice.Manager {
	b.serviceManagerOnce.Do(func() {
		b.serviceManager = grpcservice.NewManager()
	})

	return b.serviceManager
}
