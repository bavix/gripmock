package storage

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/samber/lo"
	"google.golang.org/grpc/codes"

	"github.com/bavix/gripmock/v3/internal/domain/types"
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
		zerolog.Ctx(ctx).Info().Msg("Stub loading completed")
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
	stubs, err := s.readStubWithContext(ctx, filePath)
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
// This variant does not perform deprecation logging because no context is provided.
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

// readStubWithContext performs stub reading with context-aware logging.
func (s *Extender) readStubWithContext(ctx context.Context, path string) ([]*stuber.Stub, error) {
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

	// Parse as raw data first to detect format and handle v4 inputs properly
	var rawList []map[string]any
	if err := jsondecoder.UnmarshalSlice(file, &rawList); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal file %s: %v", path, string(file))
	}

	stubs := make([]*stuber.Stub, 0, len(rawList))

	for _, rawStub := range rawList {
		stub := &stuber.Stub{}

		// Check if this is v4 format (has outputs field)
		if _, hasOutputs := rawStub["outputs"]; hasOutputs {
			// Handle v4 format
			if err := s.unmarshalV4Stub(rawStub, stub); err != nil {
				return nil, errors.Wrapf(err, "failed to unmarshal v4 stub from %s", path)
			}
		} else {
			// Handle legacy format
			zerolog.Ctx(ctx).
				Warn().
				Str("file", path).
				Msg("[DEPRECATED] legacy stub format detected; please migrate to v4 outputs")

			if err := s.unmarshalLegacyStub(rawStub, stub); err != nil {
				return nil, errors.Wrapf(err, "failed to unmarshal legacy stub from %s", path)
			}
		}

		stubs = append(stubs, stub)
	}

	return stubs, nil
}

// unmarshalV4Stub unmarshals a v4 format stub from raw data.
//

