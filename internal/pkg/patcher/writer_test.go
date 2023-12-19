package patcher_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/internal/pkg/patcher"
)

func TestWriterWrapper_OptionUpdate(t *testing.T) {
	pile, err := os.OpenFile(
		"./../../../protogen/example/multi-files/file1.proto",
		os.O_RDONLY,
		0o444,
	)
	require.NoError(t, err)

	var result bytes.Buffer

	tmp := patcher.NewWriterWrapper(&result, "protogen/patcher/v1")

	_, err = io.Copy(tmp, pile)
	require.NoError(t, err)
	require.NoError(t, pile.Close())

	require.Contains(t, result.String(), "github.com/bavix/gripmock/protogen/patcher/v1")
}

func TestWriterWrapper_OptionInsert(t *testing.T) {
	pile := bytes.NewReader([]byte(`syntax = "proto3";

package multifiles;

service Gripmock1 {
  rpc SayHello (Request1) returns (Reply1);
}

message Request1 {
  string name = 1;
}

message Reply1 {
  string message = 1;
  int32 return_code = 2;
}`))

	var result bytes.Buffer

	tmp := patcher.NewWriterWrapper(&result, "protogen/patcher/v2")

	_, err := io.Copy(tmp, pile)
	require.NoError(t, err)

	require.Contains(t, result.String(), "github.com/bavix/gripmock/protogen/patcher/v2")
}

func TestWriterWrapper_SyntaxError(t *testing.T) {
	pile := bytes.NewReader([]byte(`
package multifiles;

service Gripmock1 {
  rpc SayHello (Request1) returns (Reply1);
}

message Request1 {
  string name = 1;
}

message Reply1 {
  string message = 1;
  int32 return_code = 2;
}`))

	var result bytes.Buffer

	tmp := patcher.NewWriterWrapper(&result, "protogen/patcher/v3")

	_, err := io.Copy(tmp, pile)
	require.ErrorIs(t, patcher.ErrSyntaxNotFound, err)
}
