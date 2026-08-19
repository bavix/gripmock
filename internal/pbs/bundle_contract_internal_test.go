package pbs

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/infra/protobundle"
)

func TestEmbeddedBundlesResolveWellKnownImports(t *testing.T) {
	t.Parallel()

	resolver, err := NewResolver()
	require.NoError(t, err)

	required := []string{
		"google/protobuf/any.proto",
		"google/protobuf/descriptor.proto",
		"google/protobuf/duration.proto",
		"google/protobuf/empty.proto",
		"google/protobuf/field_mask.proto",
		"google/protobuf/struct.proto",
		"google/protobuf/timestamp.proto",
		"google/protobuf/wrappers.proto",
		"google/api/annotations.proto",
		"google/api/http.proto",
		"google/api/field_behavior.proto",
		"google/api/client.proto",
		"google/api/resource.proto",
		"google/rpc/status.proto",
		"google/rpc/code.proto",
		"google/rpc/error_details.proto",
		"google/longrunning/operations.proto",
		"google/type/date.proto",
		"google/type/money.proto",
		"google/type/latlng.proto",
	}

	for _, path := range required {
		result, err := resolver.FindFileByPath(path)
		require.NoErrorf(t, err, "bundle no longer carries %s", path)
		require.NotNil(t, result.Proto, "%s resolved to a nil descriptor", path)
		require.Equal(t, path, result.Proto.GetName())
	}
}

func TestEmbeddedBundlesAreSubstantial(t *testing.T) {
	t.Parallel()

	pb, err := protobufIndex()
	require.NoError(t, err)
	require.Greater(t, len(pb), 10, "well-known types bundle looks truncated")

	ga, err := googleapisIndex()
	require.NoError(t, err)
	require.Greater(t, len(ga), 100, "googleapis bundle looks truncated")
}

func TestLargeBundleIsNotDecodedForWellKnownTypes(t *testing.T) {
	t.Parallel()

	resolver, err := NewResolver()
	require.NoError(t, err)
	require.Empty(t, resolver.items, "embedded resolver must not hold decoded sets")
	require.NotEmpty(t, resolver.index, "well-known types are resolved without the large bundle")
	require.Len(t, resolver.index, 19, "only the protobuf bundle is decoded up front")
}

func TestUnknownImportIsNotFound(t *testing.T) {
	t.Parallel()

	resolver, err := NewResolver()
	require.NoError(t, err)

	_, err = resolver.FindFileByPath("definitely/not/a/real.proto")
	require.Error(t, err)
}

func TestConcurrentLookupsAreRaceFree(t *testing.T) {
	t.Parallel()

	embedded, err := NewResolver()
	require.NoError(t, err)

	pb, err := protobundle.Decode(protobuf)
	require.NoError(t, err)

	resolvers := []*ThirdPartyResolver{
		embedded,
		{items: []*descriptorpb.FileDescriptorSet{pb}},
	}

	paths := []string{
		"google/protobuf/timestamp.proto",
		"google/protobuf/any.proto",
		"nope/missing.proto",
	}

	for _, resolver := range resolvers {
		var wg sync.WaitGroup

		for range 16 {
			wg.Go(func() {
				for _, path := range paths {
					_, _ = resolver.FindFileByPath(path)
				}
			})
		}

		wg.Wait()
	}
}

func BenchmarkFindFileByPath(b *testing.B) {
	resolver, err := NewResolver()
	require.NoError(b, err)

	for b.Loop() {
		_, _ = resolver.FindFileByPath("google/api/annotations.proto")
	}
}
