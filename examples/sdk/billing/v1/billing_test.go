package billingv1_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	billingv1 "github.com/bavix/gripmock/v3/examples/sdk/billing/v1"
	sdk "github.com/bavix/gripmock/v3/pkg/sdk"
)

const startupBudget = 15 * time.Second

func newClient(t *testing.T) (*sdk.Server, billingv1.BillingServiceClient) {
	t.Helper()

	srv := sdk.NewServer(t,
		sdk.WithHealthCheckTimeout(startupBudget),
		sdk.WithFileDescriptor(billingv1.File_examples_sdk_billing_v1_billing_proto),
	)

	return srv, billingv1.NewBillingServiceClient(srv.Conn())
}

func TestChargeInvoiceMatchesOnRequestFields(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectUnary(billingv1.BillingService_ChargeInvoice_FullMethodName).
		Match("invoice_id", "INV-1001", "amount_cents", 2500).
		Return("transaction_id", "TX-1", "status", "settled", "remaining_cents", 0)

	resp, err := client.ChargeInvoice(t.Context(), &billingv1.ChargeInvoiceRequest{
		InvoiceId:   "INV-1001",
		CustomerId:  "CUST-7",
		AmountCents: 2500,
	})

	require.NoError(t, err)
	require.Equal(t, "TX-1", resp.GetTransactionId())
	require.Equal(t, "settled", resp.GetStatus())
}

func TestChargeInvoiceRejectsUnmatchedRequest(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectUnary(billingv1.BillingService_ChargeInvoice_FullMethodName).
		Match("invoice_id", "INV-1001").
		Return("transaction_id", "TX-1", "status", "settled")

	_, err := client.ChargeInvoice(t.Context(), &billingv1.ChargeInvoiceRequest{InvoiceId: "INV-9999"})

	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestRetryEventuallySucceeds(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectUnary(billingv1.BillingService_ChargeInvoice_FullMethodName).
		Match("invoice_id", "INV-2002").
		ReturnError(codes.Unavailable, "processor busy").
		NextWillReturn("transaction_id", "TX-2", "status", "settled")

	req := &billingv1.ChargeInvoiceRequest{InvoiceId: "INV-2002", AmountCents: 100}

	_, err := client.ChargeInvoice(t.Context(), req)
	require.Equal(t, codes.Unavailable, status.Code(err))

	resp, err := client.ChargeInvoice(t.Context(), req)
	require.NoError(t, err)
	require.Equal(t, "TX-2", resp.GetTransactionId())
}

func TestInsufficientFundsCarriesErrorDetails(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectUnary(billingv1.BillingService_ChargeInvoice_FullMethodName).
		Match(sdk.Matches("invoice_id", `INV-\d+`)).
		ReturnErrorWithDetails(codes.FailedPrecondition, "insufficient funds", map[string]any{
			"type":   "type.googleapis.com/google.rpc.ErrorInfo",
			"reason": "INSUFFICIENT_FUNDS",
			"domain": "billing.example",
		})

	_, err := client.ChargeInvoice(t.Context(), &billingv1.ChargeInvoiceRequest{InvoiceId: "INV-3003"})

	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	require.NotEmpty(t, status.Convert(err).Details())
}

func TestBalanceResponseCarriesMetadata(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectUnary(billingv1.BillingService_GetBalance_FullMethodName).
		Match("customer_id", "CUST-7").
		ReturnHeaders(map[string]string{"x-ledger-shard": "eu-3"}).
		ReturnTrailers(map[string]string{"x-query-cost": "2"}).
		Return("balance_cents", 12_000, "currency", "EUR")

	var header, trailer metadata.MD

	resp, err := client.GetBalance(t.Context(),
		&billingv1.GetBalanceRequest{CustomerId: "CUST-7"},
		grpc.Header(&header), grpc.Trailer(&trailer))

	require.NoError(t, err)
	require.EqualValues(t, 12_000, resp.GetBalanceCents())
	require.Equal(t, []string{"eu-3"}, header.Get("x-ledger-shard"))
	require.Equal(t, []string{"2"}, trailer.Get("x-query-cost"))
}

func TestOnlyAuthorisedTenantsAreServed(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectUnary(billingv1.BillingService_GetBalance_FullMethodName).
		WithHeader(sdk.Equals("x-tenant", "acme")).
		Match("customer_id", "CUST-9").
		Return("balance_cents", 750, "currency", "USD")

	req := &billingv1.GetBalanceRequest{CustomerId: "CUST-9"}

	resp, err := client.GetBalance(metadata.AppendToOutgoingContext(t.Context(), "x-tenant", "acme"), req)
	require.NoError(t, err)
	require.EqualValues(t, 750, resp.GetBalanceCents())

	_, err = client.GetBalance(t.Context(), req)
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestCallBudgetIsVerified(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectUnary(billingv1.BillingService_GetBalance_FullMethodName).
		Match("customer_id", "CUST-8").
		Twice().
		Return("balance_cents", 500, "currency", "USD")

	req := &billingv1.GetBalanceRequest{CustomerId: "CUST-8"}

	require.Error(t, srv.ExpectationsWereMetContext(t.Context()))

	for range 2 {
		_, err := client.GetBalance(t.Context(), req)
		require.NoError(t, err)
	}

	require.Equal(t, 2, srv.Called(billingv1.BillingService_GetBalance_FullMethodName))
	require.Equal(t, 2, srv.TotalCalls())
}

func TestHistoryRecordsFailedCalls(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectUnary(billingv1.BillingService_ChargeInvoice_FullMethodName).
		Match("invoice_id", "INV-4004").
		Once().
		ReturnError(codes.PermissionDenied, "account frozen")

	_, err := client.ChargeInvoice(t.Context(), &billingv1.ChargeInvoiceRequest{InvoiceId: "INV-4004"})
	require.Error(t, err)

	history := srv.History()
	require.Len(t, history, 1)
	require.Equal(t, "ChargeInvoice", history[0].Method)
	require.Equal(t, uint32(codes.PermissionDenied), history[0].Code)
}
