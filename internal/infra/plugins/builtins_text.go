package plugins

import (
	"encoding/json"
	"fmt"
	"strings"

	infrafaker "github.com/bavix/gripmock/v3/internal/infra/faker"
)

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
	return map[string]any{
		"json": func(v any) string {
			b, err := json.Marshal(v)
			if err != nil {
				return ""
			}

			return string(b)
		},
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
