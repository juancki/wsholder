
package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"./connectionPool"
	"./pb"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
)
// Add connection to REDIS and save when a connection gets into front.

// Config
// TODO add config for front_server port and back_server port
// TODO add config set for redis storge directions

// logs
// TODO fix `rpc error: code = Canceled desc = context canceled` error
// TODO add connection debug option/service that indicates for 
//      a Replications Msg which connection ids where forwarded and which not.


var pool connectionPool.ConnectionPool

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedWsBackServer
}



func (s *server) Replicate(reps pb.WsBack_ReplicateServer) (error) {
	log.Printf("Received it!")
        loop := 0
        for true {
	    // rep := new(pb.ReplicationMsg)
            rep, err := reps.Recv()
            if err != nil{
                // connexion droped
                break
            }
            // TODO send rep to channel
            serveReplication(rep)
            log.Println(rep)
            loop+=1
        }
	return nil
}


func serveReplication(rep *pb.ReplicationMsg) {
    // TODO read rep from channel and be a for true gorutine
    for _, conn_uuid := range rep.CUuids{
        c,err := pool.GetHandler(conn_uuid)
        if err != nil{
            // TODO: Connection not found, delete from REDIS ?
            log.Print("Connection ",conn_uuid," was not found")
            continue
        }
        if c.IsClosed(){
            c.Close()
            pool.Remove(conn_uuid)
            log.Print("Dropping connection:",conn_uuid)
            continue
        }
        msg := &pb.UniMsg{}
        msg.Msg = rep.Msg
        msg.MsgMime = rep.MsgMime
        bts, err := proto.Marshal(msg)
        c.Write(bts)
    }
}


func connId(conn net.Conn) uint64{
    addr := conn.RemoteAddr().String()
    ind := strings.Index(addr,":")
    port, _ := strconv.ParseUint(addr[ind+1:],10,16)
    if ind <= 5{
        // IPv6
        // ip := IP.parseIPv6(addr) 
        return 6
    }else{
        // IPv4
        ip := net.ParseIP(addr[0:ind])
        if ip == nil {
            log.Print("IP ParsingERROR:", addr[0:ind])
        }
        log.Print("Len size:", len(ip))
        dst := make([]byte,8)
        // TODO: Include ws holder ip as identifier
        dst[0] = 0
        dst[1] = 0
        copy(dst[2:6],ip[12:])
        dst[6] = byte(port >> 8)
        dst[7] = byte(port &0x00FF)
        return binary.BigEndian.Uint64(dst)
    }
}


func rutineFront(){
    addr := "127.0.0.1:8080"
    log.Print("Starting ws front: ",addr)
    socket, err := net.Listen("tcp", addr)
    if err != nil {
        log.Print("Could not start tcp.")
    }

    // prepare connection pool

    // accept connection
    for true {
        connection, _ := socket.Accept()
        ID := connId(connection)
        writer := bufio.NewWriter(connection)
        writer.WriteString("Ack\n")
        writer.Flush()
        // add connection to pool
        pool.Add(uint64(ID), connection)
        // TODO add connection to REDIS
        log.Print("New connection id:",ID)
    }
}

func rutineBack(){
    addr := "127.0.0.1:8090"
    log.Print("Starting ws back: ",addr)
    lis, err := net.Listen("tcp",addr)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    s := grpc.NewServer()
    pb.RegisterWsBackServer(s, &server{})
    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}

func main() {
    fmt.Println("Starting wsholder, will listen 8080 and 8090 for front and back")
    fmt.Println("---------------------------------------------------------------")
    pool = *connectionPool.NewConnectionPool()
    go rutineFront()
    go rutineBack()
    // End rutineBack()

    // count of connections in pool

    for true {
        time.Sleep(5 * time.Second)
        size := pool.Size()
        log.Print("Size: ",size)
    }
}
