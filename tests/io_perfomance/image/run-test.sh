#!/bin/bash
if [ -z "$GIT_BRANCH" ]; then FOLDER=FIO-tests-$(date +%H-%M-%d-%m-%Y)-"$EVE_VERSION"; else FOLDER="$GIT_BRANCH"; fi
export FOLDERNAME="$FOLDER"

# Git configurate
echo "Started configuring GitHub"
git config --global http.sslverify false
(cd ~ && git clone https://"$GIT_LOGIN":"$GIT_TOKEN"@github.com/"$GIT_LOGIN"/"$GIT_REPO")
git config --global user.email "fio_test@example.com"
git config --global user.name "FIO"
echo "GitHub configuration is done"

# File management
echo "Started setting up directories"
mkdir ~/"$GIT_REPO"/"$FOLDERNAME"
mkdir ~/"$GIT_REPO"/"$FOLDERNAME"/Configs
mkdir ~/"$GIT_REPO"/"$FOLDERNAME"/Configs/Test-results
mkdir ~/"$GIT_REPO"/"$FOLDERNAME"/Configs/Test-results/Iostat
cp README.md ~/"$GIT_REPO"/"$FOLDERNAME"/
echo "Setting up directories is end"

# Create FIO config
echo "Create config for"
result=$(./mkconfig -type="$FIO_OPTYPE" -bs="$FIO_BS" -jobs="$FIO_JOBS" -depth="$FIO_DEPTH" -time="$FIO_TIME")
export IOSTAT_COUNT="$result"
cp config.fio ~/"$GIT_REPO"/"$FOLDERNAME"/Configs/

# Running IOSTAT
echo "Running IOSTAT $result"
./run-iostat.sh &

# Running FIO
echo "Running FIO"
fio config.fio --output-format=normal,json > ~/"$GIT_REPO"/"$FOLDERNAME"/Configs/Test-results/fio-results

echo "Result FIO generate start"
./fioconv ~/"$GIT_REPO"/"$FOLDERNAME"/Configs/Test-results/fio-results ~/"$GIT_REPO"/"$FOLDERNAME"/SUMMARY.csv
echo "Result FIO generate done"

# Create a new folder in the GIT repository and push the changes
echo "Create a folder and start posting results to GIT"
(cd ~/"$GIT_REPO"/ && git add ~/"$GIT_REPO"/"$FOLDERNAME" && git commit -m "io-results $FOLDERNAME" && git push)
echo "FIO tests are end branch: $FOLDERNAME"

sleep 30m
