package protoresolver

import (
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type Fallback struct {
	Primary  protodesc.Resolver
	Fallback protodesc.Resolver
}

//nolint:ireturn
func (r *Fallback) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	if r.Primary != nil {
		if fd, err := r.Primary.FindFileByPath(path); err == nil {
			return fd, nil
		}
	}

	if r.Fallback != nil {
		return r.Fallback.FindFileByPath(path)
	}

	return nil, protoregistry.NotFound
}

//nolint:ireturn
func (r *Fallback) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	if r.Primary != nil {
		if desc, err := r.Primary.FindDescriptorByName(name); err == nil {
			return desc, nil
		}
	}

	if r.Fallback != nil {
		return r.Fallback.FindDescriptorByName(name)
	}

	return nil, protoregistry.NotFound
}
