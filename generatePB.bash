## Generate .pb.go files 


AUX=$PATH

PATH=$PATH:$GOPATH/bin
protoc -I ./proto_src ./proto_src/*.proto --go_out=plugins=grpc:pb

PATH=$AUX