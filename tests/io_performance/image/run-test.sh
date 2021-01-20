#!/bin/bash
if [ -z "$GIT_FOLDER" ]; then FOLDER=FIO-tests-$(date +%H-%M-%d-%m-%Y)-"$EVE_VERSION"; else FOLDER="$GIT_FOLDER"; fi
export FOLDER_GIT="$FOLDER"

# File management
echo "Started setting up directories"
mkdir ~/"$FOLDER_GIT"
mkdir ~/"$FOLDER_GIT"/Configs
mkdir ~/"$FOLDER_GIT"/Configs/Test-results
mkdir ~/"$FOLDER_GIT"/Configs/Test-results/Iostat
cp README.md ~/"$FOLDER_GIT"/
echo "Setting up directories is end"

# Create FIO config
echo "Create config for"
result=$(./mkconfig -type="$FIO_OPTYPE" -bs="$FIO_BS" -jobs="$FIO_JOBS" -depth="$FIO_DEPTH" -time="$FIO_TIME")
export IOSTAT_COUNT="$result"
cp config.fio ~/"$FOLDER_GIT"/Configs/

# Running IOSTAT
echo "Running IOSTAT $result"
./run-iostat.sh &

# Running FIO
echo "Running FIO"
fio config.fio --output-format=normal,json > ~/"$FOLDER_GIT"/Configs/Test-results/fio-results

echo "Result FIO generate start"
./fioconv ~/"$FOLDER_GIT"/Configs/Test-results/fio-results ~/"$FOLDER_GIT"/SUMMARY.csv
echo "Result FIO generate done"

# Git configurate
echo "Started configuring GitHub"
git config --global http.sslverify false
(cd ~ && git clone https://"$GIT_LOGIN":"$GIT_TOKEN"@github.com/"$GIT_LOGIN"/"$GIT_REPO")
git config --global user.email "fio_test@example.com"
git config --global user.name "FIO"
echo "GitHub configuration is done"

mv ~/"$FOLDER_GIT"/ ~/"$GIT_REPO"/

# Create a new folder in the GIT repository and push the changes
echo "Create a folder and start posting results to GIT"
if [ -z "$GIT_BRANCH" ]
then (cd ~/"$GIT_REPO"/ && git add ~/"$GIT_REPO"/"$FOLDER_GIT" && git commit -m "io-results $FOLDER_GIT" && git push)
else (cd ~/"$GIT_REPO"/ && git checkout -b "$GIT_BRANCH" && git add ~/"$GIT_REPO"/"$FOLDER_GIT" && git commit -m "io-results $FOLDER_GIT" && git push --set-upstream origin "$GIT_BRANCH")
fi
echo "FIO tests are end: $FOLDER_GIT"

sleep 30m
