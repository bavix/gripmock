package sdk_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/bavix/gripmock/v3/pkg/sdk"
)

func TestMatchersRejectNonMatching(t *testing.T) {
	t.Parallel()

	t.Run("glob rejects non-match", func(t *testing.T) {
		t.Parallel()

		srv, fds := newServer(t)
		defer func() { _ = srv.Close() }()

		srv.ExpectUnary("/test.Greeter/SayHello").
			Match(sdk.Glob("name", "A*")).
			Return("message", "glob")

		require.Equal(t, "glob", getMsg(t, sayHello(t, srv, fds, "Alex")))
		require.Error(t, sayHelloErr(t, srv, fds, "Bob"),
			"Glob(name, A*) must not match Bob")
	})

	t.Run("anyof rejects non-match", func(t *testing.T) {
		t.Parallel()

		srv, fds := newServer(t)
		defer func() { _ = srv.Close() }()

		srv.ExpectUnary("/test.Greeter/SayHello").
			Match(sdk.AnyOf(
				sdk.Equals("name", "Alex"),
				sdk.Equals("name", "Bob"),
			)).
			Return("message", "anyof")

		require.Equal(t, "anyof", getMsg(t, sayHello(t, srv, fds, "Alex")))
		require.Error(t, sayHelloErr(t, srv, fds, "Zoe"),
			"AnyOf(Alex, Bob) must not match Zoe")
	})
}

func TestResetClearsHistoryInPlace(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").Return("message", "hi")
	sayHello(t, srv, fds, "a")
	require.Equal(t, 1, srv.Called("/test.Greeter/SayHello"))

	srv.Reset()
	require.Equal(t, 0, srv.TotalCalls(), "Reset must empty history")

	srv.ExpectUnary("/test.Greeter/SayHello").Return("message", "hi2")
	sayHello(t, srv, fds, "b")
	require.Equal(t, 1, srv.Called("/test.Greeter/SayHello"),
		"post-Reset calls must be recorded")
	require.Equal(t, 1, srv.TotalCalls())
}

func TestConfigAfterTerminalPanics(t *testing.T) {
	t.Parallel()

	srv, _ := newServer(t)
	defer func() { _ = srv.Close() }()

	require.Panics(t, func() {
		srv.ExpectUnary("/test.Greeter/SayHello").
			Return("message", "x").
			Times(3)
	}, "Times() after Return() must panic, not silently no-op")
}

func TestVerifyExactUnderCallFails(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").Match("name", "x").Times(2).Return("message", "y")
	sayHello(t, srv, fds, "x")

	require.Error(t, srv.ExpectationsWereMet(), "1 != 2 must fail exact verify")
}

func TestVerifyExactMatchPasses(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").Match("name", "x").Times(2).Return("message", "y")
	sayHello(t, srv, fds, "x")
	sayHello(t, srv, fds, "x")

	require.NoError(t, srv.ExpectationsWereMet())
}

func TestVerifyPerStubIndependence(t *testing.T) {
	t.Parallel()

	srv, fds := newServer(t)
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").Match("name", "a").Once().Return("message", "a")
	srv.ExpectUnary("/test.Greeter/SayHello").Match("name", "b").Twice().Return("message", "b")

	sayHello(t, srv, fds, "a")
	sayHello(t, srv, fds, "b")
	sayHello(t, srv, fds, "b")

	require.NoError(t, srv.ExpectationsWereMet())
}

func TestWithProtoFilesCompiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.proto")
	require.NoError(t, os.WriteFile(path, []byte(testProto), 0o600))

	fds := compileInline(t, testProto, "test.proto")

	srv := sdk.NewTestServer(t, sdk.WithProtoFiles(path))
	defer func() { _ = srv.Close() }()

	srv.ExpectUnary("/test.Greeter/SayHello").
		Match("name", "Alex").
		Return("message", "from-proto-file")

	require.Equal(t, "from-proto-file", getMsg(t, sayHello(t, srv, fds, "Alex")))
}
