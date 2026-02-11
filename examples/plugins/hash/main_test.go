package main

import (
	"crypto/md5" //nolint:gosec
	"crypto/sha256"
	"encoding/hex"
	"hash/crc32"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/pkg/plugintest"
)

func TestHashPlugin(t *testing.T) {
	t.Parallel()

	reg := plugintest.NewRegistry()
	Register(reg)

	fnCRC, ok := plugintest.LookupFunc(reg, "crc32")
	require.True(t, ok, "crc32 not registered")

	outCRC, err := plugintest.Call(t.Context(), fnCRC, "hello")
	require.NoError(t, err)

	wantCRC := crc32.ChecksumIEEE([]byte("hello"))
	require.Equal(t, wantCRC, outCRC)

	fnSHA, ok := plugintest.LookupFunc(reg, "sha256")
	require.True(t, ok, "sha256 not registered")

	outSHA, err := plugintest.Call(t.Context(), fnSHA, "hello")
	require.NoError(t, err)

	wantSHA := sha256.Sum256([]byte("hello"))
	require.Equal(t, hex.EncodeToString(wantSHA[:]), outSHA)

	fnMD5, ok := plugintest.LookupFunc(reg, "md5")
	require.True(t, ok, "md5 not registered")

	outMD5, err := plugintest.Call(t.Context(), fnMD5, "hello")
	require.NoError(t, err)

	wantMD5 := md5.Sum([]byte("hello")) //nolint:gosec
	require.Equal(t, hex.EncodeToString(wantMD5[:]), outMD5)
}
