package sdk

import (
	"context"
	"errors"
)

// Run starts an embedded gRPC mock server or connects to a remote gripmock. Blocks until healthy.
func Run(ctx context.Context, opts ...Option) (Mock, error) {
	o := &options{healthyTimeout: defaultHealthyTimeout}
	for _, opt := range opts {
		opt(o)
	}
	if o.remoteAddr != "" {
		return runRemote(ctx, o)
	}
	if len(o.descriptorFiles) == 0 && o.mockFromAddr == "" {
		return nil, errors.New("gripmock: descriptors required (use WithDescriptors or MockFrom)")
	}
	if o.mockFromAddr != "" {
		fds, err := resolveDescriptorsFromReflection(ctx, o.mockFromAddr)
		if err != nil {
			return nil, err
		}
		o.descriptorFiles = fds.GetFile()
	}
	return runEmbedded(ctx, o)
}
