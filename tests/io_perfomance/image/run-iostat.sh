#!/bin/bash

(( "COUNT_ITER="$FIO_TIME"/3" ))

#Running IOSTAT
for (( i = 1; i <= "$IOSTAT_COUNT"; i++ ))
do
iostat -xm 3 "$COUNT_ITER" > ~/"$GIT_REPO"/"$FOLDERNAME"/Configs/Test-results/Iostat/iostat-fio-group-"$i"
done
