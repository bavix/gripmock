package sdk_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bavix/gripmock/v3/pkg/sdk"
	multiversepb "github.com/bavix/gripmock/v3/pkg/sdk/internal/examplefds/gen/examples/projects/multiverse"
	"github.com/bavix/gripmock/v3/pkg/sdk/internal/fdstest"
	"github.com/bavix/gripmock/v3/pkg/sdk/internal/testkit"
)

func TestExmpRemoteWithGeneratedDescriptorsUploadsImports(t *testing.T) {
	t.Parallel()

	grpcAddr := testkit.StartHealthGRPC(t)
	fds := fdstest.DescriptorSetFromFile(multiversepb.File_examples_projects_multiverse_service_proto)
	descriptorUploaded := make(chan struct{}, 1)
	stubPosted := make(chan struct{}, 1)
	errCh := make(chan error, 5)

	rest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleUploadsServer(w, r, descriptorUploaded, stubPosted, errCh)
	}))
	t.Cleanup(rest.Close)

	mock, err := sdk.Run(t,
		sdk.WithRemote(grpcAddr, rest.URL),
		sdk.WithDescriptors(fds),
		sdk.WithHealthCheckTimeout(time.Second),
	)
	require.NoError(t, err)

	err = mock.Stub(sdk.By(multiversepb.MultiverseService_Ping_FullMethodName)).
		Reply(sdk.Data("reply", "pong")).
		Times(1).
		Commit()
	require.NoError(t, err)

	require.NoError(t, mock.Verify().VerifyStubTimesErr())
	require.NoError(t, mock.Close())
	requireSignal(t, descriptorUploaded)
	requireSignal(t, stubPosted)

	select {
	case err := <-errCh:
		t.Fatal(err)
	default:
	}
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

func handleUploadsServer(w http.ResponseWriter, r *http.Request, descriptorUploaded, stubPosted chan struct{}, errCh chan error) {
	handlerErr := func(msg string, args ...any) {
		select {
		case errCh <- fmt.Errorf(msg, args...):
		default:
		}
	}

	switch r.URL.Path {
	case "/api/descriptors":
		handleDescriptorUpload(w, r, descriptorUploaded, handlerErr)
	case "/api/stubs":
		handleStubUpload(w, r, stubPosted)
	case "/api/verify", "/api/stubs/batchDelete":
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func handleDescriptorUpload(w http.ResponseWriter, r *http.Request, descriptorUploaded chan struct{}, handlerErr func(string, ...any)) {
	if r.Method != http.MethodPost {
		handlerErr("expected POST, got %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)

		return
	}

	raw, err := io.ReadAll(r.Body)
	if err != nil {
		handlerErr("read body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	payload := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(raw, payload); err != nil {
		handlerErr("unmarshal: %v", err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	if !exmpHasFile(payload, "examples/projects/multiverse/service.proto") {
		handlerErr("missing service.proto in descriptor set")
	}
	if !exmpHasFile(payload, "google/protobuf/timestamp.proto") {
		handlerErr("missing timestamp.proto in descriptor set")
	}

	nonBlockingSignal(descriptorUploaded)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"message":"ok"}`))
}

func handleStubUpload(w http.ResponseWriter, r *http.Request, stubPosted chan struct{}) {
	if r.Method == http.MethodPost {
		nonBlockingSignal(stubPosted)
		w.WriteHeader(http.StatusOK)

		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("[]"))
}
