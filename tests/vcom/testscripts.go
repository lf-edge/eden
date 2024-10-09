package vcom

const testScript = `#!/usr/bin/env python3

"""Test vsock communication with the host"""
import socket

CID = socket.VMADDR_CID_HOST
PORT = 2000
s = socket.socket(socket.AF_VSOCK, socket.SOCK_STREAM)
s.connect((CID, PORT))
s.sendall(b"{\"channel\":2,\"request\":1}")
response = s.recv(1024)
print(response.decode('utf-8'))
s.close()`
