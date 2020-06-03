
package main

import (
	"encoding/base64"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/golang/protobuf/proto"
	mydb "github.com/juancki/authloc/dbutils"
	cPool "github.com/juancki/wsholder/connectionPool"
	"github.com/juancki/wsholder/pb"
	"google.golang.org/grpc"
)
// TODO Add connection to REDIS and save when a connection gets into front.

// Config
// Add machine IP to the cuuid ids generated.

// logs
// TODO add connection debug option/service that indicates for 
//      a Replications Msg which connection ids where forwarded and which not.

// TODO clean logs

// TODO create tests wsholder + Redis
// TODO create docker composer to set everything up at same time

// TODO Send emtpy message to long lived connections to know if they are still on.
//  how Add connections to Datastructure (hashmap+double linkedlist) 
//              + To access a connection is as easy as now. with the key (CUUID)
//              + On write to connection -> llist remove -> llist append. Connections recently written upon will be located in the begining 
//              + Have a process checking the older connections until gets to a high ratio of working connections
//              + On error close connection and free the memory.


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



func replicateErr(errchan chan cPool.Uuid){
    for uuid := range errchan{
        log.Println("Error channel: ",uuid)
        // Delete from cPool
        pool.Remove(uuid)
        // Delete from Redis geo
        err := rgeoclient.GeoRemoveCuuid(uuid)
        if err!=nil{
            log.Print("Error trying to geoRemove: ", err)
        }
    }
}

func (s *server) Replicate(reps pb.WsBack_ReplicateServer) (error) {
        errchan := make(chan cPool.Uuid,10)
        go replicateErr(errchan)
	log.Printf("Received it!")
        loop := 0
        for true {
	    // rep := new(pb.ReplicationMsg)
            rep, err := reps.Recv()
            if err != nil{
                // connexion droped
                log.Print("#Messages received: ", loop, ". Stop reason: ", err)
                return nil
            }
            // TODO send rep to channel
            serveReplication(rep,errchan)
            loop+=1
        }
	return nil
}


func serveReplication(rep *pb.ReplicationMsg, errchan chan cPool.Uuid) {
    // TODO read rep from channel and be a for true gorutine
    for _, conn_uuid := range rep.CUuids{
        c,err := pool.GetHandler(conn_uuid)
        if err != nil{
            // TODO: Connection not found, delete from REDIS ?
            log.Print("Connection ",conn_uuid," was not found")
            continue
        }
        // TODO if c.IsClosed(){
        // }
        msg := &pb.UniMsg{}
        msg.Meta = rep.Meta
        msg.Msg = rep.Msg
        bts, err := proto.Marshal(msg)
        if err != nil {
            log.Print("Unable to marshal message")
            continue
        }
        bts2 := make([]byte,len(bts)+binary.MaxVarintLen64)
        binary.PutUvarint(bts2,uint64(len(bts)))
        copy(bts2[binary.MaxVarintLen64:],bts)
        go c.Write(bts2, errchan)
    }
}


func base64_2_string(b64 string) string {
    coorbts , err := base64.StdEncoding.DecodeString(b64)
    if err != nil {
        return ""
    }
    return string(coorbts)
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
    ID := cPool.ConnId(cn)
    cn.Write(make([]byte,binary.MaxVarintLen64)) // Send empty message
    bts := make([]byte,100) // TODO Decide token size
    read, err := cn.Read(bts)
    if err != nil{
        log.Println("Connection")
        return err
    }
    // Append if exists
    token := string(bts[1:read-1])
    // UPDATE REDIS
    row, err := rclient.AppendCuuidIfExists(token,ID)
    if err != nil {
        log.Println("Redis")
        return err
    }
    // ADD to POOL 
    pool.Add(cPool.Uuid(ID), cn)
    // value from the key,value store
    base64Coor := strings.Split(row, ":")[0]
    base64User := strings.Split(row, ":")[1]
    long,lat := coorFromBase64(base64Coor)
    geoloc := &redis.GeoLocation{}
    geoloc.Latitude = lat
    geoloc.Longitude= long
    geoloc.Name = cPool.Uuid2base64(ID)
    rgeoclient.Lock()
    geoAdd := rgeoclient.Redis.GeoAdd(WORLD,geoloc)
    rgeoclient.Unlock()
    rclient.SetUseridCuuid(base64_2_string(base64User), ID)
    if geoAdd.Err() != nil{
        return geoAdd.Err()
    }
    log.Println("Geoadd token: ", token, " UUID: ", ID, " / ", cPool.Uuid2base64(ID))
    return nil
}

func manageFrontError(c net.Conn){
    c.SetWriteDeadline(time.Now().Add(time.Millisecond*2))
    c.Write([]byte("Error in connection"))
    c.Close()
    pool.Remove(cPool.ConnId(c))
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
        connection, err := socket.Accept()
        if err != nil{
            fmt.Println(err)
            continue
        }
        err = addToPoolAndUpdateRedis(connection)
        if err != nil{
            fmt.Println(err)
            manageFrontError(connection)
        }
    }
}

func rutineBack(addr string){
    // addr := "127.0.0.1:8090"
    log.Print("Starting gRPC ws back: ",addr)
    lis, err := net.Listen("tcp",addr)
    if err != nil {
        panic(err)
    }
    log.Println("gRPC ws back server started")
    // TODO add Keep Alive parameters
    s := grpc.NewServer()
    pb.RegisterWsBackServer(s, &server{})
    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}


var rclient *mydb.Redis
var rgeoclient *mydb.Redis

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
    rclient = mydb.NewRedis(*redis)
    for rclient == nil{
        rclient = mydb.NewRedis(*redis)
        time.Sleep(time.Second*1)
    }
    rgeoclient = nil
    rgeoclient  = mydb.NewRedis(*redisGeo)
    for rgeoclient == nil{
        rgeoclient  = mydb.NewRedis(*redisGeo)
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
        time.Sleep(100 * time.Second)
    }
}
