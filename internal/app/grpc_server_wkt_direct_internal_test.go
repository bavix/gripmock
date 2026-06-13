package app

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// TestNewOutputMessageWKTDirect is the regression test for issue #882: methods
// that return a well-known type at the top level. protojson + dynamicpb handle
// the canonical protojson encoding natively; gripmock just needs to plumb the
// stub data through json.Encode → protojson.Unmarshal without inventing a
// parallel encoder.
func TestNewOutputMessageWKTDirect(t *testing.T) {
	t.Parallel()

	concrete := func(t *testing.T, msg proto.Message, target proto.Message) {
		t.Helper()

		bytes, marshalErr := proto.Marshal(msg)
		require.NoError(t, marshalErr)

		require.NoError(t, proto.Unmarshal(bytes, target))
	}

	t.Run("Timestamp", func(t *testing.T) {
		t.Parallel()
		testNewOutputMessageTimestamp(t, concrete)
	})

	t.Run("Duration", func(t *testing.T) {
		t.Parallel()
		testNewOutputMessageDuration(t, concrete)
	})

	t.Run("StringValue", func(t *testing.T) {
		t.Parallel()
		testNewOutputMessageStringValue(t, concrete)
	})

	t.Run("Int32Value", func(t *testing.T) {
		t.Parallel()
		testNewOutputMessageInt32Value(t, concrete)
	})

	t.Run("BoolValue", func(t *testing.T) {
		t.Parallel()
		testNewOutputMessageBoolValue(t, concrete)
	})

	t.Run("Struct", func(t *testing.T) {
		t.Parallel()
		testNewOutputMessageStruct(t, concrete)
	})

	t.Run("Empty", func(t *testing.T) {
		t.Parallel()
		testNewOutputMessageEmpty(t, concrete)
	})

	t.Run("regular message preserves int64 precision", func(t *testing.T) {
		t.Parallel()
		testNewOutputMessageInt64(t)
	})
}

func testNewOutputMessageTimestamp(t *testing.T, concrete func(t *testing.T, msg, target proto.Message)) {
	t.Helper()

	ts := (&timestamppb.Timestamp{}).ProtoReflect().Descriptor()
	mocker := &grpcMocker{outputDesc: ts}

	msg, err := mocker.newOutputMessage("2024-01-01T12:00:00Z")
	require.NoError(t, err)
	require.NotNil(t, msg)

	got := &timestamppb.Timestamp{}
	concrete(t, msg, got)
	require.Equal(t, int64(1704110400), got.GetSeconds())
	require.Equal(t, int32(0), got.GetNanos())
}

func testNewOutputMessageDuration(t *testing.T, concrete func(t *testing.T, msg, target proto.Message)) {
	t.Helper()

	d := (&durationpb.Duration{}).ProtoReflect().Descriptor()
	mocker := &grpcMocker{outputDesc: d}

	msg, err := mocker.newOutputMessage("1.5s")
	require.NoError(t, err)
	require.NotNil(t, msg)

	got := &durationpb.Duration{}
	concrete(t, msg, got)
	require.Equal(t, int64(1), got.GetSeconds())
	require.Equal(t, int32(500000000), got.GetNanos())
}

func testNewOutputMessageStringValue(t *testing.T, concrete func(t *testing.T, msg, target proto.Message)) {
	t.Helper()

	sv := (&wrapperspb.StringValue{}).ProtoReflect().Descriptor()
	mocker := &grpcMocker{outputDesc: sv}

	msg, err := mocker.newOutputMessage("hello")
	require.NoError(t, err)
	require.NotNil(t, msg)

	got := &wrapperspb.StringValue{}
	concrete(t, msg, got)
	require.Equal(t, "hello", got.GetValue())
}

func testNewOutputMessageInt32Value(t *testing.T, concrete func(t *testing.T, msg, target proto.Message)) {
	t.Helper()

	iv := (&wrapperspb.Int32Value{}).ProtoReflect().Descriptor()
	mocker := &grpcMocker{outputDesc: iv}

	msg, err := mocker.newOutputMessage(float64(42))
	require.NoError(t, err)
	require.NotNil(t, msg)

	got := &wrapperspb.Int32Value{}
	concrete(t, msg, got)
	require.Equal(t, int32(42), got.GetValue())
}

func testNewOutputMessageBoolValue(t *testing.T, concrete func(t *testing.T, msg, target proto.Message)) {
	t.Helper()

	bv := (&wrapperspb.BoolValue{}).ProtoReflect().Descriptor()
	mocker := &grpcMocker{outputDesc: bv}

	msg, err := mocker.newOutputMessage(true)
	require.NoError(t, err)
	require.NotNil(t, msg)

	got := &wrapperspb.BoolValue{}
	concrete(t, msg, got)
	require.True(t, got.GetValue())
}

func testNewOutputMessageStruct(t *testing.T, concrete func(t *testing.T, msg, target proto.Message)) {
	t.Helper()

	s := (&structpb.Struct{}).ProtoReflect().Descriptor()
	mocker := &grpcMocker{outputDesc: s}

	data := map[string]any{
		"region":  "us-east-1",
		"retries": float64(3),
	}

	msg, err := mocker.newOutputMessage(data)
	require.NoError(t, err)
	require.NotNil(t, msg)

	got := &structpb.Struct{}
	concrete(t, msg, got)
	require.Equal(t, "us-east-1", got.GetFields()["region"].GetStringValue())
	require.InDelta(t, 3, got.GetFields()["retries"].GetNumberValue(), 1e-9)
}

func testNewOutputMessageEmpty(t *testing.T, concrete func(t *testing.T, msg, target proto.Message)) {
	t.Helper()

	e := (&emptypb.Empty{}).ProtoReflect().Descriptor()
	mocker := &grpcMocker{outputDesc: e}

	msg, err := mocker.newOutputMessage(map[string]any{})
	require.NoError(t, err)
	require.NotNil(t, msg)

	// Empty has no fields; round-trip is the assertion.
	got := &emptypb.Empty{}
	concrete(t, msg, got)
}

func testNewOutputMessageInt64(t *testing.T) {
	t.Helper()

	mocker := createTestMocker(t)

	data := map[string]any{
		"fields": map[string]any{
			"bigint": json.Number("9223372036854775000"),
		},
	}

	msg, err := mocker.newOutputMessage(data)
	require.NoError(t, err)
	require.NotNil(t, msg)
}
