package watcher

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"

	"github.com/bavix/gripmock/v3/internal/config"
)

type StubWatcher struct {
	enabled     bool
	interval    time.Duration
	watcherType string
}

// debouncer coalesces rapid file change events into a single notification.
// Each file path has its own timer that resets on each new event.
type debouncer struct {
	mu     sync.Mutex
	timers map[string]*time.Timer
	output chan<- string
	done   chan struct{}
	closed bool
	wg     sync.WaitGroup
}

func newDebouncer(output chan<- string) *debouncer {
	return &debouncer{
		timers: make(map[string]*time.Timer),
		output: output,
		done:   make(chan struct{}),
	}
}

func (d *debouncer) add(path string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return
	}

	// Stop() == true means the old timer's callback will NOT run, so balance its
	// wg.Add here — otherwise stopAll's wg.Wait would block forever on it.
	if t, ok := d.timers[path]; ok {
		if t.Stop() {
			d.wg.Done()
		}
	}

	d.wg.Add(1)
	d.timers[path] = time.AfterFunc(100*time.Millisecond, func() { //nolint:mnd
		defer d.wg.Done()

		d.mu.Lock()
		delete(d.timers, path)
		closed := d.closed
		d.mu.Unlock()

		if closed {
			return
		}

		// Select on done so a fired timer never blocks (no consumer) or sends
		// after shutdown; stopAll closes done and waits for in-flight sends
		// before the output channel is closed, so this can never send on a
		// closed channel.
		select {
		case d.output <- path:
		case <-d.done:
		}
	})
}

// stopAll signals shutdown, stops pending timers, and blocks until every
// in-flight timer callback has returned — so the caller may safely close the
// output channel afterwards.
func (d *debouncer) stopAll() {
	d.mu.Lock()

	if d.closed {
		d.mu.Unlock()

		return
	}

	d.closed = true
	close(d.done)

	// Balance the wg.Add of every timer stopped before firing (its callback, and
	// thus its wg.Done, will never run) so wg.Wait can't deadlock.
	for _, t := range d.timers {
		if t.Stop() {
			d.wg.Done()
		}
	}

	d.timers = make(map[string]*time.Timer)
	d.mu.Unlock()

	d.wg.Wait()
}

func NewStubWatcher(
	cfg config.Config,
) *StubWatcher {
	watcherType := string(cfg.StubWatcherType)

	if !slices.Contains(
		[]string{
			string(config.WatcherFSNotify),
			string(config.WatcherTimer),
		},
		watcherType,
	) {
		watcherType = string(config.WatcherFSNotify)
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

	if s.watcherType == string(config.WatcherFSNotify) {
		return s.notify(ctx, folderPath)
	}

	return s.ticker(ctx, folderPath)
}

//nolint:cyclop
func (s *StubWatcher) notify(ctx context.Context, folderPath string) (<-chan string, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	ch := make(chan string)
	d := newDebouncer(ch)

	go func() {
		// Registered first, so it runs LAST — only after stopAll has stopped
		// every timer and drained in-flight sends, guaranteeing no timer
		// callback sends on ch after it is closed.
		defer close(ch)
		defer func() {
			if r := recover(); r != nil {
				zerolog.Ctx(ctx).
					Error().
					Interface("panic", r).
					Msg("Panic recovered in fsnotify watcher goroutine")
			}

			d.stopAll()

			err := watcher.Close()
			if err != nil {
				zerolog.Ctx(ctx).Warn().Err(err).Msg("failed to close file watcher")
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok || event.Op == fsnotify.Chmod {
					continue
				}

				s.handleFsnotifyEvent(ctx, watcher, d, event)
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

					if info.IsDir() || !isStub(currentPath) {
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
func (s *StubWatcher) handleFsnotifyEvent(ctx context.Context, watcher *fsnotify.Watcher, d *debouncer, event fsnotify.Event) {
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
		d.add(event.Name)
	}
}
