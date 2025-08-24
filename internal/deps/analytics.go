package deps

import (
	"github.com/bavix/gripmock/v3/internal/app/port"
	"github.com/bavix/gripmock/v3/internal/infra/store/memory"
)

// Analytics returns a singleton instance of the analytics repository.
//
//nolint:ireturn
func (b *Builder) Analytics() port.AnalyticsRepository {
	b.analyticsOnce.Do(func() {
		b.analytics = memory.NewInMemoryAnalytics()
	})

	return b.analytics
}
