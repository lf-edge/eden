# Data flow from running EVE

Eden support several commands to show information coming from EVE in raw format.

## LOGS

To view logs from EVE you can use the following command:

```bash
./eden log

Scans the ADAM logs for correspondence with regular expressions requests to json fields.

Usage:
eden log [field:regexp ...] [flags]

Flags:
-f, --follow          Monitor changes in selected directory
--format string   Format to print logs, supports: lines, json (default "lines")
-h, --help            help for log
-o, --out strings     Fields to print. Whole message if empty.
--tail uint       Show only last N lines

Global Flags:
--config string      Name of config (default "default")
-v, --verbosity string   Log level (debug, info, warn, error, fatal, panic (default "info")
```

For example: `eden log --tail=1 --format=json` will output something like:

```bash
{"severity":"info","source":"zedagent","iid":"1533","content":"{\"file\":\"/pillar/types/zedroutertypes.go:1044\",\"func\":\"github.com/lf-edge/eve/pkg/pillar/types.DeviceNetworkStatus.LogModify\",\"ifname\":\"eth1\",\"last-error\":\"\",\"last-failed\":\"0001-01-01T00:00:00Z\",\"last-succeeded\":\"2021-05-17T14:49:46.899694181Z\",\"level\":\"info\",\"log_event_type\":\"log\",\"msg\":\"DeviceNetworkStatus port modify\",\"obj_key\":\"devicenetwork_status-global\",\"obj_type\":\"devicenetwork_status\",\"old-last-error\":\"\",\"old-last-failed\":\"0001-01-01T00:00:00Z\",\"old-last-succeeded\":\"2021-05-17T14:44:46.824730731Z\",\"pid\":1533,\"source\":\"zedagent\",\"time\":\"2021-05-17T14:49:46.930133344Z\"}\n","msgid":3555,"timestamp":{"seconds":1621262986,"nanos":930133344},"filename":"/pillar/types/zedroutertypes.go:1044","function":"github.com/lf-edge/eve/pkg/pillar/types.DeviceNetworkStatus.LogModify"}
```

## INFO messages

To view info messages from EVE you can use the following command:

```bash
./eden info

Scans the ADAM Info for correspondence with regular expressions requests to json fields.

Usage:
  eden info [field:regexp ...] [flags]

Flags:
  -f, --follow        Monitor changes in selected directory
  -h, --help          help for info
  -o, --out strings   Fields to print. Whole message if empty.
      --tail uint     Show only last N lines

Global Flags:
      --config string      Name of config (default "default")
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic (default "info")
```

For example: `eden info --tail=1` will output something like:

