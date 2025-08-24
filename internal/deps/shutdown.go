package deps

import "context"

func (s *Builder) Shutdown(ctx context.Context) {
	s.ender.Wait(ctx)
}
