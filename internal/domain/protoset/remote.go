package protoset

import (
	"context"

	"google.golang.org/protobuf/types/descriptorpb"
)

type RemoteClient interface {
	FetchDescriptorSet(ctx context.Context, source *Source) (*descriptorpb.FileDescriptorSet, error)
}
