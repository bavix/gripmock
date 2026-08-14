package app

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

//nolint:ireturn // protoreflect exposes descriptors only as interfaces.
func methodDescWithIdempotency(t *testing.T, level descriptorpb.MethodOptions_IdempotencyLevel) protoreflect.MethodDescriptor {
	t.Helper()

	fd := &descriptorpb.FileDescriptorProto{
		Name:    new("get_test.proto"),
		Package: new("gettest"),
		Syntax:  new("proto3"),
		Dependency: []string{
			"google/protobuf/empty.proto",
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{
			Name: new("Svc"),
			Method: []*descriptorpb.MethodDescriptorProto{{
				Name:       new("Get"),
				InputType:  new(".google.protobuf.Empty"),
				OutputType: new(".google.protobuf.Empty"),
				Options:    &descriptorpb.MethodOptions{IdempotencyLevel: new(level)},
			}},
		}},
	}

	emptyFD := protodesc.ToFileDescriptorProto(emptypb.File_google_protobuf_empty_proto)

	files, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{emptyFD, fd},
	})
	require.NoError(t, err)

	desc, err := files.FindDescriptorByName("gettest.Svc.Get")
	require.NoError(t, err)

	method, ok := desc.(protoreflect.MethodDescriptor)
	require.True(t, ok)

	return method
}

func TestMethodAllowsGET(t *testing.T) {
	t.Parallel()

	require.True(t, methodAllowsGET(methodDescWithIdempotency(t, descriptorpb.MethodOptions_NO_SIDE_EFFECTS)))
	require.False(t, methodAllowsGET(methodDescWithIdempotency(t, descriptorpb.MethodOptions_IDEMPOTENT)))
	require.False(t, methodAllowsGET(methodDescWithIdempotency(t, descriptorpb.MethodOptions_IDEMPOTENCY_UNKNOWN)))
	require.False(t, methodAllowsGET(nil))
}

//nolint:funlen // one subtest per documented GET query form.
func TestConnectGetRequestDecodesQuery(t *testing.T) {
	t.Parallel()

	desc := methodDescWithIdempotency(t, descriptorpb.MethodOptions_NO_SIDE_EFFECTS)
	payload := `{"name":"Alex"}`

	t.Run("percent encoded json", func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet,
			"/s/m?connect=v1&encoding=json&message="+url.QueryEscape(payload), nil)

		body, err := connectGetRequest(r, desc)
		require.NoError(t, err)
		require.Equal(t, payload, string(body))
		require.Equal(t, contentTypeJSON, connectGetContentType(r)) //nolint:testifylint
	})

	t.Run("base64 payload", func(t *testing.T) {
		t.Parallel()

		enc := base64.RawURLEncoding.EncodeToString([]byte(payload))
		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet,
			"/s/m?connect=v1&encoding=json&base64=1&message="+enc, nil)

		body, err := connectGetRequest(r, desc)
		require.NoError(t, err)
		require.Equal(t, payload, string(body))
	})

	t.Run("padded base64 accepted", func(t *testing.T) {
		t.Parallel()

		enc := url.QueryEscape(base64.URLEncoding.EncodeToString([]byte(payload)))
		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/s/m?base64=1&encoding=json&message="+enc, nil)

		body, err := connectGetRequest(r, desc)
		require.NoError(t, err)
		require.Equal(t, payload, string(body))
	})

	t.Run("defaults to proto encoding", func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/s/m?connect=v1", nil)

		_, err := connectGetRequest(r, desc)
		require.NoError(t, err)
		require.Equal(t, contentTypeProto, connectGetContentType(r))
	})

	t.Run("compression rejected", func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/s/m?compression=gzip&message=x", nil)

		_, err := connectGetRequest(r, desc)
		require.ErrorIs(t, err, errConnectGetCompressed)
	})

	t.Run("method without the option is refused", func(t *testing.T) {
		t.Parallel()

		r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/s/m?message=x", nil)

		_, err := connectGetRequest(r, methodDescWithIdempotency(t, descriptorpb.MethodOptions_IDEMPOTENT))
		require.ErrorIs(t, err, errConnectGetNotIdempotent)
	})
}
