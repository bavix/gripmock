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
	"github.com/rs/zerolog"
	"github.com/samber/lo"

	"github.com/bavix/gripmock/v3/internal/infra/jsondecoder"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
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

	if isDirectory(pathDir) {
		ch, err := s.watcher.Watch(ctx, pathDir)
		if err != nil {
			return
		}

		var wg sync.WaitGroup

		for file := range ch {
			zerolog.Ctx(ctx).
				Debug().
				Str("path", file).
				Msg("Updating stub")

			wg.Go(func() {
				defer func() {
					if r := recover(); r != nil {
						zerolog.Ctx(ctx).
							Error().
							Interface("panic", r).
							Str("file", file).
							Msg("Panic recovered while processing stub file")
					}
				}()

				s.readByFile(ctx, file)
			})
		}

		wg.Wait()
	}
}

func (s *Extender) readFromPath(ctx context.Context, pathDir string) {
	if !isDirectory(pathDir) {
		s.handleFilePath(ctx, pathDir)

		return
	}

	s.handleDirectoryPath(ctx, pathDir)
}

func (s *Extender) handleFilePath(ctx context.Context, filePath string) {
	if s.isStubFile(filePath) {
		s.readByFile(ctx, filePath)
	}
}

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

func (s *Extender) handleFileReadError(ctx context.Context, filePath string, err error) {
	zerolog.Ctx(ctx).
		Err(err).
		Str("file", filePath).
		Msg("failed to read file")

	if existingIDs, exists := s.mapIDsByFile[filePath]; exists {
		s.storage.DeleteByID(existingIDs...)
		delete(s.mapIDsByFile, filePath)
	}
}

func (s *Extender) handleFirstTimeLoad(filePath string, stubs []*stuber.Stub) {
	for _, stub := range stubs {
		if stub.ID == uuid.Nil {
			stub.ID = uuid.New()
		}
	}

	s.mapIDsByFile[filePath] = s.storage.PutMany(stubs...)
}

func (s *Extender) handleExistingFileUpdate(filePath string, stubs []*stuber.Stub, existingIDs uuid.UUIDs) {
	currentIDs := s.extractCurrentIDs(stubs)
	unusedIDs := lo.Without(existingIDs, currentIDs...)
	newIDs := s.generateNewIDs(stubs, unusedIDs)

	if removedIDs := lo.Without(existingIDs, newIDs...); len(removedIDs) > 0 {
		s.storage.DeleteByID(removedIDs...)
	}

	if len(stubs) > 0 {
		s.mapIDsByFile[filePath] = s.storage.PutMany(stubs...)
	} else {
		delete(s.mapIDsByFile, filePath)
	}
}

func (s *Extender) extractCurrentIDs(stubs []*stuber.Stub) uuid.UUIDs {
	currentIDs := make(uuid.UUIDs, 0, len(stubs))
	for _, stub := range stubs {
		if stub.ID != uuid.Nil {
			currentIDs = append(currentIDs, stub.ID)
		}
	}

	return currentIDs
}

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

func (s *Extender) checkUniqIDs(ctx context.Context, filePath string, stubs []*stuber.Stub) {
	if s.loaded.Load() {
		return
	}

	s.muUniqueIDs.Lock()
	defer s.muUniqueIDs.Unlock()

	for _, stub := range stubs {
		if stub.ID == uuid.Nil {
			continue
		}

		if _, exists := s.uniqueIDs[stub.ID]; exists {
			zerolog.Ctx(ctx).
				Warn().
				Str("file", filePath).
				Str("id", stub.ID.String()).
				Msg("duplicate stub ID")
		}

		s.uniqueIDs[stub.ID] = struct{}{}
	}
}

func genID(stub *stuber.Stub, freeIDs uuid.UUIDs) (uuid.UUID, uuid.UUIDs) {
	if stub.ID != uuid.Nil {
		return stub.ID, freeIDs
	}

	if len(freeIDs) > 0 {
		return freeIDs[0], freeIDs[1:]
	}

	return uuid.New(), nil
}

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

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}
