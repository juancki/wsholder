package connectionPool

import (
	"bufio"
	"errors"
	"log"
	"net"
	"sync"
	"time"
	"runtime"
	"strconv"
	"strings"
	"bytes"
	"encoding/base64"
	"encoding/binary"
)

type (Uuid = uint64)
// Converstion types utils
// Uuid <---> []byte
func Uuid2Bytes(uuid Uuid) []byte{
    bts := make([]byte, binary.MaxVarintLen64)
    binary.BigEndian.PutUint64(bts, uuid)
    return bts
}

func Bytes2Uuid(bts []byte) Uuid{
    r := binary.BigEndian.Uint64(bts)
    return r
}
// Returns string from UUID currently uint64
func Uuid2base64(uuid Uuid) string{
    bts := Uuid2Bytes(uuid)
    return base64.StdEncoding.EncodeToString(bts)
}


// Returns uuid based on base64 encoding
func Base64_2_uuid(str string) (Uuid,error){
    bts,err := base64.StdEncoding.DecodeString(str)
    if err != nil{
        log.Print("Base64_2_uuid input: `",str,"`")
        return 0, err
    }
    return Bytes2Uuid(bts), nil
}
// ConnectionPool is a thread safe list of net.Conn instances
type ConnectionPool struct {
    mutex sync.RWMutex
    list  map[Uuid]*ConnectionObj // Custom struct to manage net.Conn
}

type ConnectionObj struct {
    mutex sync.RWMutex
    time time.Time
    conn net.Conn
    uuid Uuid
}

// Closes socekt.
func (cobj *ConnectionObj) Close(){
    cobj.mutex.Lock()
    cobj.conn.Close()
    cobj.mutex.Unlock()
}

func getGID() uint64 {
    b := make([]byte, 64)
    b = b[:runtime.Stack(b, false)]
    b = bytes.TrimPrefix(b, []byte("goroutine "))
    b = b[:bytes.IndexByte(b, ' ')]
    n, _ := strconv.ParseUint(string(b), 10, 64)
    return n
}

func ConnId(conn net.Conn) Uuid{
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
        return Bytes2Uuid(dst)
    }
}


// Writes bytes.
func (cobj *ConnectionObj) Write(b []byte, errchan chan Uuid){
    cobj.mutex.Lock()
    nn, err := cobj.conn.Write(b)
    cobj.mutex.Unlock()
    if err != nil || nn != len(b) {
        log.Print("Bytes not sent uuid: ",cobj.uuid,err)
        errchan <- ConnId(cobj.conn) //TODO change ConnId for cobj.Uuid
        return
    }
    log.Print("Bytes sent ", cobj.uuid, " ",len(b))
}

// Writes String.
func (cobj *ConnectionObj) WriteString(s string){
    cobj.mutex.Lock()
    writer := bufio.NewWriter(cobj.conn)
    writer.WriteString(s)
    writer.Flush()
    cobj.mutex.Unlock()
}

// Closes socekt.
func (cobj *ConnectionObj) IsClosed() bool {
	cobj.mutex.Lock()
    // TODO find a better function.
    one := make([]byte, 1)
    _, err := cobj.conn.Read(one);
	cobj.mutex.Unlock()
    if err == nil{
        return false
    }
    return false

}
// NewConnectionPool is the factory method to create new connection pool
func NewConnectionPool() *ConnectionPool {
	pool := &ConnectionPool{
		list: make(map[Uuid]*ConnectionObj),
	}
	return pool
}

// Add collection to pool
func (pool *ConnectionPool) Add(conn_uuid Uuid,connection net.Conn) {
    conn := new(ConnectionObj)
    conn.time = time.Now()
    conn.conn = connection
    conn.uuid = conn_uuid
	pool.mutex.Lock()
	pool.list[conn_uuid] = conn
	pool.mutex.Unlock()
}

// Get connection by id, obj returned not thread safe, use GetHandler
func (pool *ConnectionPool) Get(conn_uuid Uuid) net.Conn {
	pool.mutex.RLock()
	connectionObj,ok := pool.list[conn_uuid]
	pool.mutex.RUnlock()
    if !ok{
        return nil
    }
	return (*connectionObj).conn
}

// Get connection by id, obj returned thread safe
func (pool *ConnectionPool) GetHandler(conn_uuid Uuid) (*ConnectionObj,error){
    pool.mutex.RLock()
    connectionObj,ok := pool.list[conn_uuid]
    pool.mutex.RUnlock()
    if !ok{
        return nil,errors.New("Connection with uuid not found")
    }
    return connectionObj,nil
}
// Remove connection from pool
func (pool *ConnectionPool) Remove(conn_uuid Uuid) {
	pool.mutex.Lock()
	delete(pool.list, conn_uuid)
	pool.mutex.Unlock()
}

// Size of connections pool
func (pool *ConnectionPool) Size() int {
    pool.mutex.Lock()
    l := len(pool.list)
    pool.mutex.Unlock()
    return l
}

// Range iterates over pool
func (pool *ConnectionPool) Range(callback func(*ConnectionObj, Uuid)) {
	pool.mutex.RLock()
	for conn_uuid, connection := range pool.list {
		callback(connection, conn_uuid)
	}
	pool.mutex.RUnlock()
}


