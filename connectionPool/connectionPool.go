package connectionPool

import (
	"bufio"
	"errors"
	"log"
	"net"
	"sync"
	"time"
)

// ConnectionPool is a thread safe list of net.Conn instances
type ConnectionPool struct {
    mutex sync.RWMutex
    list  map[uint64]*ConnectionObj // Custom struct to manage net.Conn
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
		list: make(map[uint64]*ConnectionObj),
	}
	return pool
}

// Add collection to pool
func (pool *ConnectionPool) Add(conn_uuid uint64,connection net.Conn) {
    conn := new(ConnectionObj)
    conn.time = time.Now()
    conn.conn = connection
	pool.mutex.Lock()
	pool.list[conn_uuid] = conn
	pool.mutex.Unlock()
}

// Get connection by id, obj returned not thread safe, use GetHandler
func (pool *ConnectionPool) Get(conn_uuid uint64) net.Conn {
	pool.mutex.RLock()
	connectionObj,ok := pool.list[conn_uuid]
	pool.mutex.RUnlock()
    if !ok{
        return nil
    }
	return (*connectionObj).conn
}

// Get connection by id, obj returned thread safe
func (pool *ConnectionPool) GetHandler(conn_uuid uint64) (*ConnectionObj,error){
    pool.mutex.RLock()
    connectionObj,ok := pool.list[conn_uuid]
    pool.mutex.RUnlock()
    if !ok{
        return nil,errors.New("Connection with uuid not found")
    }
    return connectionObj,nil
}
// Remove connection from pool
func (pool *ConnectionPool) Remove(conn_uuid uint64) {
	pool.mutex.Lock()
	delete(pool.list, conn_uuid)
	pool.mutex.Unlock()
}

// Size of connections pool
func (pool *ConnectionPool) Size() int {
	return len(pool.list)
}

// Range iterates over pool
func (pool *ConnectionPool) Range(callback func(*ConnectionObj, uint64)) {
	pool.mutex.RLock()
	for conn_uuid, connection := range pool.list {
		callback(connection, conn_uuid)
	}
	pool.mutex.RUnlock()
}


