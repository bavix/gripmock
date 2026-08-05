package app

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/bavix/gripmock/v3/internal/domain/protoset"
)

// anyResponseResolver compiles wkt_any_response.proto (which defines a service
// returning google.protobuf.Any in the response) and returns GlobalFiles as
// the resolver — Build registers all files including well-known dependencies.
func anyResponseResolver(t *testing.T) protodesc.Resolver {
	t.Helper()

	_, err := protoset.Build(t.Context(), nil, []string{
		filepath.Join("..", "..", "examples", "types", "well-known-types", "wkt_any_response.proto"),
	}, nil)
	require.NoError(t, err)

	return protoregistry.GlobalFiles
}

// anyResponseMocker returns a grpcMocker configured for the GetWrapped method
// of wkt.AnyResponseService. If resolver is nil, descriptorResolver is unset.
func anyResponseMocker(t *testing.T, resolver protodesc.Resolver) *grpcMocker {
	t.Helper()

	if resolver == nil {
		resolver = anyResponseResolver(t)
	}

	svcDesc, err := resolver.FindDescriptorByName("wkt.AnyResponseService")
	require.NoError(t, err)

	method := svcDesc.(protoreflect.ServiceDescriptor).Methods().ByName("GetWrapped")
	require.NotNil(t, method)

	return &grpcMocker{outputDesc: method.Output(), descriptorResolver: resolver}
}

func TestDescriptorTypeResolver_FindMessageByURL(t *testing.T) {
	t.Parallel()

	resolver := &descriptorTypeResolver{resolver: anyResponseResolver(t)}

	tests := []struct {
		name string
		url  string
		want protoreflect.FullName
	}{
		{"custom type", "type.googleapis.com/wkt.CustomResult", "wkt.CustomResult"},
		{"well-known type", "type.googleapis.com/google.protobuf.Any", "google.protobuf.Any"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mt, err := resolver.FindMessageByURL(tt.url)
			require.NoError(t, err)
			assert.Equal(t, tt.want, mt.Descriptor().FullName())
		})
	}

	t.Run("unknown type", func(t *testing.T) {
		t.Parallel()

		_, err := resolver.FindMessageByName("nonexistent.Message")
		require.Error(t, err)
	})
}

func TestDescriptorTypeResolver_ExtensionsNotSupported(t *testing.T) {
	t.Parallel()

	resolver := &descriptorTypeResolver{resolver: anyResponseResolver(t)}

	_, err := resolver.FindExtensionByName("any.Extension")
	require.ErrorIs(t, err, protoregistry.NotFound)

	_, err = resolver.FindExtensionByNumber("any.Message", 1)
	require.ErrorIs(t, err, protoregistry.NotFound)
}

func TestNewOutputMessage_AnyTypeResolution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		typeURL   string
		fields    map[string]any
		wantType  string
		wantField string
		wantValue any
	}{
		{
			name:      "custom type",
			typeURL:   "type.googleapis.com/wkt.CustomResult",
			fields:    map[string]any{"value": "hello", "code": 42},
			wantType:  "type.googleapis.com/wkt.CustomResult",
			wantField: "value",
			wantValue: "hello",
		},
		{
			name:      "well-known type",
			typeURL:   "type.googleapis.com/google.protobuf.StringValue",
			fields:    map[string]any{"value": "hello-wkt"},
			wantType:  "type.googleapis.com/google.protobuf.StringValue",
			wantField: "value",
			wantValue: "hello-wkt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			files := anyResponseResolver(t)
			mocker := anyResponseMocker(t, files)

			stubResult := map[string]any{"@type": tt.typeURL}
			for k, v := range tt.fields {
				stubResult[k] = v
			}

			msg, err := mocker.newOutputMessage(map[string]any{
				"name":   "op-1",
				"done":   true,
				"result": stubResult,
			})
			require.NoError(t, err)

			jsonBytes, err := protojson.MarshalOptions{
				Resolver: &descriptorTypeResolver{resolver: files},
			}.Marshal(msg)
			require.NoError(t, err)

			var raw map[string]any
			require.NoError(t, json.Unmarshal(jsonBytes, &raw))

			result := raw["result"].(map[string]any)
			assert.Equal(t, tt.wantType, result["@type"])
			assert.Equal(t, tt.wantValue, result[tt.wantField])
		})
	}
}

func TestNewOutputMessage_AnyWithoutResolver(t *testing.T) {
	t.Parallel()

	files := anyResponseResolver(t)

	svcDesc, err := files.FindDescriptorByName("wkt.AnyResponseService")
	require.NoError(t, err)

	method := svcDesc.(protoreflect.ServiceDescriptor).Methods().ByName("GetWrapped")
	mocker := &grpcMocker{outputDesc: method.Output(), descriptorResolver: nil}

	_, err = mocker.newOutputMessage(map[string]any{
		"result": map[string]any{
			"@type": "type.googleapis.com/wkt.CustomResult",
			"value": "hello",
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to resolve")

	msg, err := mocker.newOutputMessage(map[string]any{
		"result": map[string]any{
			"@type": "type.googleapis.com/google.protobuf.StringValue",
			"value": "still-works",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, msg)
}
