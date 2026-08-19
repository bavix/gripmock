package app

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestSerializeErrorStatusUsesProtocolDetailShape(t *testing.T) {
	t.Parallel()

	st, err := status.New(codes.ResourceExhausted, "quota exhausted").
		WithDetails(&errdetails.ErrorInfo{Reason: "QUOTA", Domain: "e2e"})
	require.NoError(t, err)

	body := serializeErrorStatus(st)

	require.Equal(t, "resource_exhausted", body.Code)
	require.Equal(t, "quota exhausted", body.Message)
	require.Len(t, body.Details, 1)
	require.Equal(t, "google.rpc.ErrorInfo", body.Details[0].Type,
		"the type travels bare, without the Any URL prefix")

	raw, err := base64.RawStdEncoding.DecodeString(body.Details[0].Value)
	require.NoError(t, err)

	var info errdetails.ErrorInfo

	require.NoError(t, proto.Unmarshal(raw, &info))
	require.Equal(t, "QUOTA", info.GetReason())

	require.JSONEq(t, `{
		"@type": "type.googleapis.com/google.rpc.ErrorInfo",
		"domain": "e2e",
		"reason": "QUOTA"
	}`, string(body.Details[0].Debug), "the readable rendering stays, as debug")
}

func TestSerializeErrorStatusOmitsEmptyDetails(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(serializeErrorStatus(status.New(codes.NotFound, "missing")))
	require.NoError(t, err)

	require.JSONEq(t, `{"code":"not_found","message":"missing"}`, string(encoded))
}
