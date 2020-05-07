// Package main implements a client for Greeter service.
package main

import (
	"context"
	"flag"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/juancki/wsholder/pb"
	"google.golang.org/grpc"
)


// TODO Add file for input
func main() {
        address := "localhost:"
        port := flag.String("port", "50066", "port to connect")
        messageContent := flag.String("msg", "This auto content", "message to send")
        connUuids := flag.String("uuids", "1,2","uuids with commas")
        count_max := flag.Int("max", 1,"Number of msgs to send.")
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
		"Content-Type": "text",
	}
	msg := []byte(*messageContent)

	rep := new(pb.ReplicationMsg)
        rep.Meta = new(pb.Metadata)
	rep.CUuids=ids
	rep.Meta.MsgMime = mime
	rep.Msg = msg
        for count:=0; count<*count_max; count++{
	    err = r.Send(rep)
	    if err != nil {
	        log.Print("could not greet: ", err, " ",count)
	    }
            log.Print("Message ",count," sent.")
            time.Sleep(time.Second*1)
        }
}
