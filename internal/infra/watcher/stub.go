package watcher

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gripmock/environment"
	"github.com/rs/zerolog"
)

type StubWatcher struct {
	enabled     bool
	interval    time.Duration
	watcherType string
}

func NewStubWatcher(
	cfg environment.Config,
) *StubWatcher {
	watcherType := string(cfg.StubWatcherType)

	if !slices.Contains(
		[]string{
			string(environment.WatcherFSNotify),
			string(environment.WatcherTimer),
		},
		watcherType,
	) {
		watcherType = string(environment.WatcherFSNotify)
	}

	return &StubWatcher{
		enabled:     cfg.StubWatcherEnabled,
		interval:    cfg.StubWatcherInterval,
		watcherType: watcherType,
	}
}

func (s *StubWatcher) Watch(ctx context.Context, folderPath string) (<-chan string, error) {
	if !s.enabled {
		ch := make(chan string)
		close(ch)

		return ch, nil
	}

	zerolog.Ctx(ctx).Info().
		Str("type", s.watcherType).
		Msg("Tracking changes in stubs")

	if s.watcherType == string(environment.WatcherFSNotify) {
		return s.notify(ctx, folderPath)
	}

	return s.ticker(ctx, folderPath)
}

func (s *StubWatcher) notify(ctx context.Context, folderPath string) (<-chan string, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	ch := make(chan string)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				zerolog.Ctx(ctx).
					Error().
					Interface("panic", r).
					Msg("Panic recovered in fsnotify watcher goroutine")
			}

			_ = watcher.Close()
		}()
		defer close(ch)

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok || event.Op == fsnotify.Chmod {
					continue
				}

				s.handleFsnotifyEvent(ctx, watcher, ch, event)
			}
		}
	}()

	_ = filepath.Walk(folderPath, func(currentPath string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		zerolog.Ctx(ctx).Err(watcher.Add(currentPath)).
			Str("path", currentPath).
			Msg("Adding directory to watcher")

		return nil
	})

	return ch, nil
}

func (s *StubWatcher) ticker(ctx context.Context, folderPath string) (<-chan string, error) {
	ch := make(chan string)

	stubFiles := make(map[string]time.Time)

	zerolog.Ctx(ctx).
		Info().
		Str("interval", s.interval.String()).
		Msg("Starting stub ticker watcher")

	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		defer close(ch)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = filepath.Walk(folderPath, func(currentPath string, info fs.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if info.IsDir() || isStub(currentPath) {
						return nil
					}

					if lastModifyTime, ok := stubFiles[currentPath]; ok && info.ModTime().Equal(lastModifyTime) {
						return nil
					}

					ch <- currentPath

					stubFiles[currentPath] = info.ModTime()

					return nil
				})
			}
		}
	}()

	return ch, nil
}

func isStub(path string) bool {
	return strings.HasSuffix(path, ".json") ||
		strings.HasSuffix(path, ".yaml") ||
		strings.HasSuffix(path, ".yml")
}

// handleFsnotifyEvent handles a single fsnotify event with panic recovery.
func (s *StubWatcher) handleFsnotifyEvent(ctx context.Context, watcher *fsnotify.Watcher, ch chan<- string, event fsnotify.Event) {
	defer func() {
		if r := recover(); r != nil {
			zerolog.Ctx(ctx).
				Error().
				Interface("panic", r).
				Str("file", event.Name).
				Msg("Panic recovered while processing fsnotify event")
		}
	}()

	info, err := os.Stat(event.Name)
	if err == nil && info.IsDir() {
		zerolog.Ctx(ctx).Err(watcher.Add(event.Name)).
			Str("path", event.Name).
			Msg("Adding directory to watcher")
	}

	if isStub(event.Name) {
		ch <- event.Name
	}
}
