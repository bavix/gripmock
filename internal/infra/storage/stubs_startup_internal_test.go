package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/config"
	"github.com/bavix/gripmock/v3/internal/infra/watcher"
)

const waitTimeout = 2 * time.Second

func requireWaitReturns(t *testing.T, extender *Extender) {
	t.Helper()

	done := make(chan struct{})

	go func() {
		extender.Wait(t.Context())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(waitTimeout):
		t.Fatal("Wait did not return after the stubs were signalled as loaded")
	}
}

func TestSignalLoadedReleasesWait(t *testing.T) {
	t.Parallel()

	extender, _ := newLoader(t)

	extender.SignalLoaded()
	requireWaitReturns(t, extender)
	require.True(t, extender.loaded.Load())
}

func TestWaitReturnsWhenTheContextEnds(t *testing.T) {
	t.Parallel()

	extender, _ := newLoader(t)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	done := make(chan struct{})

	go func() {
		extender.Wait(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(waitTimeout):
		t.Fatal("Wait ignored a cancelled context")
	}

	require.False(t, extender.loaded.Load(),
		"a cancelled wait must not claim the stubs were loaded")
}

func TestReadFromPathSyncLoadsBeforeItReturns(t *testing.T) {
	t.Parallel()

	extender, budgerigar := newLoader(t)
	extender.watcher = watcher.NewStubWatcher(config.Config{})

	dir := t.TempDir()
	writeStubFile(t, dir, "greeter.json", `[{
		"service": "helloworld.Greeter",
		"method": "SayHello",
		"input": {"equals": {"name": "startup"}},
		"output": {"data": {"message": "loaded"}}
	}]`)

	extender.ReadFromPathSync(t.Context(), dir)

	require.Len(t, budgerigar.All(), 1,
		"the stubs must be in storage by the time the call returns")
	requireWaitReturns(t, extender)
}

func TestReadFromPathSyncWithoutAPathStillReleasesWait(t *testing.T) {
	t.Parallel()

	extender, budgerigar := newLoader(t)

	extender.ReadFromPathSync(t.Context(), "")

	require.Empty(t, budgerigar.All())
	requireWaitReturns(t, extender)
}
