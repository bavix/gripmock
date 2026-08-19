package pbs

import (
	_ "embed"
	"sync"

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

//nolint:gochecknoglobals // memoization has to outlive individual resolvers;
var (
	protobufIndex   = sync.OnceValues(func() (fileIndex, error) { return decodeIndex(protobuf) })
	googleapisIndex = sync.OnceValues(func() (fileIndex, error) { return decodeIndex(googleapis) })
)

type fileIndex map[string]*descriptorpb.FileDescriptorProto

func decodeIndex(compressed []byte) (fileIndex, error) {
	fds, err := protobundle.Decode(compressed)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode embedded descriptor")
	}

	return indexOf([]*descriptorpb.FileDescriptorSet{fds}), nil
}

func indexOf(sets []*descriptorpb.FileDescriptorSet) fileIndex {
	size := 0
	for _, set := range sets {
		size += len(set.GetFile())
	}

	index := make(fileIndex, size)

	for _, set := range sets {
		for _, file := range set.GetFile() {
			if _, seen := index[file.GetName()]; !seen {
				index[file.GetName()] = file
			}
		}
	}

	return index
}

type ThirdPartyResolver struct {
	items []*descriptorpb.FileDescriptorSet

	indexOnce sync.Once
	index     fileIndex

	embedded bool
}

func NewResolver() (*ThirdPartyResolver, error) {
	index, err := protobufIndex()
	if err != nil {
		return nil, err
	}

	return &ThirdPartyResolver{index: index, embedded: true}, nil
}

func (p *ThirdPartyResolver) FindFileByPath(path string) (protocompile.SearchResult, error) {
	p.indexOnce.Do(func() {
		if p.index == nil {
			p.index = indexOf(p.items)
		}
	})

	if file, ok := p.index[path]; ok {
		return protocompile.SearchResult{Proto: file}, nil
	}

	if p.embedded {
		index, err := googleapisIndex()
		if err != nil {
			return protocompile.SearchResult{}, err
		}

		if file, ok := index[path]; ok {
			return protocompile.SearchResult{Proto: file}, nil
		}
	}

	return protocompile.SearchResult{}, protoregistry.NotFound
}
