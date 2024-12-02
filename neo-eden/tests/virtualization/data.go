package virtualization

import tk "github.com/lf-edge/eden/pkg/evetestkit"

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

var azureIoTTestScripts = []tk.TestScript{
	{
		Name:           "test_aziot_1.4.0.sh",
		MakeExecutable: true,
		Content: `#!/bin/bash
export DEBIAN_FRONTEND=noninteractive

MS_PROD="https://packages.microsoft.com/config/ubuntu/20.04/packages-microsoft-prod.deb"
AZIOT_IDENTITY_SERVICE="https://github.com/Azure/azure-iotedge/releases/download/1.4.0/aziot-identity-service_1.4.0-1_ubuntu20.04_amd64.deb"
AZIOT_EDGE="https://github.com/Azure/azure-iotedge/releases/download/1.4.0/aziot-edge_1.4.0-1_ubuntu20.04_amd64.deb"
EVE_TOOLS="https://github.com/shjala/eve-tools-deb/raw/main/lfedge-eve-tools-3.3-ubuntu20.04.deb"

# install microsoft repository
wget $MS_PROD -O packages-microsoft-prod.deb
sudo dpkg -i packages-microsoft-prod.deb
rm packages-microsoft-prod.deb

# install pre-requisites
sudo apt-get update
sudo apt-get install -y moby-engine tpm2-tools libprotobuf-dev libprotoc-dev net-tools libssl-dev
sudo apt-get purge -y aziot-identity-service aziot-edge

# install aziot-identity-servic, this is needed by aziot-edge
wget $AZIOT_IDENTITY_SERVICE -O aziot-identity-service.deb
sudo dpkg -i aziot-identity-service.deb
rm aziot-identity-service.deb

# install aziot-edge
wget $AZIOT_EDGE -O aziot-edge.deb
sudo dpkg -i aziot-edge.deb
rm aziot-edge.deb

# install eve-tools, and patch aziot-idendity-service to use eve-tools
# for communication with TPM
wget $EVE_TOOLS -O eve-tools.deb
dpkg-deb -R eve-tools.deb .
sudo cp -r usr/ /
rm eve-tools.deb

# generate certificates needed by aziot-certd and aziot-keyd
git clone https://github.com/Azure/iotedge.git
cd iotedge || exit
git checkout 1.4.0
cd ..
mkdir test-certs
cd test-certs || exit
cp ../iotedge/tools/CACertificates/*.cnf .
cp ../iotedge/tools/CACertificates/certGen.sh .
./certGen.sh create_root_and_intermediate
./certGen.sh create_edge_device_ca_certificate test_aziot_eden_cert
cd ..

>config.toml cat <<-EOF
## DPS provisioning with TPM
[provisioning]
source = "dps"
global_endpoint = "https://global.azure-devices-provisioning.net"
id_scope = "$ID_SCOPE"
[provisioning.attestation]
method = "tpm"
registration_id = "$REGISTRATION_ID"
[edge_ca]
cert = "file:///home/ubuntu/test-certs/certs/iot-edge-device-ca-test_aziot_eden_cert-full-chain.cert.pem"
pk = "file:///home/ubuntu/test-certs/private/iot-edge-device-ca-test_aziot_eden_cert.key.pem"
EOF

sudo cp config.toml /etc/aziot/config.toml
sudo iotedge config apply

rm config.toml
rm -rf usr/ DEBIAN/ iotedge/`,
	},
}
