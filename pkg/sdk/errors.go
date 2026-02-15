package sdk

import "errors"

var (
	ErrDescriptorsRequired = errors.New("gripmock: descriptors required (use WithDescriptors or MockFrom)")
	ErrNoUsableServicesFoundViaReflection = errors.New("no services found via reflection (or only grpc.reflection/grpc.health)")
	ErrUnexpectedResponse = errors.New("unexpected response: not ListServicesResponse")
)