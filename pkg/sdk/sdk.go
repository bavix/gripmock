// Package sdk provides an embedded gRPC mock server for tests.
//
// # Quick Start
//
//	srv := sdk.NewServer(t, sdk.WithFileDescriptor(helloworld.File_service_proto))
//	defer srv.Close()
//
//	srv.ExpectUnary("/helloworld.Greeter/SayHello").
//	    Match("name", "Alex").
//	    Return("message", "Hi Alex")
//
//	client := helloworld.NewGreeterClient(srv.Conn())
//	resp, err := client.SayHello(t.Context(), &pb.HelloRequest{Name: "Alex"})
//	// resp.Message == "Hi Alex"
//
// # Unary
//
//	srv.ExpectUnary("/svc/Method").
//	    Match("field", "value").
//	    Return("responseField", "responseValue")
//
// With typed protobuf matching (use key-value pairs or Match):
//
//	srv.ExpectUnary("/svc/Method").
//	    Match("field", "value").
//	    Return("responseField", "responseValue")
//
// With sequential responses (first call→r1, second→r2):
//
//	srv.ExpectUnary("/svc/Method").
//	    Match("step", "process").
//	    Return("status", "first").
//	    NextWillReturn("status", "second")
//
// With delay:
//
//	srv.ExpectUnary("/svc/Method").
//	    Match("field", "value").
//	    Return(Delay(100*time.Millisecond, "responseField", "responseValue"))
//
// # Server Stream
//
//	srv.ExpectServerStream("/svc/Stream").
//	    Match("query", "test").
//	    SendStream(
//	        map[string]any{"id": "1", "title": "result 1"},
//	    )
//
// # Client Stream
//
//	srv.ExpectClientStream("/svc/Stream").
//	    Match(sdk.Matches("value", "\\d+")).
//	    Return("result", 42.0)
//
// # Bidirectional Stream
//
//	srv.ExpectBidirectionalStream("/svc/Bidi").
//	    Run(func(ctx context.Context, stream any) error {
//	        return nil
//	    })
//
// # Effects
//
//	effect := sdk.Upsert("svc", "NextMethod").
//	    Match("step", "complete").
//	    Return("status", "done").
//	    Build()
//	srv.ExpectUnary("/svc/Method").
//	    Match("step", "begin").
//	    Effect(effect).
//	    Return("status", "started")
//
// # Verification
//
//	err := srv.ExpectationsWereMet()
//	n := srv.Called("/svc/Method")
//	total := srv.TotalCalls()
//	history := srv.History()
//
// # Options
//
//	srv := sdk.NewServer(t,
//	    sdk.WithProtoFiles("service.proto"),
//	    sdk.WithHealthCheckTimeout(10*time.Second),
//	)
package sdk

import "context"

// UnaryHandler processes a unary gRPC request and returns a response.
type UnaryHandler func(ctx context.Context, in any) (any, error)

// ServerStreamHandler processes a server-stream request.
// Handler receives the request and a stream for sending responses.
type ServerStreamHandler func(ctx context.Context, in any, stream any) error

// ClientStreamHandler processes a client-stream request.
// Read all incoming messages from the stream, then return a single response.
type ClientStreamHandler func(ctx context.Context, stream any) (any, error)

// BidirectionalHandler processes a bidirectional stream.
// Handler manages both send and receive on the stream.
type BidirectionalHandler func(ctx context.Context, stream any) error
