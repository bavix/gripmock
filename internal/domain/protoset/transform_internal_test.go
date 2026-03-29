package protoset

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

type mockRemoteClient struct {
	fn func(source *Source) *descriptorpb.FileDescriptorSet
}

func (m *mockRemoteClient) FetchDescriptorSet(
	ctx context.Context,
	source *Source,
) (*descriptorpb.FileDescriptorSet, error) {
	return m.fn(source), nil
}

func TestNewProcessor(t *testing.T) {
	t.Parallel()

	// Test newProcessor function
	initialImports := []string{"/path1", "/path2"}
	processor := newProcessor(initialImports, nil)

	require.NotNil(t, processor)
	require.Equal(t, initialImports, processor.imports)
	require.NotNil(t, processor.seenDirs)
	require.NotNil(t, processor.seenFiles)
	require.Equal(t, []string{ProtoExt}, processor.allowedProtoExts)
	require.Equal(t, []string{ProtobufSetExt, ProtoSetExt}, processor.allowedDescExts)
}

func TestProcessorResult(t *testing.T) {
	t.Parallel()

	// Test processor.result() method
	processor := newProcessor([]string{"/import1"}, nil)
	processor.protos = []string{"file1.proto", "file2.proto"}
	processor.descriptors = []string{"file1.pb", "file2.protoset"}

	result := processor.result()

	require.NotNil(t, result)
	require.Equal(t, []string{"/import1"}, result.imports)
	require.Equal(t, []string{"file1.proto", "file2.proto"}, result.protos)
	require.Equal(t, []string{"file1.pb", "file2.protoset"}, result.descriptors)
}

func TestProcessorAddImport(t *testing.T) {
	t.Parallel()

	// Test processor.addImport method
	processor := newProcessor([]string{}, nil)
	ctx := t.Context()

	// Test adding new import
	importPath := "/new/path"
	processor.addImport(ctx, importPath)

	absPath, err := filepath.Abs(importPath)
	require.NoError(t, err)
	require.Contains(t, processor.imports, absPath)
	require.True(t, processor.seenDirs[absPath])

	// Test adding duplicate import
	processor.addImport(ctx, importPath)
	require.Len(t, processor.imports, 1) // Should not add duplicate
}

func TestFindPathByImports(t *testing.T) {
	t.Parallel()

	// Test findPathByImports function
	testCases := []struct {
		name           string
		filePath       string
		imports        []string
		expectedImport string
		expectedPath   string
	}{
		{
			name:           "File in import path",
			filePath:       "/import1/path/to/file.proto",
			imports:        []string{"/import1", "/import2"},
			expectedImport: "/import1",
			expectedPath:   "path/to/file.proto",
		},
		{
			name:           "File not in any import path",
			filePath:       "/other/path/file.proto",
			imports:        []string{"/import1", "/import2"},
			expectedImport: "",
			expectedPath:   "file.proto",
		},
		{
			name:           "Empty imports",
			filePath:       "/path/to/file.proto",
			imports:        []string{},
			expectedImport: "",
			expectedPath:   "file.proto",
		},
		{
			name:           "File in longest import path",
			filePath:       "/import1/subdir/file.proto",
			imports:        []string{"/import1", "/import1/subdir"},
			expectedImport: "/import1/subdir",
			expectedPath:   "file.proto",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			importPath, relPath := findPathByImports(tc.filePath, tc.imports)
			require.Equal(t, filepath.FromSlash(tc.expectedImport), importPath)
			require.Equal(t, filepath.FromSlash(tc.expectedPath), relPath)
		})
	}
}

func TestProcessorPRocessFileProtoFile(t *testing.T) {
	t.Parallel()

	// Test processing proto file
	processor := newProcessor([]string{}, nil)
	ctx := t.Context()

	// Create temporary proto file
	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "test.proto")
	err := os.WriteFile(protoFile, []byte("syntax = \"proto3\";"), 0o600)
	require.NoError(t, err)

	source, err := ParseSource(protoFile)
	require.NoError(t, err)

	err = ProcessSource(ctx, source, processor)
	require.NoError(t, err)
	require.Contains(t, processor.protos, "test.proto")
}

func TestProcessorPRocessFileDescriptorFile(t *testing.T) {
	t.Parallel()

	// Test processing descriptor file
	processor := newProcessor([]string{}, nil)
	ctx := t.Context()

	// Create temporary descriptor file
	tempDir := t.TempDir()
	descFile := filepath.Join(tempDir, "test.pb")

	// Create a minimal FileDescriptorSet
	fds := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			{
				Name: new("test.proto"),
			},
		},
	}

	descData, err := proto.Marshal(fds)
	require.NoError(t, err)

	err = os.WriteFile(descFile, descData, 0o600)
	require.NoError(t, err)

	source, err := ParseSource(descFile)
	require.NoError(t, err)

	err = ProcessSource(ctx, source, processor)
	require.NoError(t, err)
	require.Contains(t, processor.descriptors, descFile)
}

