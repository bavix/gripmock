package rest

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

var errTest = errors.New("test error")

// mockServer implements ServerInterface for testing.
type mockServer struct {
	called map[string]bool
}

func newMockServer() *mockServer {
	return &mockServer{called: make(map[string]bool)}
}

func (m *mockServer) Liveness(w http.ResponseWriter, _ *http.Request) {
	m.called["Liveness"] = true

	w.WriteHeader(http.StatusOK)
}

func (m *mockServer) Readiness(w http.ResponseWriter, _ *http.Request) {
	m.called["Readiness"] = true

	w.WriteHeader(http.StatusOK)
}

func (m *mockServer) ServicesList(w http.ResponseWriter, _ *http.Request) {
	m.called["ServicesList"] = true

	w.WriteHeader(http.StatusOK)
}

func (m *mockServer) ServiceMethodsList(w http.ResponseWriter, _ *http.Request, _ string) {
	m.called["ServiceMethodsList"] = true

	w.WriteHeader(http.StatusOK)
}

func (m *mockServer) PurgeStubs(w http.ResponseWriter, _ *http.Request) {
	m.called["PurgeStubs"] = true

	w.WriteHeader(http.StatusOK)
}

func (m *mockServer) ListStubs(w http.ResponseWriter, _ *http.Request) {
	m.called["ListStubs"] = true

	w.WriteHeader(http.StatusOK)
}

func (m *mockServer) AddStub(w http.ResponseWriter, _ *http.Request) {
	m.called["AddStub"] = true

	w.WriteHeader(http.StatusOK)
}

func (m *mockServer) AddDescriptors(w http.ResponseWriter, _ *http.Request) {
	m.called["AddDescriptors"] = true

	w.WriteHeader(http.StatusOK)
}

func (m *mockServer) BatchStubsDelete(w http.ResponseWriter, _ *http.Request) {
	m.called["BatchStubsDelete"] = true

	w.WriteHeader(http.StatusOK)
}

func (m *mockServer) SearchStubs(w http.ResponseWriter, _ *http.Request) {
	m.called["SearchStubs"] = true

	w.WriteHeader(http.StatusOK)
}

func (m *mockServer) ListUnusedStubs(w http.ResponseWriter, _ *http.Request) {
	m.called["ListUnusedStubs"] = true

	w.WriteHeader(http.StatusOK)
}

func (m *mockServer) ListUsedStubs(w http.ResponseWriter, _ *http.Request) {
	m.called["ListUsedStubs"] = true

	w.WriteHeader(http.StatusOK)
}

func (m *mockServer) ListHistory(w http.ResponseWriter, _ *http.Request) {
	m.called["ListHistory"] = true

	_ = json.NewEncoder(w).Encode(HistoryList{}) //nolint:errchkjson
}

func (m *mockServer) VerifyCalls(w http.ResponseWriter, _ *http.Request) {
	m.called["VerifyCalls"] = true

	w.WriteHeader(http.StatusOK)
}

func (m *mockServer) DeleteStubByID(w http.ResponseWriter, _ *http.Request, _ ID) {
	m.called["DeleteStubByID"] = true

	w.WriteHeader(http.StatusOK)
}

func (m *mockServer) FindByID(w http.ResponseWriter, _ *http.Request, _ ID) {
	m.called["FindByID"] = true

	w.WriteHeader(http.StatusOK)
}

func (m *mockServer) PatchStubByID(w http.ResponseWriter, _ *http.Request, _ ID) {
	m.called["PatchStubByID"] = true

	w.WriteHeader(http.StatusOK)
}

