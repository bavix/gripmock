package app

import (
	"fmt"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/goccy/go-json"
	_ "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

var (
	errDetailTypeRequired = errors.New("field 'type' is required")
	errDetailTypeNonEmpty = errors.New("field 'type' must be a non-empty string")
	errInvalidDetailType  = errors.New("invalid detail type URL")
	errDetailNotMessage   = errors.New("detail type is not a message")
	errUnknownDetailType  = errors.New("unknown detail type")
	errDetailUnmarshal    = errors.New("failed to unmarshal detail payload")
)

//nolint:nilnil
func (m *grpcMocker) statusFromOutput(output stuber.Output) (*status.Status, error) {
	if output.Error == "" && output.Code == nil {
		return nil, nil
	}

	if output.Code != nil && *output.Code == codes.OK {
		return nil, nil
	}

	code := codes.Aborted
	if output.Code != nil {
		code = *output.Code
	}

	st := status.New(code, output.Error)

	return m.attachDetails(st, output.Details)
}

func (m *grpcMocker) attachDetails(st *status.Status, details []map[string]any) (*status.Status, error) {
	if len(details) == 0 {
		return st, nil
	}

	anyDetails := make([]*anypb.Any, 0, len(details))

	for i, detail := range details {
		msg, err := m.detailMessage(detail)
		if err != nil {
			return nil, fmt.Errorf("invalid output.details[%d]: %w", i, err)
		}

		anyDetail, err := anypb.New(msg)
		if err != nil {
			return nil, fmt.Errorf("failed to convert detail to Any: %w", err)
		}

		anyDetails = append(anyDetails, anyDetail)
	}

	stProto := st.Proto()
	stProto.Details = append(stProto.Details, anyDetails...)

	return status.FromProto(stProto), nil
}

//nolint:cyclop,ireturn
func (m *grpcMocker) detailMessage(detail map[string]any) (proto.Message, error) {
	typeURLRaw, ok := detail["type"]
	if !ok {
		return nil, errDetailTypeRequired
	}

	typeURL, ok := typeURLRaw.(string)
	if !ok || strings.TrimSpace(typeURL) == "" {
		return nil, errDetailTypeNonEmpty
	}

	desc, err := m.resolveMessageDescriptor(typeURL)
	if err != nil {
		return nil, err
	}

	payload := deepCopyMapAny(detail)
	delete(payload, "type")

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal detail payload: %w", err)
	}

	msg := dynamicpb.NewMessage(desc)
	if err := protojson.Unmarshal(data, msg); err == nil {
		return msg, nil
	}

	if value, hasValue := payload["value"]; hasValue && len(payload) == 1 {
		valueData, marshalErr := json.Marshal(value)
		if marshalErr == nil {
			if fallbackErr := protojson.Unmarshal(valueData, msg); fallbackErr == nil {
				return msg, nil
			}
		}
	}

	return nil, fmt.Errorf("%w to %s", errDetailUnmarshal, desc.FullName())
}

//nolint:ireturn
func (m *grpcMocker) resolveMessageDescriptor(typeURL string) (protoreflect.MessageDescriptor, error) {
	fullName := parseTypeURL(typeURL)
	if fullName == "" {
		return nil, fmt.Errorf("%w: %q", errInvalidDetailType, typeURL)
	}

	if m.descriptorResolver != nil {
		desc, err := m.descriptorResolver.FindDescriptorByName(fullName)
		if err == nil {
			if msgDesc, ok := desc.(protoreflect.MessageDescriptor); ok {
				return msgDesc, nil
			}
		}
	}

	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(fullName)
	if err == nil {
		msgDesc, ok := desc.(protoreflect.MessageDescriptor)
		if !ok {
			return nil, fmt.Errorf("%w: %q", errDetailNotMessage, fullName)
		}

		return msgDesc, nil
	}

	msgType, typeErr := protoregistry.GlobalTypes.FindMessageByName(fullName)
	if typeErr == nil {
		return msgType.Descriptor(), nil
	}

	return nil, fmt.Errorf("%w: %q", errUnknownDetailType, fullName)
}

func parseTypeURL(typeURL string) protoreflect.FullName {
	typeURL = strings.TrimSpace(typeURL)
	if typeURL == "" {
		return ""
	}

	if idx := strings.LastIndex(typeURL, "/"); idx >= 0 {
		typeURL = typeURL[idx+1:]
	}

	typeURL = strings.TrimPrefix(typeURL, ".")

	return protoreflect.FullName(typeURL)
}
