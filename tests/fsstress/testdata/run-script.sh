#!/bin/bash

# disable background updates
sudo systemctl disable --now unattended-upgrades
sudo apt-get update -y
sudo apt-get install autoconf libtool fssync make coreutils pkg-config -y

mkdir tests
git clone https://github.com/linux-test-project/ltp.git
cd ~/ltp && make autotools && ./configure && cd testcases/kernel/fs/fsstress && make
nohup sudo sh -c "while true; do top -o %MEM -b -n 1 |head -n 30 >/dev/console; sleep 600; done" &>/dev/null </dev/null &
nohup ./fsstress -d ~/tests -S -n 5 -p 100 -l 0 -s 1000 -v -c &>/dev/null </dev/null &
