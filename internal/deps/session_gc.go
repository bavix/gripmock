package deps

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/bavix/gripmock/v3/internal/config"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/lifecycle"
	"github.com/bavix/gripmock/v3/internal/infra/session"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func StartSessionGC(ctx context.Context, cfg config.Config, bg *stuber.Budgerigar, hs *history.MemoryStore, ender *lifecycle.Manager) {
	interval := cfg.SessionGCInterval
	ttl := cfg.SessionGCTTL

	if interval <= 0 || ttl <= 0 {
		return
	}

	ticker := time.NewTicker(interval)

	ender.Add(func(_ context.Context) error {
		ticker.Stop()

		return nil
	})

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				cleanupExpiredSessions(ctx, now, ttl, bg, hs)
			}
		}
	}()
}

func cleanupExpiredSessions(ctx context.Context, now time.Time, ttl time.Duration, bg *stuber.Budgerigar, hs *history.MemoryStore) {
	expired := session.Expired(now, ttl)
	if len(expired) == 0 {
		return
	}

	logger := zerolog.Ctx(ctx)

	for _, sessionID := range expired {
		// Atomically re-check + forget: if the session was re-touched after the
		// Expired() snapshot, skip it so we don't delete its just-added data.
		if !session.ForgetIfExpired(sessionID, now, ttl) {
			continue
		}

		deletedStubs := bg.DeleteSession(sessionID)
		deletedHistory := 0

		if hs != nil {
			deletedHistory = hs.DeleteSession(sessionID)
		}

		if deletedStubs > 0 || deletedHistory > 0 {
			logger.Debug().
				Str("session", sessionID).
				Int("deleted_stubs", deletedStubs).
				Int("deleted_history", deletedHistory).
				Msg("session GC cleanup")
		}
	}
}
