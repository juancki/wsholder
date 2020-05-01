
package main

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/golang/protobuf/proto"
	cPool "github.com/juancki/wsholder/connectionPool"
	"github.com/juancki/wsholder/pb"
	"google.golang.org/grpc"
)
// TODO Add connection to REDIS and save when a connection gets into front.

// Config

// logs
// TODO add connection debug option/service that indicates for 
//      a Replications Msg which connection ids where forwarded and which not.
// TODO clean logs

// TODO create tests wsholder + Redis
// TODO create docker composer to set everything up at same time

// TODO find way the function cPool.Base64_2_Uuid was not working and now does (remove log line)

// FEATURES 
// TODO TODO Define incomming messages structure, sending bytes through socket not the greatest idea.
// TODO send messages to Postgre to save them for posterity.
// TODO know when a connection is closed reliably. (*connectionPool) IsClosed does not work
// TODO delete entries from rgeoclient when front connections drop.

// TODO create library for Redis/Postgre connections ¿?

var pool cPool.ConnectionPool
const WORLD = "world"

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
                log.Print("connection dropped")
                break
            }
            log.Print(rep)
            // TODO send rep to channel
            serveReplication(rep)
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
        // if c.IsClosed(){
        //     c.Close()
        //     pool.Remove(conn_uuid)
        //     log.Print("Dropping connection:",conn_uuid)
        //     continue
        // }
        msg := &pb.UniMsg{}
        msg.Msg = rep.Msg
        msg.MsgMime = rep.MsgMime
        bts, err := proto.Marshal(msg)
        if err != nil {
            log.Print("Unable to marshal message")
            continue
        }
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

func coorFromBase64(str string) (float64,float64){
    // log.Println("entering coorFromBase64")
    splits := strings.SplitN(str,":",3)
    b64 := splits[0]
    // log.Println("input string: ",str)
    // log.Println("Section: ",b64)
    coorbts , _ := base64.StdEncoding.DecodeString(b64)
    coorstring := string(coorbts)
    // log.Println("Decoded: ",coorstring)

    ind := strings.Split(coorstring,":")
    if len(ind) < 2{
        return -1,-1
    }
    long,_ := strconv.ParseFloat(ind[0],64)
    lat,_ := strconv.ParseFloat(ind[1],64)
    return long,lat
}



func addToPoolAndUpdateRedis(cn net.Conn) error {
    ID := connId(cn)
    writer := bufio.NewWriter(cn)
    writer.WriteString("Ack\n")
    writer.Flush()
    // ADD to POOL 
    pool.Add(cPool.Uuid(ID), cn)
    log.Println("New connection id:",ID)
    bts := make([]byte,100) // Decide token size
    read, err := cn.Read(bts)
    if err != nil{
        return err
    }
    token := string(bts[1:read-1])
    // UPDATE REDIS
    log.Println(token)
    log.Println("$2a$04$1B2TiNo0fhO1j0pXa/FPYeGOxOm9CRfkVN3pIb5Vaknwfl936/CpC")
    appendError := rclient.redis.Append(token,cPool.Uuid2base64(ID)).Err()
    rclient.Get("$2a$04$1B2TiNo0fhO1j0pXa/FPYeGOxOm9CRfkVN3pIb5Vaknwfl936/CpC")
    value, err := rclient.Get(token)
    if appendError != nil {
        log.Println(appendError)
        return appendError
    }
    if err != nil {
        log.Println(err)
        return err
    }
    // value from the key,value store
    log.Println("The value is: ", value)
    long,lat := coorFromBase64(value)
    geoloc := &redis.GeoLocation{}
    geoloc.Latitude = lat
    geoloc.Longitude= long
    geoloc.Name = cPool.Uuid2base64(ID)
    rgeoclient.Lock()
    geoAdd := rgeoclient.redis.GeoAdd(WORLD,geoloc)
    rgeoclient.Unlock()
    if geoAdd.Err() != nil{
        log.Println(geoAdd.Err())
        return geoAdd.Err()
    }
    log.Println("Geoadd succesful")
    return nil
}

