#!/bin/sh

set -e
protoc reader.proto --go_out=plugins=grpc:.
go build
