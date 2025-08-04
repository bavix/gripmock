package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gripmock/environment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStubWatcher(t *testing.T) {
	// Test with valid FSNotify config
	cfg := environment.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     environment.WatcherFSNotify,
		StubWatcherInterval: time.Second,
	}

	watcher := NewStubWatcher(cfg)
	assert.NotNil(t, watcher)
	assert.True(t, watcher.enabled)
	assert.Equal(t, time.Second, watcher.interval)
	assert.Equal(t, string(environment.WatcherFSNotify), watcher.watcherType)
}

func TestNewStubWatcher_WithTimer(t *testing.T) {
	// Test with Timer config
	cfg := environment.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     environment.WatcherTimer,
		StubWatcherInterval: 2 * time.Second,
	}

	watcher := NewStubWatcher(cfg)
	assert.NotNil(t, watcher)
	assert.True(t, watcher.enabled)
	assert.Equal(t, 2*time.Second, watcher.interval)
	assert.Equal(t, string(environment.WatcherTimer), watcher.watcherType)
}

func TestNewStubWatcher_WithInvalidType(t *testing.T) {
	// Test with invalid watcher type
	cfg := environment.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     "invalid_type",
		StubWatcherInterval: time.Second,
	}

	watcher := NewStubWatcher(cfg)
	assert.NotNil(t, watcher)
	assert.True(t, watcher.enabled)
	// Should default to FSNotify
	assert.Equal(t, string(environment.WatcherFSNotify), watcher.watcherType)
}

func TestNewStubWatcher_Disabled(t *testing.T) {
	// Test with disabled watcher
	cfg := environment.Config{
		StubWatcherEnabled:  false,
		StubWatcherType:     environment.WatcherFSNotify,
		StubWatcherInterval: time.Second,
	}

	watcher := NewStubWatcher(cfg)
	assert.NotNil(t, watcher)
	assert.False(t, watcher.enabled)
}

func TestStubWatcher_Watch_Disabled(t *testing.T) {
	// Test watching when disabled
	cfg := environment.Config{
		StubWatcherEnabled: false,
	}
	watcher := NewStubWatcher(cfg)
	ctx := context.Background()

	ch, err := watcher.Watch(ctx, "/tmp")
	require.NoError(t, err)
	assert.NotNil(t, ch)

	// Channel should be closed immediately
	_, ok := <-ch
	assert.False(t, ok, "Channel should be closed when watcher is disabled")
}

func TestStubWatcher_Watch_WithFSNotify(t *testing.T) {
	// Test watching with FSNotify
	cfg := environment.Config{
		StubWatcherEnabled: true,
		StubWatcherType:    environment.WatcherFSNotify,
	}
	watcher := NewStubWatcher(cfg)
	ctx := context.Background()

	// Create temporary directory
	tempDir := t.TempDir()

	ch, err := watcher.Watch(ctx, tempDir)
	require.NoError(t, err)
	assert.NotNil(t, ch)

	// Clean up
	_, cancel := context.WithCancel(ctx)
	cancel()
}

func TestStubWatcher_Watch_WithTimer(t *testing.T) {
	// Test watching with Timer
	cfg := environment.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     environment.WatcherTimer,
		StubWatcherInterval: 100 * time.Millisecond,
	}
	watcher := NewStubWatcher(cfg)
	ctx := context.Background()

	// Create temporary directory
	tempDir := t.TempDir()

	ch, err := watcher.Watch(ctx, tempDir)
	require.NoError(t, err)
	assert.NotNil(t, ch)

	// Clean up
	_, cancel := context.WithCancel(ctx)
	cancel()
}

func TestStubWatcher_notify_WithInvalidPath(t *testing.T) {
	// Test notify with invalid path
	cfg := environment.Config{
		StubWatcherEnabled: true,
		StubWatcherType:    environment.WatcherFSNotify,
	}
	watcher := NewStubWatcher(cfg)
	ctx := context.Background()

	// Use non-existent path
	ch, err := watcher.notify(ctx, "/non/existent/path")
	require.NoError(t, err)
	assert.NotNil(t, ch)

	// Clean up
	_, cancel := context.WithCancel(ctx)
	cancel()
}

func TestStubWatcher_ticker_WithInvalidPath(t *testing.T) {
	// Test ticker with invalid path
	cfg := environment.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     environment.WatcherTimer,
		StubWatcherInterval: 100 * time.Millisecond,
	}
	watcher := NewStubWatcher(cfg)
	ctx := context.Background()

	// Use non-existent path
	ch, err := watcher.ticker(ctx, "/non/existent/path")
	require.NoError(t, err)
	assert.NotNil(t, ch)

	// Clean up
	_, cancel := context.WithCancel(ctx)
	cancel()
}

