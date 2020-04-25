package main

import (
    "log"
    "time"
    "fmt"
    "net"
    "bufio"
    "strconv"
    "encoding/binary"
    "strings"
    // "google.golang.org/protobuf"
    "github.com/golang/protobuf/proto"
    "./connectionPool"
    "./proto_pb"
)


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

func backManager(pool *connectionPool.ConnectionPool, back_conn net.Conn){
    reader := bufio.NewReader(back_conn)
    repMsg := &proto_pb.ReplicationMessage{}

    for true {
        if err := proto.Unmarshal(reader, repMsg); err != nil {
            log.Fatalln("Failed to parse address book:", err)
        }
        for index, conn_uuid := range message.CUuids(){
            c := connectionPool.Get(conn_uuid)
            if c.IsClosed(){
                c.Close()
                connectionPool.Remove(conn_uuid)
                log.Print("Dropping connection:",conn_uuid)
                continue
            }
            writer := bufio.NewWriter(targetConnection)
            writer.WriteString("Some message\n")
            writer.Flush()
        }
    }
    // pool.Range(func(cobj *connectionPool.ConnectionObj, connId uint64) {
    //     if cobj == nil || cobj.IsClosed(){
    //         cobj.Close()
    //         log.Print("Dropping connection:",connId)
    //     }else{
    //         cobj.WriteString("Some message\n")
    //         // END Testing if net.Conn is open or close
    //     }
    // })
    // for true{
    //     time.Sleep(2 * time.Second)
    //     }
    // back_con.Close()
    // // remove connection from bool
    // // connectionPool.Remove(connectionId)

}

func rutineBack(connectionPool *connectionPool.ConnectionPool){
    addr := "127.0.0.1:8090"
    log.Print("Starting ws back: ",addr)
    socket1, _ := net.Listen("tcp", addr)
    for true{
        connection, _ := socket1.Accept()
        log.Print("New backendconnection: ", connection.RemoteAddr().String())
        go backManager(connectionPool, connection)
    }
}


func rutineFront(connectionPool *connectionPool.ConnectionPool){
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
        i := connId(connection)
        writer := bufio.NewWriter(connection)
        writer.WriteString("Ack\n")
        writer.Flush()
        // add connection to pool
        connectionPool.Add(uint64(i), connection)
        log.Print("New connection id:",i)
    }
}

func main(){
    fmt.Println("Starting wsholder, will listen 8080 and 8090 for front and back")
    fmt.Println("---------------------------------------------------------------")
    connectionPool := connectionPool.NewConnectionPool()
    go rutineFront(connectionPool)
    go rutineBack(connectionPool)

    // count of connections in pool

    for true {
        time.Sleep(2 * time.Second)
        size := connectionPool.Size()
        log.Print("Size: %d",size)
    }

}

