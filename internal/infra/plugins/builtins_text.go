package plugins

import (
	"errors"
	"fmt"
	"strings"

	"github.com/goccy/go-json"

	infrafaker "github.com/bavix/gripmock/v3/internal/infra/faker"
)

var errJSONArgs = errors.New("json requires a single value")

func fakerFuncs() map[string]any {
	return map[string]any{
		"faker": infrafaker.New,
	}
}

func stringFuncs() map[string]any {
	return map[string]any{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": titleCase,
		"join":  strings.Join,
		"split": strings.Split,
	}
}

func titleCase(s string) string {
	return strings.ToTitle(s)
}

func jsonFuncs() map[string]any {
	encode := func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, errJSONArgs
		}

		b, err := json.Marshal(args[0])
		if err != nil {
			return nil, err
		}

		return string(b), nil
	}

	return map[string]any{
		"json":   encode,
		"toJson": encode,
	}
}

func formatFuncs() map[string]any {
	return map[string]any{
		"sprintf": fmt.Sprintf,
		"str": func(v any) string {
			switch t := v.(type) {
			case string:
				return t
			case json.Number:
				return t.String()
			default:
				return fmt.Sprint(v)
			}
		},
	}
}
