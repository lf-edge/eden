#!/usr/bin/env python3

import xml.etree.ElementTree as ET

tree = ET.parse("/dev/stdin")
root = tree.getroot()

for index, result in enumerate(root.findall("Result")):
    for item in ("Title", "Scale"):
        print(f"{index}:{item}:{result.find(item).text}")
    value = result.find("Data").find("Entry").find("Value").text
    print(f"{index}:Value:{value}")
    print("")
