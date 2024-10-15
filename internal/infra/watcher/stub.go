package watcher

import (
	"context"
	"io/fs"
	"path"
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
		Str("folder", folderPath).
		Str("type", s.watcherType).
		Msg("Watching stub files")

	if s.watcherType == string(environment.WatcherFSNotify) {
		return s.notify(ctx, folderPath)
	}

	return s.ticker(ctx, folderPath)
}

func (s *StubWatcher) notify(ctx context.Context, folderPath string) (<-chan string, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	ch := make(chan string)

	go func() {
		defer watcher.Close()
		defer close(ch)

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok || event.Op == fsnotify.Chmod {
					continue
				}

				stubPath := path.Join(folderPath, event.Name)

				if isStub(stubPath) {
					zerolog.Ctx(ctx).
						Debug().
						Str("path", event.Name).
						Msg("updating stub")

					ch <- event.Name
				}
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
			Msg("watching stub directory")

		return nil
	})

	return ch, nil
}

//nolint:gocognit
func (s *StubWatcher) ticker(ctx context.Context, folderPath string) (<-chan string, error) {
	ch := make(chan string)

	stubFiles := make(map[string]time.Time, 128) //nolint:mnd

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

					if info.IsDir() {
						return nil
					}

					if lastModifyTime, ok := stubFiles[currentPath]; ok {
						if info.ModTime().Equal(lastModifyTime) {
							return nil
						}
					}

					if isStub(currentPath) {
						stubFiles[currentPath] = info.ModTime()

						zerolog.Ctx(ctx).
							Debug().
							Str("path", currentPath).
							Msg("updating stub")

						ch <- currentPath
					}

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
