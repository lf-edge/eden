#!/bin/sh

export EDEN_CONFIG="default"
export WORKFLOW="virtualization"

echo "Running virtualization tests"
../../dist/bin/neoeden.virtualization.testsuite -test.v
