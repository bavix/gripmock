package pbs

import (
	_ "embed"

	"github.com/bufbuild/protocompile"
	"github.com/cockroachdb/errors"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/infra/protobundle"
)

//go:embed googleapis.pbs
var googleapis []byte

//go:embed protobuf.pbs
var protobuf []byte

type ThirdPartyResolver struct {
	items []*descriptorpb.FileDescriptorSet
}

func NewResolver() (*ThirdPartyResolver, error) {
	resolver := &ThirdPartyResolver{
		items: make([]*descriptorpb.FileDescriptorSet, 0, 2), //nolint:mnd
	}

	for _, compressed := range [][]byte{googleapis, protobuf} {
		fds, err := protobundle.Decode(compressed)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode embedded descriptor")
		}

		resolver.items = append(resolver.items, fds)
	}

	return resolver, nil
}

func (p *ThirdPartyResolver) FindFileByPath(path string) (protocompile.SearchResult, error) {
	for _, pb := range p.items {
		for _, file := range pb.GetFile() {
			if file.GetName() == path {
				return protocompile.SearchResult{Proto: file}, nil
			}
		}
	}

	return protocompile.SearchResult{}, protoregistry.NotFound
}
