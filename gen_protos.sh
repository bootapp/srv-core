#!/usr/bin/env bash
mkdir -p proto/core
protoc -I$GOPATH/src/github.com/bootapp/protos/core \
       --go_out=plugins=grpc:./proto/core \
        $GOPATH/src/github.com/bootapp/protos/core/core_common.proto \
        $GOPATH/src/github.com/bootapp/protos/core/dal_user.proto

#------------------- grpc
protoc -I$GOPATH/src/github.com/bootapp/protos/core \
    -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
    --go_out=plugins=grpc:./proto/core \
    $GOPATH/src/github.com/bootapp/protos/core/srv_user.proto \
    $GOPATH/src/github.com/bootapp/protos/core/srv_security.proto
#------------------- restful gateway
protoc -I$GOPATH/src/github.com/bootapp/protos/core \
    -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
    --grpc-gateway_out=logtostderr=true:./proto/core \
    $GOPATH/src/github.com/bootapp/protos/core/srv_user.proto \
    $GOPATH/src/github.com/bootapp/protos/core/srv_security.proto
#------------------- swagger
protoc -I$GOPATH/src/github.com/bootapp/protos/core \
    -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
    --swagger_out=logtostderr=true:./proto/core \
    $GOPATH/src/github.com/bootapp/protos/core/srv_user.proto \
    $GOPATH/src/github.com/bootapp/protos/core/srv_security.proto
