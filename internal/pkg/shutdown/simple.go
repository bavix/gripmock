package shutdown

import (
	"context"

	"github.com/rs/zerolog"
)

type fn func(context.Context) error

type Shutdown struct {
	fn []fn
}

func New() *Shutdown {
	return &Shutdown{fn: []fn{}}
}

func (s *Shutdown) Add(fn fn) {
	s.fn = append(s.fn, fn)
}

func (s *Shutdown) Do(ctx context.Context) {
	for i := len(s.fn) - 1; i >= 0; i-- {
		if err := s.fn[i](ctx); err != nil {
			zerolog.Ctx(ctx).Err(err).Send()
		}
	}
}
