package sourceclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/domain/protoset"
)

type mockBSRClient struct {
	fds *descriptorpb.FileDescriptorSet
}

func (m *mockBSRClient) FetchDescriptorSet(
	ctx context.Context,
	module, version string,
) (*descriptorpb.FileDescriptorSet, error) {
	return m.fds, nil
}

type mockReflectClient struct {
	fds *descriptorpb.FileDescriptorSet
}

func (m *mockReflectClient) FetchDescriptorSet(
	ctx context.Context,
	source *protoset.Source,
) (*descriptorpb.FileDescriptorSet, error) {
	return m.fds, nil
}

func TestRouterFetchDescriptorSetBySourceType(t *testing.T) {
	t.Parallel()

	bsrName := "bsr.proto"
	reflectName := "reflect.proto"

	bsrFDS := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: &bsrName}}}
	reflectFDS := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{Name: &reflectName}}}

	r := NewRouter(&mockBSRClient{fds: bsrFDS}, &mockReflectClient{fds: reflectFDS})

	fromBSR, err := r.FetchDescriptorSet(t.Context(), &protoset.Source{Type: protoset.SourceBufBuild, Module: "buf.build/acme/api"})
	require.NoError(t, err)
	require.Equal(t, "bsr.proto", fromBSR.GetFile()[0].GetName())

	fromReflect, err := r.FetchDescriptorSet(t.Context(), &protoset.Source{Type: protoset.SourceReflect, ReflectAddress: "localhost:50051"})
	require.NoError(t, err)
	require.Equal(t, "reflect.proto", fromReflect.GetFile()[0].GetName())
}

func TestRouterFetchDescriptorSetErrors(t *testing.T) {
	t.Parallel()

	r := NewRouter(nil, nil)

	_, err := r.FetchDescriptorSet(t.Context(), nil)
	require.ErrorContains(t, err, "source is nil")

	_, err = r.FetchDescriptorSet(t.Context(), &protoset.Source{Type: protoset.SourceBufBuild})
	require.ErrorContains(t, err, "bsr client is not configured")

	_, err = r.FetchDescriptorSet(t.Context(), &protoset.Source{Type: protoset.SourceReflect})
	require.ErrorContains(t, err, "reflect client is not configured")

	_, err = r.FetchDescriptorSet(t.Context(), &protoset.Source{Type: protoset.SourceProto})
	require.ErrorContains(t, err, "unsupported remote source type")
}
