package app

import (
	"encoding/base64"
	"fmt"

	"github.com/goccy/go-json"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// MessageConverter provides methods for converting protobuf messages to map representations.
type MessageConverter struct{}

// NewMessageConverter creates a new MessageConverter instance.
func NewMessageConverter() *MessageConverter {
	return &MessageConverter{}
}

// ConvertToMap converts a protobuf message to a map[string]any representation.
func (c *MessageConverter) ConvertToMap(msg proto.Message) map[string]any {
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
		result[fieldName] = c.convertValue(fd, value)

		return true
	})

	return result
}

// convertValue converts a protobuf field value to its Go representation.
func (c *MessageConverter) convertValue(fd protoreflect.FieldDescriptor, value protoreflect.Value) any {
	switch {
	case fd.IsList():
		return c.convertList(fd, value.List())
	case fd.IsMap():
		return c.convertMap(fd, value.Map())
	default:
		return c.convertScalar(fd, value)
	}
}

// convertList converts a protobuf list to a Go slice.
func (c *MessageConverter) convertList(fd protoreflect.FieldDescriptor, list protoreflect.List) []any {
	result := make([]any, list.Len())
	elemType := fd.Message()

	for i := range list.Len() {
		elem := list.Get(i)

		if elemType != nil {
			result[i] = c.ConvertToMap(elem.Message().Interface())
		} else {
			result[i] = c.convertScalar(fd, elem)
		}
	}

	return result
}

// convertMap converts a protobuf map to a Go map.
func (c *MessageConverter) convertMap(fd protoreflect.FieldDescriptor, m protoreflect.Map) map[string]any {
	result := make(map[string]any)
	keyType := fd.MapKey()
	valType := fd.MapValue().Message()

	m.Range(func(key protoreflect.MapKey, val protoreflect.Value) bool {
		convertedKey, ok := c.convertScalar(keyType, key.Value()).(string)
		if !ok {
			return true
		}

		if valType != nil {
			result[convertedKey] = c.ConvertToMap(val.Message().Interface())
		} else {
			result[convertedKey] = c.convertScalar(fd.MapValue(), val)
		}

		return true
	})

	return result
}

// convertScalar converts a protobuf scalar value to its Go representation.
//
//nolint:cyclop
func (c *MessageConverter) convertScalar(fd protoreflect.FieldDescriptor, value protoreflect.Value) any {
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
			return c.ConvertToMap(value.Message().Interface())
		}

		return nil
	case protoreflect.GroupKind:
		// GroupKind is deprecated and not commonly used
		return fmt.Sprintf("group type: %v", fd.Kind())
	default:
		return fmt.Sprintf("unknown type: %v", fd.Kind())
	}
}
