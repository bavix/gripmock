package deps

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/config"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/pkg/session"
)

//nolint:paralleltest
func TestBuilderCleanupExpiredSessions_RemovesTouchedSessionData(t *testing.T) {
	// Arrange
	b := NewBuilder(WithConfig(config.Config{HistoryEnabled: true}))
	b.Budgerigar().PutMany(
		&stuber.Stub{
			ID:      uuid.New(),
			Service: "svc.Greeter",
			Method:  "SayHello",
			Session: "A",
			Output:  stuber.Output{Data: map[string]any{"message": "A"}},
		},
		&stuber.Stub{
			ID:      uuid.New(),
			Service: "svc.Greeter",
			Method:  "SayHello",
			Session: "B",
			Output:  stuber.Output{Data: map[string]any{"message": "B"}},
		},
	)

	store := b.HistoryStore()
	store.Record(history.CallRecord{Service: "svc.Greeter", Method: "SayHello", Session: "A"})
	store.Record(history.CallRecord{Service: "svc.Greeter", Method: "SayHello", Session: "B"})

	session.Touch("A")

	// Act
	b.cleanupExpiredSessions(context.Background(), time.Now(), 0)

	// Assert
	all := b.Budgerigar().All()
	require.Len(t, all, 1)
	require.Equal(t, "B", all[0].Session)

	records := store.All()
	require.Len(t, records, 1)
	require.Equal(t, "B", records[0].Session)
}

//nolint:paralleltest
func TestBuilderCleanupExpiredSessions_DoesNotDeleteGlobalSession(t *testing.T) {
	// Arrange
	b := NewBuilder(WithConfig(config.Config{HistoryEnabled: true}))
	b.Budgerigar().PutMany(
		&stuber.Stub{
			ID:      uuid.New(),
			Service: "svc.Greeter",
			Method:  "SayHello",
			Session: "",
			Output:  stuber.Output{Data: map[string]any{"message": "GLOBAL"}},
		},
		&stuber.Stub{
			ID:      uuid.New(),
			Service: "svc.Greeter",
			Method:  "SayHello",
			Session: "A",
			Output:  stuber.Output{Data: map[string]any{"message": "A"}},
		},
	)

	store := b.HistoryStore()
	store.Record(history.CallRecord{Service: "svc.Greeter", Method: "SayHello", Session: ""})
	store.Record(history.CallRecord{Service: "svc.Greeter", Method: "SayHello", Session: "A"})

	session.Touch("A")

	// Act
	b.cleanupExpiredSessions(context.Background(), time.Now(), 0)

	// Assert
	all := b.Budgerigar().All()
	require.Len(t, all, 1)
	require.Empty(t, all[0].Session)
	require.Equal(t, "GLOBAL", all[0].Output.Data["message"])

	records := store.All()
	require.Len(t, records, 1)
	require.Empty(t, records[0].Session)
}
