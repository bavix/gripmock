package session

import (
	"slices"
	"sync"
	"time"
)

type Tracker struct {
	mu       sync.RWMutex
	lastSeen map[string]time.Time
}

func NewTracker() *Tracker {
	return &Tracker{lastSeen: make(map[string]time.Time)}
}

func (t *Tracker) Touch(sessionID string, at time.Time) {
	if sessionID == "" {
		return
	}

	t.mu.Lock()
	t.lastSeen[sessionID] = at
	t.mu.Unlock()
}

func (t *Tracker) Forget(sessionID string) {
	if sessionID == "" {
		return
	}

	t.mu.Lock()
	delete(t.lastSeen, sessionID)
	t.mu.Unlock()
}

// IDs returns all currently-tracked session IDs, sorted.
func (t *Tracker) IDs() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	ids := make([]string, 0, len(t.lastSeen))
	for sessionID := range t.lastSeen {
		ids = append(ids, sessionID)
	}

	slices.Sort(ids)

	return ids
}

func (t *Tracker) Expired(now time.Time, ttl time.Duration) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	expired := make([]string, 0, len(t.lastSeen))
	for sessionID, seenAt := range t.lastSeen {
		if ttl <= 0 || now.Sub(seenAt) >= ttl {
			expired = append(expired, sessionID)
		}
	}

	slices.Sort(expired)

	return expired
}

//nolint:gochecknoglobals
var defaultTracker = NewTracker()

func Touch(sessionID string) {
	defaultTracker.Touch(sessionID, time.Now())
}

func Forget(sessionID string) {
	defaultTracker.Forget(sessionID)
}

// IDs returns all session IDs seen by the default tracker, sorted.
func IDs() []string {
	return defaultTracker.IDs()
}

func Expired(now time.Time, ttl time.Duration) []string {
	return defaultTracker.Expired(now, ttl)
}
