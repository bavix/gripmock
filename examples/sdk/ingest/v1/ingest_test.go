package ingestv1_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ingestv1 "github.com/bavix/gripmock/v3/examples/sdk/ingest/v1"
	sdk "github.com/bavix/gripmock/v3/pkg/sdk"
)

const source = "checkout-api"

const startupBudget = 15 * time.Second

func newClient(t *testing.T) (*sdk.Server, ingestv1.IngestServiceClient) {
	t.Helper()

	srv := sdk.NewServer(t,
		sdk.WithHealthCheckTimeout(startupBudget),
		sdk.WithFileDescriptor(ingestv1.File_examples_sdk_ingest_v1_ingest_proto),
	)

	return srv, ingestv1.NewIngestServiceClient(srv.Conn())
}

func upload(
	t *testing.T,
	client ingestv1.IngestServiceClient,
	entries ...*ingestv1.UploadBatchRequest,
) (*ingestv1.UploadBatchResponse, error) {
	t.Helper()

	stream, err := client.UploadBatch(t.Context())
	require.NoError(t, err)

	for _, entry := range entries {
		require.NoError(t, stream.Send(entry))
	}

	return stream.CloseAndRecv()
}

func entry(level, message string, tags ...string) *ingestv1.UploadBatchRequest {
	return &ingestv1.UploadBatchRequest{Source: source, Level: level, Message: message, Tags: tags}
}

func TestBatchIsAcceptedWhenEveryEntryMatches(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectClientStream(ingestv1.IngestService_UploadBatch_FullMethodName).
		Match(sdk.Equals("source", source)).
		Return("batch_id", "BATCH-1", "accepted", 2, "rejected", 0)

	receipt, err := upload(t, client, entry("info", "order created"), entry("warn", "retrying payment"))

	require.NoError(t, err)
	require.Equal(t, "BATCH-1", receipt.GetBatchId())
	require.EqualValues(t, 2, receipt.GetAccepted())
}

func TestBatchIsRejectedWhenAnEntryDiffers(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectClientStream(ingestv1.IngestService_UploadBatch_FullMethodName).
		Match(sdk.Equals("source", source)).
		Return("batch_id", "BATCH-1", "accepted", 1)

	_, err := upload(t, client,
		entry("info", "order created"),
		&ingestv1.UploadBatchRequest{Source: "billing-worker", Level: "info", Message: "charge ok"},
	)

	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestOrderedBatchIsMatchedPositionally(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectClientStream(ingestv1.IngestService_UploadBatch_FullMethodName).
		MatchSequence(
			sdk.Equals("level", "info"),
			sdk.Equals("level", "warn"),
			sdk.Equals("level", "error"),
		).
		Return("batch_id", "BATCH-ORDERED", "accepted", 3)

	receipt, err := upload(t, client,
		entry("info", "started"),
		entry("warn", "slow response"),
		entry("error", "gateway timeout"),
	)

	require.NoError(t, err)
	require.Equal(t, "BATCH-ORDERED", receipt.GetBatchId())
}

func TestOutOfOrderBatchDoesNotMatchSequence(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectClientStream(ingestv1.IngestService_UploadBatch_FullMethodName).
		MatchSequence(sdk.Equals("level", "info"), sdk.Equals("level", "warn")).
		Return("batch_id", "BATCH-ORDERED", "accepted", 2)

	_, err := upload(t, client, entry("warn", "slow response"), entry("info", "started"))

	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestEitherSeverityIsAccepted(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectClientStream(ingestv1.IngestService_UploadBatch_FullMethodName).
		Match(sdk.AnyOf(sdk.Equals("level", "warn"), sdk.Equals("level", "error"))).
		Return("batch_id", "BATCH-SEVERE", "accepted", 2)

	receipt, err := upload(t, client, entry("warn", "slow"), entry("error", "timeout"))

	require.NoError(t, err)
	require.Equal(t, "BATCH-SEVERE", receipt.GetBatchId())
}

func TestTagOrderDoesNotAffectMatching(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectClientStream(ingestv1.IngestService_UploadBatch_FullMethodName).
		Match(sdk.IgnoreArrayOrder(sdk.Equals("tags", []any{"payments", "eu-west"}))).
		Return("batch_id", "BATCH-TAGGED", "accepted", 1)

	receipt, err := upload(t, client, entry("info", "charge ok", "eu-west", "payments"))

	require.NoError(t, err)
	require.Equal(t, "BATCH-TAGGED", receipt.GetBatchId())
}

func TestReceiptIsComputedFromTheBatch(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectClientStream(ingestv1.IngestService_UploadBatch_FullMethodName).
		Match(sdk.Matches("source", `.+`)).
		Run(func(_ context.Context, messages []any) (any, error) {
			accepted, rejected := 0, 0

			for _, message := range messages {
				if field(message, "level") == "error" {
					rejected++

					continue
				}

				accepted++
			}

			return map[string]any{
				"batch_id": "BATCH-" + field(messages[0], "source"),
				"accepted": accepted,
				"rejected": rejected,
			}, nil
		})

	receipt, err := upload(t, client,
		entry("info", "a"),
		entry("error", "b"),
		entry("error", "c"),
		entry("debug", "d"),
	)

	require.NoError(t, err)
	require.Equal(t, "BATCH-"+source, receipt.GetBatchId())
	require.EqualValues(t, 2, receipt.GetAccepted())
	require.EqualValues(t, 2, receipt.GetRejected())
}

func TestOversizedBatchIsRefused(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectClientStream(ingestv1.IngestService_UploadBatch_FullMethodName).
		Match(sdk.Matches("message", "payload too large")).
		ReturnErrorWithDetails(codes.ResourceExhausted, "batch exceeds quota", map[string]any{
			"type": "type.googleapis.com/google.rpc.QuotaFailure",
			"violations": []any{
				map[string]any{"subject": "batch/" + source, "description": "batch exceeds the 1 MiB limit"},
			},
		})

	_, err := upload(t, client, entry("error", "payload too large for one request"))

	require.Error(t, err)
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
	require.NotEmpty(t, status.Convert(err).Details())
}

func field(message any, key string) string {
	fields, ok := message.(map[string]any)
	if !ok {
		return ""
	}

	value, _ := fields[key].(string)

	return value
}
