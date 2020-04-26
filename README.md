# wsholder
```
# in your .bashrc or similar file
export GOPATH=$HOME/go
```
export GOPATH=$HOME/go
# Be sure you have recent golang version installed

go install google.golang.org/protobuf/cmd/protoc-gen-go


# Add to bashrc
protoc -I=. --go_out=. replicationMessage.proto



```
go get golang.org/x/net/http2
mkdir -p $GOPATH/src/google.golang.org/grpc
cp $GOPATH/src/github.com/grpc/grpc-go/* -R $GOPATH/src/google.golang.org/grpc
```
google.golang.org/genproto/googleapis/rpc/status
golang.org/x/sys/unix
