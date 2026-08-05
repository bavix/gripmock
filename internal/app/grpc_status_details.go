package app

import (
	"fmt"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/goccy/go-json"
	_ "google.golang.org/genproto/googleapis/rpc/errdetails" // registers errdetails message types for status detail unmarshalling
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"

	protosetinfra "github.com/bavix/gripmock/v3/internal/infra/protoset"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

var (
	errDetailTypeRequired = errors.New("field 'type' is required")
	errDetailTypeNonEmpty = errors.New("field 'type' must be a non-empty string")
	errInvalidDetailType  = errors.New("invalid detail type URL")
	errUnknownDetailType  = errors.New("unknown detail type")
	errDetailUnmarshal    = errors.New("failed to unmarshal detail payload")
)

func (m *grpcMocker) statusFromOutput(output stuber.Output) (*status.Status, error) {
	return statusFromOutputWithDetails(output, m.typeResolver)
}

//nolint:nilnil
func statusFromOutputWithDetails(output stuber.Output, resolver *protosetinfra.TypeResolver) (*status.Status, error) {
	st := outputStatusBase(output)
	if st == nil {
		return nil, nil
	}

	return attachDetails(st, output.Details, resolver)
}

func attachDetails(st *status.Status, details []map[string]any, resolver *protosetinfra.TypeResolver) (*status.Status, error) {
	if len(details) == 0 {
		return st, nil
	}

	anyDetails := make([]*anypb.Any, 0, len(details))

	for i, detail := range details {
		msg, err := detailMessage(detail, resolver)
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
func detailMessage(detail map[string]any, resolver *protosetinfra.TypeResolver) (proto.Message, error) {
	typeURLRaw, ok := detail["type"]
	if !ok {
		return nil, errDetailTypeRequired
	}

	typeURL, ok := typeURLRaw.(string)
	if !ok || strings.TrimSpace(typeURL) == "" {
		return nil, errDetailTypeNonEmpty
	}

	desc, err := resolveMessageDescriptor(typeURL, resolver)
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
			if protojson.Unmarshal(valueData, msg) == nil {
				return msg, nil
			}
		}
	}

	return nil, fmt.Errorf("%w to %s", errDetailUnmarshal, desc.FullName())
}

//nolint:ireturn
func resolveMessageDescriptor(typeURL string, resolver *protosetinfra.TypeResolver) (protoreflect.MessageDescriptor, error) {
	fullName := protosetinfra.ParseTypeURL(typeURL)
	if fullName == "" {
		return nil, fmt.Errorf("%w: %q", errInvalidDetailType, typeURL)
	}

	msgType, err := resolver.FindMessageByName(fullName)
	if err != nil {
		return nil, fmt.Errorf("%w: %q", errUnknownDetailType, fullName)
	}

	return msgType.Descriptor(), nil
}
