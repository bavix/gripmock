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

func TestNewProcessor(t *testing.T) {
	t.Parallel()

	// Test newProcessor function
	initialImports := []string{"/path1", "/path2"}
	processor := newProcessor(initialImports)

	require.NotNil(t, processor)
	require.Equal(t, initialImports, processor.imports)
	require.NotNil(t, processor.seenDirs)
	require.NotNil(t, processor.seenFiles)
	require.Equal(t, []string{ProtoExt}, processor.allowedProtoExts)
	require.Equal(t, []string{ProtobufSetExt, ProtoSetExt}, processor.allowedDescExts)
}

func TestProcessor_Result(t *testing.T) {
	t.Parallel()

	// Test processor.result() method
	processor := newProcessor([]string{"/import1"})
	processor.protos = []string{"file1.proto", "file2.proto"}
	processor.descriptors = []string{"file1.pb", "file2.protoset"}

	result := processor.result()

	require.NotNil(t, result)
	require.Equal(t, []string{"/import1"}, result.imports)
	require.Equal(t, []string{"file1.proto", "file2.proto"}, result.protos)
	require.Equal(t, []string{"file1.pb", "file2.protoset"}, result.descriptors)
}

func TestProcessor_AddImport(t *testing.T) {
	t.Parallel()

	// Test processor.addImport method
	processor := newProcessor([]string{})
	ctx := context.Background()

	// Test adding new import
	processor.addImport(ctx, "/new/path")
	require.Contains(t, processor.imports, "/new/path")
	require.True(t, processor.seenDirs["/new/path"])

	// Test adding duplicate import
	processor.addImport(ctx, "/new/path")
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
			require.Equal(t, tc.expectedImport, importPath)
			require.Equal(t, tc.expectedPath, relPath)
		})
	}
}

func TestProcessor_ProcessFile_ProtoFile(t *testing.T) {
	t.Parallel()

	// Test processing proto file
	processor := newProcessor([]string{})
	ctx := context.Background()

	// Create temporary proto file
	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "test.proto")
	err := os.WriteFile(protoFile, []byte("syntax = \"proto3\";"), 0o600)
	require.NoError(t, err)

	err = processor.processFile(ctx, protoFile)
	require.NoError(t, err)
	require.Contains(t, processor.protos, "test.proto")
}

func TestProcessor_ProcessFile_DescriptorFile(t *testing.T) {
	t.Parallel()

	// Test processing descriptor file
	processor := newProcessor([]string{})
	ctx := context.Background()

	// Create temporary descriptor file
	tempDir := t.TempDir()
	descFile := filepath.Join(tempDir, "test.pb")

	// Create a minimal FileDescriptorSet
	fds := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			{
				Name: proto.String("test.proto"),
			},
		},
	}

	descData, err := proto.Marshal(fds)
	require.NoError(t, err)

	err = os.WriteFile(descFile, descData, 0o600)
	require.NoError(t, err)

	err = processor.processFile(ctx, descFile)
	require.NoError(t, err)
	require.Contains(t, processor.descriptors, descFile)
}

func TestProcessor_ProcessFile_UnsupportedFile(t *testing.T) {
	t.Parallel()

	// Test processing unsupported file
	processor := newProcessor([]string{})
	ctx := context.Background()

	// Create temporary unsupported file
	tempDir := t.TempDir()
	unsupportedFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(unsupportedFile, []byte("not a proto file"), 0o600)
	require.NoError(t, err)

	err = processor.processFile(ctx, unsupportedFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported file type")
}

func TestProcessor_ProcessDirectory(t *testing.T) {
	t.Parallel()

	// Test processing directory
	processor := newProcessor([]string{})
	ctx := context.Background()

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
				Name: proto.String("test.proto"),
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

func TestProcessor_Process_WithContextCancellation(t *testing.T) {
	t.Parallel()

	// Test processing with context cancellation
	processor := newProcessor([]string{})
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	err := processor.process(ctx, []string{"/some/path"})
	require.Error(t, err)
	require.Equal(t, context.Canceled, err)
}

func TestBuild_WithValidPaths(t *testing.T) {
	t.Parallel()

	// Test Build with valid paths
	ctx := context.Background()

	// Create temporary directory with proto file
	tempDir := t.TempDir()
	protoFile := filepath.Join(tempDir, "test.proto")
	err := os.WriteFile(protoFile, []byte("syntax = \"proto3\";"), 0o600)
	require.NoError(t, err)

	results, err := Build(ctx, []string{tempDir}, []string{protoFile})
	require.NoError(t, err)
	require.NotNil(t, results)
	require.Len(t, results, 1)
}

func TestBuild_WithDuplicatePaths(t *testing.T) {
	t.Parallel()

	// Test Build with duplicate paths
	ctx := context.Background()

	// Create temporary directory
	tempDir := t.TempDir()

	// Test with duplicate imports
	results, err := Build(ctx, []string{tempDir, tempDir}, []string{})
	require.NoError(t, err)
	require.NotNil(t, results)

	// Test with duplicate paths
	protoFile := filepath.Join(tempDir, "test.proto")
	err = os.WriteFile(protoFile, []byte("syntax = \"proto3\";"), 0o600)
	require.NoError(t, err)

	results, err = Build(ctx, []string{tempDir}, []string{protoFile, protoFile})
	require.NoError(t, err)
	require.NotNil(t, results)
}
