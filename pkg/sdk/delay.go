package sdk

import (
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

// DelayItem wraps response data with a time delay.
// Created via Delay function. Used inside Return, SendStream, Send.
type DelayItem struct {
	Delay time.Duration
	Data  map[string]any
}

// Delay wraps response data with a time delay.
//
// Usage in Return:
//
//	Return(Delay(100*time.Millisecond, "msg", "hello"))
//	Return(Delay(100*ms, &pb.Response{Message: "hi"}))
//	Return(Delay(100*ms, map[string]any{"msg": "hello"}))
//
// Usage in SendStream/Send:
//
//	SendStream(
//	    map[string]any{"msg": "first"},
//	    Delay(200*ms, "msg", "second"),
//	)
//
//	Send(Delay(150*ms, "msg", "third"))
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

// extractDelay extracts a DelayItem from the first argument if present.
// Returns (delay, data) where delay=0 means no delay was requested.
func extractDelay(kv []any, errPrefix string) (time.Duration, map[string]any) {
	if len(kv) == 1 {
		if d, ok := kv[0].(DelayItem); ok {
			return d.Delay, d.Data
		}
	}

	return 0, parseKVPairs(kv, errPrefix)
}

// injectStreamDelay injects _gripmock delay into a stream item.
// If item is a DelayItem, adds the delay marker to the data map.
// Returns the data map ready for the stream.
func injectStreamDelay(item any) any {
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

// extractDelayItem extracts the stream data from kv arguments.
// If first arg is DelayItem, returns the data map with delay injected.
// Otherwise parses KV pairs and returns a map.
func extractDelayItem(kv []any) any {
	if len(kv) == 1 {
		if d, ok := kv[0].(DelayItem); ok {
			return injectStreamDelay(d)
		}
	}

	return parseKVPairs(kv, "sdk.NextWillReturn")
}
