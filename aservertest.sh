#!/bin/bash 

pkill -9 client

go build client.go

for ds in {2..8}
do
    ./client -p 1100$ds -local -s 127.0.0.1:30000 &
done
