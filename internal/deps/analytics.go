package deps

import "github.com/bavix/gripmock/v3/internal/infra/store/memory"

// Analytics returns a singleton instance of the analytics repository.
func (b *Builder) Analytics() *memory.InMemoryAnalytics {
	b.analyticsOnce.Do(func() {
		b.analytics = memory.NewInMemoryAnalytics()
	})

	return b.analytics
}
