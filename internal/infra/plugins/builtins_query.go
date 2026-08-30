package plugins

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

var (
	errWhereArgs = errors.New("where requires a field, an operator, a value and a collection")
	errWhereOp   = errors.New("where supports eq, ne, gt, gte, lt and lte")
	errPageArgs  = errors.New("page requires an offset, a limit and a collection")
	errCountArgs = errors.New("countBy requires a field and a collection")
)

func queryFuncs() map[string]any {
	return map[string]any{
		"where":   where,
		"page":    page,
		"countBy": countBy,
	}
}

func where(args ...any) (any, error) {
	const whereArity = 4

	if len(args) != whereArity {
		return nil, errWhereArgs
	}

	field, op, want := stringOf(args[0]), stringOf(args[1]), args[2]
	items := asList(args[3])

	keep, err := comparator(op)
	if err != nil {
		return nil, err
	}

	out := make(jsonList, 0, len(items))

	for _, item := range items {
		if keep(fieldOf(item, field), want) {
			out = append(out, item)
		}
	}

	return out, nil
}

func page(args ...any) (any, error) {
	const pageArity = 3

	if len(args) != pageArity {
		return nil, errPageArgs
	}

	items := asList(args[2])

	offset, _ := convertToInt(args[0])
	offset = min(max(offset, 0), len(items))

	limit, _ := convertToInt(args[1])
	if limit <= 0 {
		limit = len(items)
	}

	end := min(offset+limit, len(items))

	return items[offset:end:end], nil
}

func countBy(args ...any) (any, error) {
	const countArity = 2

	if len(args) != countArity {
		return nil, errCountArgs
	}

	field := stringOf(args[0])
	items := asList(args[1])
	counts := make(map[string]int, len(items))

	for _, item := range items {
		counts[stringOf(fieldOf(item, field))]++
	}

	out := make(jsonMap, len(counts))
	for key, count := range counts {
		out[key] = json.Number(strconv.Itoa(count))
	}

	return out, nil
}

func fieldOf(item any, field string) any {
	values := asMap(item)
	if values == nil {
		return item
	}

	return values[field]
}

func comparator(op string) (func(a, b any) bool, error) {
	compare := func(a, b any) int {
		va, okA := convertToFloat64(a)
		vb, okB := convertToFloat64(b)

		if okA && okB {
			return cmp.Compare(va, vb)
		}

		return cmp.Compare(fmt.Sprint(a), fmt.Sprint(b))
	}

	ops := map[string]func(int) bool{
		"eq":  func(r int) bool { return r == 0 },
		"ne":  func(r int) bool { return r != 0 },
		"gt":  func(r int) bool { return r > 0 },
		"gte": func(r int) bool { return r >= 0 },
		"lt":  func(r int) bool { return r < 0 },
		"lte": func(r int) bool { return r <= 0 },
	}

	match, ok := ops[op]
	if !ok {
		return nil, errWhereOp
	}

	return func(a, b any) bool {
		return match(compare(a, b))
	}, nil
}
