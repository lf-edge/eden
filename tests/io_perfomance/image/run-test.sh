#!/bin/bash
FOLDERNAME=FIO-tests-$(date +%H-%M-%d-%m-%Y)-"$EVE_VERSION"
export FOLDERNAME

#Git configurate
echo "Started configuring GitHub"
git config --global http.sslverify false
(cd ~ && git clone https://"$GIT_LOGIN":"$GIT_TOKEN"@github.com/"$GIT_LOGIN"/"$GIT_REPO")
git config --global user.email "fio_test@example.com"
git config --global user.name "FIO"
echo "GitHub configuration is done"

#File management
echo "Started setting up directories"
mkdir ~/"$GIT_REPO"/"$FOLDERNAME"
mkdir ~/"$GIT_REPO"/"$FOLDERNAME"/Configs
mkdir ~/"$GIT_REPO"/"$FOLDERNAME"/Configs/Test-results
mkdir ~/"$GIT_REPO"/"$FOLDERNAME"/Configs/Test-results/Iostat
touch ~/"$GIT_REPO"/"$FOLDERNAME"/SUMMARY.csv
cp README.md ~/"$GIT_REPO"/"$FOLDERNAME"/
cp config.fio ~/"$GIT_REPO"/"$FOLDERNAME"/Configs/
echo "Setting up directories is end"

#Create a snapshot of the hardware
lshw -short > ~/"$GIT_REPO"/"$FOLDERNAME"/HARDWARE.cfg

#Running IOSTAT
echo "Running IOSTAT"
./run-iostat.sh &

#Running FIO
echo "Running FIO"
fio config.fio > ~/"$GIT_REPO"/"$FOLDERNAME"/Configs/Test-results/fio-results

#Create a new branch in the GIT repository and push the changes
echo "Create a branch and start posting results to GIT"
(cd ~/"$GIT_REPO"/ && git checkout -b "$FOLDERNAME" && git add ~/"$GIT_REPO"/"$FOLDERNAME" && git commit -m "fio-results" && git push --set-upstream origin "$FOLDERNAME")
echo "FIO tests are end branch:""$FOLDERNAME"
sleep 30m
