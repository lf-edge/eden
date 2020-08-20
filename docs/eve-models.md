Eden has different ways of Eve deployment: 

# Qemu deployment
This is default deployment
Once you run `eden config add default` eden generates config with this type of deployment. 
Also it generates qemu.conf in ~/.eden/ with qemu setings 

This installation type requires qemu package installed. Eve runs in qemu on the same machine. 
This is useful for debugging and getting started, but not that useful in production 

Note that KVM requires that the virtual machine host's processor has virtualization support

# GCP deployment
This deployment type is activated  by flag `--devmodel GCP`
like `make CONFIG='--devmodel GCP' run`
The line above prepares the image for GCP. Use `eden utils gcp` to upload it to Google cloud

# Raspberry 4  deployment
This deployment type is activated  by flag `--devmodel RPi`
like `make CONFIG='--devmodel RPi' run
The line above makes an image for Raspberry. Use simple SD card burner to burn the image and put the card into Raspberry and power it up.

Each deployment requires to run 
```
eden setup
eden start
eden eve onboard
```
