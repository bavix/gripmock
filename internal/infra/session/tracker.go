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

// ForgetIfExpired atomically forgets the session only if it is still expired
// (last seen at or before now-ttl), returning whether it was forgotten. This
// closes the TOCTOU window in the GC: a session re-touched after the expiry
// snapshot was taken keeps a fresh lastSeen, so it is NOT forgotten and its
// stubs/history are preserved.
func (t *Tracker) ForgetIfExpired(sessionID string, now time.Time, ttl time.Duration) bool {
	if sessionID == "" {
		return false
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	seenAt, ok := t.lastSeen[sessionID]
	if !ok {
		return false
	}

	if ttl > 0 && now.Sub(seenAt) < ttl {
		return false // re-touched since the snapshot — keep it
	}

	delete(t.lastSeen, sessionID)

	return true
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

func ForgetIfExpired(sessionID string, now time.Time, ttl time.Duration) bool {
	return defaultTracker.ForgetIfExpired(sessionID, now, ttl)
}

// IDs returns all session IDs seen by the default tracker, sorted.
func IDs() []string {
	return defaultTracker.IDs()
}

func Expired(now time.Time, ttl time.Duration) []string {
	return defaultTracker.Expired(now, ttl)
}
