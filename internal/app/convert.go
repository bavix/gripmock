package app

import (
	"encoding/base64"
	"strconv"

	"github.com/goccy/go-json"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type convertScope struct {
	seen  map[protoreflect.Message]struct{}
	depth int
	max   int
}

func newConvertScope(maxDepth int) *convertScope {
	if maxDepth <= 0 {
		maxDepth = defaultConvertDepth
	}

	return &convertScope{
		seen: make(map[protoreflect.Message]struct{}),
		max:  maxDepth,
	}
}

func (c *convertScope) enter(msg protoreflect.Message) bool {
	if msg == nil || !msg.IsValid() {
		return false
	}

	if c.depth >= c.max {
		return false
	}

	if _, ok := c.seen[msg]; ok {
		return false
	}

	c.seen[msg] = struct{}{}
	c.depth++

	return true
}

func (c *convertScope) exit() {
	c.depth--
}

func convertToMap(msg proto.Message) map[string]any {
	if msg == nil {
		return nil
	}

	return convertToMapVisited(msg.ProtoReflect(), newConvertScope(defaultConvertDepth))
}

func convertToMapWithDepth(msg proto.Message, maxDepth int) map[string]any {
	if msg == nil {
		return nil
	}

	return convertToMapVisited(msg.ProtoReflect(), newConvertScope(maxDepth))
}

func convertToMapVisited(message protoreflect.Message, scope *convertScope) map[string]any {
	if !scope.enter(message) {
		return nil
	}
	defer scope.exit()

	desc := message.Descriptor()
	result := make(map[string]any, desc.Fields().Len())

	for i := range desc.Fields().Len() {
		fd := desc.Fields().Get(i)

		if fd.Cardinality() == protoreflect.Repeated && !message.Has(fd) {
			continue
		}

		fieldName := string(fd.Name())
		result[fieldName] = convertValueVisited(fd, message.Get(fd), scope)
	}

	return result
}

func convertValueVisited(fd protoreflect.FieldDescriptor, value protoreflect.Value, scope *convertScope) any {
	switch {
	case fd.IsList():
		return convertListVisited(fd, value.List(), scope)
	case fd.IsMap():
		return convertMapVisited(fd, value.Map(), scope)
	default:
		return convertScalarVisited(fd, value, scope)
	}
}

func convertListVisited(fd protoreflect.FieldDescriptor, list protoreflect.List, scope *convertScope) []any {
	result := make([]any, list.Len())
	elemType := fd.Message()

	for i := range list.Len() {
		elem := list.Get(i)

		if elemType != nil {
			if m := elem.Message(); m.IsValid() {
				result[i] = convertToMapVisited(m, scope)
			}
		} else {
			result[i] = convertScalarVisited(fd, elem, scope)
		}
	}

	return result
}

func convertMapVisited(fd protoreflect.FieldDescriptor, m protoreflect.Map, scope *convertScope) map[string]any {
	result := make(map[string]any)
	keyType := fd.MapKey()
	valType := fd.MapValue().Message()

	m.Range(func(key protoreflect.MapKey, val protoreflect.Value) bool {
		convertedKey, ok := convertScalar(keyType, key.Value()).(string)
		if !ok {
			return true
		}

		if valType != nil {
			if m := val.Message(); m.IsValid() {
				result[convertedKey] = convertToMapVisited(m, scope)
			}
		} else {
			result[convertedKey] = convertScalar(fd.MapValue(), val)
		}

		return true
	})

	return result
}

func convertScalar(fd protoreflect.FieldDescriptor, value protoreflect.Value) any {
	return convertScalarVisited(fd, value, nil)
}

//nolint:cyclop
func convertScalarVisited(fd protoreflect.FieldDescriptor, value protoreflect.Value, scope *convertScope) any {
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
		return float64(value.Float())
	case protoreflect.DoubleKind:
		return value.Float()
	case protoreflect.StringKind:
		return value.String()
	case protoreflect.BytesKind:
		return base64.StdEncoding.EncodeToString(value.Bytes())
	case protoreflect.EnumKind:
		if fd.Enum().FullName() == nullValue {
			return nil
		}

		desc := fd.Enum().Values().ByNumber(value.Enum())
		if desc != nil {
			return string(desc.Name())
		}

		return ""
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if scope == nil {
			return convertToMap(value.Message().Interface())
		}

		m := value.Message()
		if !m.IsValid() {
			return nil
		}

		return convertToMapVisited(m, scope)
	default:
		return nil
	}
}

