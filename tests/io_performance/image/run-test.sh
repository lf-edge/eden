#!/bin/bash
if [ -z "$GIT_FOLDER" ]; then FOLDER=FIO-tests-$(date +%H-%M-%d-%m-%Y)-"$EVE_VERSION"; else FOLDER="$GIT_FOLDER"; fi
export FOLDER_GIT="$FOLDER"

# File management
echo "Started setting up directories"
mkdir -p /data/"$FOLDER_GIT"
mkdir -p /data/"$FOLDER_GIT"/Configs
mkdir -p /data/"$FOLDER_GIT"/Configs/Test-results
mkdir -p /data/"$FOLDER_GIT"/Configs/Test-results/Iostat
cp README.md /data/"$FOLDER_GIT"/
echo "Setting up directories is end"

# Create FIO config
echo "Create config for FIO"
result=$(./mkconfig -type="$FIO_OPTYPE" -bs="$FIO_BS" -jobs="$FIO_JOBS" -depth="$FIO_DEPTH" -time="$FIO_TIME")
export IOSTAT_COUNT="$result"
cp config.fio /data/"$FOLDER_GIT"/Configs/

# Running IOSTAT
echo "Running IOSTAT $result"
./run-iostat.sh &

if [[ -z "$GIT_REPO" || -z "$GIT_LOGIN" || -z "$GIT_TOKEN" ]] && [ -z "$GIT_LOCAL" ]
then
    # Running FIO
    echo "Running FIO without publish results on github.com"
    fio config.fio --output-format=normal
    echo "FIO tests are end"
else
    # Running FIO
    echo "Running FIO"
    fio config.fio --output-format=normal,json > /data/"$FOLDER_GIT"/Configs/Test-results/fio-results

    echo "Result FIO generate start"
    ./fioconv /data/"$FOLDER_GIT"/Configs/Test-results/fio-results /data/"$FOLDER_GIT"/SUMMARY.csv
    echo "Result FIO generate done"

    # Removing garbage
    rm /data/"$FOLDER_GIT"/fio.test.file

    if [ -z "$GIT_LOCAL" ]
    then
        # Git configurate
        echo "Started configuring GitHub"
        git config --global http.sslverify false
        (cd ~ && git clone https://"$GIT_LOGIN":"$GIT_TOKEN"@github.com/"$GIT_REPO")
        git config --global user.email "fio_test@example.com"
        git config --global user.name "FIO"
        IFS='/' read -ra git_direct <<< "$GIT_REPO"
        echo "GitHub configuration is done"

        if [ -z "$GIT_PATH" ]
        then mv /data/"$FOLDER_GIT"/ ~/"${git_direct[1]}"/
        else mv /data/"$FOLDER_GIT"/ ~/"${git_direct[1]}"/"$GIT_PATH"/
        fi

        # Create a new folder in the GIT repository and push the changes
        echo "Create a folder and start posting results to GIT"
        if [ -z "$GIT_BRANCH" ]
        then (cd ~/"${git_direct[1]}"/ && git add . && git commit -m "fio-results $FOLDER_GIT" && git push)
        else (cd ~/"${git_direct[1]}"/ && git checkout -b "$GIT_BRANCH" && git add . && git commit -m "fio-results $FOLDER_GIT" && git push --set-upstream origin "$GIT_BRANCH")
        fi
    fi
    echo "FIO tests are end: $FOLDER_GIT"

fi
sleep 30m
