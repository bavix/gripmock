package protobuf

import (
	"encoding/base64"
	"fmt"

	"github.com/goccy/go-json"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ScalarConverter provides unified conversion for protobuf scalar values.
type ScalarConverter struct{}

// NewScalarConverter creates a new converter instance.
func NewScalarConverter() *ScalarConverter {
	return &ScalarConverter{}
}

// ConvertScalar converts a protobuf field value to its Go representation.
//
//nolint:cyclop
func (c *ScalarConverter) ConvertScalar(fd protoreflect.FieldDescriptor, value protoreflect.Value) any {
	const nullValue = "google.protobuf.NullValue"

	switch fd.Kind() {
	case protoreflect.BoolKind:
		return value.Bool()
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return json.Number(value.String())
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return json.Number(value.String())
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return json.Number(value.String())
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return json.Number(value.String())
	case protoreflect.FloatKind:
		return json.Number(value.String())
	case protoreflect.DoubleKind:
		return json.Number(value.String())
	case protoreflect.StringKind:
		return value.String()
	case protoreflect.BytesKind:
		return base64.StdEncoding.EncodeToString(value.Bytes())
	case protoreflect.EnumKind:
		if fd.Enum().FullName() == nullValue {
			return nil
		}

		// Get the enum descriptor for the value
		desc := fd.Enum().Values().ByNumber(value.Enum())
		if desc != nil {
			return string(desc.Name())
		}

		return value.Enum()
	case protoreflect.MessageKind:
		if value.Message().IsValid() {
			return c.ConvertMessage(value.Message().Interface())
		}

		return nil
	case protoreflect.GroupKind:
		// GroupKind is deprecated and not commonly used
		return fmt.Sprintf("group type: %v", fd.Kind())
	default:
		return fmt.Sprintf("unknown type: %v", fd.Kind())
	}
}

// ConvertMessage converts a protobuf message to a map representation.
// This method should be implemented by the caller or extended as needed.
func (c *ScalarConverter) ConvertMessage(msg proto.Message) map[string]any {
	if msg == nil {
		return nil
	}

	result := make(map[string]any)
	message := msg.ProtoReflect()

	message.Range(func(fd protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		if !message.Has(fd) {
			return true
		}

		fieldName := string(fd.Name())

		// Handle different field types
		switch {
		case fd.IsList():
			result[fieldName] = c.convertList(fd, value.List())
		case fd.IsMap():
			result[fieldName] = c.convertMap(fd, value.Map())
		default:
			result[fieldName] = c.ConvertScalar(fd, value)
		}

		return true
	})

	return result
}

// convertList converts a protobuf list to a Go slice.
func (c *ScalarConverter) convertList(fd protoreflect.FieldDescriptor, list protoreflect.List) []any {
	result := make([]any, list.Len())

	for i := range list.Len() {
		result[i] = c.ConvertScalar(fd, list.Get(i))
	}

	return result
}

// convertMap converts a protobuf map to a Go map.
func (c *ScalarConverter) convertMap(fd protoreflect.FieldDescriptor, mapVal protoreflect.Map) map[string]any {
	result := make(map[string]any)

	mapVal.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
		keyStr := key.String()
		result[keyStr] = c.ConvertScalar(fd.MapValue(), value)

		return true
	})

	return result
}
