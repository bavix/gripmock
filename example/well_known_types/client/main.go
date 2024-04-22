package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	empty "google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/bavix/gripmock/protogen/example/well_known_types"
)

// in order to generate this .pb.go you need to have https://github.com/google/protobuf.git cloned
// then use it as protobuf_dir below
// protoc --go_out=plugins=grpc:${GOPATH}/src -I=.. -I=<protobuf_dir>  ../wkt.proto
//
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

	r, err := c.ApiInfo(context.Background(), &empty.Empty{})
	if err != nil {
		log.Fatalf("error from grpc: %v", err)
	}

	if r.GetName() != "Gripmock" {
		log.Fatalf("expecting api name: Gripmock, but got '%v' instead", r.GetName())
	}

	log.Printf("Api Name: %v", r.GetName())
}
