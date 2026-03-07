package sdk

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWithRemoteAssignsRemoteConfig(t *testing.T) {
	t.Parallel()

	// Arrange
	o := &options{}

	// Act
	WithRemote("localhost:4770", "http://localhost:4771")(o)

	// Assert
	require.Equal(t, "localhost:4770", o.remoteAddr)
	require.Equal(t, "http://localhost:4771", o.remoteRestURL)
}

func TestWithRemoteNormalizesRemoteConfig(t *testing.T) {
	t.Parallel()

	// Arrange
	o := &options{}

	// Act
	WithRemote(" localhost:4770/ ", " localhost:4771/ ")(o)

	// Assert
	require.Equal(t, "localhost:4770", o.remoteAddr)
	require.Equal(t, "http://localhost:4771", o.remoteRestURL)
}

func TestWithHTTPClientAssignsClient(t *testing.T) {
	t.Parallel()

	// Arrange
	client := &http.Client{}
	o := &options{}

	// Act
	WithHTTPClient(client)(o)

	// Assert
	require.Same(t, client, o.httpClient)
}

func TestWithSessionTTLAssignsTTL(t *testing.T) {
	t.Parallel()

	// Arrange
	o := &options{}

	// Act
	WithSessionTTL(2 * time.Minute)(o)

	// Assert
	require.Equal(t, 2*time.Minute, o.sessionTTL)
}

func TestDefaultSessionTTL(t *testing.T) {
	t.Parallel()

	require.Equal(t, 60*time.Second, defaultSessionTTL)
}

func TestWithGRPCTimeoutAssignsTimeout(t *testing.T) {
	t.Parallel()

	// Arrange
	o := &options{}

	// Act
	WithGRPCTimeout(3 * time.Second)(o)

	// Assert
	require.Equal(t, 3*time.Second, o.grpcTimeout)
}

func TestWithSessionTrimsSessionID(t *testing.T) {
	t.Parallel()

	// Arrange
	o := &options{}

	// Act
	WithSession("  my-session  ")(o)

	// Assert
	require.Equal(t, "my-session", o.session)
}
