package plugins

import (
	"cmp"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

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

func stringFuncs() map[string]any {
	return map[string]any{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": titleCase,
		"join":  strings.Join,
		"split": strings.Split,
	}
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

func numberFuncs() map[string]any {
	return map[string]any{
		"int": func(v any) int {
			if f, ok := convertToFloat64(v); ok {
				return int(f)
			}

			return 0
		},
		"int64": func(v any) int64 {
			if f, ok := convertToFloat64(v); ok {
				return int64(f)
			}

			return 0
		},
		"float": func(v any) float64 {
			if f, ok := convertToFloat64(v); ok {
				return f
			}

			return 0
		},
		"decimal": func(v any) json.Number {
			if f, ok := convertToFloat64(v); ok {
				if math.Trunc(f) == f {
					return json.Number(strconv.FormatFloat(f, 'f', 1, 64))
				}

				return json.Number(strconv.FormatFloat(f, 'g', -1, 64))
			}

			return json.Number("0")
		},
	}
}

func arrayFuncs() map[string]any {
	return map[string]any{
		"extract": extract,
	}
}

func extract(collection any, key any) any {
	k := fmt.Sprint(key)

	switch c := collection.(type) {
	case map[string]any:
		return c[k]
	case map[string]string:
		return c[k]
	case []any:
		if _, ok := convertToInt(key); ok {
			return extractFromSlice(len(c), key, func(i int) any { return c[i] })
		}

		return extractFromObjects(c, k)
	case []string:
		return extractFromSlice(len(c), key, func(i int) any { return c[i] })
	}

	return nil
}

func extractFromSlice(length int, key any, getter func(int) any) any {
	idx, ok := convertToInt(key)
	if !ok || idx < 0 || idx >= length {
		return nil
	}

	return getter(idx)
}

func extractFromObjects(items []any, key string) any {
	out := make([]any, 0, len(items))

	for _, item := range items {
		switch m := item.(type) {
		case map[string]any:
			if v, ok := m[key]; ok {
				out = append(out, v)
			}
		case map[string]string:
			if v, ok := m[key]; ok {
				out = append(out, v)
			}
		}
	}

	return out
}

func compareFuncs() map[string]any {
	cmpFn := func(a, b any) (int, bool) {
		va, okA := convertToFloat64(a)
		if !okA {
			return 0, false
		}

		vb, okB := convertToFloat64(b)
		if !okB {
			return 0, false
		}

		return cmp.Compare(va, vb), true
	}

	return map[string]any{
		"gt": func(a, b any) bool {
			if r, ok := cmpFn(a, b); ok {
				return r > 0
			}

			return false
		},
		"lt": func(a, b any) bool {
			if r, ok := cmpFn(a, b); ok {
				return r < 0
			}

			return false
		},
		"gte": func(a, b any) bool {
			if r, ok := cmpFn(a, b); ok {
				return r >= 0
			}

			return false
		},
		"lte": func(a, b any) bool {
			if r, ok := cmpFn(a, b); ok {
				return r <= 0
			}

			return false
		},
		"eq": func(a, b any) bool {
			if r, ok := cmpFn(a, b); ok {
				return r == 0
			}

			return false
		},
	}
}

func mathFuncs() map[string]any {
	return map[string]any{
		"round": func(v any) float64 {
			if f, ok := convertToFloat64(v); ok {
				return math.Round(f)
			}

			return 0
		},
		"floor": func(v any) float64 {
			if f, ok := convertToFloat64(v); ok {
				return math.Floor(f)
			}

			return 0
		},
		"ceil": func(v any) float64 {
			if f, ok := convertToFloat64(v); ok {
				return math.Ceil(f)
			}

			return 0
		},
		"add": add,
		"sub": subtract,
		"div": divide,
		"mod": modulo,
		"sum": sum,
		"mul": product,
		"avg": average,
		"min": minValue,
		"max": maxValue,
	}
}

func encodingFuncs() map[string]any {
	conv := conversionFuncs{}
	b64 := base64Funcs{}
	u := uuidHelper{}

	return map[string]any{
		"bytes":         conv.StringToBytes,
		"string2base64": b64.StringToBase64,
		"bytes2base64":  b64.BytesToBase64,
		"uuid2base64":   u.UUIDToBase64,
		"uuid2bytes":    u.UUIDToBytes,
		"uuid2int64":    u.UUIDToInt64,
	}
}

type conversionFuncs struct{}

func (conversionFuncs) StringToBytes(s string) []byte { return []byte(s) }

type base64Funcs struct{}

func (base64Funcs) StringToBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
func (base64Funcs) BytesToBase64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

type uuidHelper struct{}

func (uuidHelper) UUIDToBase64(id string) (string, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(parsed[:]), nil
}

func (uuidHelper) UUIDToBytes(id string) ([]byte, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	return parsed[:], nil
}

func (uuidHelper) UUIDToInt64(id string) (string, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return "", err
	}

	bytes := parsed[:]

	//nolint:gosec // we intentionally reinterpret UUID halves as signed
	high := int64(binary.LittleEndian.Uint64(bytes[:8]))
	//nolint:gosec // we intentionally reinterpret UUID halves as signed
	low := int64(binary.LittleEndian.Uint64(bytes[8:]))

	return fmt.Sprintf(`{"high":%d,"low":%d}`, high, low), nil
}

func timeFuncs() map[string]any {
	return map[string]any{
		"now":    time.Now,
		"unix":   time.Time.Unix,
		"format": time.Time.Format,
	}
}

func uuidFuncMap() map[string]any {
	return map[string]any{
		"uuid": func() string {
			return uuid.New().String()
		},
	}
}

func convertToFloat64(v any) (float64, bool) {
	switch value := v.(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case json.Number:
		f, err := value.Float64()
		if err == nil {
			return f, true
		}
	case string:
		f, err := strconv.ParseFloat(value, 64)
		if err == nil {
			return f, true
		}
	default:
		f, err := strconv.ParseFloat(fmt.Sprint(value), 64)
		if err == nil {
			return f, true
		}
	}

	return 0, false
}

func convertToInt(v any) (int, bool) {
	switch value := v.(type) {
	case int:
		return value, true
	case int64:
		return int(value), true
	case float64:
		return int(value), true
	case float32:
		return int(value), true
	case json.Number:
		if i, err := value.Int64(); err == nil {
			return int(i), true
		}
	case string:
		return parseIntString(value)
	default:
		return parseIntString(fmt.Sprint(value))
	}

	return 0, false
}

func parseIntString(s string) (int, bool) {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}

	return i, true
}

