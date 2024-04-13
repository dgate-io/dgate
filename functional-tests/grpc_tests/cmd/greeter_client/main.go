package main

import (
	"context"
	"crypto/tls"
	"flag"
	"log"
	"time"

	pb "grpc_tests/helloworld"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultName = "world"
)

var (
	addr      = flag.String("addr", "localhost:50051", "the address to connect to")
	name      = flag.String("name", defaultName, "Name to greet")
	_insecure = flag.Bool("insecure", false, "Use insecure connection")
)

func main() {
	flag.Parse()
	// Set up a connection to the server.
	dailOpts := []grpc.DialOption{}
	if *_insecure {
		dailOpts = append(dailOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		dailOpts = append(dailOpts, grpc.WithTransportCredentials(
			credentials.NewTLS(&tls.Config{
				InsecureSkipVerify: true,
			}),
		))
	}
	conn, err := grpc.Dial(*addr, dailOpts...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()
	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: *name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.GetMessage())
}
