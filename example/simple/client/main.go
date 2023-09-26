package main

import (
	"context"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/bavix/gripmock/protogen/example/simple"
)

//nolint:gomnd
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// Set up a connection to the server.
	conn, err := grpc.DialContext(ctx, "localhost:4770", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewGripmockClient(conn)

	// Contact the server and print out its response.
	name := "tokopedia"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	r, err := c.SayHello(context.Background(), &pb.Request{Name: name})
	if err != nil {
		log.Fatalf("error from grpc: %v", err)
	}
	if r.ReturnCode != 1 {
		log.Fatalf("grpc server returned code: %d, expected code: %d", r.ReturnCode, 1)
	}
	log.Printf("Greeting: %s (return code %d)", r.Message, r.ReturnCode)

	name = "world"
	r, err = c.SayHello(context.Background(), &pb.Request{Name: name})
	if err != nil {
		log.Fatalf("error from grpc: %v", err)
	}
	if r.ReturnCode != 1 {
		log.Fatalf("grpc server returned code: %d, expected code: %d", r.ReturnCode, 1)
	}
	log.Printf("Greeting: %s (return code %d)", r.Message, r.ReturnCode)

	name = "simple2"
	r, err = c.SayHello(context.Background(), &pb.Request{Name: name})
	if err != nil {
		log.Fatalf("error from grpc: %v", err)
	}
	if r.ReturnCode != 2 {
		log.Fatalf("grpc server returned code: %d, expected code: %d", r.ReturnCode, 2)
	}
	log.Printf("Greeting: %s (return code %d)", r.Message, r.ReturnCode)

	name = "simple3"
	r, err = c.SayHello(context.Background(), &pb.Request{Name: name})
	if err != nil {
		log.Fatalf("error from grpc: %v", err)
	}
	if r.ReturnCode != 3 {
		log.Fatalf("grpc server returned code: %d, expected code: %d", r.ReturnCode, 3)
	}
	log.Printf("Greeting: %s (return code %d)", r.Message, r.ReturnCode)

	md := metadata.New(map[string]string{"Authorization": "Basic dXNlcjp1c2Vy"})
	ctx = metadata.NewOutgoingContext(context.Background(), md)

	var headers metadata.MD

	name = "simple3"
	r, err = c.SayHello(ctx, &pb.Request{Name: name}, grpc.Header(&headers))
	if err != nil {
		log.Fatalf("error from grpc: %v", err)
	}
	if r.ReturnCode != 0 {
		log.Fatalf("grpc server returned code: %d, expected code: %d", r.ReturnCode, 0)
	}
	header := headers["result"]
	if len(header) == 0 {
		log.Fatal("the service did not return headers")
	}
	if header[0] != "ok" {
		log.Fatal("the service returned an incorrect header")
	}
	log.Printf("Greeting: %s (return code %d)", r.Message, r.ReturnCode)

	name = "simple3"
	r, err = c.SayHello(context.Background(), &pb.Request{Name: name}, grpc.UseCompressor(gzip.Name))
	if err != nil {
		log.Fatalf("error from grpc: %v", err)
	}
	if r.ReturnCode != 3 {
		log.Fatalf("grpc server returned code: %d, expected code: %d", r.ReturnCode, 3)
	}
	log.Printf("Greeting (gzip): %s (return code %d)", r.Message, r.ReturnCode)

	name = "error"
	r, err = c.SayHello(context.Background(), &pb.Request{Name: name})
	if err == nil {
		log.Fatalf("Expected error, but return %d", r.ReturnCode)
	}
	log.Printf("Greeting error: %s", err)

	name = "error_code"
	r, err = c.SayHello(context.Background(), &pb.Request{Name: name})
	if err == nil {
		log.Fatalf("Expected error, but return %d", r.ReturnCode)
	}

	s, ok := status.FromError(err)
	if !ok {
		log.Fatalf("Expected to get error status: %v", err)
	}

	if s.Code() != codes.InvalidArgument {
		log.Fatalf("Expected to get error status %d, got: %d", codes.InvalidArgument, s.Code())
	}

	log.Printf("Greeting error: %s, code: %d", err, s.Code())

	r, err = c.SayHello(context.Background(), &pb.Request{Vint64: 72057594037927936, Vuint64: 18446744073709551615})
	if err != nil {
		log.Fatalf("error from grpc: %v", err)
	}
	if r.Message != "72057594037927936 18446744073709551615" {
		log.Fatalf("failed to get valid message: %v", r.Message)
	}
	log.Printf("Greeting: %s (return code %d)", r.Message, r.ReturnCode)
}
