#!/bin/bash 

pkill -9 client

go build client.go

for ds in {1..9}
do
    ./client -p 1100$ds -local -s 127.0.0.1:30000 &
done
./client -p 11010 -local -s 127.0.0.1:30000 &
for ds in {11..25}
do
    ./client -p 110$ds -local -s 127.0.0.1:30000 &
done
for ds in {26..50}
do
    ./client -p 120$ds -local -s 127.0.0.1:30000 &
done
for ds in {51..75}
do
    ./client -p 140$ds -local -s 127.0.0.1:30000 &
done
for ds in {76..99}
do
    ./client -p 180$ds -local -s 127.0.0.1:30000 &
done
./client -p 18100 -local -s 127.0.0.1:30000 &
