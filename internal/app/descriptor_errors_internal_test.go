package app

import (
	stderrors "errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	errTestInvalidWireFormat = stderrors.New("proto: cannot parse invalid wire-format data")
	errTestDuplicateSymbol   = stderrors.New("duplicate symbol")
)

func TestInvalidFileDescriptorSetError_HasStableKind(t *testing.T) {
	t.Parallel()

	err := invalidFileDescriptorSetError(errTestInvalidWireFormat)
	require.EqualError(t, err, "invalid FileDescriptorSet: proto: cannot parse invalid wire-format data")
	require.ErrorIs(t, err, ErrInvalidFileDescriptorSet)
	require.NotErrorIs(t, err, ErrRegisterDescriptorFile)
}

func TestRegisterDescriptorFileError_HasStableKind(t *testing.T) {
	t.Parallel()

	err := registerDescriptorFileError("service.proto", errTestDuplicateSymbol)
	require.EqualError(t, err, "failed to register file service.proto: duplicate symbol")
	require.ErrorIs(t, err, ErrRegisterDescriptorFile)
	require.NotErrorIs(t, err, ErrInvalidFileDescriptorSet)
}