```bash
ztype: ZiDevice
devId: a9ee33b7-a5f7-4a5b-b1c3-fce73fbabd6f
[0]: ztype:ZiDevice devId:"a9ee33b7-a5f7-4a5b-b1c3-fce73fbabd6f" dinfo:{machineArch:"x86_64" cpuArch:"unknown" platform:"unknown" ncpu:4 memory:3928 storage:7369 powerCycleCounter:-1 minfo:{manufacturer:"QEMU" productName:"Standard PC (Q35 + ICH9, 2009)" version:"pc-q35-5.2" serialNumber:"31415926" UUID:"Not Settable" biosVendor:"EFI Development Kit II / OVMF" biosVersion:"0.0.0" biosReleaseDate:"02/06/2015"} network:{macAddr:"52:54:00:12:34:56" devName:"eth0" IPAddrs:"192.168.0.10" IPAddrs:"fec0::38fe:ce67:fcf7:def1" IPAddrs:"fec0::f309:dd8a:2b6b:db21" IPAddrs:"fe80::f3b:569c:ffb9:71bd" defaultRouters:"192.168.0.2" defaultRouters:"fe80::2" dns:{DNSservers:"192.168.0.3"} up:true location:{UnderlayIP:"109.71.177.101" Hostname:"101.177.smarthome.spb.ru" City:"Saint Petersburg" Country:"RU" Loc:"59.9386,30.3141" Org:"AS31376 Smart Telecom Limited" Postal:"190000"} uplink:true networkErr:{timestamp:{seconds:1621262986 nanos:903656353}} localName:"eth0" proxy:{}} network:{macAddr:"52:54:00:12:34:57" devName:"eth1" IPAddrs:"192.168.0.11" IPAddrs:"fec0::39e7:4582:e951:8836" IPAddrs:"fec0::82f6:c59:a169:efa8" IPAddrs:"fe80::5054:ff:fe12:3457" defaultRouters:"192.168.0.2" defaultRouters:"fe80::2" dns:{DNSservers:"192.168.0.3"} up:true location:{UnderlayIP:"109.71.177.101" Hostname:"101.177.smarthome.spb.ru" City:"Saint Petersburg" Country:"RU" Loc:"59.9386,30.3141" Org:"AS31376 Smart Telecom Limited" Postal:"190000"} networkErr:{timestamp:{seconds:1621262986 nanos:899694181}} localName:"eth1" proxy:{}} assignableAdapters:{type:PhyIoNetEth name:"eth0" members:"eth0" usedByBaseOS:true ioAddressList:{macAddress:"52:54:00:12:34:56"} usage:PhyIoUsageShared err:{description:"Not PCI virtio0" timestamp:{seconds:1621261186 nanos:139984208}}} assignableAdapters:{type:PhyIoNetEth name:"eth1" members:"eth1" usedByBaseOS:true ioAddressList:{macAddress:"52:54:00:12:34:57"} usage:PhyIoUsageMgmtAndApps err:{description:"Not PCI virtio1" timestamp:{seconds:1621261186 nanos:141401690}}} assignableAdapters:{type:PhyIoUSB name:"USB0" members:"USB0:1" usage:PhyIoUsageDedicated} assignableAdapters:{type:PhyIoUSB name:"USB1" members:"USB0:2" usage:PhyIoUsageDedicated} assignableAdapters:{type:PhyIoUSB name:"USB2" members:"USB0:3" usage:PhyIoUsageDedicated} assignableAdapters:{type:PhyIoUSB name:"USB3" members:"USB1:1" usage:PhyIoUsageDedicated} assignableAdapters:{type:PhyIoUSB name:"USB4" members:"USB1:2" usage:PhyIoUsageDedicated} assignableAdapters:{type:PhyIoUSB name:"USB5" members:"USB1:3" usage:PhyIoUsageDedicated} assignableAdapters:{type:PhyIoUSB name:"USB6" members:"USB2:1" usage:PhyIoUsageDedicated} assignableAdapters:{type:PhyIoUSB name:"USB7" members:"USB2:2" usage:PhyIoUsageDedicated} assignableAdapters:{type:PhyIoUSB name:"USB8" members:"USB2:3" usage:PhyIoUsageDedicated} assignableAdapters:{type:PhyIoUSB name:"USB9" members:"USB3:1" usage:PhyIoUsageDedicated} assignableAdapters:{type:PhyIoUSB name:"USB10" members:"USB3:2" usage:PhyIoUsageDedicated} assignableAdapters:{type:PhyIoUSB name:"USB11" members:"USB3:3" usage:PhyIoUsageDedicated} dns:{DNSservers:"192.168.0.3:53"} storageList:{mountPath:"/persist/vault/downloader"} storageList:{mountPath:"/persist/vault/volumes"} storageList:{mountPath:"/persist/log"} storageList:{mountPath:"/persist/vault/verifier"} storageList:{device:"sda2" total:300} storageList:{mountPath:"/persist/tmp"} storageList:{mountPath:"/persist/newlog"} storageList:{mountPath:"/persist/checkpoint"} storageList:{mountPath:"/persist/status"} storageList:{device:"sda3" total:300} storageList:{mountPath:"/persist/certs"} storageList:{device:"sda" total:8192} storageList:{device:"sda4" total:1} storageList:{device:"sda9" total:7553} storageList:{mountPath:"/persist/containerd"} storageList:{mountPath:"/config" total:1} storageList:{device:"sda1" total:36} storageList:{mountPath:"/persist/clear/volumes"} storageList:{mountPath:"/persist" total:7369 storageLocation:true} storageList:{mountPath:"/" total:1964} bootTime:{seconds:1621261079} swList:{activated:true partitionLabel:"IMGA" partitionDevice:"/dev/sda2" partitionState:"active" status:INSTALLED shortVersion:"0.0.0-master-37a88878-kvm-amd64" downloadProgress:100 userStatus:UPDATED} swList:{partitionLabel:"IMGB" partitionDevice:"/dev/sda3" partitionState:"unused" status:INITIAL} HostName:"a9ee33b7-a5f7-4a5b-b1c3-fce73fbabd6f" metricItems:{key:"lte-networks"} lastRebootReason:"NORMAL: First boot of device - at 2021-05-17T14:18:39.843661991Z" lastRebootTime:{seconds:1621261119 nanos:843661991} systemAdapter:{status:{version:1 key:"zedagent" timePriority:{seconds:1621261186 nanos:77231499} lastSucceeded:{seconds:1621262986 nanos:903707053} ports:{ifname:"eth0" name:"eth0" isMgmt:true free:true dhcpType:4 ntpServer:"<nil>" proxy:{} defaultRouters:"<nil>" dns:{} err:{timestamp:{seconds:1621262986 nanos:903656353}} usage:PhyIoUsageShared networkUUID:"6822e35f-c1b8-43ca-b344-0bbc0ece8cf1"} ports:{ifname:"eth1" name:"eth1" free:true dhcpType:4 ntpServer:"<nil>" proxy:{} defaultRouters:"<nil>" dns:{} err:{timestamp:{seconds:1621262986 nanos:899694181}} usage:PhyIoUsageMgmtAndApps networkUUID:"6822e35f-c1b8-43ca-b344-0bbc0ece8cf2"}} status:{version:1 key:"lastresort" timePriority:{} lastFailed:{seconds:1621261145 nanos:66278397} ports:{ifname:"eth0" name:"eth0" isMgmt:true free:true dhcpType:4 ntpServer:"<nil>" proxy:{} defaultRouters:"<nil>" dns:{} err:{description:"SendOnIntf to https://mydomain.adam:3333/api/v2/edgedevice/ping reqlen 0 statuscode 401 Unauthorized" timestamp:{seconds:1621261145 nanos:65604290}} usage:PhyIoUsageShared} ports:{ifname:"eth1" name:"eth1" isMgmt:true free:true dhcpType:4 ntpServer:"<nil>" proxy:{} defaultRouters:"<nil>" dns:{} err:{} usage:PhyIoUsageMgmtAndApps}}} HSMStatus:NOTFOUND HSMInfo:"Not Available" dataSecAtRestInfo:{status:DATASEC_AT_REST_DISABLED info:"TPM is either absent or not in use" vaultList:{name:"Application Data Store" status:DATASEC_AT_REST_DISABLED vaultErr:{description:"TPM is either absent or not in use" timestamp:{seconds:1621261193 nanos:163352964}}}} sec_info:{sha_root_ca:"߄[\x17'\xdd\xfe\x9fI\xedm\x00\xbcҦ\xebG\x1f\x9d\x1c\xb5rv\xf9\xf7(\x87\x84\xce[I\xec" sha_tls_root_ca:"\xbb\x14Q\xb83\\\xd0\xef\x0f\x8dnQQT\xc9MvO\x1d\xdd\x0b$\x7f^a\x99\xae;-\xee\xc90"} configItemStatus:{configItems:{key:"app.allow.vnc" value:{value:"true"}} configItems:{key:"debug.default.loglevel" value:{value:"info"}} configItems:{key:"debug.default.remote.loglevel" value:{value:"warning"}} configItems:{key:"newlog.allow.fastupload" value:{value:"true"}} configItems:{key:"timer.config.interval" value:{value:"5"}} configItems:{key:"timer.metric.interval" value:{value:"10"}}} rebootConfigCounter:1000 last_boot_reason:BOOT_REASON_FIRST hardware_watchdog_present:true capabilities:{HWAssistedVirtualization:true IOVirtualization:true}} atTimeStamp:{seconds:1621262986 nanos:961325577}
atTimeStamp: seconds:1621262986 nanos:961325577
```

