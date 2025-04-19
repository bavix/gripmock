package deps

import "context"

func (s *Builder) Shutdown(ctx context.Context) {
	s.ender.Do(ctx)
}