func TestStubWatcher_ticker_WithValidPath(t *testing.T) {
	// Test ticker with valid path
	cfg := environment.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     environment.WatcherTimer,
		StubWatcherInterval: 100 * time.Millisecond,
	}
	watcher := NewStubWatcher(cfg)
	ctx := context.Background()

	// Create temporary directory with stub files
	tempDir := t.TempDir()

	// Create stub files
	stubFile1 := filepath.Join(tempDir, "test1.json")
	err := os.WriteFile(stubFile1, []byte(`{"test": "data1"}`), 0o600)
	require.NoError(t, err)

	stubFile2 := filepath.Join(tempDir, "test2.yaml")
	err = os.WriteFile(stubFile2, []byte(`test: data2`), 0o600)
	require.NoError(t, err)

	stubFile3 := filepath.Join(tempDir, "test3.yml")
	err = os.WriteFile(stubFile3, []byte(`test: data3`), 0o600)
	require.NoError(t, err)

	ch, err := watcher.ticker(ctx, tempDir)
	require.NoError(t, err)
	assert.NotNil(t, ch)

	// Wait a bit for the ticker to process files
	time.Sleep(200 * time.Millisecond)

	// Clean up
	_, cancel := context.WithCancel(ctx)
	cancel()
}

func TestStubWatcher_ticker_WithModifiedFiles(t *testing.T) {
	// Test ticker with modified files
	cfg := environment.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     environment.WatcherTimer,
		StubWatcherInterval: 100 * time.Millisecond,
	}
	watcher := NewStubWatcher(cfg)
	ctx := context.Background()

	// Create temporary directory
	tempDir := t.TempDir()

	// Create initial stub file
	stubFile := filepath.Join(tempDir, "test.json")
	err := os.WriteFile(stubFile, []byte(`{"test": "initial"}`), 0o600)
	require.NoError(t, err)

	ch, err := watcher.ticker(ctx, tempDir)
	require.NoError(t, err)
	assert.NotNil(t, ch)

	// Wait for initial processing
	time.Sleep(200 * time.Millisecond)

	// Modify the file
	err = os.WriteFile(stubFile, []byte(`{"test": "modified"}`), 0o600)
	require.NoError(t, err)

	// Wait for modification to be detected
	time.Sleep(200 * time.Millisecond)

	// Clean up
	_, cancel := context.WithCancel(ctx)
	cancel()
}

func TestStubWatcher_ticker_WithNonStubFiles(t *testing.T) {
	// Test ticker with non-stub files
	cfg := environment.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     environment.WatcherTimer,
		StubWatcherInterval: 100 * time.Millisecond,
	}
	watcher := NewStubWatcher(cfg)
	ctx := context.Background()

	// Create temporary directory
	tempDir := t.TempDir()

	// Create non-stub files
	nonStubFile1 := filepath.Join(tempDir, "test1.txt")
	err := os.WriteFile(nonStubFile1, []byte("not a stub"), 0o600)
	require.NoError(t, err)

	nonStubFile2 := filepath.Join(tempDir, "test2.xml")
	err = os.WriteFile(nonStubFile2, []byte("<xml>not a stub</xml>"), 0o600)
	require.NoError(t, err)

	ch, err := watcher.ticker(ctx, tempDir)
	require.NoError(t, err)
	assert.NotNil(t, ch)

	// Wait a bit for the ticker to process files
	time.Sleep(200 * time.Millisecond)

	// Clean up
	_, cancel := context.WithCancel(ctx)
	cancel()
}

func TestStubWatcher_ticker_WithSubdirectories(t *testing.T) {
	// Test ticker with subdirectories
	cfg := environment.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     environment.WatcherTimer,
		StubWatcherInterval: 100 * time.Millisecond,
	}
	watcher := NewStubWatcher(cfg)
	ctx := context.Background()

	// Create temporary directory structure
	tempDir := t.TempDir()

	// Create subdirectory
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0o750)
	require.NoError(t, err)

	// Create stub file in subdirectory
	stubFile := filepath.Join(subDir, "test.json")
	err = os.WriteFile(stubFile, []byte(`{"test": "subdir"}`), 0o600)
	require.NoError(t, err)

	ch, err := watcher.ticker(ctx, tempDir)
	require.NoError(t, err)
	assert.NotNil(t, ch)

	// Wait a bit for the ticker to process files
	time.Sleep(200 * time.Millisecond)

	// Clean up
	_, cancel := context.WithCancel(ctx)
	cancel()
}

