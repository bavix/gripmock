package protoconv

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/cockroachdb/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

var (
	errExpectedMap       = errors.New("expected map")
	errExpectedArray     = errors.New("expected array")
	errExpectedString    = errors.New("expected string")
	errExpectedBool      = errors.New("expected bool")
	errExpectedNumber    = errors.New("expected number")
	errExpectedEnumValue = errors.New("expected enum value")
	errUnsupportedMapKey = errors.New("unsupported map key kind")
	errUnsupportedKind   = errors.New("unsupported kind")
	errUnknownEnum       = errors.New("unknown enum")
)

const wktPrefix = "google.protobuf."

func MapToProto(desc protoreflect.MessageDescriptor, data map[string]any) (*dynamicpb.Message, error) {
	msg := dynamicpb.NewMessage(desc)
	if err := SetMessageFromMap(msg.ProtoReflect(), data); err != nil {
		return nil, err
	}

	return msg, nil
}

func SetMessageFromMap(msg protoreflect.Message, data map[string]any) error {
	if len(data) == 0 {
		return nil
	}

	fields := msg.Descriptor().Fields()

	for k, v := range data {
		if v == nil {
			continue
		}

		fd := resolveField(fields, k)
		if fd == nil {
			continue
		}

		if err := setField(msg, fd, v); err != nil {
			return fmt.Errorf("field %q: %w", k, err)
		}
	}

	return nil
}

//nolint:ireturn
func resolveField(fields protoreflect.FieldDescriptors, name string) protoreflect.FieldDescriptor {
	if fd := fields.ByName(protoreflect.Name(name)); fd != nil {
		return fd
	}

	return fields.ByJSONName(name)
}

func setField(msg protoreflect.Message, fd protoreflect.FieldDescriptor, value any) error {
	switch {
	case fd.IsMap():
		return setMapField(msg, fd, value)
	case fd.IsList():
		return setListField(msg, fd, value)
	case fd.Kind() == protoreflect.MessageKind && isWKT(fd.Message()):
		return setWKTField(msg, fd, value)
	default:
		return setScalarField(msg, fd, value)
	}
}

func isWKT(md protoreflect.MessageDescriptor) bool {
	return strings.HasPrefix(string(md.FullName()), wktPrefix)
}

func setWKTField(msg protoreflect.Message, fd protoreflect.FieldDescriptor, value any) error {
	pv, err := wktValueFromAny(fd.Message(), value)
	if err != nil {
		return err
	}

	msg.Set(fd, pv)

	return nil
}

// wktValueFromAny converts a value to a protoreflect.Value for a WKT message via protojson.
func wktValueFromAny(md protoreflect.MessageDescriptor, v any) (protoreflect.Value, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return protoreflect.Value{}, err
	}

	wktMsg := dynamicpb.NewMessage(md)
	if err := protojson.Unmarshal(b, wktMsg); err != nil {
		return protoreflect.Value{}, err
	}

	return protoreflect.ValueOfMessage(wktMsg.ProtoReflect()), nil
}

// messageValueFromAny converts a value to a protoreflect.Value for a regular (non-WKT) message.
func messageValueFromAny(md protoreflect.MessageDescriptor, v any) (protoreflect.Value, error) {
	itemData, ok := v.(map[string]any)
	if !ok {
		return protoreflect.Value{}, errors.Wrapf(errExpectedMap, "got %T", v)
	}

	itemMsg := dynamicpb.NewMessage(md)
	if err := SetMessageFromMap(itemMsg.ProtoReflect(), itemData); err != nil {
		return protoreflect.Value{}, err
	}

	return protoreflect.ValueOfMessage(itemMsg.ProtoReflect()), nil
}

func setMapField(msg protoreflect.Message, fd protoreflect.FieldDescriptor, value any) error {
	m, ok := value.(map[string]any)
	if !ok {
		return errors.Wrapf(errExpectedMap, "got %T", value)
	}

	keyDesc := fd.MapKey()
	valDesc := fd.MapValue()
	valMsg := valDesc.Message()
	resultMap := msg.Mutable(fd).Map()

	for k, v := range m {
		if v == nil {
			continue
		}

		kv, err := mapKeyValue(keyDesc, k)
		if err != nil {
			return err
		}

		mk := kv.MapKey()

		pv, err := elementValue(valDesc, valMsg, v)
		if err != nil {
			return err
		}

		resultMap.Set(mk, pv)
	}

	return nil
}

//nolint:exhaustive
func mapKeyValue(fd protoreflect.FieldDescriptor, key string) (protoreflect.Value, error) {
	switch fd.Kind() {
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(key), nil
	case protoreflect.BoolKind:
		b, err := strconv.ParseBool(key)
		if err != nil {
			return protoreflect.Value{}, err
		}

		return protoreflect.ValueOfBool(b), nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		n, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}

		return intKindValue(fd, n), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		n, err := strconv.ParseUint(key, 10, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}

		return uintKindValue(fd, n), nil
	default:
		return protoreflect.Value{}, errors.Wrapf(errUnsupportedMapKey, "%v", fd.Kind())
	}
}

func setListField(msg protoreflect.Message, fd protoreflect.FieldDescriptor, value any) error {
	arr, ok := value.([]any)
	if !ok {
		return errors.Wrapf(errExpectedArray, "got %T", value)
	}

	list := msg.Mutable(fd).List()
	elemMsg := fd.Message()

	for i, elem := range arr {
		if elem == nil {
			continue
		}

		pv, err := elementValue(fd, elemMsg, elem)
		if err != nil {
			return fmt.Errorf("list[%d]: %w", i, err)
		}

		list.Append(pv)
	}

	return nil
}