func TestHandler_Routes(t *testing.T) {
	t.Parallel()

	mock := newMockServer()
	handler := Handler(mock)
	require.NotNil(t, handler)

	validUUID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	tests := []struct {
		method string
		path   string
		call   string
	}{
		{http.MethodGet, "/health/liveness", "Liveness"},
		{http.MethodGet, "/health/readiness", "Readiness"},
		{http.MethodGet, "/services", "ServicesList"},
		{http.MethodGet, "/services/myservice/methods", "ServiceMethodsList"},
		{http.MethodDelete, "/stubs", "PurgeStubs"},
		{http.MethodGet, "/stubs", "ListStubs"},
		{http.MethodPost, "/stubs", "AddStub"},
		{http.MethodPost, "/descriptors", "AddDescriptors"},
		{http.MethodPost, "/stubs/batchDelete", "BatchStubsDelete"},
		{http.MethodPost, "/stubs/search", "SearchStubs"},
		{http.MethodGet, "/stubs/unused", "ListUnusedStubs"},
		{http.MethodGet, "/stubs/used", "ListUsedStubs"},
		{http.MethodDelete, "/stubs/" + validUUID.String(), "DeleteStubByID"},
		{http.MethodGet, "/stubs/" + validUUID.String(), "FindByID"},
	}

	for _, tt := range tests {
		t.Run(tt.call, func(t *testing.T) {
			t.Parallel()

			m := newMockServer()
			h := Handler(m)
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			if tt.path == "/services/myservice/methods" {
				req = mux.SetURLVars(req, map[string]string{"serviceID": "myservice"})
			}

			if tt.path == "/stubs/"+validUUID.String() {
				req = mux.SetURLVars(req, map[string]string{"uuid": validUUID.String()})
			}

			h.ServeHTTP(rec, req)
			require.True(t, m.called[tt.call], "handler %s should have been called", tt.call)
		})
	}
}

