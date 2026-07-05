package sdk

import "context"

// Run starts an embedded gRPC mock server or connects to a remote gripmock.
// Returns a Mock for v1 compatibility.
//
// Deprecated: use NewServer and the v2 API instead.
func Run(t TestingT, opts ...Option) (Mock, error) { //nolint:ireturn
	srv, err := initServer(t, opts...)
	if err != nil {
		return nil, err
	}

	return &mockServer{Server: srv}, nil
}

func startServer(ctx context.Context, o *options) (Mock, error) { //nolint:ireturn
	if o.remoteAddr != "" {
		return runRemote(ctx, o)
	}

	if len(o.descriptorFiles) == 0 && o.mockFromAddr == "" {
		return nil, ErrDescriptorsRequired
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
