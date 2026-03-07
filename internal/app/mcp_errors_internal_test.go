package app

import (
	stderrors "errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var errTestIllegalBase64Data = stderrors.New("illegal base64 data")

func TestMcpInvalidArgErrorHasStableKind(t *testing.T) {
	t.Parallel()

	err := mcpInvalidArgError("limit must be a non-negative integer")
	require.ErrorIs(t, err, ErrMCPInvalidArgument)
	require.NotErrorIs(t, err, ErrMCPToolNotFound)
}

func TestMcpMethodNotFoundHasStableKind(t *testing.T) {
	t.Parallel()

	err := mcpMethodNotFound("unknown tool: x")
	require.ErrorIs(t, err, ErrMCPToolNotFound)
	require.NotErrorIs(t, err, ErrMCPInvalidArgument)
}

func TestMcpInvalidRequestErrorHasStableKind(t *testing.T) {
	t.Parallel()

	err := mcpInvalidRequestError()
	require.EqualError(t, err, "invalid JSON-RPC request")
	require.ErrorIs(t, err, ErrMCPInvalidRequest)
	require.NotErrorIs(t, err, ErrMCPInvalidArgument)
}

func TestMcpRPCMethodNotFoundErrorHasStableKind(t *testing.T) {
	t.Parallel()

	err := mcpRPCMethodNotFoundError()
	require.EqualError(t, err, "method not found")
	require.ErrorIs(t, err, ErrMCPToolNotFound)
	require.NotErrorIs(t, err, ErrMCPInvalidRequest)
}

func TestMcpDescriptorSetBase64ArgErrorHasStableKind(t *testing.T) {
	t.Parallel()

	err := mcpDescriptorSetBase64ArgError(errTestIllegalBase64Data)
	require.EqualError(t, err, "invalid descriptorSetBase64: illegal base64 data")
	require.ErrorIs(t, err, ErrMCPInvalidArgument)
	require.ErrorIs(t, err, errTestIllegalBase64Data)
	require.NotErrorIs(t, err, ErrMCPToolNotFound)
}

func TestMcpDescriptorRegistrationArgErrorPreservesDescriptorKind(t *testing.T) {
	t.Parallel()

	descriptorErr := registerDescriptorFileError("service.proto", errTestIllegalBase64Data)
	err := mcpDescriptorRegistrationArgError(descriptorErr)

	require.EqualError(t, err, "failed to register file service.proto: illegal base64 data")
	require.ErrorIs(t, err, ErrMCPInvalidArgument)
	require.ErrorIs(t, err, ErrRegisterDescriptorFile)
	require.NotErrorIs(t, err, ErrMCPToolNotFound)
}
