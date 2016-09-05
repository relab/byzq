#! /bin/bash

set -e

go build

./authsrv -port=8080 -key keys/server1 &
./authsrv -port=8081 -key keys/server1 &
./authsrv -port=8082 -key keys/server1 &
./authsrv -port=8083 -key keys/server1 &
# ./authsrv -port=8084 -key keys/server1 &

echo "running, enter to stop"

read && killall authsrv 
