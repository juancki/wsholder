package main

import (
    "log"
//    "fmt"
//    "strings"
    "net"
//    "bufio"
//    "strconv"
//    "encoding/binary"
//    "strings"

    "google.golang.org/grpc"
//    "../connectionPool"
    "../../pb"
)

const (
	port = ":50051"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedServiceServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) LotsOfReplies(in *pb.Empty, lotsreplies pb.Service_LotsOfRepliesServer) (error) {
	log.Printf("Received it!")
	ids := []uint64{1,2}
	var mime = map[string]string{
		"Content-type": "text",
		"Content-length": "32",
	}
	str := "My content of the message"
	msg := []byte(str)

	rep := new(pb.ReplicationMsg)
	rep.CUuids=ids
	rep.MsgMime = mime
	rep.Msg = msg
	lotsreplies.Send(rep)
// 	rep := &pb.ReplicationMsg{
// 		CUuids : ids,
// 		MsgMime : mime,
// 		Msg : msg,
// 		}
	// return rep, nil
	return nil
}

func main() {
	log.Print("Starting Server on port: ",port)
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterServiceServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}


