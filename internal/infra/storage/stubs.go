package storage

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/gripmock/stuber"
	"github.com/rs/zerolog"
	"github.com/samber/lo"

	"github.com/bavix/gripmock/internal/infra/watcher"
	"github.com/bavix/gripmock/pkg/jsondecoder"
	"github.com/bavix/gripmock/pkg/yaml2json"
)

type Extender struct {
	storage      *stuber.Budgerigar
	convertor    *yaml2json.Convertor
	ch           chan struct{}
	watcher      *watcher.StubWatcher
	mapIDsByFile map[string]uuid.UUIDs
}

func NewStub(
	storage *stuber.Budgerigar,
	convertor *yaml2json.Convertor,
	watcher *watcher.StubWatcher,
) *Extender {
	return &Extender{
		storage:      storage,
		convertor:    convertor,
		ch:           make(chan struct{}),
		watcher:      watcher,
		mapIDsByFile: make(map[string]uuid.UUIDs),
	}
}

func (s *Extender) Wait() {
	<-s.ch
}

func (s *Extender) ReadFromPath(ctx context.Context, pathDir string) {
	if pathDir == "" {
		close(s.ch)

		return
	}

	s.readFromPath(ctx, pathDir)
	close(s.ch)

	ch, err := s.watcher.Watch(ctx, pathDir)
	if err != nil {
		return
	}

	for file := range ch {
		s.readByFile(ctx, file)
	}
}

// readFromPath reads all the stubs from the given directory and its subdirectories,
// and adds them to the server's stub store.
// The stub files can be in yaml or json format.
// If a file is in yaml format, it will be converted to json format.
func (s *Extender) readFromPath(ctx context.Context, pathDir string) {
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

		// If the file is not a stub file, skip it.
		if !strings.HasSuffix(file.Name(), ".json") &&
			!strings.HasSuffix(file.Name(), ".yaml") &&
			!strings.HasSuffix(file.Name(), ".yml") {
			continue
		}

		s.readByFile(ctx, path.Join(pathDir, file.Name()))
	}
}

func (s *Extender) readByFile(ctx context.Context, currentFile string) {
	// Read the stub file and add it to the server's stub store.
	stubs, err := s.readStub(currentFile)
	if err != nil {
		zerolog.Ctx(ctx).
			Err(err).
			Str("file", currentFile).
			Msg("read file")

		return
	}

	// Inserts or updates the stubs in the server's stub store.
	oldIDs, ok := s.mapIDsByFile[currentFile]
	if !ok {
		s.mapIDsByFile[currentFile] = s.storage.PutMany(stubs...)

		return
	}

	// Get the IDs of the stubs that are already in the file.
	fillIDs := make(uuid.UUIDs, 0, len(stubs))
	for _, stub := range stubs {
		if stub.ID != uuid.Nil {
			fillIDs = append(fillIDs, stub.ID)
		}
	}

	// Get the IDs of the stubs that are not in the file.
	freeIDs := lo.Without(oldIDs, fillIDs...)

	// Generate new IDs for the stubs that are not in the file.
	newIDs := make(uuid.UUIDs, 0, len(stubs))
	for _, stub := range stubs {
		if stub.ID == uuid.Nil {
			stub.ID, freeIDs = genID(stub, freeIDs)
		}

		newIDs = append(newIDs, stub.ID)
	}

	// Delete the stubs that have been removed from the file.
	if deletes := lo.Intersect(oldIDs, newIDs); len(deletes) > 0 {
		s.storage.DeleteByID(deletes...)
	}

	// Upsert the stubs that have been added to the file.
	if len(stubs) > 0 {
		s.mapIDsByFile[currentFile] = s.storage.PutMany(stubs...)
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
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		file, err = s.convertor.Execute(path, file)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal file %s: %w", path, err)
		}
	}

	var stubs []*stuber.Stub
	if err := jsondecoder.UnmarshalSlice(file, &stubs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file %s: %v %w", path, string(file), err)
	}

	return stubs, nil
}
