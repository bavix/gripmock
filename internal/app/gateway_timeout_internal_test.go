package app

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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

func TestWithRequestTimeoutSetsDeadline(t *testing.T) {
	t.Parallel()

	hdr := http.Header{}
	hdr.Set(headerConnectTimeoutMs, "50")

	ctx, cancel, err := withRequestTimeout(t.Context(), hdr)
	defer cancel()

	require.NoError(t, err)

	deadline, ok := ctx.Deadline()
	require.True(t, ok, "the caller asked for a deadline, it must be applied")
	require.WithinDuration(t, time.Now().Add(50*time.Millisecond), deadline, 20*time.Millisecond)
}

func TestWithRequestTimeoutLeavesContextAloneWhenAbsent(t *testing.T) {
	t.Parallel()

	ctx, cancel, err := withRequestTimeout(t.Context(), http.Header{})
	defer cancel()

	require.NoError(t, err)

	_, ok := ctx.Deadline()
	require.False(t, ok)
}
