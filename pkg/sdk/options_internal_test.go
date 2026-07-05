package sdk

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
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

func TestWithRemoteKeepsEmptyRestURLWhenNotProvided(t *testing.T) {
	t.Parallel()

	// Arrange
	o := &options{}

	// Act
	WithRemote("localhost:4770", "")(o)

	// Assert
	require.Equal(t, "localhost:4770", o.remoteAddr)
	require.Equal(t, "", o.remoteRestURL) //nolint:testifylint
}

func TestRemoteDeprecatedAlias(t *testing.T) {
	t.Parallel()

	// Arrange
	o := &options{}

	// Act
	WithRemote("127.0.0.1:7770", "http://127.0.0.1:4771")(o)

	// Assert
	require.Equal(t, "127.0.0.1:7770", o.remoteAddr)
	require.Equal(t, "http://127.0.0.1:4771", o.remoteRestURL)
}

func TestWithDescriptorsSkipsNilFiles(t *testing.T) {
	t.Parallel()

	// Arrange
	o := &options{}
	name := "svc.proto"
	fds := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{
		nil,
		{Name: proto.String(name)}, //nolint:modernize
		nil,
		{Name: proto.String(name)}, //nolint:modernize
	}}

	// Act
	WithDescriptors(fds)(o)

	// Assert
	require.Len(t, o.descriptorFiles, 1)
	require.Equal(t, name, o.descriptorFiles[0].GetName())
}

func TestDeriveRestUrlFromGrpcAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		addr string
		want string
	}{
		{"127.0.0.1:4770", "http://127.0.0.1:4771"},
		{"localhost:4770", "http://localhost:4771"},
		{"[::1]:4770", "http://[::1]:4771"},
		{"[2001:db8::1]:4770", "http://[2001:db8::1]:4771"},
	}
	for _, tt := range tests {
		got := deriveRestURLFromGrpcAddr(tt.addr)
		require.Equal(t, tt.want, got, "addr=%q", tt.addr)
	}
}

func TestWithFileDescriptor(t *testing.T) {
	t.Parallel()

	o := &options{}
	WithFileDescriptor(wrapperspb.File_google_protobuf_wrappers_proto)(o)

	require.Len(t, o.descriptorFiles, 1)
	require.Equal(t, "google/protobuf/wrappers.proto", o.descriptorFiles[0].GetName())
}
