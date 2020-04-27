// Package main implements a client for Greeter service.
package main

import (
    "context"
    "log"
    "flag"
    "time"

    "google.golang.org/grpc"
    "../../pb"
)


func main() {
	// Set up a connection to the server.
        port := flag.String("port", "50051", "port to connect")
        flag.Parse()
        address := "localhost:"+*port
        log.Print("Accessing: ",address)
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
        log.Print("Connected: ",address)
	defer conn.Close()
	c := pb.NewServiceClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.LotsOfReplies(ctx, &pb.Empty{})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
        for true {
	    recv,err := r.Recv()
	    if err != nil {
	        log.Fatalf("could not greet: %v", err)
	    }
	    log.Printf("",recv)
        }
}
