package stuber_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/bavix/features"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func newBudgerigar() *stuber.Budgerigar {
	return stuber.NewBudgerigar(features.New())
}

func runFindByTests(t *testing.T, create func() *stuber.Budgerigar) {
	t.Helper()

	s := create()

	require.Empty(t, s.All())

	s.PutMany(
		&stuber.Stub{ID: uuid.New(), Service: "Greeter1", Method: "SayHello1"},
		&stuber.Stub{ID: uuid.New(), Service: "Greeter1", Method: "SayHello1"},
		&stuber.Stub{ID: uuid.New(), Service: "Greeter2", Method: "SayHello2"},
		&stuber.Stub{ID: uuid.New(), Service: "Greeter3", Method: "SayHello2"},
		&stuber.Stub{ID: uuid.New(), Service: "Greeter4", Method: "SayHello3"},
		&stuber.Stub{ID: uuid.New(), Service: "Greeter5", Method: "SayHello3"},
		&stuber.Stub{ID: uuid.New(), Service: "Greeter1", Method: "SayHello3"},
	)

	require.Len(t, s.All(), 7)
}

func runFindBySortedTests(t *testing.T, create func() *stuber.Budgerigar) {
	t.Helper()

	s := create()

	stub1 := &stuber.Stub{ID: uuid.New(), Service: "Greeter1", Method: "SayHello1", Priority: 10}
	stub2 := &stuber.Stub{ID: uuid.New(), Service: "Greeter1", Method: "SayHello1", Priority: 30}
	stub3 := &stuber.Stub{ID: uuid.New(), Service: "Greeter1", Method: "SayHello1", Priority: 20}
	stub4 := &stuber.Stub{ID: uuid.New(), Service: "Greeter2", Method: "SayHello2", Priority: 50}

	s.PutMany(stub1, stub2, stub3, stub4)

	results, err := s.FindBy("Greeter1", "SayHello1")
	require.NoError(t, err)
	require.Len(t, results, 3)

	require.Equal(t, 30, results[0].Priority)
	require.Equal(t, 20, results[1].Priority)
	require.Equal(t, 10, results[2].Priority)

	results, err = s.FindBy("Greeter2", "SayHello2")
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, 50, results[0].Priority)

	_, err = s.FindBy("Greeter3", "SayHello3")
	require.ErrorIs(t, err, stuber.ErrServiceNotFound)
}
