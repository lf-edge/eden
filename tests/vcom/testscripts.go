package vcom

const testScript = `#!/usr/bin/env python3
import socket
import sys
import os

# Add the path to the generated protobuf files
sys.path.append('.')
import messages_pb2

VMADDR_CID_LOCAL = socket.VMADDR_CID_HOST  # change to 1 for unix.VMADDR_CID_LOCAL
TPM_EK_HANDLE = 0x81000001 # etpm.TpmEKHdl value
VSOCK_PORT = 2000

def vsock_http_request(cid, port, method, path, body=None, headers=None):
    """Make HTTP request over VSOCK"""
    if headers is None:
        headers = {}

    sock = socket.socket(socket.AF_VSOCK, socket.SOCK_STREAM)
    try:
        sock.connect((cid, port))
        request_lines = [f"{method} {path} HTTP/1.1"]
        request_lines.append("Host: vsock")
        request_lines.append("Connection: close")
        for key, value in headers.items():
            request_lines.append(f"{key}: {value}")
        if body:
            request_lines.append(f"Content-Length: {len(body)}")
        request_lines.append("")
        http_request = "\r\n".join(request_lines) + "\r\n"

        # Send request
        sock.send(http_request.encode('utf-8'))
        if body:
            sock.send(body)

        # Read response - first read headers
        response_data = b""
        headers_end = b"\r\n\r\n"
        while headers_end not in response_data:
            chunk = sock.recv(1024)
            if not chunk:
                break
            response_data += chunk
        headers_part = response_data.split(headers_end)[0].decode('utf-8')
        content_length = 0
        for line in headers_part.split('\r\n'):
            if line.lower().startswith('content-length:'):
                content_length = int(line.split(':', 1)[1].strip())
                break
        body_start = response_data.find(headers_end) + len(headers_end)
        body_received = len(response_data) - body_start

        while body_received < content_length:
            chunk = sock.recv(min(4096, content_length - body_received))
            if not chunk:
                break
            response_data += chunk
            body_received += len(chunk)

        # Parse HTTP response
        response_str = response_data.decode('utf-8', errors='ignore')
        if '\r\n\r\n' in response_str:
            headers_part, body_part = response_str.split('\r\n\r\n', 1)
        else:
            headers_part = response_str
            body_part = ""

        # Parse status line
        status_line = headers_part.split('\r\n')[0]
        status_code = int(status_line.split()[1])

        # Return binary body for protobuf
        body_start = response_data.find(b'\r\n\r\n')
        if body_start != -1:
            binary_body = response_data[body_start + 4:]
        else:
            binary_body = b""

        return status_code, binary_body

    finally:
        sock.close()

def decode_key_attr(attr):
    """Decode key attributes"""
    flags = {
        "FlagFixedTPM": 0x00000002,
        "FlagStClear": 0x00000004,
        "FlagFixedParent": 0x00000010,
        "FlagSensitiveDataOrigin": 0x00000020,
        "FlagUserWithAuth": 0x00000040,
        "FlagAdminWithPolicy": 0x00000080,
        "FlagNoDA": 0x00000400,
        "FlagRestricted": 0x00010000,
        "FlagDecrypt": 0x00020000,
        "FlagSign": 0x00040000,
    }

    attr_list = []
    for name, value in flags.items():
        if attr & value != 0:
            attr_list.append(name)

    if not attr_list:
        return "NO ATTRIBUTES"

    return " | ".join(attr_list)

def test_valid_get_public():
    """Test getting public key from TPM via VSOCK HTTP"""
    try:
        request = messages_pb2.TpmRequestGetPub()
        request.index = TPM_EK_HANDLE
        serialized_request = request.SerializeToString()

        print(f"Sending TPM GetPub request via VSOCK (CID: {VMADDR_CID_LOCAL}, Port: {VSOCK_PORT})...")
        status_code, response_body = vsock_http_request(
            cid=VMADDR_CID_LOCAL,
            port=VSOCK_PORT,
            method="POST",
            path="/tpm/getpub",
            body=serialized_request
        )

        # Check status
        if status_code != 200:
            print(f"Error: expected status 200, got {status_code}")
            return False

        # Parse protobuf response
        tmp_resp = messages_pb2.TpmResponseGetPub()
        tmp_resp.ParseFromString(response_body)

        # Validate response
        if len(tmp_resp.public) == 0:
            print("Error: expected non-empty EK, got empty")
            return False

        # Print results like Go version
        print(f"TPM EK: {tmp_resp.public[:16].hex()}...")
        print(f"TPM EK Algorithm: {tmp_resp.algorithm}")
        print(f"TPM EK Attributes: {decode_key_attr(tmp_resp.attributes)}")
        return True

    except Exception as e:
        print(f"Error occurred: {e}")
        import traceback
        traceback.print_exc()
        return False

def main():
    """Main function"""
    print("Testing TPM Get Public Key via VSOCK HTTP...")
    success = test_valid_get_public()

    if success:
        print("\nTest passed!")
        sys.exit(0)
    else:
        print("\nTest failed!")
        sys.exit(1)

if __name__ == '__main__':
    main()`

