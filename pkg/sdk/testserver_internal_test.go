package sdk

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const TestHealthTimeout = 15 * time.Second

func NewTestServer(t TestingT, opts ...Option) *Server {
	return NewServer(t, append([]Option{WithHealthCheckTimeout(TestHealthTimeout)}, opts...)...)
}

func TestWaitForHealthyReportsWhyItGaveUp(t *testing.T) {
	t.Parallel()

	conn, err := grpc.NewClient("passthrough:///127.0.0.1:1",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	defer func() { _ = conn.Close() }()

	err = waitForHealthy(t.Context(), conn, 200*time.Millisecond)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrServerNotHealthy)
	require.Contains(t, err.Error(), "127.0.0.1:1")
	require.Contains(t, err.Error(), "last check:")
	require.Contains(t, err.Error(), "waited")
}