func manageFrontError(c net.Conn){
    c.SetWriteDeadline(time.Now().Add(time.Millisecond*2))
    c.Write([]byte("Error in connection"))
    c.Close()
    pool.Remove(connId(c))
}

func rutineFront(addr string){
    // addr := "127.0.0.1:8080"
    log.Print("Starting TCP ws front: ",addr)
    socket, err := net.Listen("tcp", addr)
    if err != nil {
        log.Print("Could not start tcp.")
    }
    log.Println("TCP ws back server started")

    // accept connection
    for true {
        connection, _ := socket.Accept()
        err = addToPoolAndUpdateRedis(connection)
        if err != nil{
            manageFrontError(connection)
        }
    }
}

func rutineBack(addr string){
    // addr := "127.0.0.1:8090"
    log.Print("Starting gRPC ws back: ",addr)
    lis, err := net.Listen("tcp",addr)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    log.Println("gRPC ws back server started")
    s := grpc.NewServer()
    pb.RegisterWsBackServer(s, &server{})
    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}

type Redis struct{
    redis *redis.Client
    sync.Mutex
}


// type Postgre struct{
//     pg sql.DB
//     sync.Mutex
// }


func NewRedis(connURL string) *Redis{
    // redis://password@netloc:port/dbnum
    // redis does not have a username
    key := "postgresql://"
    if strings.HasPrefix(connURL,key){
        connURL = connURL[len(key):]
    }
    dbnum,_ := strconv.Atoi(strings.Split(connURL,"/")[1])
    connURL = strings.Split(connURL,"/")[0]

    pass := strings.Split(connURL,"@")[0]
    log.Print("`",pass,"`\t",len(pass))
    addr := strings.Split(connURL,"@")[1]
    client := redis.NewClient(&redis.Options{
            Addr:     addr,
            Password: "", // no password set
            DB:       dbnum,  // use default DB
    })

    pong, err := client.Ping().Result()
    if err != nil{
        // Exponential backoff
        log.Print("Un able to connect to REDIS `", dbnum,"` at: `", addr, "` with password: `",pass,"`.")
        log.Print(err)
        return nil
    }
    log.Print("I said PING, Redis said: ",pong)
    return &Redis{redis: client}

}

func (*Redis) Get(key string) (string,error){
    rclient.Lock()
    val, err := rclient.redis.Get(key).Result()
    rclient.Unlock()
    if err != nil {
        log.Println("Not succesful: ", key, " ", err)
        return "",err
    }
    log.Println("Succesful: ", key, " ", val)
    return val, nil
}

var rclient *Redis
var rgeoclient *Redis

func main() {
    // flag for configuration via command line
    frontport := flag.String("front-port", "localhost:8080", "port to connect (clients)")
    backport := flag.String("back-port", "localhost:8090", "port to connect (server)")
    redis:= flag.String("redis", "@localhost:6379/0", "format password@IPAddr:port")
    redisGeo := flag.String("redisGeo", "@localhost:6379/1", "format password@IPAddr:port")
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
    rclient = nil
    for rclient == nil{
        rclient = NewRedis(*redis)
        time.Sleep(time.Second*1)
    }
    rgeoclient = nil
    for rgeoclient == nil{
        rgeoclient  = NewRedis(*redisGeo)
        time.Sleep(time.Second*1)
    }
    // Starting server
    fmt.Println("Starting wsholder...") // ,*frontport,"for front, ",*backport," for back")
    fmt.Println("--------------------------------------------------------------- ")
    pool = *cPool.NewConnectionPool()
    go rutineFront(fport)
    go rutineBack(bport)
    // End rutineBack()

    // count of connections in pool

    for true {
        // TODO: Logging stay alive, remove before prod
        size := pool.Size()
        log.Print("Size: ",size)
        time.Sleep(10 * time.Second)
    }
}
