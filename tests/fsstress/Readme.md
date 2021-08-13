
# Fsstress test

This a test to Eden that "stress" the filesystem on the guest VM using the ffsstress utility from the LTP package.

The test is started with the following command:
./eden test ./tests/fsstress

If the test is launched outside the eden directory, then when starting the test, you must pass the *-scriptpath* parameter where you need to specify the absolute path to the *testdata/run-script.sh* file.

You can adjust time of test with environment variable `FSSTRESS_TIME` (default is `96h`).

## Test structure

* eden.fsstress.tests.txt - escript scenario file
* /testdata - a folder with custom escripts for a workload and script for running on guest VM
* fsstress_tests.txt - main test file
