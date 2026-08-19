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

	o := &options{}

	WithRemote("localhost:4770", "http://localhost:4771")(o)

	require.Equal(t, "localhost:4770", o.remoteAddr)
	require.Equal(t, "http://localhost:4771", o.remoteRestURL)
}

func TestWithRemoteNormalizesRemoteConfig(t *testing.T) {
	t.Parallel()

	o := &options{}

	WithRemote(" localhost:4770/ ", " localhost:4771/ ")(o)

	require.Equal(t, "localhost:4770", o.remoteAddr)
	require.Equal(t, "http://localhost:4771", o.remoteRestURL)
}

func TestWithHTTPClientAssignsClient(t *testing.T) {
	t.Parallel()

	client := &http.Client{}
	o := &options{}

	WithHTTPClient(client)(o)

	require.Same(t, client, o.httpClient)
}

func TestWithSessionTTLAssignsTTL(t *testing.T) {
	t.Parallel()

	o := &options{}

	WithSessionTTL(2 * time.Minute)(o)

	require.Equal(t, 2*time.Minute, o.sessionTTL)
}

func TestDefaultSessionTTL(t *testing.T) {
	t.Parallel()

	require.Equal(t, 60*time.Second, defaultSessionTTL)
}

func TestWithGRPCTimeoutAssignsTimeout(t *testing.T) {
	t.Parallel()

	o := &options{}

	WithGRPCTimeout(3 * time.Second)(o)

	require.Equal(t, 3*time.Second, o.grpcTimeout)
}

func TestWithSessionTrimsSessionID(t *testing.T) {
	t.Parallel()

	o := &options{}

	WithSession("  my-session  ")(o)

	require.Equal(t, "my-session", o.session)
}

func TestWithRemoteKeepsEmptyRestURLWhenNotProvided(t *testing.T) {
	t.Parallel()

	o := &options{}

	WithRemote("localhost:4770", "")(o)

	require.Equal(t, "localhost:4770", o.remoteAddr)
	require.Equal(t, "", o.remoteRestURL) //nolint:testifylint
}

func TestWithRemoteSetsExplicitRestURL(t *testing.T) {
	t.Parallel()

	o := &options{}

	WithRemote("127.0.0.1:7770", "http://127.0.0.1:4771")(o)

	require.Equal(t, "127.0.0.1:7770", o.remoteAddr)
	require.Equal(t, "http://127.0.0.1:4771", o.remoteRestURL)
}

func TestWithDescriptorsSkipsNilFiles(t *testing.T) {
	t.Parallel()

	o := &options{}
	name := "svc.proto"
	fds := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{
		nil,
		{Name: proto.String(name)}, //nolint:modernize
		nil,
		{Name: proto.String(name)}, //nolint:modernize
	}}

	WithDescriptors(fds)(o)

	require.Len(t, o.descriptorFiles, 1)
	require.Equal(t, name, o.descriptorFiles[0].GetName())
}

func TestWithFileDescriptor(t *testing.T) {
	t.Parallel()

	o := &options{}
	WithFileDescriptor(wrapperspb.File_google_protobuf_wrappers_proto)(o)

	require.Len(t, o.descriptorFiles, 1)
	require.Equal(t, "google/protobuf/wrappers.proto", o.descriptorFiles[0].GetName())
}
