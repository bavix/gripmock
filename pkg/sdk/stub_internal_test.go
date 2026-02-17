package sdk

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func TestStubBuilderCommit_AssignsNonNilID(t *testing.T) {
	t.Parallel()

	// Arrange
	var got *stuber.Stub
	b := &stubBuilderCore{
		service: "helloworld.Greeter",
		method:  "SayHello",
		onCommit: func(stub *stuber.Stub) {
			got = stub
		},
	}

	// Act
	b.Unary("name", "Bob", "message", "Hello Bob").Commit()

	// Assert
	require.NotNil(t, got)
	require.NotEqual(t, uuid.Nil, got.ID)
}
