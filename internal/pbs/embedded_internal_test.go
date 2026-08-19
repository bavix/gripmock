package pbs

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestNewResolver(t *testing.T) {
	t.Parallel()

	resolver, err := NewResolver()
	require.NoError(t, err)
	require.NotNil(t, resolver)

	wkt, err := resolver.FindFileByPath("google/protobuf/timestamp.proto")
	require.NoError(t, err)
	require.NotNil(t, wkt.Proto)

	lazy, err := resolver.FindFileByPath("google/api/annotations.proto")
	require.NoError(t, err)
	require.NotNil(t, lazy.Proto)
}

func TestNewResolverEmbeddedData(t *testing.T) {
	t.Parallel()

	require.NotEmpty(t, googleapis)
	require.NotEmpty(t, protobuf)
}

func TestThirdPartyResolverFIndFileByPathExistingFile(t *testing.T) {
	t.Parallel()

	resolver, err := NewResolver()
	require.NoError(t, err)

	result, err := resolver.FindFileByPath("google/protobuf/descriptor.proto")
	if err == nil {
		require.NotNil(t, result)
		require.NotNil(t, result.Proto)
	} else {
		require.Equal(t, protoregistry.NotFound, err)
	}
}

func TestThirdPartyResolverFIndFileByPathNonExistentFile(t *testing.T) {
	t.Parallel()

	resolver, err := NewResolver()
	require.NoError(t, err)

	result, err := resolver.FindFileByPath("non/existent/file.proto")
	require.Error(t, err)
	require.Equal(t, protoregistry.NotFound, err)
	require.Empty(t, result)
}

func TestThirdPartyResolverFIndFileByPathEmptyPath(t *testing.T) {
	t.Parallel()

	resolver, err := NewResolver()
	require.NoError(t, err)

	result, err := resolver.FindFileByPath("")
	require.Error(t, err)
	require.Equal(t, protoregistry.NotFound, err)
	require.Empty(t, result)
}

func TestThirdPartyResolverFIndFileByPathNilResolver(t *testing.T) {
	t.Parallel()

	var resolver *ThirdPartyResolver = nil
	if resolver != nil {
		result, err := resolver.FindFileByPath("test.proto")
		require.Error(t, err)
		require.Empty(t, result)
	}
}

func TestThirdPartyResolverStruct(t *testing.T) {
	t.Parallel()

	resolver := &ThirdPartyResolver{
		items: []*descriptorpb.FileDescriptorSet{},
	}

	require.NotNil(t, resolver)
	require.NotNil(t, resolver.items)
	require.Empty(t, resolver.items)
}

func TestThirdPartyResolverWithEmptyItems(t *testing.T) {
	t.Parallel()

	resolver := &ThirdPartyResolver{
		items: []*descriptorpb.FileDescriptorSet{},
	}

	result, err := resolver.FindFileByPath("test.proto")
	require.Error(t, err)
	require.Equal(t, protoregistry.NotFound, err)
	require.Empty(t, result)
}

func TestThirdPartyResolverWithNilItems(t *testing.T) {
	t.Parallel()

	resolver := &ThirdPartyResolver{
		items: nil,
	}

	result, err := resolver.FindFileByPath("test.proto")
	require.Error(t, err)
	require.Equal(t, protoregistry.NotFound, err)
	require.Empty(t, result)
}

func TestThirdPartyResolverWithSingleItem(t *testing.T) {
	t.Parallel()

	resolver := &ThirdPartyResolver{
		items: []*descriptorpb.FileDescriptorSet{
			{
				File: []*descriptorpb.FileDescriptorProto{
					{
						Name: new("test.proto"),
					},
				},
			},
		},
	}

	result, err := resolver.FindFileByPath("test.proto")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Proto)
	require.Equal(t, "test.proto", result.Proto.GetName())
}

func TestThirdPartyResolverWithMultipleItems(t *testing.T) {
	t.Parallel()

	resolver := &ThirdPartyResolver{
		items: []*descriptorpb.FileDescriptorSet{
			{
				File: []*descriptorpb.FileDescriptorProto{
					{
						Name: new("first.proto"),
					},
				},
			},
			{
				File: []*descriptorpb.FileDescriptorProto{
					{
						Name: new("second.proto"),
					},
				},
			},
		},
	}

	result, err := resolver.FindFileByPath("first.proto")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "first.proto", result.Proto.GetName())

	result, err = resolver.FindFileByPath("second.proto")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "second.proto", result.Proto.GetName())

	_, err = resolver.FindFileByPath("third.proto")
	require.Error(t, err)
	require.Equal(t, protoregistry.NotFound, err)
}

