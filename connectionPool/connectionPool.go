package connectionPool

import (
	"bufio"
	"errors"
	"log"
	"net"
	"sync"
	"time"
	"encoding/base64"
	"encoding/binary"
)

type (Uuid = uint64)
// Converstion types utils
// Returns string from UUID currently uint64
func Uuid2base64(uuid Uuid) string{
    bts := make([]byte,8)
    binary.PutUvarint(bts,uuid)
    return base64.StdEncoding.EncodeToString(bts)
}
// Returns uuid based on base64 encoding
func Base64_2_uuid(str string) (Uuid,error){
    bts,err := base64.StdEncoding.DecodeString(str)
    if err != nil{
        return 0, err
    }
    result, l :=  binary.Uvarint(bts)

    if l == 0{
        return 0, errors.New("Unable to cast fronm base64 to uuid in base64_2_uuid")
    }
    return Uuid(result),nil
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
}

// Closes socekt.
func (cobj *ConnectionObj) Close(){
    cobj.mutex.Lock()
    cobj.conn.Close()
    cobj.mutex.Unlock()
}

// Writes bytes.
func (cobj *ConnectionObj) Write(b []byte){
    cobj.mutex.Lock()
    writer := bufio.NewWriter(cobj.conn)
    _,err  :=writer.Write(b)
    if err != nil {
        log.Fatal(err)
    }
    writer.Flush()
    cobj.mutex.Unlock()
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
	return len(pool.list)
}

// Range iterates over pool
func (pool *ConnectionPool) Range(callback func(*ConnectionObj, Uuid)) {
	pool.mutex.RLock()
	for conn_uuid, connection := range pool.list {
		callback(connection, conn_uuid)
	}
	pool.mutex.RUnlock()
}


