#!/bin/bash

export DEBIAN_FRONTEND=noninteractive

sudo apt-get update
sudo apt-get install -y tpm2-tools

# Make TPM devices accessible, 777 is OK for testing
sudo chmod 777 /dev/tpm*

# Create the endorsement key (EK) and storage root key (SRK)
if ! tpm2_getcap handles-persistent | grep 0x81010001; then
    tpm2_createek -c 0x81010001 -G rsa -u ek.pub
    tpm2_createprimary -Q -C o -c srk.ctx > /dev/null
    tpm2_evictcontrol -c srk.ctx 0x81000001 > /dev/null
    tpm2_flushcontext -t > /dev/null
fi

if [ ! -f ek.pub ]; then
    echo "EK creation failed"
    exit 1
fi

if [ ! -f srk.ctx ]; then
    echo "SRK creation failed"
    exit 1
fi

if ! tpm2_getcap handles-persistent | grep 0x81010001; then
    echo "EK not found"
    exit 1
fi

if ! tpm2_getcap handles-persistent | grep 0x81000001; then
    echo "SRK not found"
    exit 1
fi

exit 0