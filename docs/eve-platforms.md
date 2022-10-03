# EVE Platforms

If you are running EVE on one of the following platforms, you may need to
build or deploy EVE, or run `eden` commands, with additional considerations.

## VMs and Cloud

EVE uses virtualization; to run in VM-based environments, including most cloud
instances, see [virtual EVE](./virtual-eve.md).

## VirtualBox support

Eden can be used with VirtualBox.
Tested on VirtualBox 6.1 with nested virtualization.

1. Set a devmodel and make the EVE image in Virtual Disk Image (VDI) format
    ```console
    eden config add default --devmodel VBox
    eden setup
    ```
1.  Start eden and onboard Eve. `eve_live` VM will start in VirtualBox at the same time.
    ```console
    eden start
    eden eve onboard
    ```

It now is ready to use.

## Google Cloud support

1. Make an image
1. Upload it to GCP and run
1. start eden (if not started yet)
1. Onboard eve (use `eden eve onboard`)

Since the controller (Adam) runs on the Manager, and it needs to be accessible
from the running EVE device, ensure that the Manager is accessible from your GCP
instance. This means that the Manager must be one of:

* accessible over the Internet
* accessible via a VPN
* running inside GCP in the same project and VPC as the EVE device

In addition, you must set the network access rules to enable traffic.

1. Make an image, specifying the IP of Adam
    ```console
    eden config add default --devmodel GCP
    eden config set default --key adam.eve-ip --value <IP of Adam/Eden>
    eden setup
    ```
1. Upload the image to GCP and run it. You will need [a google service key json](https://cloud.google.com/iam/docs/creating-managing-service-account-keys)
    ```console
    eden utils gcp image -k <PATH TO SERVICE KEY FILE> -p <PROJECT ON GCP> --image-name <NAME OF IMAGE ON GCP> upload <PATH TO EVE IMAGE>
    eden utils gcp vm -k <PATH TO SERVICE KEY FILE> -p <PROJECT ON GCP> --image-name <NAME OF IMAGE ON GCP> --vm-name=<NAME OF VM ON GCP> run
    ```
1. Configure the firewall and make sure Adam is exposed in the network
    ```console
    BWD=$(./eden utils gcp vm get-ip --vm-name eve-eden-one -k <google json key path>)
    ./eden utils gcp firewall -k <google json key path>  --source-range $BWD --name <firewall_rule_name>
    ```
1. Start eden and onboard Eve
    ```console
    eden start
    eden eve onboard
    ```

Your device should onboard.

To see the GCP device console logs:

```console
eden utils gcp vm -k <google json key path> -p <PROJECT ON GCP> --vm-name=<NAME OF VM ON GCP> log
```

`eden utils gcp` also supports:

* --bucket-name  for images
* --machine-type for vm

For all options, run `eden utils gcp --help`.

## Raspberry Pi 4 support

1. If you already have EVE on your SD and want to try the new version, please format SD card with zeroes (at least first 700 MB).
1. Install EVE on Raspberry SD card
   1. Prepare Raspberry image
      ```console
      eden config add default --devmodel RPi4
      eden config set default --key adam.eve-ip --value <IP of Adam/Eden>
      eden setup
      eden start
      ```
   1. You now have an `.img` that can be transferred to SD card:
[installing-images](https://www.raspberrypi.org/documentation/installation/installing-images/).
      * MacOS:
        ```console
        diskutil list
        diskutil unmountDisk /dev/diskN
        sudo dd bs=1m if=path_of_your_image.img of=/dev/rdiskN; sync
        sudo diskutil eject /dev/rdiskN
        ```
1. Put the SD card into your Raspberry Pi and power it on
1. Connect to the Raspberry Pi and run some app.
    ```console
    eden eve onboard
    eden pod deploy -p 8028:80 docker://nginx
    ```

You now have nginx available on public EVE IP at port 8028

To get the status of the deployment:

```console
eden status
eden pod ps
```

If you have an SD card of 32GB or larger, you can boot the Windows 10 ARM64 image:

```console
eden pod deploy docker://itmoeve/eci-windows:2004-compressed-arm64 --vnc-display=1 --memory=2GB --cpus=2 -p 3389:3389
```

The `-p 3389:3389` flag enables port forwarding for RDP.

Once it is running - check with `eden pod ps` - connect via VNC
on the public EVE IP at port 5901 with credentials `IEUser:Passw0rd!`,
or via RDP on port 3389.

### Raspberry Pi 4 WiFi support

To enable WIFi on your Raspberry Pi:

```console
eden config add default --devmodel RPi4 --ssid <Your SSID>
eden setup
eden start
```

You will be asked for a WiFi password upon setup and first reboot. If a WiFi
doesn't require password just press return button when asked for a password.

## Rack of Labs platform

Rack of Labs (RoL) is our development platform for testing and developing EVE on baremetal devices.
It is currently under active development. Access to this system is open only after raising the VPN tunnel.

We can use this system for running tests on RPi4.

1. Make network bootable EVE image.
2. Start eden.
3. Create device rent in RoL.
4. Waiting for device will be ready.
5. Onboard eve (use `eden eve onboard`)
6. Get the logs from the UART console.
7. Close device rent in RoL.

Since the controller (Adam) runs on the Manager, and it needs to be accessible
from the running EVE device, ensure that the Manager is accessible from your GCP
instance. This means that the Manager must be one of:

* accessible over the Internet
* accessible via a VPN

1. Make network bootable EVE image for arm64.

    ```console
    eden config add default --devmodel=general --arch=arm64
    eden config set default --key adam.eve-ip --value <IP of Adam/Eden>
    eden config set default --key registry.ip --value <IP of Adam/Eden>
    eden setup -v debug --netboot=true
    ```

2. Start eden

   ```console
   eden start
   ```

3. Create device rent in RoL platform.

   ```console
   export ROL_API_URL="http://10.10.88.2/api/"
   export ROL_API_KEY="777cb750-8690-4852-89e6-9990c8eb6887"
   export PROJECT_ID="333cb350-82c0-c232-89e6-7770c8eb6887"
   eden rol rent create -m "raspberry" --model "pi_4_model_b_8gb" -p "$PROJECT_ID" -n "name_of_the_rent"
   ```

   The output of the last command returns the lease identifier to standard output, or an error to stderr if we have problems.

   `http://10.10.88.2/api/` - local address of the system in VPN network.

   `777cb750-8690-4852-89e6-9990c8eb6887` - example API key.
4. Wait 3-4 minutes and check the machine status of the device. If the state is ready, we can move on to the next step.

   `eden rol rent get -i "$rent_id" -p "$PROJECT_ID" | jq -r .machineState`
5. Onboard eve (use `eden eve onboard`)
6. Let's get the logs from the UART console, so we can write them to a file for further analysis.

   `eden rol rent console-output -i "$rent_id" -p "$PROJECT_ID" > uart.log`
7. When you no longer need the device, you need to close the rent.
   `eden rol rent close -i "$rent_id" -p "$PROJECT_ID"`
