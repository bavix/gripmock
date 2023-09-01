package main

import (
	"context"
	"log"
	"os"
	"time"

	oneof "github.com/bavix/gripmock/protogen/example/one-of"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// Set up a connection to the server.
	conn, err := grpc.DialContext(ctx, "localhost:4770", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := oneof.NewGripmockClient(conn)

	// Contact the server and print out its response.
	name := "bavix"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	r, err := c.SayHello(context.Background(), &oneof.Request{Name: name})
	if err != nil {
		log.Fatalf("error from grpc: %v", err)
	}
	log.Printf("Reply1: %s", r.GetReply1())
	log.Printf("Reply2: %s", r.GetReply2())
	log.Printf("ReplyType: %s", r.GetReplyType())
}