## Metrics messages

To view metrics messages from EVE you can use the following command:

```bash
./eden metric

Scans the ADAM metrics for correspondence with regular expressions requests to json fields.

Usage:
  eden metric [field:regexp ...] [flags]

Flags:
  -f, --follow        Monitor changes in selected metrics
  -h, --help          help for metric
  -o, --out strings   Fields to print. Whole message if empty.
      --tail uint     Show only last N lines

Global Flags:
      --config string      Name of config (default "default")
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic (default "info")
```

For example: `eden metric --tail=1` will output something like:

```bash
DevID: a9ee33b7-a5f7-4a5b-b1c3-fce73fbabd6f     AtTimeStamp: 2021-05-17 14:56:08.096166558 +0000 UTC    Dm: memory:{usedMem:476 availMem:3452 usedPercentage:12.118126272912424 availPercentage:87.88187372708758} network:{iName:"eth0" txBytes:6748987 rxBytes:72164442 txPkts:34085 rxPkts:80542 localName:"eth0"} network:{iName:"eth1" txBytes:83686 rxBytes:92301 txPkts:486 rxPkts:430 localName:"eth1"} zedcloud:{ifName:"eth0" success:1371 lastSuccess:{seconds:1621263366 nanos:235463285} urlMetrics:{url:"https://mydomain.adam:3333/api/v2/edgedevice/id/a9ee33b7-a5f7-4a5b-b1c3-fce73fbabd6f/flowlog" sentMsgCount:1 sentByteCount:816 recvMsgCount:1 total_time_spent:9} urlMetrics:{url:"https://mydomain.adam:3333/api/v2/edgedevice/config" sentMsgCount:1 recvMsgCount:1 recvByteCount:197 total_time_spent:16} urlMetrics:{url:"https://mydomain.adam:3333/api/v2/edgedevice/uuid" sentMsgCount:1 recvMsgCount:1 recvByteCount:10 total_time_spent:8} urlMetrics:{url:"docker://index.docker.io/itmoeve/eclient@sha256:a1f26a56ef2fee1d5ee254cbda33fb7a5844f7d7e2e99668347733e88b1a1f75" sentMsgCount:1 sentByteCount:1024 recvMsgCount:1 recvByteCount:765 total_time_spent:1653} urlMetrics:{url:"docker://index.docker.io/itmoeve/eclient@sha256:c51ff6ae8403909a1cd6fcc9ec52309fbcf4b91948905d5ee6be056407c3d4f3" sentMsgCount:1 sentByteCount:1024 recvMsgCount:1 recvByteCount:1645 total_time_spent:1661} urlMetrics:{url:"docker://index.docker.io/itmoeve/eclient@sha256:f9625b9acd847c7633a8227ce4450c4a0645f83923482ef836cbe53ce1098067" sentMsgCount:1 sentByteCount:1024 recvMsgCount:1 recvByteCount:444 total_time_spent:1631} urlMetrics:{url:"https://mydomain.adam:3333/api/v2/edgedevice/id/a9ee33b7-a5f7-4a5b-b1c3-fce73fbabd6f/metrics" sentMsgCount:343 sentByteCount:2485161 recvMsgCount:343 total_time_spent:3292} urlMetrics:{url:"https://mydomain.adam:3333/api/v2/edgedevice/certs" sentMsgCount:2 recvMsgCount:2 recvByteCount:5448 total_time_spent:5} urlMetrics:{url:"docker://index.docker.io/itmoeve/eclient@sha256:a4b77138cbadd7341e855095ec7f7ff57eb7db0d0e7a5478f21cac89ab79374b" sentMsgCount:1 sentByteCount:1024 recvMsgCount:1 recvByteCount:119 total_time_spent:1614} urlMetrics:{url:"docker://index.docker.io/itmoeve/eclient@sha256:5aa46b441e6f215479a8de4fb64fef561b2103ae91d630b7214fea51c3a20a28" sentMsgCount:1 sentByteCount:1024 recvMsgCount:1 recvByteCount:158 total_time_spent:1680} urlMetrics:{url:"https://mydomain.adam:3333/api/v2/edgedevice/register" sentMsgCount:1 sentByteCount:899 recvMsgCount:1 total_time_spent:297} urlMetrics:{url:"docker://index.docker.io/itmoeve/eclient@sha256:051e2b8d242baf92d678f63b84ed4a4af5a8bc3efe11487164c1e2413190e85d" sentMsgCount:1 sentByteCount:1024 recvMsgCount:1 recvByteCount:3229 total_time_spent:1600} urlMetrics:{url:"docker://index.docker.io/itmoeve/eclient@sha256:2b61c0590645f44cde086dc05885c0fe1ae6c46f17b7e44cc16259a04520f4d6" sentMsgCount:1 sentByteCount:1024 recvMsgCount:1 recvByteCount:1039 total_time_spent:1592} urlMetrics:{url:"docker://index.docker.io/itmoeve/eclient@sha256:83ee3a23efb7c75849515a6d46551c608b255d8402a4d3753752b88e0dc188fa" sentMsgCount:1 sentByteCount:1024 recvMsgCount:1 recvByteCount:28565893 total_time_spent:5859} urlMetrics:{url:"docker://index.docker.io/itmoeve/eclient@sha256:654864fa19a37c13059f91f4f5e227d96c9ace3aaa59b53ef1d2f37a67794127" sentMsgCount:1 sentByteCount:1024 recvMsgCount:1 recvByteCount:6523 total_time_spent:1542} urlMetrics:{url:"docker://index.docker.io/itmoeve/eclient@sha256:57a7e84f11b2df67e5c485852c2dbd08c678b51ed69043152829a28216c88d9d" sentMsgCount:1 sentByteCount:1024 recvMsgCount:1 recvByteCount:36576501 total_time_spent:6659} urlMetrics:{url:"https://mydomain.adam:3333/api/v2/edgedevice/id/a9ee33b7-a5f7-4a5b-b1c3-fce73fbabd6f/config" sentMsgCount:680 sentByteCount:46713 recvMsgCount:680 recvByteCount:6830 total_time_spent:5277} urlMetrics:{url:"https://mydomain.adam:3333/api/v2/edgedevice/id/a9ee33b7-a5f7-4a5b-b1c3-fce73fbabd6f/info" sentMsgCount:237 sentByteCount:139946 recvMsgCount:237 total_time_spent:885} urlMetrics:{url:"https://mydomain.adam:3333/api/v2/edgedevice/id/a9ee33b7-a5f7-4a5b-b1c3-fce73fbabd6f/attest" sentMsgCount:3 sentByteCount:2484 recvMsgCount:3 recvByteCount:351 total_time_spent:176} urlMetrics:{url:"docker://index.docker.io/itmoeve/eclient@sha256:0d6f6830ca9a91a2707b4bdcb6d4bda90a1a81b3e5bf3ce6cf2c6b131fe7d45a" sentMsgCount:1 sentByteCount:1024 recvMsgCount:1 recvByteCount:120 total_time_spent:1556} urlMetrics:{url:"docker://index.docker.io/itmoeve/eclient@sha256:db98fc6f11f08950985a203e07755c3262c680d00084f601e7304b768c83b3b1" sentMsgCount:1 sentByteCount:1024 recvMsgCount:1 recvByteCount:843 total_time_spent:1762} urlMetrics:{url:"docker://index.docker.io/itmoeve/eclient@sha256:126ad37f6270cd8f55a9fad211a06845b805c1e7caed5dd1f2832d4007c98695" sentMsgCount:1 sentByteCount:1024 recvMsgCount:1 recvByteCount:370 total_time_spent:1693} urlMetrics:{url:"docker://index.docker.io/itmoeve/eclient@sha256:c280633a416de433f317dd64395c5669d4483dd153104367b911c7735026a38d" sentMsgCount:1 sentByteCount:1024 recvMsgCount:1 recvByteCount:3021 total_time_spent:1134} urlMetrics:{url:"docker://index.docker.io/itmoeve/eclient@sha256:f611acd52c6cad803b06b5ba932e4aabd0f2d0d5a4d050c81de2832fcb781274" sentMsgCount:1 sentByteCount:1024 recvMsgCount:1 recvByteCount:162 total_time_spent:1575} urlMetrics:{url:"https://mydomain.adam:3333/api/v2/edgedevice/id/a9ee33b7-a5f7-4a5b-b1c3-fce73fbabd6f/apps/instanceid/dbd53bf1-d7f7-4f7a-ac27-fc0621be50ba/newlogs" sentMsgCount:2 sentByteCount:4267 recvMsgCount:2 total_time_spent:20} urlMetrics:{url:"https://mydomain.adam:3333/api/v2/edgedevice/id/a9ee33b7-a5f7-4a5b-b1c3-fce73fbabd6f/newlogs" sentMsgCount:85 sentByteCount:176050 recvMsgCount:85 total_time_spent:1288}} zedcloud:{ifName:"eth1" success:5 lastSuccess:{seconds:1621261186 nanos:210610374} urlMetrics:{url:"https://mydomain.adam:3333/api/v2/edgedevice/id/a9ee33b7-a5f7-4a5b-b1c3-fce73fbabd6f/metrics" sentMsgCount:1 sentByteCount:438 recvMsgCount:1 total_time_spent:60} urlMetrics:{url:"https://mydomain.adam:3333/api/v2/edgedevice/id/a9ee33b7-a5f7-4a5b-b1c3-fce73fbabd6f/attest" sentMsgCount:1 sentByteCount:2 recvMsgCount:1 recvByteCount:123 total_time_spent:4} urlMetrics:{url:"https://mydomain.adam:3333/api/v2/edgedevice/id/a9ee33b7-a5f7-4a5b-b1c3-fce73fbabd6f/info" sentMsgCount:2 sentByteCount:6540 recvMsgCount:2 total_time_spent:12} urlMetrics:{url:"https://mydomain.adam:3333/api/v2/edgedevice/uuid" sentMsgCount:1 recvMsgCount:1 recvByteCount:10 total_time_spent:7}} disk:{mountPath:"/persist" total:7369 used:35 free:6941} disk:{mountPath:"/persist/vault/downloader"} disk:{disk:"sda4" readBytes:1 readCount:213 writeCount:25 total:1} disk:{mountPath:"/persist/log"} disk:{mountPath:"/persist/clear/volumes"} disk:{mountPath:"/persist/checkpoint"} disk:{disk:"sda2" readBytes:109 readCount:3678 total:300} disk:{mountPath:"/persist/containerd" used:1} disk:{mountPath:"/persist/certs"} disk:{mountPath:"/persist/status"} disk:{disk:"sda" readBytes:141 writeBytes:946 readCount:5308 writeCount:38181 total:8192} disk:{mountPath:"/persist/vault/verifier"} disk:{disk:"sda1" readBytes:6 readCount:503 total:36} disk:{disk:"sda9" readBytes:4 writeBytes:945 readCount:144 writeCount:37071 total:7553} disk:{disk:"sda3" readBytes:20 readCount:641 total:300} disk:{mountPath:"/" total:1964 free:1964} disk:{mountPath:"/config" total:1 free:1} disk:{mountPath:"/persist/tmp"} disk:{mountPath:"/persist/vault/volumes"} disk:{mountPath:"/persist/newlog"} cpuMetric:{upTime:{seconds:2289} total:33} runtimeStorageOverheadMB:35 systemServicesMemoryMB:{usedMem:476 availMem:3452 usedPercentage:12 availPercentage:88} cipher:{agent_name:"downloader" failure_count:4074837394752758774 last_failure:{seconds:1621261216 nanos:942838209} tc:{} tc:{error_code:CIPHER_ERROR_NOT_READY} tc:{error_code:CIPHER_ERROR_DECRYPT_FAILED} tc:{error_code:CIPHER_ERROR_UNMARSHAL_FAILED} tc:{error_code:CIPHER_ERROR_CLEARTEXT_FALLBACK} tc:{error_code:CIPHER_ERROR_MISSING_FALLBACK} tc:{error_code:CIPHER_ERROR_NO_CIPHER} tc:{error_code:CIPHER_ERROR_NO_DATA count:4074837394752758774}} acl:{} newlog:{failSentStartTime:{seconds:1621261165 nanos:962416566} currentUploadIntv:3 logfileTimeout:10 maxGzipFileSize:26968 avgGzipFileSize:2125 deviceMetrics:{numGzipBytesWrite:173710 numBytesWrite:2194978 numInputEvent:3578 numGzipFileRetry:81} appMetrics:{numGzipBytesWrite:4267 numBytesWrite:28357 numInputEvent:144 numGzipFileRetry:2} top10_input_sources:{key:"baseosmgr" value:2} top10_input_sources:{key:"domainmgr" value:2} top10_input_sources:{key:"downloader" value:13} top10_input_sources:{key:"kernel" value:5} top10_input_sources:{key:"nim" value:8} top10_input_sources:{key:"verifier" value:5} top10_input_sources:{key:"volumemgr" value:22} top10_input_sources:{key:"zedagent" value:14} top10_input_sources:{key:"zedbox" value:6} top10_input_sources:{key:"zedrouter" value:2}} zedbox:{numGoRoutines:439} last_received_config:{seconds:1621261555 nanos:513166958} last_processed_config:{seconds:1621261555 nanos:517204083}      Am: []  Nm: [networkID:"96ed0239-6ec3-4c50-88a8-650101ded47c" networkVersion:"1" instType:2 displayname:"pensive_lewin" networkStats:{rx:{} tx:{}}]   Vm: []
```