func TestProcessorPRocessFileUnsupportedFile(t *testing.T) {
	t.Parallel()

	// Test processing unsupported file
	processor := newProcessor([]string{}, nil)
	ctx := t.Context()

	// Create temporary unsupported file
	tempDir := t.TempDir()
	unsupportedFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(unsupportedFile, []byte("not a proto file"), 0o600)
	require.NoError(t, err)

	source, err := ParseSource(unsupportedFile)
	require.NoError(t, err)

	err = ProcessSource(ctx, source, processor)
	require.NoError(t, err)
	require.Contains(t, processor.protos, "test.txt")
}

func TestProcessorProcessDirectory(t *testing.T) {
	t.Parallel()

	// Test processing directory
	processor := newProcessor([]string{}, nil)
	ctx := t.Context()

	// Create temporary directory with mixed files
	tempDir := t.TempDir()

	// Create proto file
	protoFile := filepath.Join(tempDir, "test.proto")
	err := os.WriteFile(protoFile, []byte("syntax = \"proto3\";"), 0o600)
	require.NoError(t, err)

	// Create descriptor file
	descFile := filepath.Join(tempDir, "test.pb")
	fds := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			{
				Name: new("test.proto"),
			},
		},
	}
	descData, err := proto.Marshal(fds)
	require.NoError(t, err)
	err = os.WriteFile(descFile, descData, 0o600)
	require.NoError(t, err)

	// Create unsupported file
	unsupportedFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(unsupportedFile, []byte("not a proto file"), 0o600)
	require.NoError(t, err)

	err = processor.processDirectory(ctx, tempDir)
	require.NoError(t, err)

	// Should have added the directory as import
	require.Contains(t, processor.imports, tempDir)

	// Should have processed proto and descriptor files
	require.Contains(t, processor.protos, "test.proto")
	require.Contains(t, processor.descriptors, descFile)

	// Should not have processed unsupported file
	require.NotContains(t, processor.protos, "test.txt")
	require.NotContains(t, processor.descriptors, unsupportedFile)
}

func TestProcessorProcessReflectNamespacesDuplicateDescriptorFiles(t *testing.T) {
	t.Parallel()

	processor := newProcessor([]string{}, &mockRemoteClient{fn: func(source *Source) *descriptorpb.FileDescriptorSet {
		name := "service.proto"

		return &descriptorpb.FileDescriptorSet{
			File: []*descriptorpb.FileDescriptorProto{{Name: &name}},
		}
	}})

	err := processor.ProcessReflect(t.Context(), &Source{Type: SourceReflect, Raw: "grpc://localhost:4444"})
	require.NoError(t, err)
	require.Len(t, processor.descriptorSets, 1)

	err = processor.ProcessReflect(t.Context(), &Source{Type: SourceReflect, Raw: "grpc://localhost:5555"})
	require.NoError(t, err)
	require.Len(t, processor.descriptorSets, 2)
	require.Equal(t,
		processor.descriptorSets[0].GetFile()[0].GetName(),
		processor.descriptorSets[1].GetFile()[0].GetName(),
	)
}

func TestProcessorPRocessWithContextCancellation(t *testing.T) {
	t.Parallel()

	// Test processing with context cancellation
	processor := newProcessor([]string{}, nil)
	ctx, cancel := context.WithCancel(t.Context())

	// Cancel context immediately
	cancel()

	err := processor.process(ctx, []string{"/some/path"})
	require.Error(t, err)
	require.Equal(t, context.Canceled, err)
}

func TestBuildWithValidPaths(t *testing.T) {
	t.Parallel()

	// Test Build with valid paths
	ctx := t.Context()

	// Create temporary directory with proto file
	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "test.proto")
	err := os.WriteFile(protoFile, []byte("syntax = \"proto3\";"), 0o600)
	require.NoError(t, err)

	results, err := Build(ctx, []string{tempDir}, []string{protoFile}, nil)
	require.NoError(t, err)
	require.NotNil(t, results)
	require.Len(t, results, 1)
}

func TestBuildWithDuplicatePaths(t *testing.T) {
	t.Parallel()

	// Test Build with duplicate paths
	ctx := t.Context()

	// Create temporary directory
	tempDir := t.TempDir()

	// Test with duplicate imports
	results, err := Build(ctx, []string{tempDir, tempDir}, []string{}, nil)
	require.NoError(t, err)
	require.NotNil(t, results)

	// Test with duplicate paths
	protoFile := filepath.Join(tempDir, "test.proto")
	err = os.WriteFile(protoFile, []byte("syntax = \"proto3\";"), 0o600)
	require.NoError(t, err)

	results, err = Build(ctx, []string{tempDir}, []string{protoFile, protoFile}, nil)
	require.NoError(t, err)
	require.NotNil(t, results)
}

func TestConfigureGetters(t *testing.T) {
	t.Parallel()

	processor := newProcessor([]string{"/import1"}, nil)
	processor.protos = []string{"a.proto", "b.proto"}
	processor.descriptors = []string{"/path/to/file.pb"}

	cfg := processor.result()
	require.NotNil(t, cfg)
	require.Equal(t, []string{"/import1"}, cfg.Imports())
	require.Equal(t, []string{"a.proto", "b.proto"}, cfg.Protos())
	require.Equal(t, []string{"/path/to/file.pb"}, cfg.Descriptors())
}

func TestFindMinimalPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		paths    []string
		expected []string
	}{
		{
			name:     "empty",
			paths:    []string{},
			expected: nil,
		},
		{
			name:     "single path",
			paths:    []string{"/a"},
			expected: []string{"/a"},
		},
		{
			name:     "parent and child - keeps parent only",
			paths:    []string{"/a/b/c", "/a"},
			expected: []string{"/a"},
		},
		{
			name:     "sibling paths - keeps both",
			paths:    []string{"/a", "/b"},
			expected: []string{"/a", "/b"},
		},
		{
			name:     "nested - keeps root",
			paths:    []string{"/a/b", "/a/b/c", "/a"},
			expected: []string{"/a"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := findMinimalPaths(tc.paths)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestBuildWithNonExistentPath(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	_, err := Build(ctx, []string{}, []string{"/non/existent/path.proto"}, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to compile descriptors")
}

func TestBuildWithNonExistentImportPath(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "test.proto")
	require.NoError(t, os.WriteFile(protoFile, []byte("syntax = \"proto3\";"), 0o600))

	_, err := Build(ctx, []string{"/non/existent/import"}, []string{protoFile}, nil)
	require.NoError(t, err) // imports can be non-existent if we only use descriptors
}

func TestBuildWithDirectoryPath(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "test.proto")
	require.NoError(t, os.WriteFile(protoFile, []byte("syntax = \"proto3\";"), 0o600))

	results, err := Build(ctx, []string{tempDir}, []string{tempDir}, nil)
	require.NoError(t, err)
	require.NotNil(t, results)
	require.Len(t, results, 1)
}

func TestBuildWithDescriptorOnly(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	tempDir := t.TempDir()
	descFile := filepath.Join(tempDir, "test.pb")
	fds := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			{Name: new("test_descriptor_only.proto")},
		},
	}
	descData, err := proto.Marshal(fds)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(descFile, descData, 0o600))

	results, err := Build(ctx, []string{}, []string{descFile}, nil)
	require.NoError(t, err)
	require.NotNil(t, results)
	require.Len(t, results, 1)
}

func TestBuildWithProtosetFile(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	tempDir := t.TempDir()
	protosetFile := filepath.Join(tempDir, "test.protoset")
	fds := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			{Name: new("test_protoset_file.proto")},
		},
	}
	descData, err := proto.Marshal(fds)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(protosetFile, descData, 0o600))

	results, err := Build(ctx, []string{}, []string{protosetFile}, nil)
	require.NoError(t, err)
	require.NotNil(t, results)
	require.Len(t, results, 1)
}

func TestProcessorPRocessNonExistentPath(t *testing.T) {
	t.Parallel()

	processor := newProcessor([]string{}, nil)
	ctx := t.Context()

	err := processor.process(ctx, []string{"/non/existent/path"})
	require.NoError(t, err)
	require.Contains(t, processor.protos, "path")
}

func TestProcessorPRocessDirectoryContextCancellation(t *testing.T) {
	t.Parallel()

	processor := newProcessor([]string{}, nil)
	ctx, cancel := context.WithCancel(t.Context())
	tempDir := t.TempDir()

	// Create a proto file so the directory has content
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "test.proto"), []byte("syntax = \"proto3\";"), 0o600))

	cancel()

	err := processor.process(ctx, []string{tempDir})
	require.Error(t, err)
	require.Equal(t, context.Canceled, err)
}

func TestProcessorAddProtoFile(t *testing.T) {
	t.Parallel()

	processor := newProcessor([]string{}, nil)
	ctx := t.Context()
	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "test.proto")
	require.NoError(t, os.WriteFile(protoFile, []byte("syntax = \"proto3\";"), 0o600))

	processor.addImport(ctx, tempDir)
	processor.AddProtoFile(ctx, protoFile)

	require.Contains(t, processor.protos, "test.proto")
}

func TestProcessorAddDescriptorFile(t *testing.T) {
	t.Parallel()

	processor := newProcessor([]string{}, nil)
	ctx := t.Context()
	tempDir := t.TempDir()
	descFile := filepath.Join(tempDir, "test.pb")
	fds := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{{Name: new("x.proto")}},
	}
	data, _ := proto.Marshal(fds)
	require.NoError(t, os.WriteFile(descFile, data, 0o600))

	processor.addImport(ctx, tempDir)
	processor.AddDescriptorFile(ctx, descFile)

	absPath, err := filepath.Abs(descFile)
	require.NoError(t, err)
	require.Contains(t, processor.descriptors, absPath)
}

func TestConstants(t *testing.T) {
	t.Parallel()

	require.Equal(t, ".proto", ProtoExt)
	require.Equal(t, ".pb", ProtobufSetExt)
	require.Equal(t, ".protoset", ProtoSetExt)
}
