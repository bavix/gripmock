package sdk

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestHelpersEquals(t *testing.T) {
	t.Parallel()
	id := Equals("key", "value")
	require.NotNil(t, id.Equals)
	require.Equal(t, "value", id.Equals["key"])
}

func TestHelpersContains(t *testing.T) {
	t.Parallel()
	id := Contains("key", "value")
	require.NotNil(t, id.Contains)
	require.Equal(t, "value", id.Contains["key"])
}

func TestHelpersMatches(t *testing.T) {
	t.Parallel()
	id := Matches("key", `\d+`)
	require.NotNil(t, id.Matches)
	require.Equal(t, `\d+`, id.Matches["key"])
}

func TestHelpersMap(t *testing.T) {
	t.Parallel()
	id := Map("a", 1, "b", "two")
	require.NotNil(t, id.Equals)
	require.Equal(t, 1, id.Equals["a"])
	require.Equal(t, "two", id.Equals["b"])
}

func TestHelpersMapPanicOddArgs(t *testing.T) {
	t.Parallel()
	require.PanicsWithValue(t, "sdk.Map: need pairs (key, value), got 3 args", func() {
		Map("a", 1, "b")
	})
}

func TestHelpersMapPanicNonStringKey(t *testing.T) {
	t.Parallel()
	require.PanicsWithValue(t, "sdk.Map: key at 0 must be string, got int", func() {
		Map(123, "value")
	})
}

func TestHelpersData(t *testing.T) {
	t.Parallel()
	out := Data("msg", "hello", "n", 42)
	require.NotNil(t, out.Data)
	require.Equal(t, "hello", out.Data["msg"])
	require.Equal(t, 42, out.Data["n"])
}

func TestHelpersDataPanicOddArgs(t *testing.T) {
	t.Parallel()
	require.PanicsWithValue(t, "sdk.Data: need pairs (key, value), got 3 args", func() {
		Data("a", 1, "b")
	})
}

func TestHelpersDataPanicNonStringKey(t *testing.T) {
	t.Parallel()
	require.PanicsWithValue(t, "sdk.Data: key at 0 must be string, got int", func() {
		Data(123, "value")
	})
}

func TestHelpersHeaderEquals(t *testing.T) {
	t.Parallel()
	h := HeaderEquals("authorization", "Bearer token")
	require.NotNil(t, h.Equals)
	require.Equal(t, "Bearer token", h.Equals["authorization"])
}

func TestHelpersHeaderMatches(t *testing.T) {
	t.Parallel()
	h := HeaderMatches("authorization", `^Bearer\s+.+$`)
	require.NotNil(t, h.Matches)
	require.Equal(t, `^Bearer\s+.+$`, h.Matches["authorization"])
}

func TestHelpersHeaderMap(t *testing.T) {
	t.Parallel()
	h := HeaderMap("x-id", "123", "x-name", "test")
	require.NotNil(t, h.Equals)
	require.Equal(t, "123", h.Equals["x-id"])
	require.Equal(t, "test", h.Equals["x-name"])
}

func TestHelpersIgnoreArrayOrder(t *testing.T) {
	t.Parallel()
	id := IgnoreArrayOrder(Equals("arr", []any{1, 2}))
	require.True(t, id.IgnoreArrayOrder)
	require.Equal(t, []any{1, 2}, id.Equals["arr"])
}

func TestHelpersMerge(t *testing.T) {
	t.Parallel()
	id := Merge(
		Equals("name", "Alex"),
		Contains("tags", "go"),
		Matches("email", `.*@test\.com`),
		IgnoreOrder(),
	)
	require.Equal(t, "Alex", id.Equals["name"])
	require.Equal(t, "go", id.Contains["tags"])
	require.Equal(t, `.*@test\.com`, id.Matches["email"])
	require.True(t, id.IgnoreArrayOrder)
}

func TestHelpersMergeOutput(t *testing.T) {
	t.Parallel()
	out := MergeOutput(
		Data("message", "Hi", "code", 200),
		ReplyHeader("x-custom", "value"),
		ReplyDelay(10*time.Millisecond),
		ReplyErrWithDetails(codes.InvalidArgument, "validation failed", map[string]any{
			"type":  "type.googleapis.com/google.protobuf.StringValue",
			"value": "merge detail",
		}),
	)
	require.Equal(t, "Hi", out.Data["message"])
	require.Equal(t, 200, out.Data["code"])
	require.Equal(t, "value", out.Headers["x-custom"])
	require.NotZero(t, out.Delay)
	require.Equal(t, "validation failed", out.Error)
	require.Len(t, out.Details, 1)
}

func TestHelpersMergeHeaders(t *testing.T) {
	t.Parallel()
	h := MergeHeaders(
		HeaderEquals("x-id", "123"),
		HeaderContains("user-agent", "test"),
	)
	require.Equal(t, "123", h.Equals["x-id"])
	require.Equal(t, "test", h.Contains["user-agent"])
}

func TestHelpersReplyOutputModifiers(t *testing.T) {
	t.Parallel()
	require.Equal(t, map[string]string{"x": "y"}, ReplyHeader("x", "y").Headers)
	require.NotZero(t, ReplyDelay(10*time.Millisecond).Delay)
	require.Equal(t, "err", ReplyErr(codes.InvalidArgument, "err").Error)
	require.Len(t, ReplyErrWithDetails(codes.InvalidArgument, "err", map[string]any{"type": "type.googleapis.com/google.protobuf.StringValue", "value": "v"}).Details, 1)
	out := StreamItem("msg", "hi")
	require.Len(t, out.Stream, 1)
	require.Equal(t, "hi", out.Stream[0].(map[string]any)["msg"])
}

