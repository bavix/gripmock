package session_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/session"
)

func TestTrackerExpiredAndForget(t *testing.T) {
	t.Parallel()

	// Arrange
	tracker := session.NewTracker()
	now := time.Now()
	tracker.Touch("A", now.Add(-2*time.Minute))
	tracker.Touch("B", now)

	// Act
	expired := tracker.Expired(now, time.Minute)
	tracker.Forget("A")
	expiredAfterForget := tracker.Expired(now, 0)

	// Assert
	require.Equal(t, []string{"A"}, expired)
	require.Equal(t, []string{"B"}, expiredAfterForget)
}

func TestTrackerIDs(t *testing.T) {
	t.Parallel()

	tracker := session.NewTracker()
	require.Empty(t, tracker.IDs())

	now := time.Now()
	tracker.Touch("Z", now)
	tracker.Touch("A", now)
	tracker.Touch("", now) // empty ignored

	require.Equal(t, []string{"A", "Z"}, tracker.IDs())

	tracker.Forget("A")
	require.Equal(t, []string{"Z"}, tracker.IDs())
}
