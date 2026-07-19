package sdk

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/pkg/sdk/internal/httpmock"
	"github.com/bavix/gripmock/v3/pkg/sdk/internal/remoteapi"
)

func newRemoteMock(t *testing.T) (*httpmock.Server, remoteapi.Client) {
	t.Helper()

	mock := httpmock.NewServer()
	t.Cleanup(mock.Close)

	return mock, remoteapi.Client{
		BaseURL:    mock.URL,
		HTTPClient: mock.HTTPServer.Client(),
	}
}

func TestRemoteAddStubs(t *testing.T) {
	t.Parallel()

	mock, client := newRemoteMock(t)

	err := client.AddStubs([]*stuber.Stub{{
		Service: "test.Service",
		Method:  "TestMethod",
		Input:   stuber.InputData{Equals: map[string]any{"key": "value"}},
		Output:  stuber.Output{Data: map[string]any{"result": "ok"}},
	}})
	require.NoError(t, err)
	require.Len(t, mock.Budgerigar.All(), 1)
}

func TestRemoteBatchDelete(t *testing.T) {
	t.Parallel()

	mock, client := newRemoteMock(t)

	require.NoError(t, client.AddStubs([]*stuber.Stub{
		{
			Service: "svc", Method: "m1",
			Input:  stuber.InputData{Equals: map[string]any{"id": "1"}},
			Output: stuber.Output{Data: map[string]any{"ok": true}},
		},
		{
			Service: "svc", Method: "m2",
			Input:  stuber.InputData{Equals: map[string]any{"id": "2"}},
			Output: stuber.Output{Data: map[string]any{"ok": true}},
		},
	}))
	require.Len(t, mock.Budgerigar.All(), 2)

	mock.Budgerigar.Clear()
	require.Empty(t, mock.Budgerigar.All())
}

func TestRemoteVerifyCalls(t *testing.T) {
	t.Parallel()

	mock, client := newRemoteMock(t)

	mock.RecordCall("svc", "method", nil, nil)
	mock.RecordCall("svc", "method", nil, nil)

	require.NoError(t, client.VerifyMethodCalled("svc", "method", 2))

	err := client.VerifyMethodCalled("svc", "method", 1)
	require.Error(t, err)
}

func TestRemoteHistory(t *testing.T) {
	t.Parallel()

	mock, client := newRemoteMock(t)

	mock.RecordCall("svc", "method", map[string]any{"req": "1"}, map[string]any{"resp": "ok"})

	history, err := client.FetchHistory()
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, "svc", history[0].Service)
	require.Equal(t, "method", history[0].Method)
}

func TestRemoteSessionIsolation(t *testing.T) {
	t.Parallel()

	mock, clientBase := newRemoteMock(t)
	clientA := remoteapi.Client{
		BaseURL:    clientBase.BaseURL,
		HTTPClient: clientBase.HTTPClient,
		Session:    "session-A",
	}

	require.NoError(t, clientA.AddStubs([]*stuber.Stub{{
		Service: "svc", Method: "m1",
		Input:  stuber.InputData{Equals: map[string]any{"id": "1"}},
		Output: stuber.Output{Data: map[string]any{"ok": true}},
	}}))

	all := mock.Budgerigar.All()
	require.Len(t, all, 1)
	require.Equal(t, "session-A", all[0].Session)

	found, _ := mock.Budgerigar.FindByQuery(stuber.Query{
		Service: "svc",
		Method:  "m1",
		Session: "session-A",
	})
	require.NotNil(t, found)

	foundB, _ := mock.Budgerigar.FindByQuery(stuber.Query{
		Service: "svc",
		Method:  "m1",
		Session: "session-B",
	})
	require.Nil(t, foundB)
}

func TestRemoteBatch(t *testing.T) {
	t.Parallel()

	mock, client := newRemoteMock(t)

	stubs := []*stuber.Stub{
		{
			Service: "svc", Method: "m1",
			Input:  stuber.InputData{Equals: map[string]any{"id": "1"}},
			Output: stuber.Output{Data: map[string]any{"ok": true}},
		},
		{
			Service: "svc", Method: "m2",
			Input:  stuber.InputData{Equals: map[string]any{"id": "2"}},
			Output: stuber.Output{Data: map[string]any{"ok": true}},
		},
		{Service: "svc", Method: "m3", Input: stuber.InputData{Equals: map[string]any{"id": "3"}}, Output: stuber.Output{Data: map[string]any{"ok": true}}}, //nolint:lll
	}

	require.NoError(t, client.AddStubs(stubs))
	require.Len(t, mock.Budgerigar.All(), 3)
}
