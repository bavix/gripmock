package sdk

import (
	"net/http"
	"time"
)

// Run starts an embedded gRPC mock server or connects to a remote gripmock. Blocks until healthy.
// Registers cleanup to verify stub Times and Close.
// t must not be nil.
func Run(t TestingT, opts ...Option) (Mock, error) {
	if t == nil {
		panic("gripmock: t must not be nil")
	}

	o := &options{healthyTimeout: defaultHealthyTimeout, sessionTTL: defaultSessionTTL}
	for _, opt := range opts {
		opt(o)
	}

	if o.httpClient == nil {
		o.httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	ctx := t.Context()

	var mock Mock
	var err error
	if o.remoteAddr != "" {
		mock, err = runRemote(ctx, o)
	} else {
		if len(o.descriptorFiles) == 0 && o.mockFromAddr == "" {
			return nil, ErrDescriptorsRequired
		}
		if o.mockFromAddr != "" {
			fds, errResolve := resolveDescriptorsFromReflection(ctx, o.mockFromAddr)
			if errResolve != nil {
				return nil, errResolve
			}
			o.descriptorFiles = fds.GetFile()
		}
		mock, err = runEmbedded(ctx, o)
	}
	if err != nil {
		return nil, err
	}

	t.Cleanup(func() {
		if err := mock.Verify().VerifyStubTimesErr(); err != nil {
			t.Error(err)
			t.Fail()
		}
		_ = mock.Close()
	})

	return mock, nil
}
