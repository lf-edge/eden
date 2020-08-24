# ansible-win-phoronix

This is a role for installing phoronix under windows.

## Requirements


  * KVM or similar
  * Ansible (2.8+?)
  * Compatible version of winrm for ansible

## User Configurable variables

  * winphoronix_shutdown: set to true to shut down at the end
  * winphoronix_sysprep: set to true to run sysprep /generalize

References:

  * <https://docs.microsoft.com/en-us/windows-hardware/manufacture/desktop/sysprep--generalize--a-windows-installation>

## Tags

  * virtio
  * xen

## End to End setup

### Download/run initial image

```
# Create and go into directory
mkdir MSEdge.Win10
cd MSEdge.Win10

# Download from https://developer.microsoft.com/en-us/microsoft-edge/tools/vms/
wget https://az792536.vo.msecnd.net/vms/VMBuild_20190311/VirtualBox/MSEdge/MSEdge.Win10.VirtualBox.zip

# Unzip
unzip ../MSEdge.Win10.VirtualBox.zip

# Unpack
mv "MSEdge - Win10.ova" MSEdge-Win10.tar
tar xaf MSEdge-Win10.tar

# Convert to QCOW2
qemu-img convert "MSEdge - Win10-disk001.vmdk" -O qcow2 -S 4k -c MSEdge-Win10-disk001.qcow2

# Start under KVM with port redirection
qemu-system-x86_64 -enable-kvm -smp cpus=2 -m 2048M -net nic -net user,hostfwd=tcp::5985-:5985,hostfwd=tcp::8022-:22  MSEdge-Win10-disk001.qcow2
```

### Ubuntu Focal Packages

  * ansible
  * python3-winrm

### Manual setup in Windows VM

  * Log in using default credential IEUser / Passw0rd!
  * Click on network connection
  * Click on "network and internet settings"
  * Click on "change connection properties"
  * Select "private" not "public"

### Sample Ansible Inventory

```
msedge-win10 ansible_user=IEUser ansible_password='Passw0rd!' ansible_connection=winrm ansible_winrm_tranport=basic ansible_host=127.0.0.1 ansible_port=5985
```
