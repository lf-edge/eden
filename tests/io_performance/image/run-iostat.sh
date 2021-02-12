#!/bin/bash
if [ -z "$FIO_TIME" ]; then TIME_IOTEST=60; else TIME_IOTEST="$FIO_TIME"; fi

(( "COUNT_ITER="$TIME_IOTEST"/3" ))

#Running IOSTAT
for (( i = 1; i <= "$IOSTAT_COUNT"; i++ ))
do
iostat -xm 3 "$COUNT_ITER" > /data/"$FOLDER_GIT"/Configs/Test-results/Iostat/iostat-fio-group-"$i"
done
