// Package main implements a client for Greeter service.
package main

import (
	"log"
        "net"

	"google.golang.org/grpc"
	"../../pb"
)

const (
	address     = "localhost:50066"
	defaultName = "world"
)


const (
	port = 50066
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedWsBackServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) Replicate(reps pb.WsBack_ReplicateServer) (error) {
	log.Printf("Received it!")
        loop := 0
        for true {
	    // rep := new(pb.ReplicationMsg)
            rep, err := reps.Recv()
            if err != nil{
                log.Println(err," ",loop)
                break
            }
            log.Println(rep)
            loop+=1
        }

	return nil
}

func main() {
	log.Print("Starting Server on port: ",port)
	lis, err := net.Listen("tcp",address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterWsBackServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}


