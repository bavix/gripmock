package runtime

import (
	"context"

	domain "github.com/bavix/gripmock/v3/internal/domain/types"
)

// ParserFactory creates appropriate parser strategies.
type ParserFactory struct {
	strategies []ParserStrategy
}

// NewParserFactory creates a new parser factory with default strategies.
func NewParserFactory() *ParserFactory {
	return &ParserFactory{
		strategies: []ParserStrategy{
			&DataParserStrategy{},
			&StatusParserStrategy{},
		},
	}
}

// Parse parses data using the appropriate strategy.
func (pf *ParserFactory) Parse(ctx context.Context, data map[string]any) (domain.OutputStrict, error) {
	for _, strategy := range pf.strategies {
		if strategy.CanHandle(data) {
			return strategy.Parse(ctx, data)
		}
	}

	// Default: return empty output
	return domain.OutputStrict{}, nil
}
