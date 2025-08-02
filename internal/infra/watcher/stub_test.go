package watcher //nolint:testpackage

import (
	"context"
	"testing"
	"time"

	"github.com/gripmock/environment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsStub(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "json file",
			path:     "test.json",
			expected: true,
		},
		{
			name:     "yaml file",
			path:     "test.yaml",
			expected: true,
		},
		{
			name:     "yml file",
			path:     "test.yml",
			expected: true,
		},
		{
			name:     "txt file",
			path:     "test.txt",
			expected: false,
		},
		{
			name:     "proto file",
			path:     "test.proto",
			expected: false,
		},
		{
			name:     "empty path",
			path:     "",
			expected: false,
		},
		{
			name:     "path with json in middle",
			path:     "test.json.backup",
			expected: false,
		},
		{
			name:     "path ending with json",
			path:     "/path/to/file.json",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isStub(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewStubWatcher(t *testing.T) {
	// Test with custom configuration
	cfg := environment.Config{
		StubWatcherEnabled:  true,
		StubWatcherInterval: time.Second,
		StubWatcherType:     environment.WatcherFSNotify,
	}
	watcher := NewStubWatcher(cfg)

	assert.True(t, watcher.enabled)
	assert.Equal(t, time.Second, watcher.interval)
	assert.Equal(t, string(environment.WatcherFSNotify), watcher.watcherType)

	// Test with custom configuration
	cfg = environment.Config{
		StubWatcherEnabled:  false,
		StubWatcherInterval: 2 * time.Second,
		StubWatcherType:     environment.WatcherTimer,
	}
	watcher = NewStubWatcher(cfg)

	assert.False(t, watcher.enabled)
	assert.Equal(t, 2*time.Second, watcher.interval)
	assert.Equal(t, string(environment.WatcherTimer), watcher.watcherType)
}

func TestStubWatcher_Watch_Disabled(t *testing.T) {
	cfg := environment.Config{
		StubWatcherEnabled: false,
	}

	watcher := NewStubWatcher(cfg)
	ctx := context.Background()

	ch, err := watcher.Watch(ctx, "/tmp")

	require.NoError(t, err)
	assert.NotNil(t, ch)

	// Channel should be closed immediately when disabled
	select {
	case _, ok := <-ch:
		assert.False(t, ok, "Channel should be closed")
	default:
		t.Error("Channel should be closed immediately")
	}
}
