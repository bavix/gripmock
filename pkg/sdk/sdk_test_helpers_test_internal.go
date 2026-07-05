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

// mustRunWithProto builds descriptors from protoPath and runs mock via Run(t, ...) (auto cleanup).
func mustRunWithProto(t *testing.T, protoPath string, opts ...Option) Mock { //nolint:ireturn
	t.Helper()

	fds := mustBuildFDS(t, protoPath)
	allOpts := append([]Option{WithDescriptors(fds)}, opts...)
	mock, err := Run(t, allOpts...)
	require.NoError(t, err)
	require.NotNil(t, mock)

	return mock
}
