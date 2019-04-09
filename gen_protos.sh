#!/usr/bin/env bash
mkdir -p proto/clients/dal-core
mkdir -p proto/server
protoc -I$GOPATH/src/github.com/bootapp/protos/dal/core \
       --go_out=plugins=grpc:./proto/clients/dal-core \
        $GOPATH/src/github.com/bootapp/protos/dal/core/User.proto \
        $GOPATH/src/github.com/bootapp/protos/dal/core/Organization.proto

#------------------- grpc
protoc -I$GOPATH/src/github.com/bootapp/protos/srv/core \
    -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
    --go_out=plugins=grpc:./proto/server \
    $GOPATH/src/github.com/bootapp/protos/srv/core/User.proto
#------------------- restful gateway
protoc -I$GOPATH/src/github.com/bootapp/protos/srv/core \
  -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
  --grpc-gateway_out=logtostderr=true:./proto/server \
  $GOPATH/src/github.com/bootapp/protos/srv/core/User.proto
#------------------- swagger
protoc -I$GOPATH/src/github.com/bootapp/protos/srv/core \
  -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
  --swagger_out=logtostderr=true:./proto/server \
  $GOPATH/src/github.com/bootapp/protos/srv/core/User.proto
