//nolint:ireturn
package protoresolver_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/bavix/gripmock/v3/internal/pkg/protoresolver"
)

type fakeResolver struct {
	findFileByPath       func(path string) (protoreflect.FileDescriptor, error)
	findDescriptorByName func(name protoreflect.FullName) (protoreflect.Descriptor, error)
	findFileByPathCalls  int
	findByNameCalls      int
}

func (f *fakeResolver) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	f.findFileByPathCalls++

	return f.findFileByPath(path)
}

func (f *fakeResolver) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	f.findByNameCalls++

	return f.findDescriptorByName(name)
}

func TestFallback_FindFileByPath_UsesPrimaryFirst(t *testing.T) {
	t.Parallel()

	// Arrange
	file := (&wrapperspb.DoubleValue{}).ProtoReflect().Descriptor().ParentFile()
	primary := &fakeResolver{
		findFileByPath: func(_ string) (protoreflect.FileDescriptor, error) {
			return file, nil
		},
		findDescriptorByName: func(_ protoreflect.FullName) (protoreflect.Descriptor, error) {
			return nil, protoregistry.NotFound
		},
	}
	fallback := &fakeResolver{
		findFileByPath: func(_ string) (protoreflect.FileDescriptor, error) {
			return nil, protoregistry.NotFound
		},
		findDescriptorByName: func(_ protoreflect.FullName) (protoreflect.Descriptor, error) {
			return nil, protoregistry.NotFound
		},
	}
	resolver := &protoresolver.Fallback{Primary: primary, Fallback: fallback}

	// Act
	got, err := resolver.FindFileByPath(file.Path())

	// Assert
	require.NoError(t, err)
	require.Equal(t, file.Path(), got.Path())
	require.Equal(t, 1, primary.findFileByPathCalls)
	require.Equal(t, 0, fallback.findFileByPathCalls)
}

func TestFallback_FindFileByPath_FallsBackOnNotFound(t *testing.T) {
	t.Parallel()

	// Arrange
	file := (&wrapperspb.DoubleValue{}).ProtoReflect().Descriptor().ParentFile()
	primary := &fakeResolver{
		findFileByPath: func(_ string) (protoreflect.FileDescriptor, error) {
			return nil, protoregistry.NotFound
		},
		findDescriptorByName: func(_ protoreflect.FullName) (protoreflect.Descriptor, error) {
			return nil, protoregistry.NotFound
		},
	}
	fallback := &fakeResolver{
		findFileByPath: func(_ string) (protoreflect.FileDescriptor, error) {
			return file, nil
		},
		findDescriptorByName: func(_ protoreflect.FullName) (protoreflect.Descriptor, error) {
			return nil, protoregistry.NotFound
		},
	}
	resolver := &protoresolver.Fallback{Primary: primary, Fallback: fallback}

	// Act
	got, err := resolver.FindFileByPath(file.Path())

	// Assert
	require.NoError(t, err)
	require.Equal(t, file.Path(), got.Path())
	require.Equal(t, 1, primary.findFileByPathCalls)
	require.Equal(t, 1, fallback.findFileByPathCalls)
}

func TestFallback_FindDescriptorByName_FallsBackOnNotFound(t *testing.T) {
	t.Parallel()

	// Arrange
	desc := (&wrapperspb.DoubleValue{}).ProtoReflect().Descriptor()
	fullName := desc.FullName()
	primary := &fakeResolver{
		findFileByPath: func(_ string) (protoreflect.FileDescriptor, error) {
			return nil, protoregistry.NotFound
		},
		findDescriptorByName: func(_ protoreflect.FullName) (protoreflect.Descriptor, error) {
			return nil, protoregistry.NotFound
		},
	}
	fallback := &fakeResolver{
		findFileByPath: func(_ string) (protoreflect.FileDescriptor, error) {
			return nil, protoregistry.NotFound
		},
		findDescriptorByName: func(_ protoreflect.FullName) (protoreflect.Descriptor, error) {
			return desc, nil
		},
	}
	resolver := &protoresolver.Fallback{Primary: primary, Fallback: fallback}

	// Act
	got, err := resolver.FindDescriptorByName(fullName)

	// Assert
	require.NoError(t, err)
	require.Equal(t, fullName, got.FullName())
	require.Equal(t, 1, primary.findByNameCalls)
	require.Equal(t, 1, fallback.findByNameCalls)
}

func TestFallback_ReturnsNotFoundWithoutResolvers(t *testing.T) {
	t.Parallel()

	// Arrange
	resolver := &protoresolver.Fallback{}

	// Act
	_, fileErr := resolver.FindFileByPath("unknown.proto")
	_, descErr := resolver.FindDescriptorByName("unknown.Service")

	// Assert
	require.ErrorIs(t, fileErr, protoregistry.NotFound)
	require.ErrorIs(t, descErr, protoregistry.NotFound)
}
