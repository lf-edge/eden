package virtualization

import tk "github.com/lf-edge/eden/pkg/evetestkit"

type TestScript struct {
	Name           string
	DstPath        string
	Content        string
	MakeExecutable bool
}

var vComLinkTestScript = []tk.TestScript{
	{
		Name:           "vcomlink_test.py",
		MakeExecutable: false,
		Content: `#!/usr/bin/env python3
"""Test vsock communication with the host"""
import socket

CID = socket.VMADDR_CID_HOST
PORT = 2000
s = socket.socket(socket.AF_VSOCK, socket.SOCK_STREAM)
s.connect((CID, PORT))
s.sendall(b"{\"channel\":2,\"request\":1}")
response = s.recv(1024)
print(response.decode('utf-8'))
s.close()
`,
	},
}

var vTPMTestScripts = []tk.TestScript{
	{
		Name:           "make_tpm_keys.sh",
		MakeExecutable: true,
		Content: `#!/bin/bash
export DEBIAN_FRONTEND=noninteractive

sudo apt-get update
sudo apt-get install -y tpm2-tools

# Make TPM devices accessible, 777 is OK for testing
sudo chmod 777 /dev/tpm*

# Create the endorsement key (EK) and storage root key (SRK)
tpm2_createek -c 0x81010001 -G rsa -u ek.pub
tpm2_createprimary -Q -C o -c srk.ctx > /dev/null
tpm2_evictcontrol -c srk.ctx 0x81000001 > /dev/null
tpm2_flushcontext -t > /dev/null

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
`,
	},
	{
		Name:           "check_tpm_keys.sh",
		MakeExecutable: true,
		Content: `#!/bin/bash
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
`,
	}}
