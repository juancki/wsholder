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