func TestStubWatcher_ticker_WithContextCancellation(t *testing.T) {
	// Test ticker with context cancellation
	cfg := environment.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     environment.WatcherTimer,
		StubWatcherInterval: 100 * time.Millisecond,
	}
	watcher := NewStubWatcher(cfg)
	ctx, cancel := context.WithCancel(context.Background())

	// Create temporary directory
	tempDir := t.TempDir()

	ch, err := watcher.ticker(ctx, tempDir)
	require.NoError(t, err)
	assert.NotNil(t, ch)

	// Cancel context immediately
	cancel()

	// Wait a bit to ensure cleanup
	time.Sleep(50 * time.Millisecond)
}

func TestStubWatcher_ticker_WithEmptyDirectory(t *testing.T) {
	// Test ticker with empty directory
	cfg := environment.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     environment.WatcherTimer,
		StubWatcherInterval: 100 * time.Millisecond,
	}
	watcher := NewStubWatcher(cfg)
	ctx := context.Background()

	// Create empty temporary directory
	tempDir := t.TempDir()

	ch, err := watcher.ticker(ctx, tempDir)
	require.NoError(t, err)
	assert.NotNil(t, ch)

	// Wait a bit for the ticker to process
	time.Sleep(200 * time.Millisecond)

	// Clean up
	_, cancel := context.WithCancel(ctx)
	cancel()
}

func TestStubWatcher_ticker_WithMixedFiles(t *testing.T) {
	// Test ticker with mixed file types
	cfg := environment.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     environment.WatcherTimer,
		StubWatcherInterval: 100 * time.Millisecond,
	}
	watcher := NewStubWatcher(cfg)
	ctx := context.Background()

	// Create temporary directory
	tempDir := t.TempDir()

	// Create mixed files
	files := map[string]string{
		"stub1.json":   `{"test": "json"}`,
		"stub2.yaml":   `test: yaml`,
		"stub3.yml":    `test: yml`,
		"nonstub1.txt": "not a stub",
		"nonstub2.xml": "<xml>not a stub</xml>",
		"nonstub3.log": "log file",
	}

	for filename, content := range files {
		filepath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filepath, []byte(content), 0o600)
		require.NoError(t, err)
	}

	ch, err := watcher.ticker(ctx, tempDir)
	require.NoError(t, err)
	assert.NotNil(t, ch)

	// Wait a bit for the ticker to process files
	time.Sleep(200 * time.Millisecond)

	// Clean up
	_, cancel := context.WithCancel(ctx)
	cancel()
}

func TestIsStub(t *testing.T) {
	// Test isStub function
	testCases := []struct {
		name     string
		path     string
		expected bool
	}{
		{"JSON file", "test.json", true},
		{"YAML file", "test.yaml", true},
		{"YML file", "test.yml", true},
		{"TXT file", "test.txt", false},
		{"XML file", "test.xml", false},
		{"LOG file", "test.log", false},
		{"JSON with path", "/path/to/test.json", true},
		{"YAML with path", "/path/to/test.yaml", true},
		{"YML with path", "/path/to/test.yml", true},
		{"TXT with path", "/path/to/test.txt", false},
		{"Empty path", "", false},
		{"Just extension", ".json", true},
		{"Just extension", ".yaml", true},
		{"Just extension", ".yml", true},
		{"Just extension", ".txt", false},
		{"Multiple dots", "test.backup.json", true},
		{"Multiple dots", "test.backup.yaml", true},
		{"Multiple dots", "test.backup.yml", true},
		{"Multiple dots", "test.backup.txt", false},
		// Note: isStub is case-sensitive, so uppercase extensions are not recognized
		{"Uppercase", "TEST.JSON", false},
		{"Uppercase", "TEST.YAML", false},
		{"Uppercase", "TEST.YML", false},
		{"Uppercase", "TEST.TXT", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isStub(tc.path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestStubWatcher_Struct(t *testing.T) {
	// Test StubWatcher struct
	watcher := &StubWatcher{
		enabled:     true,
		interval:    time.Second,
		watcherType: "test",
	}

	assert.True(t, watcher.enabled)
	assert.Equal(t, time.Second, watcher.interval)
	assert.Equal(t, "test", watcher.watcherType)
}

func TestStubWatcher_Watch_WithEmptyPath(t *testing.T) {
	// Test watching with empty path
	cfg := environment.Config{
		StubWatcherEnabled: true,
		StubWatcherType:    environment.WatcherFSNotify,
	}
	watcher := NewStubWatcher(cfg)
	ctx := context.Background()

	ch, err := watcher.Watch(ctx, "")
	require.NoError(t, err)
	assert.NotNil(t, ch)

	// Clean up
	_, cancel := context.WithCancel(ctx)
	cancel()
}
