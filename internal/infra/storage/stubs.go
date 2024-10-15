package storage

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/gripmock/stuber"
	"github.com/rs/zerolog"

	"github.com/bavix/gripmock/internal/infra/watcher"
	"github.com/bavix/gripmock/pkg/jsondecoder"
	"github.com/bavix/gripmock/pkg/yaml2json"
)

type Extender struct {
	storage   *stuber.Budgerigar
	convertor *yaml2json.Convertor
	ch        chan struct{}
	watcher   *watcher.StubWatcher
}

func NewStub(
	storage *stuber.Budgerigar,
	convertor *yaml2json.Convertor,
	watcher *watcher.StubWatcher,
) *Extender {
	return &Extender{
		storage:   storage,
		convertor: convertor,
		ch:        make(chan struct{}),
		watcher:   watcher,
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
		s.readByFile(ctx, file, true)
	}
}

// readFromPath reads all the stubs from the given directory and its subdirectories,
// and adds them to the server's stub store.
// The stub files can be in yaml or json format.
// If a file is in yaml format, it will be converted to json format.
//
// If `update` is true, the stubs will be updated in the server's stub store.
// Otherwise, the stubs will be added to the server's stub store.
func (s *Extender) readFromPath(ctx context.Context, pathDir string) {
	files, err := os.ReadDir(pathDir)
	if err != nil {
		zerolog.Ctx(ctx).
			Err(err).Str("path", pathDir).
			Msg("read directory")

		return
	}

	for _, file := range files {
		// If the file is a directory, recursively read its stubs.
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

		s.readByFile(ctx, path.Join(pathDir, file.Name()), false)
	}
}

func (s *Extender) readByFile(ctx context.Context, currentFile string, update bool) {
	// Read the stub file and add it to the server's stub store.
	stubs, err := s.readStub(currentFile)
	if err != nil {
		zerolog.Ctx(ctx).
			Err(err).
			Str("file", currentFile).
			Msg("read file")

		return
	}

	// Update or add the stubs to the server's stub store.
	if update {
		s.storage.UpdateMany(stubs...)
	} else {
		s.storage.PutMany(stubs...)
	}
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
