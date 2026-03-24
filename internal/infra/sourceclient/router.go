package sourceclient

import (
	"context"

	"github.com/cockroachdb/errors"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/internal/domain/protoset"
)

type Router struct {
	bsrClient     protoset.BSRClient
	reflectClient protoset.RemoteClient
}

var (
	errSourceNil                  = errors.New("source is nil")
	errBSRClientNotConfigured     = errors.New("bsr client is not configured")
	errReflectClientNotConfigured = errors.New("reflect client is not configured")
	errUnsupportedSourceType      = errors.New("unsupported remote source type")
)

func NewRouter(bsrClient protoset.BSRClient, reflectClient protoset.RemoteClient) *Router {
	return &Router{bsrClient: bsrClient, reflectClient: reflectClient}
}

func (r *Router) FetchDescriptorSet(ctx context.Context, source *protoset.Source) (*descriptorpb.FileDescriptorSet, error) {
	if source == nil {
		return nil, errSourceNil
	}

	switch source.Type {
	case protoset.SourceBufBuild:
		if r.bsrClient == nil {
			return nil, errBSRClientNotConfigured
		}

		return r.bsrClient.FetchDescriptorSet(ctx, source.Module, source.Version)
	case protoset.SourceReflect:
		if r.reflectClient == nil {
			return nil, errReflectClientNotConfigured
		}

		return r.reflectClient.FetchDescriptorSet(ctx, source)
	case protoset.SourceUnknown, protoset.SourceProto, protoset.SourceDescriptor, protoset.SourceDirectory:
		return nil, errors.Wrapf(errUnsupportedSourceType, "%d", source.Type)
	default:
		return nil, errors.Wrapf(errUnsupportedSourceType, "%d", source.Type)
	}
}
