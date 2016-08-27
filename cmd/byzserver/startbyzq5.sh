#! /bin/bash

set -e

go build

./byzserver -port=8080 &
./byzserver -port=8081 &
./byzserver -port=8082 &
./byzserver -port=8083 &
./byzserver -port=8084 &

echo "running, enter to stop"

read && killall byzserver 
