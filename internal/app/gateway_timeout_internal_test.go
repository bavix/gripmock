package app

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/internal/infra/types"
)

func TestRequestTimeoutParsesBothProtocols(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		header  string
		value   string
		want    time.Duration
		present bool
		wantErr bool
	}{
		{name: "connect milliseconds", header: headerConnectTimeoutMs, value: "250", want: 250 * time.Millisecond, present: true},
		{name: "connect zero", header: headerConnectTimeoutMs, value: "0", want: 0, present: true},
		{name: "connect too many digits", header: headerConnectTimeoutMs, value: "12345678901", wantErr: true},
		{name: "connect not a number", header: headerConnectTimeoutMs, value: "1s", wantErr: true},
		{name: "connect negative", header: headerConnectTimeoutMs, value: "-1", wantErr: true},
		{name: "grpc hours", header: headerGRPCTimeout, value: "2H", want: 2 * time.Hour, present: true},
		{name: "grpc minutes", header: headerGRPCTimeout, value: "3M", want: 3 * time.Minute, present: true},
		{name: "grpc seconds", header: headerGRPCTimeout, value: "5S", want: 5 * time.Second, present: true},
		{name: "grpc milliseconds", header: headerGRPCTimeout, value: "250m", want: 250 * time.Millisecond, present: true},
		{name: "grpc microseconds", header: headerGRPCTimeout, value: "10u", want: 10 * time.Microsecond, present: true},
		{name: "grpc nanoseconds", header: headerGRPCTimeout, value: "7n", want: 7 * time.Nanosecond, present: true},
		{name: "grpc unknown unit", header: headerGRPCTimeout, value: "5x", wantErr: true},
		{name: "grpc missing unit", header: headerGRPCTimeout, value: "5", wantErr: true},
		{name: "grpc too many digits", header: headerGRPCTimeout, value: "123456789S", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hdr := http.Header{}
			hdr.Set(tt.header, tt.value)

			got, present, err := requestTimeout(hdr)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.present, present)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestRequestTimeoutAbsentMeansNoDeadline(t *testing.T) {
	t.Parallel()

	_, present, err := requestTimeout(http.Header{})
	require.NoError(t, err)
	require.False(t, present)
}

func TestRequestTimeoutPrefersConnectHeader(t *testing.T) {
	t.Parallel()

	hdr := http.Header{}
	hdr.Set(headerConnectTimeoutMs, "100")
	hdr.Set(headerGRPCTimeout, "9S")

	got, present, err := requestTimeout(hdr)
	require.NoError(t, err)
	require.True(t, present)
	require.Equal(t, 100*time.Millisecond, got)
}

// The parsed timeout is worth nothing unless the gateway puts it on the
// context the mocker runs under: a stub that sleeps past the client's deadline
// must come back as deadline_exceeded, not as its data.
func TestConnectGatewayAppliesRequestTimeout(t *testing.T) {
	t.Parallel()

	registry := descriptors.NewRegistry()
	registerMultiverseDescriptors(t, t.Context(), registry)

	budgerigar := stuber.NewBudgerigar()
	budgerigar.PutMany(&stuber.Stub{
		Service: "multiverse.v1.MultiverseService",
		Method:  "Ping",
		Input:   stuber.InputData{Equals: map[string]any{"message": "slow"}},
		Output: stuber.Output{
			Data:  map[string]any{"reply": "Pong"},
			Delay: types.Duration(time.Second),
		},
	})

	gateway := NewConnectRPCGateway(budgerigar, registry, nil, nil, nil, nil)
	router := mux.NewRouter()
	router.Handle("/{service:.+}/{method}", gateway).Methods(http.MethodPost)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost,
		"/multiverse.v1.MultiverseService/Ping", strings.NewReader(`{"message":"slow"}`))
	req.Header.Set(headerContentType, contentTypeJSON)
	req.Header.Set(headerConnectTimeoutMs, "20")

	start := time.Now()

	router.ServeHTTP(rec, req)

	require.Less(t, time.Since(start), time.Second, "the deadline must cut the stub delay short")
	require.Equal(t, http.StatusGatewayTimeout, rec.Code)
	require.Contains(t, rec.Body.String(), "deadline_exceeded")
}
