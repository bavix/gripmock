package protoset

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestProcessorProcessBufBuild(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	source := &Source{
		Type:    SourceBufBuild,
		Raw:     "buf.build/grpc-ecosystem/grpc-gateway",
		Module:  "buf.build/grpc-ecosystem/grpc-gateway",
		Version: "",
	}

	descriptors := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			{
				Name:    new("test.proto"),
				Package: new("test"),
			},
		},
	}

	mockClient := &mockBufClient{descriptors: descriptors}
	processor := newProcessor([]string{}, mockClient)

	err := processor.ProcessBufBuild(ctx, source)
	require.NoError(t, err)
	require.Len(t, processor.descriptorSets, 1)
}

func TestBuildWithBufBuildClient(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	mockClient := &mockBufClient{
		descriptors: &descriptorpb.FileDescriptorSet{
			File: []*descriptorpb.FileDescriptorProto{
				{
					Name:    new("test.proto"),
					Package: new("test"),
				},
			},
		},
	}

	descriptors, err := Build(ctx, []string{}, []string{"buf.build/test/module"}, mockClient)
	require.NoError(t, err)
	require.Len(t, descriptors, 1)
}

type mockBufClient struct {
	descriptors *descriptorpb.FileDescriptorSet
}

func (m *mockBufClient) FetchDescriptorSet(ctx context.Context, source *Source) (*descriptorpb.FileDescriptorSet, error) {
	return m.descriptors, nil
}
