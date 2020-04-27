# wsholder
```
# in your .bashrc or similar file
export GOPATH=$HOME/go
```
export GOPATH=$HOME/go
# Be sure you have recent golang version installed

go install google.golang.org/protobuf/cmd/protoc-gen-go


# Generate pb/proto files with 
```
./generatePB.bash
```


# Other libraries might be necessary to download
```
go get golang.org/x/net/http2
go get github.com/grpc/grpc-go
# For some reason I'm not able to download google.golang.org/grpc
mkdir -p $GOPATH/src/google.golang.org/grpc 
cp $GOPATH/src/github.com/grpc/grpc-go/* -R $GOPATH/src/google.golang.org/grpc
```
google.golang.org/genproto/googleapis/rpc/status
golang.org/x/sys/unix
TESTING
go run main.go
go run replicate/client/main.go -port=8090 -uuids=139637976836088,139637976836098


======

DESIGN
======
This service is a man in the middle to replicate messages from the 
server to multiple clients.

The client (front) connect via TCP socket, this connection in saved 
in a key,value map: being the key the connectionId (uuid).

From the serverside (back) a gRPC server waits for messages from the
server with (uudis,message). This message is then transferred to all
the clients lited in the uuids.
