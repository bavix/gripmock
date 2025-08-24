package shutdown

import (
	"context"
	"sync"
)

// Shutdown coordinates graceful shutdown callbacks.
type Shutdown struct {
	mu  sync.Mutex
	fns []func(context.Context) error
}

// New returns a new Shutdown instance.
func New(_ any) *Shutdown { // keep signature compatible with previous usage
	return &Shutdown{}
}

// Add registers shutdown callbacks.
func (s *Shutdown) Add(fns ...func(context.Context) error) {
	s.mu.Lock()
	s.fns = append(s.fns, fns...)
	s.mu.Unlock()
}

// Wait executes all registered callbacks with the provided context.
func (s *Shutdown) Wait(ctx context.Context) {
	s.mu.Lock()
	fns := append([]func(context.Context) error(nil), s.fns...)
	s.mu.Unlock()

	var wg sync.WaitGroup
	for _, fn := range fns {
		wg.Add(1)

		go func(fn func(context.Context) error) {
			defer wg.Done()

			_ = fn(ctx)
		}(fn)
	}

	wg.Wait()
}

// Do executes all shutdown functions in reverse order.
func (s *Shutdown) Do(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := len(s.fns) - 1; i >= 0; i-- {
		if err := s.fns[i](ctx); err != nil {
			// Log error if logger is available
			// For now, we'll just ignore the error
			_ = err // explicitly ignore error
		}
	}

	s.fns = nil
}
