package app

import (
	"context"

	"google.golang.org/grpc/metadata"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func (s *mockableHealthServer) findStub(ctx context.Context, method, service string) (*stuber.Stub, bool) {
	if s.storage == nil {
		return nil, false
	}

	query := newHealthQuery(method, service)

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		query.Headers = processHeaders(md)
		query.Session = sessionFromMetadata(md)
	}

	result, err := s.storage.FindByQuery(query)
	if err != nil || result == nil || result.Found() == nil {
		return nil, false
	}

	return result.Found(), true
}

func newHealthQuery(method, service string) stuber.Query {
	return stuber.Query{
		Service: HealthServiceFullName,
		Method:  method,
		Input: []map[string]any{{
			"service": service,
		}},
	}
}
