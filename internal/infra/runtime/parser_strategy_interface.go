package runtime

import (
	"context"

	domain "github.com/bavix/gripmock/v3/internal/domain/types"
)

// ParserStrategy defines the interface for different parsing strategies.
type ParserStrategy interface {
	Parse(ctx context.Context, data map[string]any) (domain.OutputStrict, error)
	CanHandle(data map[string]any) bool
}
