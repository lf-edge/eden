#!/bin/bash

# Make TPM devices accessible, 777 is OK for testing
sudo chmod 777 /dev/tpm*

if ! tpm2_getcap handles-persistent | grep 0x81010001; then
    echo "EK not found"
    exit 1
fi

if ! tpm2_getcap handles-persistent | grep 0x81000001; then
    echo "SRK not found"
    exit 1
fi

exit 0
