package app

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/bavix/gripmock/v3/internal/infra/jsondecoder"
)

func isWellKnown(name protoreflect.FullName) bool {
	switch name {
	case "google.protobuf.Timestamp", "google.protobuf.Duration",
		"google.protobuf.Struct", "google.protobuf.Value", "google.protobuf.ListValue",
		"google.protobuf.DoubleValue", "google.protobuf.FloatValue",
		"google.protobuf.Int64Value", "google.protobuf.UInt64Value",
		"google.protobuf.Int32Value", "google.protobuf.UInt32Value",
		"google.protobuf.BoolValue", "google.protobuf.StringValue", "google.protobuf.BytesValue":
		return true
	default:
		return false
	}
}

func convertMessageVisited(message protoreflect.Message, scope *convertScope) any {
	if known, ok := wellKnownValue(message); ok {
		return known
	}

	return convertToMapVisited(message, scope)
}

func wellKnownValue(message protoreflect.Message) (any, bool) {
	if !isWellKnown(message.Descriptor().FullName()) {
		return nil, false
	}

	encoded, err := protojson.Marshal(message.Interface())
	if err != nil {
		return nil, false
	}

	var decoded any

	err = jsondecoder.Unmarshal(encoded, &decoded)
	if err != nil {
		return nil, false
	}

	return decoded, true
}
