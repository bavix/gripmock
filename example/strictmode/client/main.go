package main

import (
	"context"
	"log"
	"os"
	"time"

	grpcinterceptors "github.com/gripmock/grpc-interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	strictmode "github.com/bavix/gripmock/protogen/example/strictmode"
)

//nolint:mnd
func main() {
	// Set up a connection to the server.
	conn, err := grpc.NewClient("localhost:4770",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(grpcinterceptors.UnaryTimeoutInterceptor(5*time.Second)),
		grpc.WithChainStreamInterceptor(grpcinterceptors.StreamTimeoutInterceptor(5*time.Second)))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := strictmode.NewGripMockClient(conn)

	// Contact the server and print out its response.
	name := "GripMock Request"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	r1, err := c.SayLowerHello(context.Background(), &strictmode.SayLowerHelloRequest{Name: name}, grpc.WaitForReady(true))
	if err != nil {
		log.Fatalf("error from grpc: %v", err)
	}

	if r1.GetMessage() != "ok" {
		log.Fatalf("message is not ok: %s", r1.GetMessage())
	}

	r2, err := c.SayTitleHello(context.Background(), &strictmode.SayTitleHelloRequest{Name: name}, grpc.WaitForReady(true))
	if err != nil {
		log.Fatalf("error from grpc: %v", err)
	}

	if r2.GetMessage() != "OK" {
		log.Fatalf("message is not ok: %s", r2.GetMessage())
	}
}
