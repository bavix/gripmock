package sdk

import (
	"context"
	"runtime"
	"strconv"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/bavix/gripmock/v3/internal/infra/stuber"
	"github.com/bavix/gripmock/v3/pkg/sdk/internal/httpmock"
)

func TestFlushIsSafeWhileStubsAreRegistered(t *testing.T) {
	t.Parallel()

	const total = 200

	mock := httpmock.NewServer()
	t.Cleanup(mock.Close)

	srv := &Server{
		t:         t,
		bg:        context.Background(),
		batchMode: true,
		remote: &remoteMock{
			ctx:         context.Background(),
			restBaseURL: mock.URL,
			httpClient:  mock.HTTPServer.Client(),
		},
	}

	done := make(chan struct{})

	var wg sync.WaitGroup

	wg.Go(func() {
		defer close(done)

		for i := range total {
			srv.registerStub(&stuber.Stub{
				ID:      uuid.New(),
				Service: "race.Service",
				Method:  "Method" + strconv.Itoa(i),
				Input:   stuber.InputData{Equals: map[string]any{"i": i}},
				Output:  stuber.Output{Data: map[string]any{"ok": true}},
			})
		}
	})

	wg.Go(func() {
		for {
			select {
			case <-done:
				return
			default:
				_ = srv.Flush()

				runtime.Gosched()
			}
		}
	})

	wg.Wait()

	require.NoError(t, srv.Flush())
	require.Len(t, mock.Budgerigar.All(), total)
}
