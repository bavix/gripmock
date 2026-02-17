package session_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/pkg/session"
)

func TestTracker_ExpiredAndForget(t *testing.T) {
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
