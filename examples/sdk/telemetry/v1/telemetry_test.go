package telemetryv1_test

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	telemetryv1 "github.com/bavix/gripmock/v3/examples/sdk/telemetry/v1"
	sdk "github.com/bavix/gripmock/v3/pkg/sdk"
)

const startupBudget = 15 * time.Second

func newClient(t *testing.T) (*sdk.Server, telemetryv1.TelemetryServiceClient) {
	t.Helper()

	srv := sdk.NewServer(t,
		sdk.WithHealthCheckTimeout(startupBudget),
		sdk.WithFileDescriptor(telemetryv1.File_examples_sdk_telemetry_v1_telemetry_proto),
	)

	return srv, telemetryv1.NewTelemetryServiceClient(srv.Conn())
}

func openStream(
	t *testing.T,
	client telemetryv1.TelemetryServiceClient,
	deviceID, metric string,
) telemetryv1.TelemetryService_StreamSamplesClient {
	t.Helper()

	stream, err := client.StreamSamples(t.Context(), &telemetryv1.StreamSamplesRequest{
		DeviceId: deviceID,
		Metric:   metric,
	})
	require.NoError(t, err)

	return stream
}

func drain(stream telemetryv1.TelemetryService_StreamSamplesClient) ([]*telemetryv1.StreamSamplesResponse, error) {
	var samples []*telemetryv1.StreamSamplesResponse

	for {
		sample, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return samples, nil
		}

		if err != nil {
			return samples, err
		}

		samples = append(samples, sample)
	}
}

func TestStreamDeliversEverySample(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectServerStream(telemetryv1.TelemetryService_StreamSamples_FullMethodName).
		Match("device_id", "sensor-1", "metric", "temperature").
		SendStream(
			map[string]any{"sequence": 1, "value": 21.5, "unit": "C"},
			map[string]any{"sequence": 2, "value": 21.8, "unit": "C"},
			map[string]any{"sequence": 3, "value": 22.1, "unit": "C"},
		)

	samples, err := drain(openStream(t, client, "sensor-1", "temperature"))

	require.NoError(t, err)
	require.Len(t, samples, 3)
	require.InDelta(t, 22.1, samples[2].GetValue(), 0.001)
}

func TestStreamAbortsMidway(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectServerStream(telemetryv1.TelemetryService_StreamSamples_FullMethodName).
		Match("device_id", "sensor-2").
		SendStream(
			map[string]any{"sequence": 1, "value": 0.5, "unit": "V"},
			sdk.StreamError(codes.DataLoss, "sensor disconnected"),
		)

	samples, err := drain(openStream(t, client, "sensor-2", "voltage"))

	require.Error(t, err)
	require.Equal(t, codes.DataLoss, status.Code(err))
	require.Len(t, samples, 1)
}

func TestSlowProducerIsObservable(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	const pause = 80 * time.Millisecond

	srv.ExpectServerStream(telemetryv1.TelemetryService_StreamSamples_FullMethodName).
		Match("device_id", "sensor-3").
		Delay(pause).
		SendStream(map[string]any{"sequence": 1, "value": 9.9, "unit": "A"})

	started := time.Now()

	samples, err := drain(openStream(t, client, "sensor-3", "current"))

	require.NoError(t, err)
	require.Len(t, samples, 1)
	require.GreaterOrEqual(t, time.Since(started), pause)
}

func TestPerMessageDelayShapesTheStream(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	const gap = 60 * time.Millisecond

	srv.ExpectServerStream(telemetryv1.TelemetryService_StreamSamples_FullMethodName).
		Match("device_id", "sensor-4").
		SendStream(
			map[string]any{"sequence": 1, "value": 1, "unit": "A"},
			sdk.Delay(gap, "sequence", 2, "value", 2, "unit", "A"),
		)

	started := time.Now()

	samples, err := drain(openStream(t, client, "sensor-4", "current"))

	require.NoError(t, err)
	require.Len(t, samples, 2)
	require.EqualValues(t, 1, samples[0].GetSequence())
	require.EqualValues(t, 2, samples[1].GetSequence())
	require.GreaterOrEqual(t, time.Since(started), gap)
}

func TestRetiredDeviceFailsBeforeStreaming(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectServerStream(telemetryv1.TelemetryService_StreamSamples_FullMethodName).
		Match(sdk.Glob("device_id", "retired-*")).
		ReturnError(codes.NotFound, "device retired")

	samples, err := drain(openStream(t, client, "retired-17", "temperature"))

	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
	require.Empty(t, samples)
}

func TestSamplesCanBeAppendedAfterTheFirstBatch(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectServerStream(telemetryv1.TelemetryService_StreamSamples_FullMethodName).
		Match("device_id", "sensor-5").
		SendStream(map[string]any{"sequence": 1, "value": 10, "unit": "W"}).
		Send("sequence", 2, "value", 20, "unit", "W")

	samples, err := drain(openStream(t, client, "sensor-5", "power"))

	require.NoError(t, err)
	require.Len(t, samples, 2)
	require.InDelta(t, 20, samples[1].GetValue(), 0.001)
}
