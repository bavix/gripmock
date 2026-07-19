package plugins

import (
	"github.com/bavix/gripmock/v3/internal/infra/build"
	pkgplugins "github.com/bavix/gripmock/v3/pkg/plugins"
)

const (
	builtinSummary = "Built-in helpers preloaded by GripMock"
)

type groupDef struct {
	group       string
	defaultDesc string
	funcs       map[string]any
	overrides   map[string]string
}

func buildSpecs(groups []groupDef) []pkgplugins.FuncSpec {
	totalSize := 0
	for _, g := range groups {
		totalSize += len(g.funcs)
	}

	specs := make([]pkgplugins.FuncSpec, 0, totalSize)

	for _, g := range groups {
		for name, fn := range g.funcs {
			desc := g.defaultDesc
			if d, ok := g.overrides[name]; ok {
				desc = d
			}

			specs = append(specs, pkgplugins.FuncSpec{
				Name:        name,
				Fn:          fn,
				Description: desc,
				Group:       g.group,
			})
		}
	}

	return specs
}

func RegisterBuiltins(reg pkgplugins.Registry) {
	spec := buildSpecs(builtinGroups())

	reg.AddPlugin(builtinInfo(), []pkgplugins.SpecProvider{
		pkgplugins.Specs(spec...),
	})
}

func builtinGroups() []groupDef {
	return []groupDef{
		stringGroup(),
		jsonGroup(),
		formatGroup(),
		numberGroup(),
		arrayGroup(),
		compareGroup(),
		mathGroup(),
		timeGroup(),
		uuidGroup(),
		encodingGroup(),
		fakerGroup(),
	}
}

func fakerGroup() groupDef {
	return groupDef{
		group:       "faker",
		defaultDesc: "faker object",
		funcs:       fakerFuncs(),
		overrides: map[string]string{
			"faker": "faker object exposing various data generators",
		},
	}
}

func stringGroup() groupDef {
	return groupDef{
		group:       "string",
		defaultDesc: "string helper",
		funcs:       stringFuncs(),
		overrides: map[string]string{
			"upper": "uppercase string",
			"lower": "lowercase string",
			"title": "title case string",
			"join":  "join slice with separator",
			"split": "split string by separator",
		},
	}
}

func jsonGroup() groupDef {
	return groupDef{
		group:       "json",
		defaultDesc: "json helper",
		funcs:       jsonFuncs(),
		overrides: map[string]string{
			"json": "encode value as JSON string",
		},
	}
}

func formatGroup() groupDef {
	return groupDef{
		group:       "format",
		defaultDesc: "format helper",
		funcs:       formatFuncs(),
		overrides: map[string]string{
			"sprintf": "format via fmt.Sprintf",
			"str":     "format value with default verb",
		},
	}
}

func numberGroup() groupDef {
	return groupDef{
		group:       "number",
		defaultDesc: "number helper",
		funcs:       numberFuncs(),
		overrides: map[string]string{
			"int":     "convert to int",
			"int64":   "convert to int64",
			"float":   "convert to float64",
			"decimal": "render number as json.Number",
		},
	}
}

func arrayGroup() groupDef {
	return groupDef{
		group:       "array",
		defaultDesc: "array helper",
		funcs:       arrayFuncs(),
		overrides: map[string]string{
			"extract": "get element by key or index",
		},
	}
}

func compareGroup() groupDef {
	return groupDef{
		group:       "compare",
		defaultDesc: "compare helper",
		funcs:       compareFuncs(),
		overrides: map[string]string{
			"gt":  "a greater than b",
			"lt":  "a less than b",
			"gte": "a greater or equal b",
			"lte": "a less or equal b",
			"eq":  "a equals b",
		},
	}
}

func mathGroup() groupDef {
	return groupDef{
		group:       "math",
		defaultDesc: "math helper",
		funcs:       mathFuncs(),
		overrides: map[string]string{
			"round": "round to nearest integer",
			"floor": "floor value",
			"ceil":  "ceil value",
			"add":   "add numbers",
			"sub":   "subtract numbers",
			"div":   "divide numbers",
			"mod":   "modulo of numbers",
			"sum":   "sum slice of numbers",
			"mul":   "multiply numbers",
			"avg":   "average of numbers",
			"min":   "minimum of numbers",
			"max":   "maximum of numbers",
		},
	}
}

func timeGroup() groupDef {
	return groupDef{
		group:       "time",
		defaultDesc: "time helper",
		funcs:       timeFuncs(),
		overrides: map[string]string{
			"now":    "current time",
			"unix":   "timestamp seconds",
			"format": "format time with layout",
		},
	}
}

func uuidGroup() groupDef {
	return groupDef{
		group:       "uuid",
		defaultDesc: "uuid helper",
		funcs:       uuidFuncMap(),
		overrides: map[string]string{
			"uuid": "generate UUID v4",
		},
	}
}

func encodingGroup() groupDef {
	return groupDef{
		group:       "encoding",
		defaultDesc: "encoding helper",
		funcs:       encodingFuncs(),
		overrides: map[string]string{
			"bytes":         "string to bytes",
			"string2base64": "encode string to base64",
			"bytes2base64":  "encode bytes to base64",
			"uuid2base64":   "encode UUID to base64",
			"uuid2bytes":    "UUID to bytes",
			"uuid2int64":    "UUID high bits as int64",
		},
	}
}

func builtinInfo() pkgplugins.PluginInfo {
	return pkgplugins.PluginInfo{
		Name:         "gripmock",
		Source:       "gripmock",
		Version:      build.Version,
		Kind:         "builtin",
		Capabilities: []string{"template-funcs"},
		Authors: []pkgplugins.Author{
			{Name: "Maxim Babichev", Contact: "info@babichev.net"},
		},
		Description: builtinSummary,
	}
}
