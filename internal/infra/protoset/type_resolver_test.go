package protoset_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/infra/protoset"
)

func TestParseTypeURL(t *testing.T) {
	t.Parallel()

	for name, testCase := range map[string]struct {
		url  string
		want protoreflect.FullName
	}{
		"any url":         {"type.googleapis.com/google.rpc.ErrorInfo", "google.rpc.ErrorInfo"},
		"bare name":       {"google.rpc.ErrorInfo", "google.rpc.ErrorInfo"},
		"leading dot":     {".google.rpc.ErrorInfo", "google.rpc.ErrorInfo"},
		"surrounded":      {"  type.googleapis.com/google.rpc.ErrorInfo  ", "google.rpc.ErrorInfo"},
		"host only slash": {"example.com/", ""},
		"empty":           {"", ""},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, testCase.want, protoset.ParseTypeURL(testCase.url))
		})
	}
}

func TestNewTypeResolverWithoutResolverIsGlobal(t *testing.T) {
	t.Parallel()

	require.Same(t, protoset.GlobalTypeResolver(), protoset.NewTypeResolver(nil))
}

func TestFindMessageFallsBackToGlobalTypes(t *testing.T) {
	t.Parallel()

	resolver := protoset.NewTypeResolver(nil)

	byName, err := resolver.FindMessageByName("google.protobuf.FileDescriptorProto")
	require.NoError(t, err)
	require.Equal(t, protoreflect.FullName("google.protobuf.FileDescriptorProto"), byName.Descriptor().FullName())

	byURL, err := resolver.FindMessageByURL("type.googleapis.com/google.protobuf.FileDescriptorProto")
	require.NoError(t, err)
	require.Equal(t, byName.Descriptor().FullName(), byURL.Descriptor().FullName())
}

func TestFindMessageByURLRejectsEmptyName(t *testing.T) {
	t.Parallel()

	_, err := protoset.NewTypeResolver(nil).FindMessageByURL("")
	require.ErrorIs(t, err, protoregistry.NotFound)
}

func TestFindMessageByNamePrefersTheSuppliedResolver(t *testing.T) {
	t.Parallel()

	files, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{{
			Name:    new("gripmock_type_resolver_probe.proto"),
			Package: new("gripmock.typeprobe"),
			Syntax:  new("proto3"),
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: new("Probe"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:     new("user_id"),
					Number:   new(int32(1)),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					JsonName: new("userId"),
				}},
			}},
		}},
	})
	require.NoError(t, err)

	found, err := protoset.NewTypeResolver(files).FindMessageByName("gripmock.typeprobe.Probe")
	require.NoError(t, err)
	require.Equal(t, protoreflect.FullName("gripmock.typeprobe.Probe"), found.Descriptor().FullName())

	_, err = protoset.NewTypeResolver(nil).FindMessageByName("gripmock.typeprobe.Probe")
	require.ErrorIs(t, err, protoregistry.NotFound,
		"a descriptor known only to the supplied resolver must not leak into the global one")
}

func namingProbeFiles(t *testing.T) *protoregistry.Files {
	t.Helper()

	files, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{{
			Name:    new("gripmock_naming_probe.proto"),
			Package: new("gripmock.namingprobe"),
			Syntax:  new("proto3"),
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: new("Probe"),
				Field: []*descriptorpb.FieldDescriptorProto{{
					Name:     new("user_id"),
					Number:   new(int32(1)),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					JsonName: new("userId"),
				}},
			}},
		}},
	})
	require.NoError(t, err)

	return files
}

func TestMarshalNamingMatchesTheCaller(t *testing.T) {
	t.Parallel()

	files := namingProbeFiles(t)

	desc, err := files.FindDescriptorByName("gripmock.namingprobe.Probe")
	require.NoError(t, err)

	msgDesc, ok := desc.(protoreflect.MessageDescriptor)
	require.True(t, ok)

	message := dynamicpb.NewMessage(msgDesc)
	message.Set(message.Descriptor().Fields().ByName("user_id"), protoreflect.ValueOfString("u-1"))

	resolver := protoset.NewTypeResolver(files)

	camel, err := resolver.Marshal(message)
	require.NoError(t, err)
	require.JSONEq(t, `{"userId":"u-1"}`, string(camel))

	proto, err := resolver.MarshalProtoNames(message)
	require.NoError(t, err)
	require.JSONEq(t, `{"user_id":"u-1"}`, string(proto),
		"the gateways encode JSON responses with proto names, so this is the wire shape clients see")
}

func TestUnmarshalAcceptsBothNamings(t *testing.T) {
	t.Parallel()

	files := namingProbeFiles(t)

	desc, err := files.FindDescriptorByName("gripmock.namingprobe.Probe")
	require.NoError(t, err)

	resolver := protoset.NewTypeResolver(files)

	msgDesc, ok := desc.(protoreflect.MessageDescriptor)
	require.True(t, ok)

	for _, payload := range []string{`{"user_id":"u-1"}`, `{"userId":"u-1"}`} {
		message := dynamicpb.NewMessage(msgDesc)

		require.NoError(t, resolver.Unmarshal([]byte(payload), message), payload)
		require.Equal(t, "u-1",
			message.Get(message.Descriptor().Fields().ByName("user_id")).String(), payload)
	}
}
