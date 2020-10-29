# Test Description

This test creates and assignes a network to 2 applications, which communicate.
One has nginx, the other uses curl.
Creates 2 networks, checks internal IPs and does intercommunication

## Test structure

eden.network.tests.txt - escript scenario file

* /image - a folder with docker image
* Dockerfile with nginx, dhcpcd and curl based on Debian
* dhcpcd.conf - setup for dhcpcd
* entrypoint.sh - entrypoint for Docker which runs required workload
(curl, nginx, ip, dhcpcd)
* supervisord.conf - processing params and run strings for workloads
* /testdata - a folder with custom escripts for a workload
* test_networking.txt - main test file
