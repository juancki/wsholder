
// Package main implements a client for Greeter service.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
        "bufio"
	"flag"
	"fmt"
	"net"
	"net/http"

	"github.com/juancki/wsholder/pb"
	"google.golang.org/protobuf/proto"
)


type Person struct {
    Name string
    Pass string
    Loc string
}


type TokenResponse struct {
    Token string

}

func getJson(url string, postData []byte, target interface{}) error {
    // var myClient = &http.Client{Timeout: 10 * time.Second}
    r, err := http.Post("http://"+url, "application/json", bytes.NewBuffer(postData))
    if err != nil {
        return err
    }
    return json.NewDecoder(r.Body).Decode(target)
}


func authenticate(name string, pass string, location string, url string) (string,error){
    p := new(Person)
    p.Name = name
    p.Pass = pass
    p.Loc = location

    bts, err := json.Marshal(&p)
    if err != nil{ return "", err}

    fmt.Println("Accessing: ",url, " with: ",string(bts))
    if err != nil{ return "", err}

    token := new(TokenResponse)
    err = getJson(url, bts, token)
    if err != nil{ return "", err}
    return token.Token,nil
}

func getNextMessageLength(c net.Conn) (uint64,error){
    bts := make([]byte,binary.MaxVarintLen64)
    _, err := c.Read(bts)
    if err != nil{
        return 0, err
    }
    return binary.ReadUvarint(bytes.NewReader(bts))
}

func main() {
    // Set up a connection to the server.
    wsholder := flag.String("wsholder", "127.0.0.1:8080", "address to connect")
    authloc := flag.String("authloc", "localhost:8000", "address to connect")
    name := flag.String("name", "John", "name to auth")
    pass := flag.String("pass", "Kevin", "password")
    usetoken := flag.String("token", "", "Avoid auth and use token")
    location := flag.String("location", "13.13:20.20", "port to connect")
    flag.Parse()
    fmt.Println(*usetoken)
    fmt.Println(len(*usetoken))
    token := ""
    var err error
    if len(*usetoken)==0{
        token, err = authenticate(*name,*pass,*location,*authloc+"/auth")
        if err != nil{
            fmt.Println(err)
        }
    }else{
        fmt.Println("-token flag passed, avoiding authentication step")
        token = *usetoken
    }
    fmt.Println("Token: ", token)

    conn, err := net.Dial("tcp", *wsholder)
    if err != nil{ fmt.Print(err); return}
    sendbytes := make([]byte,len(token)+2)
    sendbytes[0] = ':'
    sendbytes[len(token)+2-1] = '\n'
    copy(sendbytes[1:],token[:])
    writer := bufio.NewWriter(conn)
    writer.WriteString(string(sendbytes))
    writer.Flush()

    for true {
        rcv := &pb.UniMsg{}
        rcv.Meta = &pb.Metadata{}
        rcv.Meta.MsgMime = make(map[string]string)
        length, err := getNextMessageLength(conn)
        if err != nil{
            fmt.Println(err)
            return
        }
        bts := make([]byte,length)
        n, err := conn.Read(bts)
        if err != nil || uint64(n) != length{
            fmt.Println(err)
            return
        }
        err = proto.Unmarshal(bts,rcv)
        if err != nil{
            fmt.Println(err)
            return
        }
        fmt.Print(len(rcv.GetMsg()),"B ")
        if rcv == nil || rcv.Meta == nil{
            continue
        }
        fmt.Print(rcv.GetMeta().Resource, " ", rcv.Meta.Poster)
        fmt.Print(rcv.GetMeta().GetMsgMime())
        if tpe, ok := rcv.GetMeta().GetMsgMime()["Content-Type"]; ok && tpe != "bytes"{
            fmt.Print(" `",string(rcv.GetMsg()),"`")
        }
        fmt.Println()
    }
}
