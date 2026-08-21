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

func TestTemplateSendsOneSamplePerRequestedMetric(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectServerStream(telemetryv1.TelemetryService_StreamSamples_FullMethodName).
		Match("device_id", "sensor-6").
		SendStreamTemplate(`
{{ range $i, $metric := split .Request.metric "," }}
  {{ dict "sequence" (add $i 1) "value" (mul (add $i 1) 1.5) "unit" $metric }}
{{ end }}`)

	samples, err := drain(openStream(t, client, "sensor-6", "C,V,A"))

	require.NoError(t, err)
	require.Len(t, samples, 3)
	require.Equal(t, "A", samples[2].GetUnit())
	require.EqualValues(t, 3, samples[2].GetSequence())
	require.InDelta(t, 4.5, samples[2].GetValue(), 0.001)
}

func TestTemplateStreamDelayIsPerMessageOnly(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	const gap = 60 * time.Millisecond

	srv.ExpectServerStream(telemetryv1.TelemetryService_StreamSamples_FullMethodName).
		Match("device_id", "sensor-7").
		Delay(gap).
		SendStreamTemplate(`{{ range $i := seq 2 }}{{ dict "sequence" $i "value" 1 "unit" "C" }}{{ end }}`)

	started := time.Now()

	samples, err := drain(openStream(t, client, "sensor-7", "temperature"))

	require.NoError(t, err)
	require.Len(t, samples, 2)

	elapsed := time.Since(started)
	require.GreaterOrEqual(t, elapsed, 2*gap)
	require.Less(t, elapsed, 4*gap)
}

func TestEffectCreatesAStreamTemplateStub(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	unlockLive := sdk.Upsert("telemetry.v1.TelemetryService", "StreamSamples").
		Match("device_id", "sensor-9").
		SendStreamTemplate(`{{ range $i := seq 2 }}{{ dict "sequence" $i "value" 7 "unit" "C" }}{{ end }}`).
		Build()

	srv.ExpectServerStream(telemetryv1.TelemetryService_StreamSamples_FullMethodName).
		Match("device_id", "sensor-8").
		Effect(unlockLive).
		SendStream(map[string]any{"sequence": 1, "value": 1, "unit": "C"})

	_, err := drain(openStream(t, client, "sensor-9", "temperature"))
	require.Error(t, err, "the child stub does not exist yet")

	primed, err := drain(openStream(t, client, "sensor-8", "temperature"))
	require.NoError(t, err)
	require.Len(t, primed, 1)

	samples, err := drain(openStream(t, client, "sensor-9", "temperature"))
	require.NoError(t, err)
	require.Len(t, samples, 2)
	require.EqualValues(t, 1, samples[1].GetSequence())
}
