package storage

import (
	"context"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/gripmock/stuber"
	"github.com/rs/zerolog"
	"github.com/samber/lo"

	"github.com/bavix/gripmock/v3/internal/infra/jsondecoder"
	"github.com/bavix/gripmock/v3/internal/infra/watcher"
	"github.com/bavix/gripmock/v3/internal/infra/yaml2json"
)

type Extender struct {
	storage      *stuber.Budgerigar
	converter    *yaml2json.Convertor
	ch           chan struct{}
	watcher      *watcher.StubWatcher
	mapIDsByFile map[string]uuid.UUIDs
	muUniqueIDs  sync.Mutex
	uniqueIDs    map[uuid.UUID]struct{}
	loaded       atomic.Bool
}

func NewStub(
	storage *stuber.Budgerigar,
	converter *yaml2json.Convertor,
	watcher *watcher.StubWatcher,
) *Extender {
	return &Extender{
		storage:      storage,
		converter:    converter,
		ch:           make(chan struct{}),
		watcher:      watcher,
		mapIDsByFile: make(map[string]uuid.UUIDs),
		uniqueIDs:    make(map[uuid.UUID]struct{}),
		loaded:       atomic.Bool{},
	}
}

func (s *Extender) Wait(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-s.ch:
		s.loaded.Store(true)
	}
}

func (s *Extender) ReadFromPath(ctx context.Context, pathDir string) {
	if pathDir == "" {
		close(s.ch)

		return
	}

	zerolog.Ctx(ctx).Info().Msg("Loading stubs from directory (preserving API stubs)")

	s.readFromPath(ctx, pathDir)
	close(s.ch)

	// Only watch directories, not individual files
	if isDirectory(pathDir) {
		ch, err := s.watcher.Watch(ctx, pathDir)
		if err != nil {
			return
		}

		for file := range ch {
			zerolog.Ctx(ctx).
				Debug().
				Str("path", file).
				Msg("Updating stub")

			s.readByFile(ctx, file)
		}
	}
}

// readFromPath reads all the stubs from the given directory and its subdirectories,
// or from a single file if a file path is provided.
// The stub files can be in yaml or json format.
// If a file is in yaml format, it will be converted to json format.
func (s *Extender) readFromPath(ctx context.Context, pathDir string) {
	if !isDirectory(pathDir) {
		s.handleFilePath(ctx, pathDir)

		return
	}

	s.handleDirectoryPath(ctx, pathDir)
}

// handleFilePath processes a single file path.
func (s *Extender) handleFilePath(ctx context.Context, filePath string) {
	if s.isStubFile(filePath) {
		s.readByFile(ctx, filePath)
	}
}

// handleDirectoryPath processes a directory path recursively.
func (s *Extender) handleDirectoryPath(ctx context.Context, pathDir string) {
	files, err := os.ReadDir(pathDir)
	if err != nil {
		zerolog.Ctx(ctx).
			Err(err).Str("path", pathDir).
			Msg("read directory")

		return
	}

	for _, file := range files {
		if file.IsDir() {
			s.readFromPath(ctx, path.Join(pathDir, file.Name()))

			continue
		}

		if s.isStubFile(file.Name()) {
			s.readByFile(ctx, path.Join(pathDir, file.Name()))
		}
	}
}

// isStubFile checks if the given filename has a stub file extension.
func (s *Extender) isStubFile(filename string) bool {
	return strings.HasSuffix(filename, ".json") ||
		strings.HasSuffix(filename, ".yaml") ||
		strings.HasSuffix(filename, ".yml")
}

func (s *Extender) readByFile(ctx context.Context, filePath string) {
	stubs, err := s.readStub(filePath)
	if err != nil {
		s.handleFileReadError(ctx, filePath, err)

		return
	}

	s.checkUniqIDs(ctx, filePath, stubs)

	existingIDs, exists := s.mapIDsByFile[filePath]
	if !exists {
		s.handleFirstTimeLoad(filePath, stubs)

		return
	}

	s.handleExistingFileUpdate(filePath, stubs, existingIDs)
}

// handleFileReadError handles errors when reading stub files.
func (s *Extender) handleFileReadError(ctx context.Context, filePath string, err error) {
	zerolog.Ctx(ctx).
		Err(err).
		Str("file", filePath).
		Msg("failed to read file")

	// Remove existing stubs from this file if it was previously loaded
	if existingIDs, exists := s.mapIDsByFile[filePath]; exists {
		s.storage.DeleteByID(existingIDs...)
		delete(s.mapIDsByFile, filePath)
	}
}

