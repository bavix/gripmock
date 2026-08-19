package sdk

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/domain/protoset"
)

func sdkProtoPath(project string) string {
	return filepath.Join("..", "..", "examples", "projects", project, "service.proto")
}

func mustBuildFDS(t *testing.T, protoPath string) *descriptorpb.FileDescriptorSet {
	t.Helper()

	ctx := t.Context()
	fdsSlice, err := protoset.Build(ctx, nil, []string{protoPath}, nil)
	require.NoError(t, err)
	require.NotEmpty(t, fdsSlice)

	return fdsSlice[0]
}

func mustServerWithProto(t *testing.T, protoPath string, opts ...Option) *Server {
	t.Helper()

	fds := mustBuildFDS(t, protoPath)

	return NewTestServer(t, append([]Option{WithDescriptors(fds)}, opts...)...)
}
