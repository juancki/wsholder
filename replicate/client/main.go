// Package main implements a client for Greeter service.
package main

import (
	"context"
        "strconv"
        "strings"
        "flag"
	"log"
	"time"

	"google.golang.org/grpc"
	"../../pb"
)


func main() {
        address := "localhost:"
        port := flag.String("port", "50066", "port to connect")
        connUuids := flag.String("uuids", "1,2","uuids with commas")
        flag.Parse()
	// Set up a connection to the server.
        log.Print("Accessing: ",address,":",*port)
	conn, err := grpc.Dial(address+*port, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
        log.Print("Connected: ",address,":",*port)
	c := pb.NewWsBackClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	defer conn.Close()
	r, err := c.Replicate(ctx)
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Print("Sending...")
        uuids := strings.Split(*connUuids,",")
	ids := []uint64{}
        for _, uuid := range uuids{
            eq,err := strconv.Atoi(uuid)
            if err == nil{
                ids = append(ids,uint64(eq))
            }else{
                log.Fatal("connection Uuid: unable to cast ",eq," to intger") 
            }
        }
        log.Print("Uuids: ", ids)

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
        for count:=0; count<10; count++{
	    err = r.Send(rep)
	    if err != nil {
	        log.Print("could not greet: ", err, " ",count)
	    }
            log.Print("Message ",count," sent.")
            time.Sleep(time.Second*1)
        }
}
