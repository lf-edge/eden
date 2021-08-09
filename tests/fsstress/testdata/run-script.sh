#!/bin/bash

sudo apt-get update -y
sudo apt-get install autoconf libtool fssync make coreutils pkg-config -y

mkdir tests
git clone https://github.com/linux-test-project/ltp.git
(cd ~/ltp/ && make autotools && ./configure && cd testcases/kernel/fs/fsstress && make && ./fsstress -d ~/tests -S -n 5 -p 100 -l 0 -s 1000 -v -c)
