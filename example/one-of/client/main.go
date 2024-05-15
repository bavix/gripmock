package main

import (
	"context"
	"log"
	"os"
	"time"

	grpcinterceptors "github.com/gripmock/grpc-interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	oneof "github.com/bavix/gripmock/protogen/example/one-of"
)

//nolint:gomnd
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

	c := oneof.NewGripmockClient(conn)

	// Contact the server and print out its response.
	name := "tokopedia"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	r, err := c.SayHello(context.Background(), &oneof.Request{Name: name}, grpc.WaitForReady(true))
	if err != nil {
		log.Fatalf("error from grpc: %v", err)
	}
	log.Printf("Reply1: %s", r.GetReply1())
	log.Printf("Reply2: %s", r.GetReply2())
	log.Printf("ReplyType: %s", r.GetReplyType())
}
