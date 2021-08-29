# Storage Information

For convenience, this documentation on retrieving information directly from a
storage device on which EVE has been flashed is available here. Eventually,
this will be moved to the primary EVE documentation.

To get information from a storage device, such as an SD card or USB drive,
on which EVE was previously flashed, do the following.

1. (macOS only) Install [brew](https://brew.sh/) and required packages:
```console
brew cask install osxfuse
brew install ext4fuse squashfuse
```
1. List partitions; you should find your device with 5 partitions on it.
   * For MacOS `diskutil list`
   * For Linux `lsblk`
1. Mount partitions of the storage device on your computer
   * For MacOS (`diskN` is the storage device card from the previous step)
```console
sudo umount /dev/diskN*
sudo squashfuse /dev/diskNs2 ~/tmp/rootfs -o allow_other
sudo ext4fuse /dev/diskNs9 ~/tmp/persist -o allow_other
```
   * For Linux (`sdN` is the storage device from the previous step)
```console
mkdir -p ~/tmp/rootfs ~/tmp/persist
sudo umount /dev/sdN*
sudo mount /dev/sdN2 ~/tmp/rootfs
sudo mount /dev/sdN9 ~/tmp/persist
```
1. Extract files and save them:
   * `syslog.txt` contains logs of EVE: `sudo cp ~/tmp/persist/rsyslog/syslog.txt ~/syslog.txt`
   * `eve-release` contains the installed version of EVE: `sudo cp ~/tmp/rootfs/etc/eve-release ~/eve-release`
1. Umount and eject SD
   * For MacOS
```console
sudo umount /dev/diskN*
sudo diskutil eject /dev/diskN
```
   * For Linux
```console
sudo umount /dev/sdN*
sudo eject /dev/sdN
```
