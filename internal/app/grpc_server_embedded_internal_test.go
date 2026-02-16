package app

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/features"
	"github.com/bavix/gripmock/v3/internal/domain/protoset"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func TestBuildFromDescriptorSet_Greeter(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	protoPath := filepath.Join("..", "..", "examples", "projects", "greeter", "service.proto")
	fdsSlice, err := protoset.Build(ctx, nil, []string{protoPath})
	require.NoError(t, err)
	require.NotEmpty(t, fdsSlice)

	budgerigar := stuber.NewBudgerigar(features.New())
	waiter := NewInstantExtender()

	server, err := BuildFromDescriptorSet(ctx, fdsSlice[0], budgerigar, waiter, nil)
	require.NoError(t, err)
	require.NotNil(t, server)

	defer server.GracefulStop()
}
