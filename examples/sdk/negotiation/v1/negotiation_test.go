package negotiationv1_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	negotiationv1 "github.com/bavix/gripmock/v3/examples/sdk/negotiation/v1"
	sdk "github.com/bavix/gripmock/v3/pkg/sdk"
)

const startupBudget = 15 * time.Second

func newClient(t *testing.T) (*sdk.Server, negotiationv1.NegotiationServiceClient) {
	t.Helper()

	srv := sdk.NewServer(t,
		sdk.WithHealthCheckTimeout(startupBudget),
		sdk.WithFileDescriptor(negotiationv1.File_examples_sdk_negotiation_v1_negotiation_proto),
	)

	return srv, negotiationv1.NewNegotiationServiceClient(srv.Conn())
}

func haggle(
	t *testing.T,
	ctx context.Context,
	client negotiationv1.NegotiationServiceClient,
	offers ...*negotiationv1.HaggleRequest,
) ([]*negotiationv1.HaggleResponse, error) {
	t.Helper()

	stream, err := client.Haggle(ctx)
	require.NoError(t, err)

	for _, offer := range offers {
		require.NoError(t, stream.Send(offer))
	}

	require.NoError(t, stream.CloseSend())

	var counters []*negotiationv1.HaggleResponse

	for {
		counter, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return counters, nil
		}

		if err != nil {
			return counters, err
		}

		counters = append(counters, counter)
	}
}

func offer(dealID string, priceCents int64) *negotiationv1.HaggleRequest {
	return &negotiationv1.HaggleRequest{DealId: dealID, PriceCents: priceCents}
}

func TestStaticCounterOffers(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectBidirectionalStream(negotiationv1.NegotiationService_Haggle_FullMethodName).
		Match("deal_id", "DEAL-1").
		SendStream(
			map[string]any{"price_cents": 9000, "verdict": "counter"},
			map[string]any{"price_cents": 8500, "verdict": "final"},
		)

	counters, err := haggle(t, t.Context(), client, offer("DEAL-1", 7000))

	require.NoError(t, err)
	require.Len(t, counters, 2)
	require.Equal(t, "final", counters[1].GetVerdict())
	require.EqualValues(t, 8500, counters[1].GetPriceCents())
}

func TestUnknownDealIsRejected(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectBidirectionalStream(negotiationv1.NegotiationService_Haggle_FullMethodName).
		Match("deal_id", "DEAL-1").
		SendStream(map[string]any{"price_cents": 9000, "verdict": "counter"})

	_, err := haggle(t, t.Context(), client, offer("DEAL-404", 7000))

	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestOnlyAuthorisedBuyersAreServed(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectBidirectionalStream(negotiationv1.NegotiationService_Haggle_FullMethodName).
		WithHeader(sdk.Equals("x-buyer-tier", "gold")).
		Match("deal_id", "DEAL-2").
		SendStream(map[string]any{"price_cents": 6000, "verdict": "accepted"})

	gold := metadata.AppendToOutgoingContext(t.Context(), "x-buyer-tier", "gold")

	counters, err := haggle(t, gold, client, offer("DEAL-2", 6000))
	require.NoError(t, err)
	require.Equal(t, "accepted", counters[0].GetVerdict())

	bronze := metadata.AppendToOutgoingContext(t.Context(), "x-buyer-tier", "bronze")

	_, err = haggle(t, bronze, client, offer("DEAL-2", 6000))
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))

	_, err = haggle(t, t.Context(), client, offer("DEAL-2", 6000))
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestNegotiationLoopIsDrivenByAHandler(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectBidirectionalStream(negotiationv1.NegotiationService_Haggle_FullMethodName).
		Run(func(_ context.Context, stream any) error {
			serverStream, ok := stream.(grpc.ServerStream)
			if !ok {
				return status.Error(codes.Internal, "unexpected stream type")
			}

			for {
				var request negotiationv1.HaggleRequest

				if err := serverStream.RecvMsg(&request); err != nil {
					if errors.Is(err, io.EOF) {
						return nil
					}

					return err
				}

				response := &negotiationv1.HaggleResponse{
					PriceCents: request.GetPriceCents() + 500,
					Verdict:    "counter",
				}

				if request.GetPriceCents() >= 9500 {
					response.PriceCents = request.GetPriceCents()
					response.Verdict = "accepted"
				}

				if err := serverStream.SendMsg(response); err != nil {
					return err
				}
			}
		})

	counters, err := haggle(t, t.Context(), client,
		offer("DEAL-3", 8000),
		offer("DEAL-3", 9000),
		offer("DEAL-3", 9500),
	)

	require.NoError(t, err)
	require.Len(t, counters, 3)
	require.EqualValues(t, 8500, counters[0].GetPriceCents())
	require.Equal(t, "counter", counters[1].GetVerdict())
	require.Equal(t, "accepted", counters[2].GetVerdict())
}

func TestHandlerIsAlsoGatedByHeaders(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectBidirectionalStream(negotiationv1.NegotiationService_Haggle_FullMethodName).
		WithHeader(sdk.Equals("x-buyer-tier", "gold")).
		Run(func(_ context.Context, stream any) error {
			serverStream, ok := stream.(grpc.ServerStream)
			if !ok {
				return status.Error(codes.Internal, "unexpected stream type")
			}

			var request negotiationv1.HaggleRequest
			if err := serverStream.RecvMsg(&request); err != nil {
				return err
			}

			return serverStream.SendMsg(&negotiationv1.HaggleResponse{
				PriceCents: request.GetPriceCents(),
				Verdict:    "accepted",
			})
		})

	gold := metadata.AppendToOutgoingContext(t.Context(), "x-buyer-tier", "gold")

	counters, err := haggle(t, gold, client, offer("DEAL-5", 4200))
	require.NoError(t, err)
	require.Equal(t, "accepted", counters[0].GetVerdict())

	_, err = haggle(t, t.Context(), client, offer("DEAL-5", 4200))
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestWalkAwayIsReportedAsAnError(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectBidirectionalStream(negotiationv1.NegotiationService_Haggle_FullMethodName).
		Run(func(_ context.Context, _ any) error {
			return status.Error(codes.Aborted, "seller walked away")
		})

	_, err := haggle(t, t.Context(), client, offer("DEAL-4", 100))

	require.Error(t, err)
	require.Equal(t, codes.Aborted, status.Code(err))
}