func TestParseFullMethodName(t *testing.T) {
	t.Parallel()

	svc, method, err := ParseFullMethodName("/helloworld.Greeter/SayHello")
	require.NoError(t, err)
	require.Equal(t, "helloworld.Greeter", svc)
	require.Equal(t, "SayHello", method)

	svc, method, err = ParseFullMethodName("helloworld.Greeter/SayHello")
	require.NoError(t, err)
	require.Equal(t, "helloworld.Greeter", svc)
	require.Equal(t, "SayHello", method)

	_, _, err = ParseFullMethodName("")
	require.EqualError(t, err, "sdk: full method name is empty")

	_, _, err = ParseFullMethodName("bad")
	require.EqualError(t, err, "sdk: invalid full method name \"bad\"")

	_, _, err = ParseFullMethodName("svc/")
	require.EqualError(t, err, "sdk: invalid full method name \"svc/\"")

	_, _, err = ParseFullMethodName("/svc/more/extra")
	require.EqualError(t, err, "sdk: invalid full method name \"svc/more/extra\"")
}

func TestStubAndVerifyWithFullMethod(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		Unary("name", "Alex", "message", "Hi Alex").
		Commit()

	msg := invokeGreeterSayHello(t, mock.Conn(), reg, t.Context(), "Alex")
	require.Equal(t, "Hi Alex", getMessageField(t, msg, "message"))

	mock.Verify().Method(By("/helloworld.Greeter/SayHello")).Called(t, 1)
}

func TestMustFullMethodHelpersPanicOnInvalidInput(t *testing.T) {
	t.Parallel()

	require.PanicsWithValue(t, "sdk: invalid full method name \"bad\"", func() {
		_, _ = MustParseFullMethodName("bad")
	})

	mock, _ := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	require.PanicsWithValue(t, "sdk: invalid full method name \"bad\"", func() {
		mock.Stub(By("bad"))
	})

	require.PanicsWithValue(t, "sdk: invalid full method name \"bad\"", func() {
		mock.Verify().Method(By("bad"))
	})
}

func TestMockStubAcceptsFullMethodName(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		Unary("name", "Alex", "message", "Hi Alex").
		Commit()

	msg := invokeGreeterSayHello(t, mock.Conn(), reg, t.Context(), "Alex")
	require.Equal(t, "Hi Alex", getMessageField(t, msg, "message"))
}

func TestMockStubPanicsOnInvalidArguments(t *testing.T) {
	t.Parallel()

	mock, _ := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	require.PanicsWithValue(t, "sdk: invalid full method name \"bad\"", func() {
		mock.Stub(By("bad"))
	})

	require.PanicsWithValue(t, "sdk: invalid full method name \"svc/\"", func() {
		mock.Stub(By("svc/"))
	})
}

func TestParseFullMethodNameValidationTable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		input     string
		wantSvc   string
		wantMth   string
		wantError string
	}{
		{name: "full-leading-slash", input: "/svc/M", wantSvc: "svc", wantMth: "M"},
		{name: "full-no-leading-slash", input: "svc/M", wantSvc: "svc", wantMth: "M"},
		{name: "full-trim-spaces", input: "  /svc/M  ", wantSvc: "svc", wantMth: "M"},
		{name: "empty", input: "", wantError: "sdk: full method name is empty"},
		{name: "only-service", input: "svc", wantError: "sdk: invalid full method name \"svc\""},
		{name: "missing-method", input: "svc/", wantError: "sdk: invalid full method name \"svc/\""},
		{name: "missing-service", input: "/M", wantError: "sdk: invalid full method name \"M\""},
		{name: "extra-slash", input: "/svc/M/extra", wantError: "sdk: invalid full method name \"svc/M/extra\""},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc, mth, err := ParseFullMethodName(tc.input)
			if tc.wantError != "" {
				require.EqualError(t, err, tc.wantError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantSvc, svc)
			require.Equal(t, tc.wantMth, mth)
		})
	}
}

func TestMockStubAndVerifierTwoSignatures(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	mock.Stub("helloworld.Greeter", "SayHello").
		Unary("name", "Alex", "message", "Hi Alex").
		Commit()

	mock.Stub(By("/helloworld.Greeter/SayHello")).
		Unary("name", "Bob", "message", "Hi Bob").
		Commit()

	msg1 := invokeGreeterSayHello(t, mock.Conn(), reg, t.Context(), "Alex")
	require.Equal(t, "Hi Alex", getMessageField(t, msg1, "message"))

	msg2 := invokeGreeterSayHello(t, mock.Conn(), reg, t.Context(), "Bob")
	require.Equal(t, "Hi Bob", getMessageField(t, msg2, "message"))

	mock.Verify().Method("helloworld.Greeter", "SayHello").Called(t, 2)
	mock.Verify().Method(By("/helloworld.Greeter/SayHello")).Called(t, 2)
}

func TestStubBatchEmbedded(t *testing.T) {
	t.Parallel()

	mock, reg := mustRunWithProtoAndReg(t, sdkProtoPath("greeter"))

	batch := NewBatch(mock)
	batch.Stub("helloworld.Greeter", "SayHello").
		When(Equals("name", "Alex")).
		Reply(Data("message", "Hi Alex")).
		Commit()
	batch.Stub(By("/helloworld.Greeter/SayHello")).
		When(Equals("name", "Bob")).
		Reply(Data("message", "Hi Bob")).
		Commit()

	require.NoError(t, batch.Commit())

	msg1 := invokeGreeterSayHello(t, mock.Conn(), reg, t.Context(), "Alex")
	require.Equal(t, "Hi Alex", getMessageField(t, msg1, "message"))

	msg2 := invokeGreeterSayHello(t, mock.Conn(), reg, t.Context(), "Bob")
	require.Equal(t, "Hi Bob", getMessageField(t, msg2, "message"))
}