## Netstat

To view network statistic messages from EVE you can use the following command:

```bash
./eden netstat

Scans the ADAM flow messages for correspondence with regular expressions to show network flow statistics
(TCP and UDP flows with IP addresses, port numbers, counters, whether dropped or accepted)

Usage:
  eden netstat [field:regexp ...] [flags]

Flags:
  -f, --follow        Monitor changes in selected directory
  -h, --help          help for netstat
  -o, --out strings   Fields to print. Whole message if empty.
      --tail uint     Show only last N lines

Global Flags:
      --config string      Name of config (default "default")
  -v, --verbosity string   Log level (debug, info, warn, error, fatal, panic (default "info")
```

For example: `eden netstat --tail=1` will output something like:

```bash
{"devId":"a9ee33b7-a5f7-4a5b-b1c3-fce73fbabd6f","scope":{"uuid":"dbd53bf1-d7f7-4f7a-ac27-fc0621be50ba","localIntf":"bn1","netInstUUID":"96ed0239-6ec3-4c50-88a8-650101ded47c"},"flows":[{"flow":{"src":"10.11.12.2","srcPort":33678,"dest":"140.82.121.3","destPort":80,"protocol":6},"aclId":1,"startTime":{"seconds":1621261310,"nanos":907129900},"endTime":{"seconds":1621261430,"nanos":141507000},"txBytes":334,"txPkts":6,"rxBytes":288,"rxPkts":5,"action":2},{"flow":{"src":"10.11.12.2","srcPort":22,"dest":"192.168.31.137","destPort":40284,"protocol":6},"inbound":true,"aclId":2,"startTime":{"seconds":1621261299,"nanos":172136400},"endTime":{"seconds":1621261419,"nanos":141512000},"txBytes":4509,"txPkts":26,"rxBytes":4947,"rxPkts":28,"action":2},{"flow":{"src":"10.11.12.2","srcPort":22,"dest":"192.168.31.137","destPort":40496,"protocol":6},"inbound":true,"aclId":2,"startTime":{"seconds":1621261309,"nanos":947387600},"endTime":{"seconds":1621261430,"nanos":141514800},"txBytes":16245,"txPkts":131,"rxBytes":9195,"rxPkts":134,"action":2},{"flow":{"src":"10.11.12.2","srcPort":33784,"dest":"173.194.73.101","destPort":80,"protocol":6},"startTime":{"seconds":1621261312,"nanos":344697600},"endTime":{"seconds":1621261447,"nanos":141518300},"txBytes":300,"txPkts":5,"action":1},{"flow":{"src":"10.11.12.2","srcPort":22,"dest":"192.168.31.137","destPort":40512,"protocol":6},"inbound":true,"aclId":2,"startTime":{"seconds":1621261311,"nanos":168963000},"endTime":{"seconds":1621261462,"nanos":141524200},"txBytes":48369,"txPkts":236,"rxBytes":13475,"rxPkts":241,"action":2}],"dnsReqs":[{"hostName":"github.com","addrs":["140.82.121.3"],"requestTime":{"seconds":1621261310,"nanos":886307600}},{"hostName":"google.com","addrs":["173.194.73.101","173.194.73.100","173.194.73.139","173.194.73.113","173.194.73.102","173.194.73.138"],"requestTime":{"seconds":1621261312,"nanos":346228200}},{"hostName":"google.com","addrs":["2a00:1450:4010:c0d::71","2a00:1450:4010:c0d::64","2a00:1450:4010:c0d::65","2a00:1450:4010:c0d::8b"],"requestTime":{"seconds":1621261312,"nanos":346235100}}]}
```
