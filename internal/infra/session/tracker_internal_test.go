package session_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/session"
)

// Regression for the session-GC TOCTOU: a session re-touched after the
// Expired() snapshot must NOT be forgotten by ForgetIfExpired.
func TestForgetIfExpiredKeepsRetouched(t *testing.T) {
	t.Parallel()

	tracker := session.NewTracker()
	base := time.Unix(1_000_000, 0)
	ttl := 10 * time.Second

	tracker.Touch("s", base)

	// Simulate a fresh request arriving after the expiry snapshot was taken.
	tracker.Touch("s", base.Add(9*time.Second))

	// GC tick at base+10s: last seen 1s ago (< ttl) → must be kept.
	require.False(t, tracker.ForgetIfExpired("s", base.Add(10*time.Second), ttl))
	require.Contains(t, tracker.IDs(), "s")

	// Once genuinely idle past the TTL → forgotten.
	require.True(t, tracker.ForgetIfExpired("s", base.Add(30*time.Second), ttl))
	require.NotContains(t, tracker.IDs(), "s")

	// Absent session → no-op false.
	require.False(t, tracker.ForgetIfExpired("missing", base.Add(30*time.Second), ttl))
}

func TestTrackerExpiredAndForget(t *testing.T) {
	t.Parallel()

	// Arrange
	tracker := session.NewTracker()
	now := time.Now()
	tracker.Touch("A", now.Add(-2*time.Minute))
	tracker.Touch("B", now)

	// Act
	expired := tracker.Expired(now, time.Minute)
	tracker.ForgetIfExpired("A", time.Now(), 0)
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

	tracker.ForgetIfExpired("A", time.Now(), 0)
	require.Equal(t, []string{"Z"}, tracker.IDs())
}
