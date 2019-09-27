#!/bin/bash 

pkill -9 client

go build client.go


for ds in {1..4}
do
    ./client -p 1100$ds -local -s 127.0.0.1:30000 &
done

# ./client -p 11010 -local -s 127.0.0.1:30000 &
# for ds in {11..12}
# do
#     ./client -p 110$ds -local -s 127.0.0.1:30000 &
# done
# for ds in {13..24}
# do
#     ./client -p 120$ds -local -s 127.0.0.1:30000 &
# done
# for ds in {25..36}
# do
#     ./client -p 130$ds -local -s 127.0.0.1:30000 &
# done
# for ds in {37..48}
# do
#     ./client -p 140$ds -local -s 127.0.0.1:30000 &
# done
# for ds in {49..60}
# do
#     ./client -p 150$ds -local -s 127.0.0.1:30000 &
# done
# for ds in {61..73}
# do
#     ./client -p 160$ds -local -s 127.0.0.1:30000 &
# done
# for ds in {74..86}
# do
#     ./client -p 170$ds -local -s 127.0.0.1:30000 &
# done
# for ds in {87..99}
# do
#     ./client -p 180$ds -local -s 127.0.0.1:30000 &
# done
# ./client -p 18100 -local -s 127.0.0.1:30000 &

# for ds in {1..9}
# do
#     ./client -p 1100$ds -local -s 127.0.0.1:30000 &
# done
# ./client -p 11010 -local -s 127.0.0.1:30000 &
# for ds in {11..90}
# do
#     ./client -p 110$ds -local -s 127.0.0.1:30000 &
# done
# for ds in {91..99}
# do
#     ./client -p 110$ds -local -s 127.0.0.1:30000 &
# done
# ./client -p 12100 -local -s 127.0.0.1:30000 &

# for ds in {101..200}
# do
#     ./client -p 12$ds -local -s 127.0.0.1:30000 &
# done
# for ds in {201..300}
# do
#     ./client -p 13$ds -local -s 127.0.0.1:30000 &
# done
# for ds in {301..400}
# do
#     ./client -p 14$ds -local -s 127.0.0.1:30000 &
# done
# for ds in {401..500}
# do
#     ./client -p 15$ds -local -s 127.0.0.1:30000 &
# done
# for ds in {501..600}
# do
#     ./client -p 16$ds -local -s 127.0.0.1:30000 &
# done
# for ds in {601..700}
# do
#     ./client -p 17$ds -local -s 127.0.0.1:30000 &
# done
# for ds in {701..800}
# do
#     ./client -p 18$ds -local -s 127.0.0.1:30000 &
# done