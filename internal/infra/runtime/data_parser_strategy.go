package runtime

import (
	"context"

	domain "github.com/bavix/gripmock/v3/internal/domain/types"
)

// DataParserStrategy handles data-type outputs.
type DataParserStrategy struct{}

func (d *DataParserStrategy) CanHandle(data map[string]any) bool {
	_, ok := data["data"]

	return ok
}

func (d *DataParserStrategy) Parse(ctx context.Context, data map[string]any) (domain.OutputStrict, error) {
	if dataMap, ok := globalParser.parseMap(data["data"]); ok {
		return domain.OutputStrict{
			Data: &domain.DataResponse{
				Content: dataMap,
			},
		}, nil
	}

	return domain.OutputStrict{}, nil
}
