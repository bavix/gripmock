package app

import (
	"fmt"
	"sort"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/bavix/gripmock/v3/internal/domain/rest"
)

func (h *RestServer) collectServices(file protoreflect.FileDescriptor, results *[]rest.Service) bool {
	services := file.Services()

	for i := range services.Len() {
		*results = append(*results, h.serviceFromDescriptor(services.Get(i), false))
	}

	return true
}

func (h *RestServer) collectAllServices() []rest.Service {
	results := make([]rest.Service, 0, servicesListCap)

	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		return h.collectServices(file, &results)
	})

	h.restDescriptors.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		return h.collectServices(file, &results)
	})

	sort.Slice(results, func(i, j int) bool {
		return results[i].Id < results[j].Id
	})

	return results
}

func (h *RestServer) serviceFromDescriptor(
	service protoreflect.ServiceDescriptor,
	includeSchemas bool,
) rest.Service {
	methods := service.Methods()
	result := rest.Service{
		Id:      string(service.FullName()),
		Name:    string(service.Name()),
		Package: string(service.ParentFile().Package()),
		Methods: make([]rest.Method, 0, methods.Len()),
	}

	for j := range methods.Len() {
		result.Methods = append(result.Methods, h.methodFromDescriptor(service, methods.Get(j), includeSchemas))
	}

	sort.Slice(result.Methods, func(i, j int) bool {
		return result.Methods[i].Id < result.Methods[j].Id
	})

	return result
}

func (h *RestServer) methodFromDescriptor(
	service protoreflect.ServiceDescriptor,
	method protoreflect.MethodDescriptor,
	includeSchemas bool,
) rest.Method {
	requestType := string(method.Input().FullName())
	responseType := string(method.Output().FullName())

	result := rest.Method{
		Id:              fmt.Sprintf("%s/%s", string(service.FullName()), string(method.Name())),
		Name:            string(method.Name()),
		MethodType:      grpcMethodType(method.IsStreamingClient(), method.IsStreamingServer()),
		RequestType:     &requestType,
		ResponseType:    &responseType,
		ClientStreaming: method.IsStreamingClient(),
		ServerStreaming: method.IsStreamingServer(),
	}

	if includeSchemas {
		result.RequestSchema = h.messageSchemaFromDescriptor(method.Input(), map[protoreflect.FullName]struct{}{})
		result.ResponseSchema = h.messageSchemaFromDescriptor(method.Output(), map[protoreflect.FullName]struct{}{})
	}

	return result
}

func (h *RestServer) messageSchemaFromDescriptor(
	message protoreflect.MessageDescriptor,
	visiting map[protoreflect.FullName]struct{},
) *rest.ProtoMessageSchema {
	fullName := message.FullName()
	if _, ok := visiting[fullName]; ok {
		return &rest.ProtoMessageSchema{
			TypeName:     string(fullName),
			Fields:       []rest.ProtoFieldSchema{},
			RecursiveRef: true,
		}
	}

	visiting[fullName] = struct{}{}
	defer delete(visiting, fullName)

	fields := message.Fields()
	result := rest.ProtoMessageSchema{
		TypeName: string(fullName),
		Fields:   make([]rest.ProtoFieldSchema, 0, fields.Len()),
	}

	for i := range fields.Len() {
		result.Fields = append(result.Fields, h.fieldSchemaFromDescriptor(fields.Get(i), visiting))
	}

	return &result
}

//nolint:funlen
func (h *RestServer) fieldSchemaFromDescriptor(
	field protoreflect.FieldDescriptor,
	visiting map[protoreflect.FullName]struct{},
) rest.ProtoFieldSchema {
	result := rest.ProtoFieldSchema{
		Name:        string(field.Name()),
		JsonName:    field.JSONName(),
		Number:      int(field.Number()),
		Kind:        field.Kind().String(),
		Cardinality: grpcCardinality(field.Cardinality()),
	}

	if oneof := field.ContainingOneof(); oneof != nil && !oneof.IsSynthetic() {
		group := string(oneof.Name())
		result.Oneof = &group
	}

	if field.IsMap() {
		result.Map = true

		keyKind := field.MapKey().Kind().String()
		result.MapKeyKind = &keyKind

		mapValue := field.MapValue()
		valueKind := mapValue.Kind().String()
		result.MapValueKind = &valueKind

		if mapValue.Kind() == protoreflect.MessageKind {
			valueTypeName := string(mapValue.Message().FullName())
			result.MapValueTypeName = &valueTypeName
		}

		if mapValue.Kind() == protoreflect.EnumKind {
			valueTypeName := string(mapValue.Enum().FullName())
			result.MapValueTypeName = &valueTypeName
		}

		if mapValue.Kind() == protoreflect.MessageKind {
			result.MapValueMessage = h.messageSchemaFromDescriptor(mapValue.Message(), visiting)
		}

		return result
	}

	if field.Kind() == protoreflect.EnumKind {
		enumTypeName := string(field.Enum().FullName())
		result.TypeName = &enumTypeName

		enumValues := make([]string, 0, field.Enum().Values().Len())
		for i := range field.Enum().Values().Len() {
			enumValues = append(enumValues, string(field.Enum().Values().Get(i).Name()))
		}

		result.EnumValues = &enumValues

		return result
	}

	if field.Kind() == protoreflect.MessageKind {
		messageTypeName := string(field.Message().FullName())
		result.TypeName = &messageTypeName
		result.Message = h.messageSchemaFromDescriptor(field.Message(), visiting)
	}

	return result
}

func grpcCardinality(cardinality protoreflect.Cardinality) rest.ProtoFieldSchemaCardinality {
	switch cardinality {
	case protoreflect.Required:
		return rest.Required
	case protoreflect.Repeated:
		return rest.Repeated
	case protoreflect.Optional:
		return rest.Optional
	default:
		return rest.Optional
	}
}

func grpcMethodType(clientStreaming bool, serverStreaming bool) rest.MethodMethodType {
	switch {
	case clientStreaming && serverStreaming:
		return rest.BidiStreaming
	case clientStreaming:
		return rest.ClientStreaming
	case serverStreaming:
		return rest.ServerStreaming
	default:
		return rest.Unary
	}
}

// liveness handles the liveness probe response.
