#!/usr/bin/env bash
mkdir -p build
cd build && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo ../srv_core.go && cd ..
cp app.properties build/
cp ./.agollo build/
docker build -t bootapp/service-core -f devops/Dockerfile .