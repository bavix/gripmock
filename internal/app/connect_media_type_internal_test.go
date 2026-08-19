package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

func connectRouterForMediaTypes(t *testing.T) *mux.Router {
	t.Helper()

	registry := descriptors.NewRegistry()
	registerMultiverseDescriptors(t, t.Context(), registry)

	gateway := NewConnectRPCGateway(t.Context(), stuber.NewBudgerigar(), registry, nil, nil, nil, nil)

	router := mux.NewRouter()
	router.Handle("/{service:.+}/{method}", gateway).Methods(http.MethodPost)

	return router
}

const unaryMethodPath = "/multiverse.v1.MultiverseService/Ping"

func postWithContentType(t *testing.T, router *mux.Router, contentType string) int {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, unaryMethodPath,
		strings.NewReader("{}"))

	if contentType != "" {
		request.Header.Set(headerContentType, contentType)
	}

	router.ServeHTTP(recorder, request)

	return recorder.Code
}

func TestConnectUnaryRejectsUnknownContentType(t *testing.T) {
	t.Parallel()

	router := connectRouterForMediaTypes(t)

	for _, contentType := range []string{"application/xml", "text/plain", "application/grpc"} {
		require.Equal(t, http.StatusUnsupportedMediaType,
			postWithContentType(t, router, contentType),
			"content type %q", contentType)
	}
}

func TestConnectUnaryServesKnownJSONContentTypes(t *testing.T) {
	t.Parallel()

	router := connectRouterForMediaTypes(t)

	for _, contentType := range []string{
		contentTypeJSON,
		contentTypeConnectJSON,
		"application/json; charset=utf-8",
		"Application/JSON",
		"application/json;charset=UTF-8",
	} {
		require.NotEqual(t, http.StatusUnsupportedMediaType,
			postWithContentType(t, router, contentType),
			"content type %q must be recognised", contentType)
		require.NotEqual(t, http.StatusBadRequest,
			postWithContentType(t, router, contentType),
			"content type %q must be decoded as JSON, not protobuf", contentType)
	}
}

func TestConnectUnaryAcceptsBinaryAndMissingContentTypes(t *testing.T) {
	t.Parallel()

	router := connectRouterForMediaTypes(t)

	for _, contentType := range []string{contentTypeProto, contentTypeConnectProto, ""} {
		require.NotEqual(t, http.StatusUnsupportedMediaType,
			postWithContentType(t, router, contentType),
			"content type %q", contentType)
	}
}

func TestUnsupportedStreamEncoding(t *testing.T) {
	t.Parallel()

	for name, testCase := range map[string]struct {
		value     string
		wantOK    bool
		wantNamed string
	}{
		"absent":         {"", true, ""},
		"identity":       {"identity", true, ""},
		"identity cased": {"Identity", true, ""},
		"gzip":           {"gzip", false, "gzip"},
		"br":             {"br", false, "br"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			header := http.Header{}
			if testCase.value != "" {
				header.Set("Connect-Content-Encoding", testCase.value)
			}

			encoding, ok := unsupportedStreamEncoding(header)

			require.Equal(t, testCase.wantOK, ok)
			require.Equal(t, testCase.wantNamed, encoding)
		})
	}
}

func TestGRPCWebContentTypeSupported(t *testing.T) {
	t.Parallel()

	for _, contentType := range []string{
		"application/grpc-web",
		"application/grpc-web+proto",
		"application/grpc-web+json",
		"application/grpc-web-text",
		"application/grpc-web-text+proto",
		"application/grpc-web-text+json",
		"Application/GRPC-Web+Proto",
		"application/grpc-web+proto; charset=utf-8",
	} {
		require.True(t, grpcWebContentTypeSupported(contentType), contentType)
	}

	for _, contentType := range []string{
		"application/grpc-web+thrift",
		"application/grpc-web-text+thrift",
		"application/grpc",
		"application/json",
	} {
		require.False(t, grpcWebContentTypeSupported(contentType), contentType)
	}

	require.True(t, grpcWebContentTypeSupported(""), "an absent content type stays lenient")
}