const protobufFile = `# Copyright (c) 2025 Zededa, Inc.
# SPDX-License-Identifier: Apache-2.0
# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: proto/messages.proto
"""Generated protocol buffer code."""
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import symbol_database as _symbol_database
from google.protobuf.internal import builder as _builder
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()




DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\x14proto/messages.proto\x12\x04vcom\"!\n\x10TpmRequestGetPub\x12\r\n\x05index\x18\x01 \x01(\r\"J\n\x11TpmResponseGetPub\x12\x0e\n\x06public\x18\x01 \x01(\x0c\x12\x11\n\talgorithm\x18\x02 \x01(\r\x12\x12\n\nattributes\x18\x03 \x01(\r\"-\n\x0eTpmRequestSign\x12\r\n\x05index\x18\x01 \x01(\r\x12\x0c\n\x04\x64\x61ta\x18\x02 \x01(\x0c\"\x91\x01\n\x0fTpmResponseSign\x12\x11\n\talgorithm\x18\x01 \x01(\t\x12\x15\n\rrsa_signature\x18\x02 \x01(\x0c\x12\x10\n\x08rsa_hash\x18\x03 \x01(\t\x12\x17\n\x0f\x65\x63\x63_signature_r\x18\x04 \x01(\x0c\x12\x17\n\x0f\x65\x63\x63_signature_s\x18\x05 \x01(\x0c\x12\x10\n\x08\x65\x63\x63_hash\x18\x06 \x01(\t\"!\n\x10TpmRequestReadNv\x12\r\n\x05index\x18\x01 \x01(\r\"!\n\x11TpmResponseReadNv\x12\x0c\n\x04\x64\x61ta\x18\x01 \x01(\x0c\"-\n\x1cTpmRequestActivateCredParams\x12\r\n\x05index\x18\x01 \x01(\r\"N\n\x1dTpmResponseActivateCredParams\x12\n\n\x02\x65k\x18\x01 \x01(\x0c\x12\x0f\n\x07\x61ik_pub\x18\x02 \x01(\x0c\x12\x10\n\x08\x61ik_name\x18\x03 \x01(\x0c\"J\n\x17TpmRequestGeneratedCred\x12\x0c\n\x04\x63red\x18\x01 \x01(\x0c\x12\x0e\n\x06secret\x18\x02 \x01(\x0c\x12\x11\n\taik_index\x18\x03 \x01(\r\"*\n\x18TpmResponseActivatedCred\x12\x0e\n\x06secret\x18\x01 \x01(\x0c\"\"\n\x11TpmRequestCertify\x12\r\n\x05index\x18\x01 \x01(\r\"A\n\x12TpmResponseCertify\x12\x0e\n\x06public\x18\x01 \x01(\x0c\x12\x0b\n\x03sig\x18\x02 \x01(\x0c\x12\x0e\n\x06\x61ttest\x18\x03 \x01(\x0c\x42\x07Z\x05vcom/b\x06proto3')

_globals = globals()
_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, _globals)
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'proto.messages_pb2', _globals)
if _descriptor._USE_C_DESCRIPTORS == False:

  DESCRIPTOR._options = None
  DESCRIPTOR._serialized_options = b'Z\005vcom/'
  _globals['_TPMREQUESTGETPUB']._serialized_start=30
  _globals['_TPMREQUESTGETPUB']._serialized_end=63
  _globals['_TPMRESPONSEGETPUB']._serialized_start=65
  _globals['_TPMRESPONSEGETPUB']._serialized_end=139
  _globals['_TPMREQUESTSIGN']._serialized_start=141
  _globals['_TPMREQUESTSIGN']._serialized_end=186
  _globals['_TPMRESPONSESIGN']._serialized_start=189
  _globals['_TPMRESPONSESIGN']._serialized_end=334
  _globals['_TPMREQUESTREADNV']._serialized_start=336
  _globals['_TPMREQUESTREADNV']._serialized_end=369
  _globals['_TPMRESPONSEREADNV']._serialized_start=371
  _globals['_TPMRESPONSEREADNV']._serialized_end=404
  _globals['_TPMREQUESTACTIVATECREDPARAMS']._serialized_start=406
  _globals['_TPMREQUESTACTIVATECREDPARAMS']._serialized_end=451
  _globals['_TPMRESPONSEACTIVATECREDPARAMS']._serialized_start=453
  _globals['_TPMRESPONSEACTIVATECREDPARAMS']._serialized_end=531
  _globals['_TPMREQUESTGENERATEDCRED']._serialized_start=533
  _globals['_TPMREQUESTGENERATEDCRED']._serialized_end=607
  _globals['_TPMRESPONSEACTIVATEDCRED']._serialized_start=609
  _globals['_TPMRESPONSEACTIVATEDCRED']._serialized_end=651
  _globals['_TPMREQUESTCERTIFY']._serialized_start=653
  _globals['_TPMREQUESTCERTIFY']._serialized_end=687
  _globals['_TPMRESPONSECERTIFY']._serialized_start=689
  _globals['_TPMRESPONSECERTIFY']._serialized_end=754
# @@protoc_insertion_point(module_scope)
`
