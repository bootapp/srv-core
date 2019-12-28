#!/usr/bin/env bash
mkdir -p proto
protoc -I$GOPATH/src/github.com/bootapp/proto-core \
       --go_out=plugins=grpc:./proto \
        $GOPATH/src/github.com/bootapp/proto-core/core_common.proto \
        $GOPATH/src/github.com/bootapp/proto-core/dal_user.proto \
        $GOPATH/src/github.com/bootapp/proto-core/dal_auth.proto

#------------------- grpc
protoc -I$GOPATH/src/github.com/bootapp/proto-core \
    -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
    --go_out=plugins=grpc:./proto \
    $GOPATH/src/github.com/bootapp/proto-core/srv_user.proto \
    $GOPATH/src/github.com/bootapp/proto-core/srv_security.proto \
    $GOPATH/src/github.com/bootapp/proto-core/srv_data_store.proto \
    $GOPATH/src/github.com/bootapp/proto-core/srv_auth.proto
#------------------- restful gateway
protoc -I$GOPATH/src/github.com/bootapp/proto-core \
    -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
    --grpc-gateway_out=logtostderr=true:./proto \
    $GOPATH/src/github.com/bootapp/proto-core/srv_user.proto \
    $GOPATH/src/github.com/bootapp/proto-core/srv_security.proto \
    $GOPATH/src/github.com/bootapp/proto-core/srv_data_store.proto \
    $GOPATH/src/github.com/bootapp/proto-core/srv_auth.proto
#------------------- swagger
#protoc -I$GOPATH/src/github.com/bootapp/proto-core \
#    -I$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
#    --swagger_out=logtostderr=true:./proto/core \
#    $GOPATH/src/github.com/bootapp/proto-core/srv_user.proto \
#    $GOPATH/src/github.com/bootapp/proto-core/srv_security.proto
