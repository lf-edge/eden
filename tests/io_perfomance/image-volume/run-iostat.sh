#!/bin/bash

#Running IOSTAT
for (( i = 1; i <= 36; i++ ))
do
iostat -xm 3 20 > ~/"$GIT_REPO"/"$FOLDERNAME"/Configs/Test-results/Iostat/iostat-fio-group-"$i"
done
