package lifecycle

import (
	"context"
	"sync"
)

// Fn describes a shutdown callback.
type Fn func(context.Context) error

// Logger is a minimal logger interface used to report shutdown errors.
type Logger interface {
	Err(err error)
}

// Manager collects shutdown callbacks and executes them in LIFO order.
type Manager struct {
	mu     sync.Mutex
	fns    []Fn
	logger Logger
}

// New creates a Manager with an optional logger.
func New(logger Logger) *Manager {
	return &Manager{
		fns:    []Fn{},
		logger: logger,
	}
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

// Do runs registered callbacks in reverse order. Errors are logged via the
// provided logger, if any, and stored callbacks are cleared after execution.
func (m *Manager) Do(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := len(m.fns) - 1; i >= 0; i-- {
		if err := m.fns[i](ctx); err != nil && m.logger != nil {
			m.logger.Err(err)
		}
	}

	m.fns = nil
}
