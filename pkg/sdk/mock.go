package sdk

import "google.golang.org/grpc"

// Mock represents a running gRPC mock server.
//
// Deprecated: use *Server returned by NewServer instead.
type Mock interface {
	Conn() *grpc.ClientConn
	Addr() string
	Stub(service, method string) StubBuilder
	History() HistoryReader
	Verify() Verifier
	Close() error
}