// handleFirstTimeLoad handles the first time loading of a file.
func (s *Extender) handleFirstTimeLoad(filePath string, stubs []*stuber.Stub) {
	// Generate new IDs for stubs without them
	for _, stub := range stubs {
		if stub.ID == uuid.Nil {
			stub.ID = uuid.New()
		}
	}

	s.mapIDsByFile[filePath] = s.storage.PutMany(stubs...)
}

// handleExistingFileUpdate handles updating an existing file with ID reuse logic.
func (s *Extender) handleExistingFileUpdate(filePath string, stubs []*stuber.Stub, existingIDs uuid.UUIDs) {
	currentIDs := s.extractCurrentIDs(stubs)
	unusedIDs := lo.Without(existingIDs, currentIDs...)
	newIDs := s.generateNewIDs(stubs, unusedIDs)

	// Remove stubs that are no longer in the file
	if removedIDs := lo.Without(existingIDs, newIDs...); len(removedIDs) > 0 {
		s.storage.DeleteByID(removedIDs...)
	}

	// Add/update stubs and update file mapping
	if len(stubs) > 0 {
		s.mapIDsByFile[filePath] = s.storage.PutMany(stubs...)
	} else {
		delete(s.mapIDsByFile, filePath)
	}
}

// extractCurrentIDs extracts current IDs from stubs.
func (s *Extender) extractCurrentIDs(stubs []*stuber.Stub) uuid.UUIDs {
	currentIDs := make(uuid.UUIDs, 0, len(stubs))
	for _, stub := range stubs {
		if stub.ID != uuid.Nil {
			currentIDs = append(currentIDs, stub.ID)
		}
	}

	return currentIDs
}

// generateNewIDs generates new IDs for stubs, reusing unused IDs first.
func (s *Extender) generateNewIDs(stubs []*stuber.Stub, unusedIDs uuid.UUIDs) uuid.UUIDs {
	newIDs := make(uuid.UUIDs, 0, len(stubs))
	for _, stub := range stubs {
		if stub.ID == uuid.Nil {
			stub.ID, unusedIDs = genID(stub, unusedIDs)
		}

		newIDs = append(newIDs, stub.ID)
	}

	return newIDs
}

// checkUniqIDs checks for unique IDs in the provided stubs.
// It logs a warning if a duplicate ID is found.
// If the Extender has already been loaded, it skips the check.
func (s *Extender) checkUniqIDs(ctx context.Context, filePath string, stubs []*stuber.Stub) {
	// If the Extender is already loaded, no need to check for unique IDs.
	if s.loaded.Load() {
		return
	}

	// The mutex is not needed now, but it may be useful in the future.
	// Lock the mutex to prevent concurrent access to the uniqIDs map.
	s.muUniqueIDs.Lock()
	defer s.muUniqueIDs.Unlock()

	// Iterate over each stub to verify uniqueness of IDs.
	for _, stub := range stubs {
		// Skip stubs without an ID.
		if stub.ID == uuid.Nil {
			continue
		}

		// Check if the ID already exists in the uniqIDs map.
		if _, exists := s.uniqueIDs[stub.ID]; exists {
			// Log a warning if a duplicate ID is found.
			zerolog.Ctx(ctx).
				Warn().
				Str("file", filePath).
				Str("id", stub.ID.String()).
				Msg("duplicate stub ID")
		}

		// Mark the stub ID as seen by adding it to the uniqIDs map.
		s.uniqueIDs[stub.ID] = struct{}{}
	}
}

// genID generates a new ID for a stub if it does not already have one.
// It also returns the remaining free IDs after generating the new ID.
func genID(stub *stuber.Stub, freeIDs uuid.UUIDs) (uuid.UUID, uuid.UUIDs) {
	// If the stub already has an ID, return it.
	if stub.ID != uuid.Nil {
		return stub.ID, freeIDs
	}

	// If there are free IDs, use the first one.
	if len(freeIDs) > 0 {
		return freeIDs[0], freeIDs[1:]
	}

	// Otherwise, generate a new ID.
	return uuid.New(), nil
}

// readStub reads a stub file and returns a slice of stubs.
// The stub file can be in yaml or json format.
// If the file is in yaml format, it will be converted to json format.
func (s *Extender) readStub(path string) ([]*stuber.Stub, error) {
	file, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file %s", path)
	}

	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		file, err = s.converter.Execute(path, file)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal file %s", path)
		}
	}

	var stubs []*stuber.Stub
	if err := jsondecoder.UnmarshalSlice(file, &stubs); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal file %s: %v", path, string(file))
	}

	return stubs, nil
}

// isDirectory checks if the given path is a directory.
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}