func TestThirdPartyResolverWithEmptyFileList(t *testing.T) {
	t.Parallel()

	resolver := &ThirdPartyResolver{
		items: []*descriptorpb.FileDescriptorSet{
			{
				File: []*descriptorpb.FileDescriptorProto{},
			},
		},
	}

	result, err := resolver.FindFileByPath("test.proto")
	require.Error(t, err)
	require.Equal(t, protoregistry.NotFound, err)
	require.Empty(t, result)
}

func TestThirdPartyResolverWithNilFileList(t *testing.T) {
	t.Parallel()

	resolver := &ThirdPartyResolver{
		items: []*descriptorpb.FileDescriptorSet{
			{
				File: nil,
			},
		},
	}

	result, err := resolver.FindFileByPath("test.proto")
	require.Error(t, err)
	require.Equal(t, protoregistry.NotFound, err)
	require.Empty(t, result)
}

func TestThirdPartyResolverWithFileWithoutName(t *testing.T) {
	t.Parallel()

	resolver := &ThirdPartyResolver{
		items: []*descriptorpb.FileDescriptorSet{
			{
				File: []*descriptorpb.FileDescriptorProto{
					{
						Name: nil,
					},
				},
			},
		},
	}

	result, err := resolver.FindFileByPath("test.proto")
	require.Error(t, err)
	require.Equal(t, protoregistry.NotFound, err)
	require.Empty(t, result)
}

func TestThirdPartyResolverWithEmptyFileName(t *testing.T) {
	t.Parallel()

	resolver := &ThirdPartyResolver{
		items: []*descriptorpb.FileDescriptorSet{
			{
				File: []*descriptorpb.FileDescriptorProto{
					{
						Name: new(""),
					},
				},
			},
		},
	}

	result, err := resolver.FindFileByPath("test.proto")
	require.Error(t, err)
	require.Equal(t, protoregistry.NotFound, err)
	require.Empty(t, result)
}

func TestThirdPartyResolverWithMatchingEmptyName(t *testing.T) {
	t.Parallel()

	resolver := &ThirdPartyResolver{
		items: []*descriptorpb.FileDescriptorSet{
			{
				File: []*descriptorpb.FileDescriptorProto{
					{
						Name: new(""),
					},
				},
			},
		},
	}

	result, err := resolver.FindFileByPath("")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Proto)
	require.Empty(t, result.Proto.GetName())
}

func TestThirdPartyResolverWithSpecialCharacters(t *testing.T) {
	t.Parallel()

	resolver := &ThirdPartyResolver{
		items: []*descriptorpb.FileDescriptorSet{
			{
				File: []*descriptorpb.FileDescriptorProto{
					{
						Name: new("test-file.proto"),
					},
					{
						Name: new("test_file.proto"),
					},
					{
						Name: new("test.file.proto"),
					},
				},
			},
		},
	}

	testCases := []string{
		"test-file.proto",
		"test_file.proto",
		"test.file.proto",
	}

	for _, fileName := range testCases {
		t.Run(fileName, func(t *testing.T) {
			t.Parallel()

			result, err := resolver.FindFileByPath(fileName)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, fileName, result.Proto.GetName())
		})
	}
}

func TestThirdPartyResolverWithLongPath(t *testing.T) {
	t.Parallel()

	resolver := &ThirdPartyResolver{
		items: []*descriptorpb.FileDescriptorSet{
			{
				File: []*descriptorpb.FileDescriptorProto{
					{
						Name: new("very/long/path/to/file.proto"),
					},
				},
			},
		},
	}

	result, err := resolver.FindFileByPath("very/long/path/to/file.proto")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "very/long/path/to/file.proto", result.Proto.GetName())
}

func TestThirdPartyResolverWithUnicodePath(t *testing.T) {
	t.Parallel()

	resolver := &ThirdPartyResolver{
		items: []*descriptorpb.FileDescriptorSet{
			{
				File: []*descriptorpb.FileDescriptorProto{
					{
						Name: new("тест/файл.proto"),
					},
				},
			},
		},
	}

	result, err := resolver.FindFileByPath("тест/файл.proto")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "тест/файл.proto", result.Proto.GetName())
}

func TestThirdPartyResolverWithDuplicateNames(t *testing.T) {
	t.Parallel()

	resolver := &ThirdPartyResolver{
		items: []*descriptorpb.FileDescriptorSet{
			{
				File: []*descriptorpb.FileDescriptorProto{
					{
						Name: new("duplicate.proto"),
					},
				},
			},
			{
				File: []*descriptorpb.FileDescriptorProto{
					{
						Name: new("duplicate.proto"),
					},
				},
			},
		},
	}

	result, err := resolver.FindFileByPath("duplicate.proto")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "duplicate.proto", result.Proto.GetName())
}
