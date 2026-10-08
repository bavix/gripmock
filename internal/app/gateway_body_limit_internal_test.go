package app

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/bavix/gripmock/v3/internal/infra/httputil"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/template"
)

func oversizedBody() io.Reader {
	return io.LimitReader(zeroReader{}, httputil.MaxBodyBytes()+1)
}

func newStructMocker(t *testing.T, bg *stuber.Budgerigar) *grpcMocker {
	t.Helper()

	structDesc := (&structpb.Struct{}).ProtoReflect().Descriptor()

	return &grpcMocker{
		budgerigar:      bg,
		templateEngine:  template.New(t.Context(), nil),
		errorFormatter:  NewErrorFormatter(),
		inputDesc:       structDesc,
		outputDesc:      structDesc,
		fullServiceName: "test.Service",
		serviceName:     "test.Service",
		methodName:      "TestMethod",
	}
}

func TestConnectRPCGateway_HandleUnary_RejectsOversizedBody(t *testing.T) {
	t.Parallel()

	bg := stuber.NewBudgerigar()
	gateway := NewConnectRPCGateway(t.Context(), bg, nil, nil, nil, nil, nil)
	mocker := newStructMocker(t, bg)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/test.Service/TestMethod", oversizedBody())
	req.Header.Set("Content-Type", "application/json")

	adapter := &httpStreamAdapter{
		baseStreamAdapter: baseStreamAdapter{
			ctx: req.Context(),
			req: req,
			w:   rec,
		},
	}

	gateway.handleUnary(mocker, adapter, nil)

	require.Equal(t, http.StatusTooManyRequests, rec.Code)

	var resp struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "resource_exhausted", resp.Code)
	require.Contains(t, resp.Message, "exceeds")
}

func TestGRPCWebGateway_HandleUnary_RejectsOversizedBody(t *testing.T) {
	t.Parallel()

	bg := stuber.NewBudgerigar()
	gateway := NewGRPCWebGateway(t.Context(), bg, nil, nil, nil, nil, nil)
	mocker := newStructMocker(t, bg)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/test.Service/TestMethod", oversizedBody())
	req.Header.Set("Content-Type", "application/grpc-web+json")

	gateway.handleUnary(mocker, newGRPCWebAdapter(req, rec, mocker))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "grpc-status: 8")
}

func TestConnectRPCGateway_HandleUnary_AcceptsBodyAtTheLimit(t *testing.T) {
	t.Parallel()

	bg := stuber.NewBudgerigar()
	bg.PutMany(&stuber.Stub{
		Service: "test.Service",
		Method:  "TestMethod",
		Output:  stuber.Output{Data: map[string]any{"name": "Alice"}},
	})

	gateway := NewConnectRPCGateway(t.Context(), bg, nil, nil, nil, nil, nil)
	mocker := newStructMocker(t, bg)

	const envelope = `{"name":"` + `"}`

	padding := int(httputil.MaxBodyBytes()) - len(envelope)
	body := `{"name":"` + strings.Repeat("a", padding) + `"}`
	require.Len(t, body, int(httputil.MaxBodyBytes()))

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/test.Service/TestMethod", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	adapter := &httpStreamAdapter{
		baseStreamAdapter: baseStreamAdapter{
			ctx: req.Context(),
			req: req,
			w:   rec,
		},
	}

	gateway.handleUnary(mocker, adapter, nil)

	require.Equal(t, http.StatusOK, rec.Code)
}
