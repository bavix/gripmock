package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/bavix/gripmock/protogen/example/ms"
)

func env(key, fallback string) string {
	if value := os.Getenv(key); len(value) > 0 {
		return value
	}

	return fallback
}

//nolint:gomnd
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	grpcPort := env("GRPC_PORT", "4770")

	conn, err := grpc.DialContext(ctx, net.JoinHostPort("localhost", grpcPort), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewMicroServiceClient(conn)

	r1, err := c.SayHello(context.Background(), &pb.Request{V1: [][]byte{
		u2bytes("ab0ed195-6ac5-4006-a98b-6978c6ed1c6b"), // 3
		u2bytes("99aebcf2-b56d-4923-9266-ab72bf5b9d0b"), // 1
		u2bytes("5659bec5-dda5-4e87-bef4-e9e37c60eb1c"), // 2
		u2bytes("77465064-a0ce-48a3-b7e4-d50f88e55093"), // 0
	}})
	if err != nil {
		log.Fatalf("error from grpc: %v", err)
	}
	log.Printf("Result: %d", r1.Code)

	r2, err := c.SayHello(context.Background(), &pb.Request{V2: []string{
		"e3484119-24e1-42d9-b4c2-7d6004ee86d9", // 1
		"c30f45d2-f8a4-4a94-a994-4cc349bca457", // 3
		"f1e9ed24-93ba-4e4f-ab9f-3942196d5c03", // 0
		"cc991218-a920-40c8-9f42-3b329c8723f2", // 2
	}})
	if err != nil {
		log.Fatalf("error from grpc: %v", err)
	}
	log.Printf("Result: %d", r2.Code)

	r3, err := c.SayHello(context.Background(), &pb.Request{V2: []string{
		"e3484119-24e1-42d9-b4c2-7d6004ee86d9", // 1
		"c30f45d2-f8a4-4a94-a994-4cc349bca457", // 3
		"f1e9ed24-93ba-4e4f-ab9f-3942196d5c03", // 0
		"cc991218-a920-40c8-9f42-3b329c8723f2", // 2
	}, V3: int64ptr(77)})
	if err != nil {
		log.Fatalf("error from grpc: %v", err)
	}
	log.Printf("Result: %d", r3.Code)
	log.Printf("v3: %d", *r3.V3)
}

func u2bytes(v string) []byte {
	u := uuid.MustParse(v)

	return u[:]
}

func int64ptr(v int64) *int64 {
	return &v
}
