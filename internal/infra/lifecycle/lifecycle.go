package lifecycle

import (
	"context"
	"slices"
	"sync"

	"github.com/rs/zerolog"
)

// Fn describes a shutdown callback.
type Fn func(context.Context) error

// Manager collects shutdown callbacks and executes them in LIFO order.
type Manager struct {
	mu  sync.Mutex
	fns []Fn
}

// New creates a Manager.
func New() *Manager {
	return &Manager{fns: []Fn{}}
}

// Add registers one or more callbacks. Nil callbacks are ignored.
func (m *Manager) Add(fns ...Fn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, fn := range fns {
		if fn == nil {
			continue
		}

		m.fns = append(m.fns, fn)
	}
}

// Do runs registered callbacks in reverse order and clears them. A callback that
// fails is reported through the context logger instead of being swallowed.
func (m *Manager) Do(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, v := range slices.Backward(m.fns) {
		err := v(ctx)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("shutdown callback failed")
		}
	}

	m.fns = nil
}
