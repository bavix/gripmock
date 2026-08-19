package onboardingv1_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	onboardingv1 "github.com/bavix/gripmock/v3/examples/sdk/onboarding/v1"
	sdk "github.com/bavix/gripmock/v3/pkg/sdk"
)

const startupBudget = 15 * time.Second

func newClient(t *testing.T) (*sdk.Server, onboardingv1.OnboardingServiceClient) {
	t.Helper()

	srv := sdk.NewServer(t,
		sdk.WithHealthCheckTimeout(startupBudget),
		sdk.WithFileDescriptor(onboardingv1.File_examples_sdk_onboarding_v1_onboarding_proto),
	)

	return srv, onboardingv1.NewOnboardingServiceClient(srv.Conn())
}

func TestConfirmationIsUnlockedBySignup(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	unlockConfirmation := sdk.Upsert("onboarding.v1.OnboardingService", "ConfirmEmail").
		Match("email", "ann@example.com").
		Return("status", "confirmed", "next_step", "choose_workspace").
		Build()

	srv.ExpectUnary(onboardingv1.OnboardingService_StartSignup_FullMethodName).
		Match("email", "ann@example.com", "plan", "team").
		Effect(unlockConfirmation).
		Return("status", "pending", "next_step", "confirm_email")

	confirm := &onboardingv1.ConfirmEmailRequest{Email: "ann@example.com", Token: "T-1"}

	_, err := client.ConfirmEmail(t.Context(), confirm)
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))

	signup, err := client.StartSignup(t.Context(), &onboardingv1.StartSignupRequest{
		Email: "ann@example.com",
		Plan:  "team",
	})
	require.NoError(t, err)
	require.Equal(t, "confirm_email", signup.GetNextStep())

	confirmed, err := client.ConfirmEmail(t.Context(), confirm)
	require.NoError(t, err)
	require.Equal(t, "confirmed", confirmed.GetStatus())
}

func TestSpecificPlanWinsOverTheFallback(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectUnary(onboardingv1.OnboardingService_StartSignup_FullMethodName).
		Match(sdk.Matches("email", `.+@.+`)).
		Return("status", "pending", "next_step", "confirm_email")

	srv.ExpectUnary(onboardingv1.OnboardingService_StartSignup_FullMethodName).
		Match("plan", "enterprise").
		Priority(10).
		Return("status", "pending", "next_step", "assign_account_manager")

	enterprise, err := client.StartSignup(t.Context(), &onboardingv1.StartSignupRequest{
		Email: "cto@example.com",
		Plan:  "enterprise",
	})
	require.NoError(t, err)
	require.Equal(t, "assign_account_manager", enterprise.GetNextStep())

	free, err := client.StartSignup(t.Context(), &onboardingv1.StartSignupRequest{
		Email: "hobbyist@example.com",
		Plan:  "free",
	})
	require.NoError(t, err)
	require.Equal(t, "confirm_email", free.GetNextStep())
}

func TestResponseIsRenderedFromTheRequest(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectUnary(onboardingv1.OnboardingService_StartSignup_FullMethodName).
		Match(sdk.Matches("plan", `.+`)).
		Return("status", "pending", "next_step", "confirm_{{.Request.plan}}")

	resp, err := client.StartSignup(t.Context(), &onboardingv1.StartSignupRequest{
		Email: "dev@example.com",
		Plan:  "team",
	})

	require.NoError(t, err)
	require.Equal(t, "confirm_team", resp.GetNextStep())
}

func TestSingleUseTokenIsConsumed(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectUnary(onboardingv1.OnboardingService_ConfirmEmail_FullMethodName).
		Match("token", "T-ONCE").
		Once().
		Return("status", "confirmed", "next_step", "choose_workspace")

	req := &onboardingv1.ConfirmEmailRequest{Email: "sam@example.com", Token: "T-ONCE"}

	confirmed, err := client.ConfirmEmail(t.Context(), req)
	require.NoError(t, err)
	require.Equal(t, "confirmed", confirmed.GetStatus())

	_, err = client.ConfirmEmail(t.Context(), req)
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
}

func TestScenarioIsRestartedByReset(t *testing.T) {
	t.Parallel()

	srv, client := newClient(t)

	srv.ExpectUnary(onboardingv1.OnboardingService_StartSignup_FullMethodName).
		Match("plan", "trial").
		Return("status", "pending", "next_step", "confirm_email")

	req := &onboardingv1.StartSignupRequest{Email: "kim@example.com", Plan: "trial"}

	first, err := client.StartSignup(t.Context(), req)
	require.NoError(t, err)
	require.Equal(t, "confirm_email", first.GetNextStep())

	srv.Reset()

	_, err = client.StartSignup(t.Context(), req)
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))

	srv.ExpectUnary(onboardingv1.OnboardingService_StartSignup_FullMethodName).
		Match("plan", "trial").
		Return("status", "blocked", "next_step", "contact_support")

	second, err := client.StartSignup(t.Context(), req)
	require.NoError(t, err)
	require.Equal(t, "contact_support", second.GetNextStep())
}
