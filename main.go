
package main

import (
    "bufio"
    "encoding/binary"
    "flag"
    "fmt"
    "log"
    "net"
    "strconv"
    "strings"
    "time"
    cPool "github.com/juancki/wsholder/connectionPool"
    "github.com/juancki/wsholder/pb"
    "github.com/golang/protobuf/proto"
    "google.golang.org/grpc"
)
// Add connection to REDIS and save when a connection gets into front.

// Config

// logs
// TODO add connection debug option/service that indicates for 
//      a Replications Msg which connection ids where forwarded and which not.


var pool cPool.ConnectionPool

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


func connId(conn net.Conn) cPool.Uuid{
    addr := conn.RemoteAddr().String()
    ind := strings.Index(addr,":")
    if ind <= 5{
        // TODO: Add support for IPv6
        log.Print("IP ParsingERRO IPv6 addr not supported, if used localhost -> use 127.0.0.1")
        return 0
    }else{
        // IPv4
        port, _ := strconv.ParseUint(addr[ind+1:],10,16)
        ip := net.ParseIP(addr[0:ind])
        if ip == nil {
            log.Print("IP ParsingERROR:", addr[0:ind])
        }
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


func rutineFront(addr string){
    // addr := "127.0.0.1:8080"
    log.Print("Starting ws front: ",addr)
    socket, err := net.Listen("tcp", addr)
    if err != nil {
        log.Print("Could not start tcp.")
    }

    // accept connection
    for true {
        connection, _ := socket.Accept()
        ID := connId(connection)
        writer := bufio.NewWriter(connection)
        writer.WriteString("Ack\n")
        writer.Flush()
        // add connection to pool
        pool.Add(cPool.Uuid(ID), connection)
        // TODO add connection to REDIS
        log.Print("New connection id:",ID)
    }
}

func rutineBack(addr string){
    // addr := "127.0.0.1:8090"
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

func setupRedis(user string, password string, addr string){
    // TODO implement
    log.Print("Redis not configured")
}

func main() {
    // flag for configuration via command line
    frontport := flag.String("front-port", "localhost:8080", "port to connect (clients)")
    backport := flag.String("back-port", "localhost:8090", "port to connect (server)")
    redis:= flag.String("redis", ":@localhost:6379", "user:password@IPAddr:port")
    flag.Parse()
    fport := *frontport
    bport := *backport
    if strings.Index(*frontport,":") == -1{
        fport = "localhost:" + *frontport
    }
    if strings.Index(*backport,":") == -1{
        bport = ("localhost:" + *backport)
    }
    // Set up redis
    redisconf := strings.Split(*redis,"@")
    redisid := strings.Split(redisconf[0],":")
    redisAddr := redisconf[1]
    // Starting server
    fmt.Println("Starting wsholder...") // ,*frontport,"for front, ",*backport," for back")
    setupRedis(redisid[0],redisid[1],redisAddr)
    fmt.Println("--------------------------------------------------------------- ")
    pool = *cPool.NewConnectionPool()
    go rutineFront(fport)
    go rutineBack(bport)
    // End rutineBack()

    // count of connections in pool

    for true {
        // TODO: Logging stay alive, remove before prod
        time.Sleep(5 * time.Second)
        size := pool.Size()
        log.Print("Size: ",size)
    }
}
