package main

import (
	"context"
	"log"
	"time"

	grpcinterceptors "github.com/gripmock/grpc-interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/bavix/gripmock/protogen/example/multi-files"
)

//nolint:gomnd
func main() {
	// Set up a connection to the server.
	conn, err := grpc.NewClient("localhost:4770", grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(grpcinterceptors.UnaryTimeoutInterceptor(5*time.Second)),
		grpc.WithChainStreamInterceptor(grpcinterceptors.StreamTimeoutInterceptor(5*time.Second)))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewGripmock1Client(conn)

	// Contact the server and print out its response.
	r, err := c.SayHello(context.Background(), &pb.Request1{Name: "tokopedia"})
	if err != nil {
		log.Fatalf("error from grpc: %v", err)
	}
	log.Printf("Greeting: %s", r.GetMessage())

	c2 := pb.NewGripmock2Client(conn)

	// Contact the server and print out its response.
	r2, err := c2.SayHello(context.Background(), &pb.Request2{Name: "tokopedia"})
	if err != nil {
		log.Fatalf("error from grpc: %v", err)
	}
	log.Printf("Greeting: %s", r2.GetMessage())
}