func convertMapNumericToStringNumber(data map[string]any, desc protoreflect.MessageDescriptor) map[string]any {
	result := make(map[string]any, len(data))

	for k, v := range data {
		var fd protoreflect.FieldDescriptor
		if desc != nil {
			fd = desc.Fields().ByName(protoreflect.Name(k))
			if fd == nil {
				fd = desc.Fields().ByJSONName(k)
			}
		}

		result[k] = convertMapValue(v, fd)
	}

	return result
}

func convertMapValue(v any, fd protoreflect.FieldDescriptor) any {
	switch val := v.(type) {
	case map[string]any:
		var nestedDesc protoreflect.MessageDescriptor
		if fd != nil && fd.Kind() == protoreflect.MessageKind {
			nestedDesc = fd.Message()
		}

		return convertMapNumericToStringNumber(val, nestedDesc)
	case []any:
		return convertMapArray(val, fd)
	case string:
		return convertStringValue(val, fd)
	case float64:
		return convertFloat64(val)
	case float32:
		return convertFloat64(float64(val))
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return convertIntLikeValue(val)
	default:
		return v
	}
}

func convertIntLikeValue(v any) any {
	switch val := v.(type) {
	case int, int8, int16, int32, int64:
		return json.Number(strconv.FormatInt(toInt64(val), 10))
	default:
		return json.Number(strconv.FormatUint(toUint64(val), 10))
	}
}

func convertStringValue(val string, fd protoreflect.FieldDescriptor) any {
	if fd == nil || !isNumericKind(fd.Kind()) {
		return val
	}

	if is64BitIntKind(fd.Kind()) {
		return val
	}

	if f, err := strconv.ParseFloat(val, 64); err == nil {
		return convertFloat64(f)
	}

	return val
}

func is64BitIntKind(k protoreflect.Kind) bool {
	switch k { //nolint:exhaustive
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return true
	default:
		return false
	}
}

func convertFloat64(f float64) json.Number {
	if isSafeInteger(f) {
		return json.Number(strconv.FormatInt(int64(f), 10))
	}

	return json.Number(strconv.FormatFloat(f, 'g', -1, 64))
}

func toInt64(v any) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int8:
		return int64(val)
	case int16:
		return int64(val)
	case int32:
		return int64(val)
	case int64:
		return val
	default:
		return 0
	}
}

func toUint64(v any) uint64 {
	switch val := v.(type) {
	case uint:
		return uint64(val)
	case uint8:
		return uint64(val)
	case uint16:
		return uint64(val)
	case uint32:
		return uint64(val)
	case uint64:
		return val
	default:
		return 0
	}
}

func isSafeInteger(f float64) bool {
	return f == float64(int64(f))
}

func convertMapArray(arr []any, fd protoreflect.FieldDescriptor) []any {
	result := make([]any, len(arr))

	for i, v := range arr {
		result[i] = convertMapValue(v, fd)
	}

	return result
}

func isNumericKind(k protoreflect.Kind) bool {
	switch k { //nolint:exhaustive
	case protoreflect.DoubleKind, protoreflect.FloatKind,
		protoreflect.Int32Kind, protoreflect.Int64Kind,
		protoreflect.Uint32Kind, protoreflect.Uint64Kind,
		protoreflect.Sint32Kind, protoreflect.Sint64Kind,
		protoreflect.Sfixed32Kind, protoreflect.Sfixed64Kind,
		protoreflect.Fixed32Kind, protoreflect.Fixed64Kind:
		return true
	default:
		return false
	}
}