// elementValue resolves a single element value for map values and list elements.
// md is non-nil for message-typed elements, nil for scalar.
func elementValue(fd protoreflect.FieldDescriptor, md protoreflect.MessageDescriptor, v any) (protoreflect.Value, error) {
	if md != nil {
		if isWKT(md) {
			return wktValueFromAny(md, v)
		}

		return messageValueFromAny(md, v)
	}

	return scalarToValue(fd, v)
}

func setScalarField(msg protoreflect.Message, fd protoreflect.FieldDescriptor, value any) error {
	if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
		pv, err := messageValueFromAny(fd.Message(), value)
		if err != nil {
			return err
		}

		msg.Set(fd, pv)

		return nil
	}

	pv, err := scalarToValue(fd, value)
	if err != nil {
		return err
	}

	msg.Set(fd, pv)

	return nil
}

//nolint:cyclop,exhaustive
func scalarToValue(fd protoreflect.FieldDescriptor, value any) (protoreflect.Value, error) {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		return boolToValue(value)
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		n, err := jsonNumberInt64(value)
		if err != nil {
			return protoreflect.Value{}, err
		}

		return intKindValue(fd, n), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		n, err := jsonNumberUint64(value)
		if err != nil {
			return protoreflect.Value{}, err
		}

		return uintKindValue(fd, n), nil
	case protoreflect.FloatKind:
		f, err := jsonNumberFloat64(value)
		if err != nil {
			return protoreflect.Value{}, err
		}

		return protoreflect.ValueOfFloat32(float32(f)), nil
	case protoreflect.DoubleKind:
		f, err := jsonNumberFloat64(value)
		if err != nil {
			return protoreflect.Value{}, err
		}

		return protoreflect.ValueOfFloat64(f), nil
	case protoreflect.StringKind:
		s, ok := value.(string)
		if !ok {
			return protoreflect.Value{}, errors.Wrapf(errExpectedString, "got %T", value)
		}

		return protoreflect.ValueOfString(s), nil
	case protoreflect.BytesKind:
		s, ok := value.(string)
		if !ok {
			return protoreflect.Value{}, errors.Wrapf(errExpectedString, "got %T", value)
		}

		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return protoreflect.Value{}, err
		}

		return protoreflect.ValueOfBytes(b), nil
	case protoreflect.EnumKind:
		return enumToValue(fd, value)
	default:
		return protoreflect.Value{}, errors.Wrapf(errUnsupportedKind, "%v", fd.Kind())
	}
}

func boolToValue(value any) (protoreflect.Value, error) {
	switch v := value.(type) {
	case bool:
		return protoreflect.ValueOfBool(v), nil
	case string:
		b, err := strconv.ParseBool(v)
		if err != nil {
			return protoreflect.Value{}, err
		}

		return protoreflect.ValueOfBool(b), nil
	}

	return protoreflect.Value{}, errors.Wrapf(errExpectedBool, "got %T", value)
}

func jsonNumberInt64(value any) (int64, error) {
	switch v := value.(type) {
	case json.Number:
		return v.Int64()
	case float64:
		return int64(v), nil
	case int:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	}

	return 0, errors.Wrapf(errExpectedNumber, "got %T", value)
}

func jsonNumberUint64(value any) (uint64, error) {
	switch v := value.(type) {
	case json.Number:
		return strconv.ParseUint(string(v), 10, 64)
	case float64:
		return uint64(v), nil
	case int:
		return uint64(v), nil //nolint:gosec
	case uint:
		return uint64(v), nil
	case string:
		return strconv.ParseUint(v, 10, 64)
	}

	return 0, errors.Wrapf(errExpectedNumber, "got %T", value)
}

func jsonNumberFloat64(value any) (float64, error) {
	switch v := value.(type) {
	case json.Number:
		return v.Float64()
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	}

	return 0, errors.Wrapf(errExpectedNumber, "got %T", value)
}

func enumToValue(fd protoreflect.FieldDescriptor, value any) (protoreflect.Value, error) {
	switch v := value.(type) {
	case string:
		if desc := fd.Enum().Values().ByName(protoreflect.Name(v)); desc != nil {
			return protoreflect.ValueOfEnum(desc.Number()), nil
		}

		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return protoreflect.Value{}, errors.Wrapf(errUnknownEnum, "%s: %q", fd.Enum().FullName(), v)
		}

		return protoreflect.ValueOfEnum(protoreflect.EnumNumber(n)), nil
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return protoreflect.Value{}, err
		}

		return protoreflect.ValueOfEnum(protoreflect.EnumNumber(n)), nil //nolint:gosec
	case float64:
		return protoreflect.ValueOfEnum(protoreflect.EnumNumber(v)), nil
	}

	return protoreflect.Value{}, errors.Wrapf(errExpectedEnumValue, "got %T", value)
}

//nolint:exhaustive
func intKindValue(fd protoreflect.FieldDescriptor, n int64) protoreflect.Value {
	switch fd.Kind() {
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return protoreflect.ValueOfInt32(int32(n)) //nolint:gosec
	default:
		return protoreflect.ValueOfInt64(n)
	}
}

//nolint:exhaustive
func uintKindValue(fd protoreflect.FieldDescriptor, n uint64) protoreflect.Value {
	switch fd.Kind() {
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return protoreflect.ValueOfUint32(uint32(n)) //nolint:gosec
	default:
		return protoreflect.ValueOfUint64(n)
	}
}
