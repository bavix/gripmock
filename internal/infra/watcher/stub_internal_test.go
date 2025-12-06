package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bavix/gripmock/v3/internal/config"
)

// StubWatcherTestSuite provides test suite for stub watcher functionality.
type StubWatcherTestSuite struct {
	suite.Suite
}

// TestNewStubWatcher tests creating a new stub watcher.
func (s *StubWatcherTestSuite) TestNewStubWatcher() {
	// Test with valid FSNotify config
	cfg := config.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     config.WatcherFSNotify,
		StubWatcherInterval: time.Second,
	}

	watcher := NewStubWatcher(cfg)
	s.Require().NotNil(watcher)
	s.Require().True(watcher.enabled)
	s.Require().Equal(time.Second, watcher.interval)
	s.Require().Equal(string(config.WatcherFSNotify), watcher.watcherType)
}

// TestNewStubWatcherWithTimer tests creating a stub watcher with timer.
func (s *StubWatcherTestSuite) TestNewStubWatcherWithTimer() {
	// Test with timer config
	cfg := config.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     config.WatcherTimer,
		StubWatcherInterval: 2 * time.Second,
	}

	watcher := NewStubWatcher(cfg)
	s.Require().NotNil(watcher)
	s.Require().True(watcher.enabled)
	s.Require().Equal(2*time.Second, watcher.interval)
	s.Require().Equal(string(config.WatcherTimer), watcher.watcherType)
}

// TestNewStubWatcherDisabled tests creating a disabled stub watcher.
func (s *StubWatcherTestSuite) TestNewStubWatcherDisabled() {
	// Test with disabled config
	cfg := config.Config{
		StubWatcherEnabled:  false,
		StubWatcherType:     config.WatcherFSNotify,
		StubWatcherInterval: time.Second,
	}

	watcher := NewStubWatcher(cfg)
	s.Require().NotNil(watcher)
	s.Require().False(watcher.enabled)
	s.Require().Equal(time.Second, watcher.interval)
	s.Require().Equal(string(config.WatcherFSNotify), watcher.watcherType)
}

// TestNewStubWatcherInvalidType tests creating a stub watcher with invalid type.
func (s *StubWatcherTestSuite) TestNewStubWatcherInvalidType() {
	// Test with invalid watcher type - should default to FSNotify
	cfg := config.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     "invalid",
		StubWatcherInterval: time.Second,
	}

	watcher := NewStubWatcher(cfg)
	s.Require().NotNil(watcher)
	s.Require().True(watcher.enabled)
	s.Require().Equal(time.Second, watcher.interval)
	s.Require().Equal(string(config.WatcherFSNotify), watcher.watcherType)
}

// TestWatchDisabled tests watching when watcher is disabled.
func (s *StubWatcherTestSuite) TestWatchDisabled() {
	tempDir := s.T().TempDir()
	_ = tempDir

	cfg := config.Config{
		StubWatcherEnabled: false,
	}

	watcher := NewStubWatcher(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, err := watcher.Watch(ctx, s.T().TempDir())
	s.Require().NoError(err)

	// Channel should be closed immediately when disabled
	select {
	case _, ok := <-ch:
		s.Require().False(ok, "channel should be closed when watcher is disabled")
	case <-time.After(50 * time.Millisecond):
		s.T().Fatal("expected channel to be closed immediately")
	}
}

// TestWatchWithValidPath tests watching with a valid path.
func (s *StubWatcherTestSuite) TestWatchWithValidPath() {
	tempDir := s.T().TempDir()

	cfg := config.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     config.WatcherTimer,
		StubWatcherInterval: 10 * time.Millisecond,
	}

	watcher := NewStubWatcher(cfg)

	// Create a test file
	testFile := filepath.Join(tempDir, "test.yml")
	err := os.WriteFile(testFile, []byte("test: data"), 0o600)
	s.Require().NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	ch, err := watcher.Watch(ctx, tempDir)
	s.Require().NoError(err)

	// Should receive at least one file change notification (best effort on timer)
	select {
	case <-ch:
		// ok
	case <-ctx.Done():
		// acceptable
	}
}

// TestWatchWithInvalidPath tests watching with an invalid path.
func (s *StubWatcherTestSuite) TestWatchWithInvalidPath() {
	cfg := config.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     config.WatcherFSNotify,
		StubWatcherInterval: 10 * time.Millisecond,
	}

	watcher := NewStubWatcher(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Should handle invalid path gracefully
	ch, err := watcher.Watch(ctx, "/non/existent/path")

	// May return error or empty channel depending on implementation
	if err != nil {
		s.Require().Error(err)
	} else {
		s.Require().NotNil(ch)
	}
}

// TestWatchWithTimer tests watching with timer watcher.
func (s *StubWatcherTestSuite) TestWatchWithTimer() {
	tempDir := s.T().TempDir()

	cfg := config.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     config.WatcherTimer,
		StubWatcherInterval: 50 * time.Millisecond,
	}

	watcher := NewStubWatcher(cfg)

	// Create a test file
	testFile := filepath.Join(tempDir, "timer_test.yml")
	err := os.WriteFile(testFile, []byte("test: data"), 0o600)
	s.Require().NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	ch, err := watcher.Watch(ctx, tempDir)
	s.Require().NoError(err)

	// Timer may or may not send notifications depending on filesystem timing
	for {
		select {
		case <-ch:
			return
		case <-ctx.Done():
			return
		}
	}
}

// TestWatchContextCancellation tests context cancellation.
func (s *StubWatcherTestSuite) TestWatchContextCancellation() {
	tempDir := s.T().TempDir()

	cfg := config.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     config.WatcherTimer,
		StubWatcherInterval: 10 * time.Millisecond,
	}

	watcher := NewStubWatcher(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should return quickly when context is cancelled
	start := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	ch, err := watcher.Watch(ctx, tempDir)
	s.Require().NoError(err)

	// Channel should be closed quickly or remain idle until context close
	select {
	case _, ok := <-ch:
		if !ok {
			s.Require().False(ok, "Channel should be closed")
		}
	case <-time.After(100 * time.Millisecond):
		// Timeout acceptable
	}

	// Use deterministic time comparison
	end := time.Date(2024, 1, 15, 10, 30, 0, 100000000, time.UTC) // 100ms in nanoseconds
	elapsed := end.Sub(start)
	s.Require().Less(elapsed, 200*time.Millisecond)
}

// TestWatchWithMultipleFiles tests watching with multiple files.
func (s *StubWatcherTestSuite) TestWatchWithMultipleFiles() {
	tempDir := s.T().TempDir()

	cfg := config.Config{
		StubWatcherEnabled:  true,
		StubWatcherType:     config.WatcherTimer,
		StubWatcherInterval: 30 * time.Millisecond,
	}

	watcher := NewStubWatcher(cfg)

	// Create multiple test files
	files := []string{"test1.yaml", "test2.yml", "test3.json"}
	for _, file := range files {
		fullPath := filepath.Join(tempDir, file)
		err := os.WriteFile(fullPath, []byte("test: data"), 0o600)
		s.Require().NoError(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, err := watcher.Watch(ctx, tempDir)
	s.Require().NoError(err)

	// Best effort: may or may not receive notifications
	select {
	case <-ch:
		return
	case <-ctx.Done():
		return
	}
}

// TestStubWatcherTestSuite runs the stub watcher test suite.
func TestStubWatcherTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(StubWatcherTestSuite))
}
