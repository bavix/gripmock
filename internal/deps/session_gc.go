package deps

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/bavix/gripmock/v3/internal/pkg/session"
)

func (b *Builder) StartSessionGC(ctx context.Context) {
	b.sessionGCOnce.Do(func() {
		interval := b.config.SessionGCInterval
		ttl := b.config.SessionGCTTL

		if interval <= 0 || ttl <= 0 {
			return
		}

		ticker := time.NewTicker(interval)

		b.ender.Add(func(_ context.Context) error {
			ticker.Stop()

			return nil
		})

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case now := <-ticker.C:
					b.cleanupExpiredSessions(ctx, now, ttl)
				}
			}
		}()
	})
}

func (b *Builder) cleanupExpiredSessions(ctx context.Context, now time.Time, ttl time.Duration) {
	expired := session.Expired(now, ttl)
	if len(expired) == 0 {
		return
	}

	logger := zerolog.Ctx(ctx)
	historyStore := b.HistoryStore()

	for _, sessionID := range expired {
		deletedStubs := b.Budgerigar().DeleteSession(sessionID)
		deletedHistory := 0

		if historyStore != nil {
			deletedHistory = historyStore.DeleteSession(sessionID)
		}

		session.Forget(sessionID)

		if deletedStubs > 0 || deletedHistory > 0 {
			logger.Debug().
				Str("session", sessionID).
				Int("deleted_stubs", deletedStubs).
				Int("deleted_history", deletedHistory).
				Msg("session GC cleanup")
		}
	}
}
