package app

import (
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bavix/gripmock/v3/internal/infra/httputil"
)

func readUnaryBody(body io.Reader) ([]byte, error) {
	if body == nil {
		return nil, nil
	}

	limit := httputil.MaxBodyBytes()

	data, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to read body")
	}

	if int64(len(data)) > limit {
		return nil, status.Errorf(codes.ResourceExhausted,
			"request body exceeds the %d byte limit", limit)
	}

	return data, nil
}
