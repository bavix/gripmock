package app

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bavix/gripmock/v3/internal/infra/types"
)

func TestGrpcMockerDelay_RespectsContextTimeout(t *testing.T) {
	t.Parallel()

	// Arrange
	m := &grpcMocker{}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()

	// Act
	err := m.delay(ctx, types.Duration(200*time.Millisecond))

	// Assert
	require.Error(t, err)
	require.Equal(t, codes.DeadlineExceeded, status.Code(err))
}

func TestGrpcMockerDelay_CompletesBeforeTimeout(t *testing.T) {
	t.Parallel()

	// Arrange
	m := &grpcMocker{}

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	// Act
	err := m.delay(ctx, types.Duration(5*time.Millisecond))

	// Assert
	require.NoError(t, err)
}
