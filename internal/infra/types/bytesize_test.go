package types_test

import (
	"encoding"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/types"
)

func TestByteSize_UnmarshalText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		text    []byte
		want    int64
		wantErr bool
	}{
		{"plain bytes", []byte("262144"), 262144, false},
		{"with K suffix", []byte("128K"), 128 * 1024, false},
		{"with M suffix", []byte("64M"), 64 * 1024 * 1024, false},
		{"with G suffix", []byte("1G"), 1024 * 1024 * 1024, false},
		{"lowercase k", []byte("128k"), 128 * 1024, false},
		{"with spaces", []byte("  64M  "), 64 * 1024 * 1024, false},
		{"empty", []byte(""), 0, false},
		{"invalid", []byte("abc"), 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var b types.ByteSize

			err := b.UnmarshalText(tt.text)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, b.Bytes)
		})
	}
}

func TestByteSize_Int64(t *testing.T) {
	t.Parallel()

	var b types.ByteSize

	b.Bytes = 12345
	require.Equal(t, int64(12345), b.Int64())
}

func TestByteSize_TextUnmarshaler(t *testing.T) {
	t.Parallel()

	var b types.ByteSize
	require.Implements(t, (*encoding.TextUnmarshaler)(nil), &b)
}