//nolint:gocognit,gocyclo,cyclop,funlen,maintidx
func (s *Extender) unmarshalV4Stub(rawStub map[string]any, stub *stuber.Stub) error {
	// Convert raw data to JSON for unmarshaling
	data, err := json.Marshal(rawStub)
	if err != nil {
		return err
	}

	// Unmarshal basic fields
	if err := json.Unmarshal(data, stub); err != nil {
		return err
	}

	// Handle v4 inputs separately

	//nolint:nestif
	if rawInputs, ok := rawStub["inputs"]; ok {
		if inputsSlice, ok := rawInputs.([]any); ok {
			for _, rawInput := range inputsSlice {
				if inputMap, ok := rawInput.(map[string]any); ok {
					matcher := types.Matcher{}

					inputBytes, mErr := json.Marshal(inputMap)
					if mErr == nil {
						if err := json.Unmarshal(inputBytes, &matcher); err == nil {
							stub.InputsV4 = append(stub.InputsV4, matcher)

							continue
						}
					}
					// Fallback: try to handle any matcher manually
					if anyData, hasAny := inputMap["any"]; hasAny {
						if anySlice, ok := anyData.([]any); ok {
							for _, anyItem := range anySlice {
								if anyMap, ok := anyItem.(map[string]any); ok {
									anyMatcher := types.Matcher{}

									anyBytes, aErr := json.Marshal(anyMap)
									if aErr == nil {
										if err := json.Unmarshal(anyBytes, &anyMatcher); err == nil {
											matcher.Any = append(matcher.Any, anyMatcher)
										}
									}
								}
							}

							stub.InputsV4 = append(stub.InputsV4, matcher)
						}
					}
				}
			}
		}
	}

	//nolint:nestif
	if len(stub.InputsV4) > 0 {
		if len(stub.InputsV4) == 1 {
			// Single input - populate Input
			v4Matcher := stub.InputsV4[0]
			stub.Input = stuber.InputData{
				Equals:           v4Matcher.Equals,
				Contains:         v4Matcher.Contains,
				Matches:          make(map[string]any),
				IgnoreArrayOrder: v4Matcher.IgnoreArrayOrder,
			}
			// Convert Matches from map[string]string to map[string]any
			for k, v := range v4Matcher.Matches {
				stub.Input.Matches[k] = v
			}
			// Handle Any matcher (OR logic)
			if len(v4Matcher.Any) > 0 {
				// Convert v4 Any matcher to legacy Any matcher
				stub.Input.Any = make([]stuber.InputData, len(v4Matcher.Any))
				for i, anyMatcher := range v4Matcher.Any {
					stub.Input.Any[i] = stuber.InputData{
						Equals:           anyMatcher.Equals,
						Contains:         anyMatcher.Contains,
						Matches:          make(map[string]any),
						IgnoreArrayOrder: anyMatcher.IgnoreArrayOrder,
					}
					// Convert Matches from map[string]string to map[string]any
					for k, v := range anyMatcher.Matches {
						stub.Input.Any[i].Matches[k] = v
					}
				}
			}
		} else {
			// Multiple inputs - populate Inputs
			stub.Inputs = make([]stuber.InputData, len(stub.InputsV4))
			for i, v4Matcher := range stub.InputsV4 {
				stub.Inputs[i] = stuber.InputData{
					Equals:           v4Matcher.Equals,
					Contains:         v4Matcher.Contains,
					Matches:          make(map[string]any),
					IgnoreArrayOrder: v4Matcher.IgnoreArrayOrder,
				}
				// Convert Matches from map[string]string to map[string]any
				for k, v := range v4Matcher.Matches {
					stub.Inputs[i].Matches[k] = v
				}
				// Handle Any matcher (OR logic)
				if len(v4Matcher.Any) > 0 {
					// Convert v4 Any matcher to legacy Any matcher
					stub.Inputs[i].Any = make([]stuber.InputData, len(v4Matcher.Any))
					for j, anyMatcher := range v4Matcher.Any {
						stub.Inputs[i].Any[j] = stuber.InputData{
							Equals:           anyMatcher.Equals,
							Contains:         anyMatcher.Contains,
							Matches:          make(map[string]any),
							IgnoreArrayOrder: anyMatcher.IgnoreArrayOrder,
						}
						// Convert Matches from map[string]string to map[string]any
						for k, v := range anyMatcher.Matches {
							stub.Inputs[i].Any[j].Matches[k] = v
						}
					}
				}
			}
		}
	}

	// Handle v4 outputs separately
	//nolint:nestif
	if rawOutputs, ok := rawStub["outputs"]; ok {
		if outputsSlice, ok := rawOutputs.([]any); ok {
			for _, rawOutput := range outputsSlice {
				if outputMap, ok := rawOutput.(map[string]any); ok {
					stub.OutputsRawV4 = append(stub.OutputsRawV4, outputMap)

					// Also populate legacy Output for backward compatibility
					if streamData, hasStream := outputMap["stream"]; hasStream {
						if streamSlice, ok := streamData.([]any); ok {
							stub.Output.Stream = streamSlice
						}
					}

					if dataValue, hasData := outputMap["data"]; hasData {
						if dataMap, ok := dataValue.(map[string]any); ok {
							stub.Output.Data = dataMap
						}
					}

					// Map v4 status to legacy error/code for gRPC execution path
					if stRaw, hasStatus := outputMap["status"]; hasStatus {
						if stMap, ok := stRaw.(map[string]any); ok {
							var codeName string
							if v, ok := stMap["code"].(string); ok {
								codeName = v
							}

							var msg string
							if v, ok := stMap["message"].(string); ok {
								msg = v
							}

							if codeName != "" {
								if c, ok := parseGrpcCode(codeName); ok {
									if c != codes.OK {
										stub.Output.Code = &c
										stub.Output.Error = msg
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Also populate legacy fields for backward compatibility with bidirectional streaming
	if len(stub.InputsV4) > 0 {
		if len(stub.InputsV4) == 1 {
			// Single input - populate Input
			v4Matcher := stub.InputsV4[0]
			stub.Input = stuber.InputData{
				Equals:           v4Matcher.Equals,
				Contains:         v4Matcher.Contains,
				Matches:          make(map[string]any),
				IgnoreArrayOrder: v4Matcher.IgnoreArrayOrder,
			}
			// Convert Matches from map[string]string to map[string]any
			for k, v := range v4Matcher.Matches {
				stub.Input.Matches[k] = v
			}
		} else {
			// Multiple inputs - populate Inputs
			stub.Inputs = make([]stuber.InputData, len(stub.InputsV4))
			for i, v4Matcher := range stub.InputsV4 {
				stub.Inputs[i] = stuber.InputData{
					Equals:           v4Matcher.Equals,
					Contains:         v4Matcher.Contains,
					Matches:          make(map[string]any),
					IgnoreArrayOrder: v4Matcher.IgnoreArrayOrder,
				}
				// Convert Matches from map[string]string to map[string]any
				for k, v := range v4Matcher.Matches {
					stub.Inputs[i].Matches[k] = v
				}
			}
		}
	}

	// Merge top-level v4 responseHeaders into legacy Output.Headers for execution path
	// so that gRPC handlers can send them using existing header logic.
	if len(stub.ResponseHeaders) > 0 {
		if stub.Output.Headers == nil {
			stub.Output.Headers = make(map[string]string, len(stub.ResponseHeaders))
		}

		for k, v := range stub.ResponseHeaders {
			stub.Output.Headers[k] = v
		}
	}

	// Note: Stub type determination is now done at the service layer
	// where we have access to MethodRegistry to get actual gRPC method characteristics

	return nil
}

// parseGrpcCode converts a string gRPC status code name into codes.Code.
// Accepts both numeric strings and symbolic names.
//

//nolint:gocyclo,cyclop,funlen
func parseGrpcCode(name string) (codes.Code, bool) {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "OK":
		return codes.OK, true
	case "CANCELLED":
		return codes.Canceled, true
	case "UNKNOWN":
		return codes.Unknown, true
	case "INVALID_ARGUMENT":
		return codes.InvalidArgument, true
	case "DEADLINE_EXCEEDED":
		return codes.DeadlineExceeded, true
	case "NOT_FOUND":
		return codes.NotFound, true
	case "ALREADY_EXISTS":
		return codes.AlreadyExists, true
	case "PERMISSION_DENIED":
		return codes.PermissionDenied, true
	case "RESOURCE_EXHAUSTED":
		return codes.ResourceExhausted, true
	case "FAILED_PRECONDITION":
		return codes.FailedPrecondition, true
	case "ABORTED":
		return codes.Aborted, true
	case "OUT_OF_RANGE":
		return codes.OutOfRange, true
	case "UNIMPLEMENTED":
		return codes.Unimplemented, true
	case "INTERNAL":
		return codes.Internal, true
	case "UNAVAILABLE":
		return codes.Unavailable, true
	case "DATA_LOSS":
		return codes.DataLoss, true
	case "UNAUTHENTICATED":
		return codes.Unauthenticated, true
	default:
		// Try to parse numeric code
		switch name {
		case "0":
			return codes.OK, true
		case "1":
			return codes.Canceled, true
		case "2":
			return codes.Unknown, true
		case "3":
			return codes.InvalidArgument, true
		case "4":
			return codes.DeadlineExceeded, true
		case "5":
			return codes.NotFound, true
		case "6":
			return codes.AlreadyExists, true
		case "7":
			return codes.PermissionDenied, true
		case "8":
			return codes.ResourceExhausted, true
		case "9":
			return codes.FailedPrecondition, true
		case "10":
			return codes.Aborted, true
		case "11":
			return codes.OutOfRange, true
		case "12":
			return codes.Unimplemented, true
		case "13":
			return codes.Internal, true
		case "14":
			return codes.Unavailable, true
		case "15":
			return codes.DataLoss, true
		case "16":
			return codes.Unauthenticated, true
		}
	}

	return codes.OK, false
}

// unmarshalLegacyStub unmarshals a legacy format stub from raw data.
func (s *Extender) unmarshalLegacyStub(rawStub map[string]any, stub *stuber.Stub) error {
	// Convert raw data to JSON for unmarshaling
	data, err := json.Marshal(rawStub)
	if err != nil {
		return err
	}

	// Unmarshal using standard JSON unmarshaling for legacy format
	return json.Unmarshal(data, stub)
}

// isDirectory checks if the given path is a directory.
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}