func titleCase(s string) string {
	return strings.ToTitle(s)
}

func add(values ...any) float64 {
	nums, ok := convertAllToFloat64(values...)
	if !ok {
		return 0
	}

	sum := 0.0
	for _, v := range nums {
		sum += v
	}

	return sum
}

func subtract(values ...any) float64 {
	nums, ok := convertAllToFloat64(values...)
	if !ok || len(nums) == 0 {
		return 0
	}

	result := nums[0]

	for _, v := range nums[1:] {
		result -= v
	}

	return result
}

func divide(values ...any) float64 {
	nums, ok := convertAllToFloat64(values...)
	if !ok || len(nums) == 0 {
		return 0
	}

	result := nums[0]

	for _, v := range nums[1:] {
		if v != 0 {
			result /= v
		}
	}

	return result
}

func modulo(values ...any) float64 {
	nums, ok := convertAllToFloat64(values...)
	if !ok || len(nums) < 2 || nums[1] == 0 {
		return 0
	}

	return math.Mod(nums[0], nums[1])
}

func sum(values ...any) float64 {
	nums, ok := convertAllToFloat64(values...)
	if !ok {
		return 0
	}

	total := 0.0
	for _, v := range nums {
		total += v
	}

	return total
}

func product(values ...any) float64 {
	nums, ok := convertAllToFloat64(values...)
	if !ok {
		return 0
	}

	prod := 1.0
	for _, v := range nums {
		prod *= v
	}

	return prod
}

func average(values ...any) float64 {
	nums, ok := convertAllToFloat64(values...)
	if !ok || len(nums) == 0 {
		return 0
	}

	total := 0.0
	for _, v := range nums {
		total += v
	}

	return total / float64(len(nums))
}

func minValue(values ...any) float64 {
	nums, ok := convertAllToFloat64(values...)
	if !ok || len(nums) == 0 {
		return 0
	}

	minVal := nums[0]

	for _, v := range nums[1:] {
		minVal = minFloat(minVal, v)
	}

	return minVal
}

func maxValue(values ...any) float64 {
	nums, ok := convertAllToFloat64(values...)
	if !ok || len(nums) == 0 {
		return 0
	}

	maxVal := nums[0]

	for _, v := range nums[1:] {
		maxVal = maxFloat(maxVal, v)
	}

	return maxVal
}

func convertAllToFloat64(values ...any) ([]float64, bool) {
	nums := make([]float64, 0, len(values))
	for _, v := range values {
		if f, ok := convertToFloat64(v); ok {
			nums = append(nums, f)
		} else {
			return nil, false
		}
	}

	return nums, true
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}

	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}

	return b
}
