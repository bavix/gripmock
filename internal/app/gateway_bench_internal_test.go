package app

import (
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bufbuild/protocompile"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/bavix/gripmock/v3/internal/domain/descriptors"
	"github.com/bavix/gripmock/v3/internal/domain/history"
	"github.com/bavix/gripmock/v3/internal/infra/proxyroutes"
	"github.com/bavix/gripmock/v3/internal/infra/stuber"
)

const gatewayProto = `
syntax = "proto3";
package gw;
service Greeter {
  rpc SayHello (HelloRequest) returns (HelloReply);
}
message HelloRequest { string name = 1; }
message HelloReply { string message = 1; }
`

//nolint:ireturn // protoreflect returns interfaces
func benchGateway(b *testing.B) (http.Handler, protoreflect.MessageDescriptor) {
	b.Helper()

	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(map[string]string{"gw.proto": gatewayProto}),
		}),
	}

	files, err := compiler.Compile(b.Context(), "gw.proto")
	if err != nil {
		b.Fatal(err)
	}

	registry := descriptors.NewRegistry()
	registry.Register(files[0])

	budgerigar := stuber.NewBudgerigar()
	budgerigar.PutMany(&stuber.Stub{
		ID:      uuid.New(),
		Service: "gw.Greeter",
		Method:  "SayHello",
		Input:   stuber.InputData{Equals: map[string]any{"name": "Alex"}},
		Output:  stuber.Output{Data: map[string]any{"message": "Hello Alex"}},
	})

	gateway := NewMultiProtocolGateway(
		b.Context(),
		budgerigar,
		registry,
		history.NewMemoryStore(0),
		&atomic.Pointer[proxyroutes.Registry]{},
		nil,
		NewErrorFormatter(),
	)

	router := mux.NewRouter()
	router.Handle("/{service}/{method}", gateway).Methods(http.MethodPost, http.MethodGet)

	return router, files[0].Messages().ByName("HelloRequest")
}

func BenchmarkGatewayConnectUnary(b *testing.B) {
	handler, _ := benchGateway(b)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		req := httptest.NewRequestWithContext(b.Context(), http.MethodPost,
			"/gw.Greeter/SayHello", strings.NewReader(`{"name":"Alex"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Connect-Protocol-Version", "1")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			b.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	}
}

func BenchmarkGatewayGRPCWebUnary(b *testing.B) {
	handler, in := benchGateway(b)

	request := dynamicpb.NewMessage(in)
	request.Set(in.Fields().ByName("name"), protoreflect.ValueOfString("Alex"))

	payload, err := proto.Marshal(request)
	if err != nil {
		b.Fatal(err)
	}

	frame := make([]byte, 5+len(payload))
	binary.BigEndian.PutUint32(frame[1:5], uint32(len(payload))) //nolint:gosec
	copy(frame[5:], payload)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		req := httptest.NewRequestWithContext(b.Context(), http.MethodPost,
			"/gw.Greeter/SayHello", strings.NewReader(string(frame)))
		req.Header.Set("Content-Type", "application/grpc-web+proto")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			b.Fatalf("status %d: %s", rec.Code, rec.Body.String())
		}
	}
}
