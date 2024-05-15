package main

import (
	"context"
	"log"
	"os"
	"time"

	grpcinterceptors "github.com/gripmock/grpc-interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/bavix/gripmock/protogen/example/multi-package"
	multipackage "github.com/bavix/gripmock/protogen/example/multi-package/bar"
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

	c := pb.NewGripmockClient(conn)

	// Contact the server and print out its response.
	name := "tokopedia"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}

	r, err := c.Greet(context.Background(), &multipackage.Bar{Name: name})
	if err != nil {
		log.Fatalf("error from grpc: %v", err)
	}

	log.Printf("Greeting: %s", r.GetResponse())
}
