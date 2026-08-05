package protoset

import (
	"strings"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

type TypeResolver struct {
	resolver protodesc.Resolver
}

//nolint:gochecknoglobals
var globalTypeResolver = &TypeResolver{}

func NewTypeResolver(resolver protodesc.Resolver) *TypeResolver {
	if resolver == nil {
		return globalTypeResolver
	}

	return &TypeResolver{resolver: resolver}
}

func ParseTypeURL(typeURL string) protoreflect.FullName {
	typeURL = strings.TrimSpace(typeURL)
	if idx := strings.LastIndex(typeURL, "/"); idx >= 0 {
		typeURL = typeURL[idx+1:]
	}

	return protoreflect.FullName(strings.TrimPrefix(typeURL, "."))
}

//nolint:ireturn
func (r *TypeResolver) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageType, error) {
	if desc := r.findDescriptor(name); desc != nil {
		if msgDesc, ok := desc.(protoreflect.MessageDescriptor); ok {
			return dynamicpb.NewMessageType(msgDesc), nil
		}
	}

	return protoregistry.GlobalTypes.FindMessageByName(name)
}

//nolint:ireturn
func (r *TypeResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	name := ParseTypeURL(url)
	if name == "" {
		return nil, protoregistry.NotFound
	}

	return r.FindMessageByName(name)
}

//nolint:ireturn
func (r *TypeResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	if desc := r.findDescriptor(field); desc != nil {
		if extDesc, ok := desc.(protoreflect.ExtensionDescriptor); ok && extDesc.IsExtension() {
			return dynamicpb.NewExtensionType(extDesc), nil
		}
	}

	return protoregistry.GlobalTypes.FindExtensionByName(field)
}

//nolint:ireturn
func (r *TypeResolver) FindExtensionByNumber(
	message protoreflect.FullName,
	field protoreflect.FieldNumber,
) (protoreflect.ExtensionType, error) {
	return protoregistry.GlobalTypes.FindExtensionByNumber(message, field)
}

//nolint:ireturn
func (r *TypeResolver) findDescriptor(name protoreflect.FullName) protoreflect.Descriptor {
	if r != nil && r.resolver != nil {
		if desc, err := r.resolver.FindDescriptorByName(name); err == nil {
			return desc
		}
	}

	if desc, err := protoregistry.GlobalFiles.FindDescriptorByName(name); err == nil {
		return desc
	}

	return nil
}
