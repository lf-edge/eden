#!/bin/sh

export EDEN_CONFIG="default"
export WORKFLOW="smoke"

echo "Running smoke tests"
../../dist/bin/neoeden.eve.testsuite -test.v
