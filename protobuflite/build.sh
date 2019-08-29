#!/bin/sh
set -e
go build
./protobuflite > test/stream.h;
(cd test && make)
./test/streamtest
