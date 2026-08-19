package sdk

import (
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// DelayItem wraps response data with a time delay.
type DelayItem struct {
	Delay time.Duration
	Data  map[string]any
}

// Delay wraps response data with a time delay.
func Delay(d time.Duration, kv ...any) DelayItem {
	if len(kv) == 1 {
		if msg, ok := kv[0].(proto.Message); ok {
			return DelayItem{Delay: d, Data: protoToMap(msg)}
		}

		if m, ok := kv[0].(map[string]any); ok {
			return DelayItem{Delay: d, Data: m}
		}
	}

	return DelayItem{Delay: d, Data: parseKVPairs(kv, "sdk.Delay")}
}

// StreamErrorItem aborts a stream at this position instead of sending a message.
type StreamErrorItem struct {
	Code    codes.Code
	Message string
	Details []map[string]any
}

func StreamError(code codes.Code, msg string, details ...map[string]any) StreamErrorItem {
	return StreamErrorItem{Code: code, Message: msg, Details: details}
}

func (e StreamErrorItem) element() map[string]any {
	meta := map[string]any{"error": e.Message, "code": uint32(e.Code)}

	if len(e.Details) > 0 {
		details := make([]any, len(e.Details))
		for i, d := range e.Details {
			details[i] = d
		}

		meta["details"] = details
	}

	return map[string]any{stuber.GripMockKey: meta}
}

func extractDelay(kv []any, errPrefix string) (time.Duration, map[string]any) {
	if len(kv) == 1 {
		if d, ok := kv[0].(DelayItem); ok {
			return d.Delay, d.Data
		}
	}

	return 0, parseKVPairs(kv, errPrefix)
}

func injectStreamDelay(item any) any {
	if e, ok := item.(StreamErrorItem); ok {
		return e.element()
	}

	if d, ok := item.(DelayItem); ok {
		m := d.Data
		if m == nil {
			m = map[string]any{}
		}

		m[stuber.GripMockKey] = map[string]any{"delay": d.Delay.String()}

		return m
	}

	return item
}

func extractDelayItem(kv []any) any {
	if len(kv) == 1 {
		return streamElement(kv[0], "sdk.NextWillReturn")
	}

	return parseKVPairs(kv, "sdk.NextWillReturn")
}

func streamElement(v any, errPrefix string) any {
	switch item := v.(type) {
	case StreamErrorItem:
		return item.element()
	case DelayItem:
		return injectStreamDelay(item)
	case string:
		return parseKVPairs([]any{item}, errPrefix)
	default:
		return item
	}
}
