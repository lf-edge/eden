# EVE for zedcontrol

In order to create image for zedcontrol you should run the following:

```console
eden config add
eden setup --zedcontrol=<domain of zedcontrol> --soft-serial=<soft-serial>
```

Where zedcontrol should be in the notation of `zedcloud.alpha.zededa.net` and soft-serial may be used to overwrite
hardware one and simplify onboarding process.  In this case you can only use `eden eve` commands to manage EVE.
Adam will not work. You can use this command in any combinations of other options of setup type.

In the output of command you will see what to use in the onboarding process in zedcontrol.
