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
	if o.descriptors == nil && o.mockFromAddr == "" {
		return nil, errors.New("gripmock: descriptors required (use WithDescriptors or MockFrom)")
	}
	if o.mockFromAddr != "" {
		fds, err := resolveDescriptorsFromReflection(ctx, o.mockFromAddr)
		if err != nil {
			return nil, err
		}
		o.descriptors = fds
	}
	return runEmbedded(ctx, o)
}
