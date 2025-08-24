package runtime

import (
	"context"

	domain "github.com/bavix/gripmock/v3/internal/domain/types"
)

// StatusParserStrategy handles status-only outputs.
type StatusParserStrategy struct{}

func (s *StatusParserStrategy) CanHandle(data map[string]any) bool {
	_, hasStatus := data["status"]
	_, hasData := data["data"]
	_, hasStream := data["stream"]

	return hasStatus && !hasData && !hasStream
}

func (s *StatusParserStrategy) Parse(ctx context.Context, data map[string]any) (domain.OutputStrict, error) {
	if status, ok := globalParser.parseMap(data["status"]); ok {
		return domain.OutputStrict{
			Status: &domain.GrpcStatus{
				Code:    globalParser.parseString(status["code"]),
				Message: globalParser.parseString(status["message"]),
			},
		}, nil
	}

	return domain.OutputStrict{}, nil
}
