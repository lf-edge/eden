#!/bin/bash
foldername=FIO-tests-$(date +%H-%M-%d-%m-%Y)

#Git configurate
git config --global http.sslverify false
(cd ~ && git clone https://"$LOGIN":"$TOKEN"@github.com/"$LOGIN"/"$GITREPO")
git config --global user.email "fio_test@example.com"
git config --global user.name "FIO"

#File management
mkdir ~/"$GITREPO"/"$foldername"
mkdir ~/"$GITREPO"/"$foldername"/configs
mkdir ~/"$GITREPO"/"$foldername"/configs/test-results
touch ~/"$GITREPO"/"$foldername"/SUMMARY.csv
cp README.md ~/"$GITREPO"/"$foldername"/
cp config.fio ~/"$GITREPO"/"$foldername"/configs/

#Create a snapshot of the hardware
lshw -short > ~/"$GITREPO"/"$foldername"/HARDWARE.cfg

#Running FIO
fio config.fio > ~/"$GITREPO"/"$foldername"/configs/test-results/fio-result

#Create a new branch in the GIT repository and push the changes
(cd ~/"$GITREPO"/ && git checkout -b "$foldername" && git add ~/"$GITREPO"/"$foldername" && git commit -m "fio-results" && git push --set-upstream origin "$foldername")
