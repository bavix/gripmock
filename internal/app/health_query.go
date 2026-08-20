package app

import (
	"context"
	"time"

	"google.golang.org/grpc/metadata"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

func (s *mockableHealthServer) findStub(ctx context.Context, method, service string) (*stuber.Stub, int, bool) {
	if s.storage == nil {
		return nil, 0, false
	}

	query := newHealthQuery(method, service)

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		query.Headers = processHeaders(md)
		query.Session = sessionFromMetadata(md)
	}

	result, err := s.storage.FindByQuery(query)
	if err != nil || result == nil || result.Found() == nil {
		return nil, 0, false
	}

	return result.Found(), result.MatchNumber(), true
}

func healthTemplateData(ctx context.Context, service string, stub *stuber.Stub, matchNumber int) template.Data {
	request := map[string]any{"service": service}

	headers := make(map[string]any)
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		headers = processHeaders(md)
	}

	return newTemplateData(request, headers, 0, time.Now(),
		[]any{request}, stub, matchNumber)
}

func newHealthQuery(method, service string) stuber.Query {
	// The health service is the ONLY caller allowed to match the reserved internal
	// gripmock health stubs; every other query keeps them hidden.
	return stuber.WithInternalStubs(stuber.Query{
		Service: HealthServiceFullName,
		Method:  method,
		Input: []map[string]any{{
			"service": service,
		}},
	})
}
