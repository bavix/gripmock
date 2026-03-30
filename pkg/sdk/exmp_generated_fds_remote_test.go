package sdk_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	multiversepb "github.com/bavix/gripmock/v3/pkg/sdk/internal/examplefds/gen/examples/projects/multiverse"
	"github.com/bavix/gripmock/v3/pkg/sdk/internal/fdstest"
	"github.com/bavix/gripmock/v3/pkg/sdk/internal/testkit"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/pkg/sdk"
)

func TestExmpRemoteWithGeneratedDescriptorsUploadsImports(t *testing.T) {
	t.Parallel()

	grpcAddr := testkit.StartHealthGRPC(t)
	fds := fdstest.DescriptorSetFromFile(multiversepb.File_examples_projects_multiverse_service_proto)
	descriptorUploaded := make(chan struct{}, 1)
	stubPosted := make(chan struct{}, 1)

	rest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/descriptors":
			require.Equal(t, http.MethodPost, r.Method)

			raw, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			payload := &descriptorpb.FileDescriptorSet{}
			require.NoError(t, proto.Unmarshal(raw, payload))
			require.True(t, exmpHasFile(payload, "examples/projects/multiverse/service.proto"))
			require.True(t, exmpHasFile(payload, "google/protobuf/timestamp.proto"))

			nonBlockingSignal(descriptorUploaded)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"message":"ok"}`))

		case "/api/stubs":
			if r.Method == http.MethodPost {
				nonBlockingSignal(stubPosted)
				w.WriteHeader(http.StatusOK)
				return
			}

			w.WriteHeader(http.StatusOK)
			require.NoError(t, json.NewEncoder(w).Encode([]map[string]any{}))

		case "/api/verify", "/api/stubs/batchDelete":
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(rest.Close)

	mock, err := sdk.Run(t,
		sdk.WithRemote(grpcAddr, rest.URL),
		sdk.WithDescriptors(fds),
		sdk.WithHealthCheckTimeout(time.Second),
	)
	require.NoError(t, err)

	mock.Stub(sdk.By(multiversepb.MultiverseService_Ping_FullMethodName)).
		Reply(sdk.Data("reply", "pong")).
		Times(1).
		Commit()

	require.NoError(t, mock.Verify().VerifyStubTimesErr())
	require.NoError(t, mock.Close())
	requireSignal(t, descriptorUploaded)
	requireSignal(t, stubPosted)
}

func nonBlockingSignal(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

func requireSignal(t *testing.T, ch chan struct{}) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("expected signal")
	}
}

func exmpHasFile(fds *descriptorpb.FileDescriptorSet, name string) bool {
	for _, file := range fds.GetFile() {
		if file.GetName() == name {
			return true
		}
	}

	return false
}
