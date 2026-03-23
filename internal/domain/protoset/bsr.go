package protoset

import (
	"context"

	"google.golang.org/protobuf/types/descriptorpb"
)

type BSRClient interface {
	FetchDescriptorSet(ctx context.Context, module, version string) (*descriptorpb.FileDescriptorSet, error)
}
