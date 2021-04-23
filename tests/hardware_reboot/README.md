# Test Description

Key purpose is to verify that a reboot (sudo shutdown -r +1) from the guest doesn’t disrupt the set of assigned adapters.
Tests working correctly only on KVM

## Test structure

eden.hardware_reboot.tests.txt - escript scenario file

* /testdata - a folder with custom escripts for a workload