func TestHandlerWithOptions_BaseURL(t *testing.T) {
	t.Parallel()

	mock := newMockServer()
	handler := HandlerWithOptions(mock, GorillaServerOptions{
		BaseURL: "/api",
	})
	req := httptest.NewRequest(http.MethodGet, "/api/health/liveness", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.True(t, mock.called["Liveness"])
}

func TestHandlerWithOptions_CustomRouter(t *testing.T) {
	t.Parallel()

	r := mux.NewRouter()
	mock := newMockServer()
	handler := HandlerFromMux(mock, r)
	req := httptest.NewRequest(http.MethodGet, "/health/liveness", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.True(t, mock.called["Liveness"])
}

func TestHandlerFromMuxWithBaseURL(t *testing.T) {
	t.Parallel()

	r := mux.NewRouter()
	mock := newMockServer()
	handler := HandlerFromMuxWithBaseURL(mock, r, "/api")
	req := httptest.NewRequest(http.MethodGet, "/api/health/liveness", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.True(t, mock.called["Liveness"])
}

func TestHandlerWithOptions_Middleware(t *testing.T) {
	t.Parallel()

	mock := newMockServer()
	called := false
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true

			next.ServeHTTP(w, r)
		})
	}
	handler := HandlerWithOptions(mock, GorillaServerOptions{
		Middlewares: []MiddlewareFunc{mw},
	})
	req := httptest.NewRequest(http.MethodGet, "/health/liveness", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.True(t, called)
	require.True(t, mock.called["Liveness"])
}

func TestHandlerWithOptions_CustomErrorHandler(t *testing.T) {
	t.Parallel()

	handler := HandlerWithOptions(newMockServer(), GorillaServerOptions{
		ErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(err.Error()))
		},
	})
	req := httptest.NewRequest(http.MethodDelete, "/stubs/bad-uuid", nil)
	req = mux.SetURLVars(req, map[string]string{"uuid": "bad-uuid"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestDeleteStubByID_InvalidUUID(t *testing.T) {
	t.Parallel()

	handler := HandlerWithOptions(newMockServer(), GorillaServerOptions{})
	req := httptest.NewRequest(http.MethodDelete, "/stubs/invalid-uuid", nil)
	req = mux.SetURLVars(req, map[string]string{"uuid": "invalid-uuid"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestFindByID_InvalidUUID(t *testing.T) {
	t.Parallel()

	handler := HandlerWithOptions(newMockServer(), GorillaServerOptions{})
	req := httptest.NewRequest(http.MethodGet, "/stubs/not-a-uuid", nil)
	req = mux.SetURLVars(req, map[string]string{"uuid": "not-a-uuid"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestErrorTypes(t *testing.T) {
	t.Parallel()

	t.Run("UnescapedCookieParamError", func(t *testing.T) {
		t.Parallel()

		err := &UnescapedCookieParamError{ParamName: "foo", Err: http.ErrNoCookie}
		require.Contains(t, err.Error(), "foo")
		require.Equal(t, http.ErrNoCookie, err.Unwrap())
	})

	t.Run("UnmarshalingParamError", func(t *testing.T) {
		t.Parallel()

		err := &UnmarshalingParamError{ParamName: "bar", Err: errTest}
		require.Contains(t, err.Error(), "bar")
		require.Equal(t, errTest, err.Unwrap())
	})

	t.Run("RequiredParamError", func(t *testing.T) {
		t.Parallel()

		err := &RequiredParamError{ParamName: "baz"}
		require.Contains(t, err.Error(), "baz")
	})

	t.Run("RequiredHeaderError", func(t *testing.T) {
		t.Parallel()

		err := &RequiredHeaderError{ParamName: "x-id", Err: http.ErrNoCookie}
		require.Contains(t, err.Error(), "x-id")
		require.Equal(t, http.ErrNoCookie, err.Unwrap())
	})

	t.Run("InvalidParamFormatError", func(t *testing.T) {
		t.Parallel()

		err := &InvalidParamFormatError{ParamName: "uuid", Err: errTest}
		require.Contains(t, err.Error(), "uuid")
		require.Equal(t, errTest, err.Unwrap())
	})

	t.Run("TooManyValuesForParamError", func(t *testing.T) {
		t.Parallel()

		err := &TooManyValuesForParamError{ParamName: "limit", Count: 5}
		require.Contains(t, err.Error(), "limit")
		require.Contains(t, err.Error(), "5")
	})
}

func TestTypes_MessageOK(t *testing.T) {
	t.Parallel()

	msg := MessageOK{
		Message: "ok",
		Time:    time.Unix(0, 0).UTC(),
	}
	data, err := json.Marshal(msg)
	require.NoError(t, err)

	var decoded MessageOK

	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, msg.Message, decoded.Message)
}

func TestTypes_Method(t *testing.T) {
	t.Parallel()

	m := Method{Id: "id1", Name: "Get"}
	data, err := json.Marshal(m)
	require.NoError(t, err)

	var decoded Method

	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, m.Id, decoded.Id)
	require.Equal(t, m.Name, decoded.Name)
}

func TestTypes_Service(t *testing.T) {
	t.Parallel()

	svc := Service{
		Id:      "svc1",
		Name:    "TestService",
		Package: "pkg",
		Methods: []Method{{Id: "m1", Name: "Get"}},
	}
	data, err := json.Marshal(svc)
	require.NoError(t, err)

	var decoded Service

	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, svc.Id, decoded.Id)
	require.Equal(t, svc.Name, decoded.Name)
}

func TestTypes_SearchRequest(t *testing.T) {
	t.Parallel()

	id := openapi_types.UUID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")) //nolint:unconvert
	req := SearchRequest{
		Service: "Svc",
		Method:  "Get",
		Data:    map[string]any{"k": "v"},
		Id:      &id,
		Headers: map[string]string{"h": "v"},
	}
	data, err := json.Marshal(req)
	require.NoError(t, err)

	var decoded SearchRequest

	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, req.Service, decoded.Service)
	require.Equal(t, req.Method, decoded.Method)
}

func TestTypes_SearchResponse(t *testing.T) {
	t.Parallel()

	resp := SearchResponse{
		Code:    codes.OK,
		Data:    map[string]any{"r": "v"},
		Error:   "",
		Headers: map[string]string{"h": "v"},
	}
	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded SearchResponse

	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, resp.Code, decoded.Code)
}

func TestTypes_StubHeaders(t *testing.T) {
	t.Parallel()

	h := StubHeaders{
		Contains: map[string]string{"a": "b"},
		Equals:   map[string]string{"c": "d"},
		Matches:  map[string]string{"e": "f"},
	}
	data, err := json.Marshal(h)
	require.NoError(t, err)

	var decoded StubHeaders

	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Equal(t, h.Contains, decoded.Contains)
}

func TestTypes_ListID(t *testing.T) {
	t.Parallel()

	id1 := openapi_types.UUID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")) //nolint:unconvert
	id2 := openapi_types.UUID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440001")) //nolint:unconvert
	list := ListID{id1, id2}
	require.Len(t, list, 2)
}

func TestTypes_StubList(t *testing.T) {
	t.Parallel()

	list := StubList{
		{Service: "S", Method: "M", Input: StubInput{Contains: map[string]any{"x": "y"}}, Output: StubOutput{Data: map[string]any{"ok": true}}},
	}
	require.Len(t, list, 1)
}
