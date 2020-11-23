#!/bin/bash
FOLDERNAME=FIO-tests-$(date +%H-%M-%d-%m-%Y)-EVE-"$EVE_VER"
export FOLDERNAME

#Git configurate
git config --global http.sslverify false
(cd ~ && git clone https://"$LOGIN":"$TOKEN"@github.com/"$LOGIN"/"$GITREPO")
git config --global user.email "fio_test@example.com"
git config --global user.name "FIO"

#File management
mkdir ~/"$GITREPO"/"$FOLDERNAME"
mkdir ~/"$GITREPO"/"$FOLDERNAME"/Configs
mkdir ~/"$GITREPO"/"$FOLDERNAME"/Configs/Test-results
mkdir ~/"$GITREPO"/"$FOLDERNAME"/Configs/Test-results/Iostat
touch ~/"$GITREPO"/"$FOLDERNAME"/SUMMARY.csv
cp README.md ~/"$GITREPO"/"$FOLDERNAME"/
cp config.fio ~/"$GITREPO"/"$FOLDERNAME"/Configs/

#Create a snapshot of the hardware
lshw -short > ~/"$GITREPO"/"$FOLDERNAME"/HARDWARE.cfg

#Running IOSTAT
./run-iostat.sh &

#Running FIO
fio config.fio > ~/"$GITREPO"/"$FOLDERNAME"/Configs/Test-results/fio-results

#Create a new branch in the GIT repository and push the changes
(cd ~/"$GITREPO"/ && git checkout -b "$FOLDERNAME" && git add ~/"$GITREPO"/"$FOLDERNAME" && git commit -m "fio-results" && git push --set-upstream origin "$FOLDERNAME")
