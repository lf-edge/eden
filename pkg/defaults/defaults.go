package defaults

import (
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

// directories and files
const (
	DefaultDist             = "dist"             //root directory
	DefaultImageDist        = "images"           //directory for images inside dist
	DefaultEserverDist      = ""                 //directory to mount eserver images
	DefaultRedisDist        = ""                 //directory for volume of redis inside dist
	DefaultRegistryDist     = ""                 //directory for volume of registry inside dist
	DefaultAdamDist         = ""                 //directory for volume of adam inside dist
	DefaultEVEDist          = "eve"              //directory for build EVE inside dist
	DefaultCertsDist        = "certs"            //directory for certs inside dist
	DefaultBinDist          = "bin"              //directory for binaries inside dist
	DefaultEdenHomeDir      = ".eden"            //directory inside HOME directory for configs
	DefaultBuildtoolsDir    = "build-tools"      //directory to store tools needed for building (e.g. linuxkit)
	DefaultCurrentDirConfig = "eden-config.yml"  //file for search config in current directory
	DefaultContextFile      = "context.yml"      //file for saving current context inside DefaultEdenHomeDir
	DefaultContextDirectory = "contexts"         //directory for saving contexts inside DefaultEdenHomeDir
	DefaultQemuFileToSave   = "qemu.conf"        //qemu config file inside DefaultEdenHomeDir
	DefaultSSHKey           = "certs/id_rsa.pub" //file for save ssh key
	DefaultConfigHidden     = ".eden-config.yml" //file to save config get --all
	DefaultConfigSaved      = "config_saved.yml" //file to save config during 'eden setup'
	DefaultSwtpmSockFile    = "swtpm-sock"       //file to communicate with swtpm
	DefaultAdditionalDisks  = 0                  //number of disks to use alongside with bootable one

	DefaultContext = "default" //default context name

	DefaultConfigEnv   = "EDEN_CONFIG"    //default env for set config
	DefaultTestArgsEnv = "EDEN_TEST_ARGS" //default env for test arguments
)

// domains, ips, ports
const (
	DefaultDomain               = "mydomain.adam"
	DefaultIP                   = "192.168.0.1"
	DefaultEVEIP                = "192.168.1.2"
	DefaultEserverPort          = 8888
	DefaultTelnetPort           = 17777
	DefaultQemuMonitorPort      = 7788
	DefaultQemuNetdevSocketPort = 7790
	DefaultSSHPort              = 2222
	DefaultEVEHost              = "127.0.0.1"
	DefaultRedisHost            = "localhost"
	DefaultRedisPort            = 6379
	DefaultAdamPort             = 3333
	DefaultRegistryPort         = 5050

	//tags, versions, repos
	DefaultEVETag               = "14.5.0-rc1" // DefaultEVETag tag for EVE image
	DefaultAdamTag              = "0.0.57"
	DefaultRedisTag             = "7"
	DefaultRegistryTag          = "2.7"
	DefaultProcTag              = "83cfe07"
	DefaultMkimageTag           = "8.5.0"
	DefaultSDNVersion           = "v1.0.0"
	DefaultImage                = "library/alpine"
	DefaultAdamContainerRef     = "lfedge/adam"
	DefaultRedisContainerRef    = "redis"
	DefaultRegistryContainerRef = "library/registry"
	DefaultProcContainerRef     = "lfedge/eden-processing"
	DefaultMkimageContainerRef  = "lfedge/eve-mkimage-raw-efi"
	DefaultEdenSDNContainerRef  = "lfedge/eden-sdn"
	DefaultEveRepo              = "https://github.com/lf-edge/eve.git"
	DefaultEveRegistry          = "lfedge/eve"
	DefaultRegistry             = "docker.io"

	DefaultSFTPUser      = "user"
	DefaultSFTPPassword  = "password"
	DefaultSFTPDirPrefix = "/eserver/run"

	DefaultEVEPlatform = "none"

	DefaultRedisPasswordFile = "redis.pass"

	DefaultEServerTag          = "4b71e2c"
	DefaultEServerContainerRef = "lfedge/eden-http-server"

	DefaultEClientTag          = "b1c1de6"
	DefaultEClientContainerRef = "lfedge/eden-eclient"

	//DefaultRepeatCount is repeat count for requests
	DefaultRepeatCount = 20
	//DefaultRepeatTimeout is time wait for next attempt
	DefaultRepeatTimeout         = 5 * time.Second
	DefaultUUID                  = "1"
	DefaultFileToSave            = "./test.tar"
	DefaultIsLocal               = false
	DefaultEVEHV                 = "kvm"
	DefaultCpus                  = 4
	DefaultMemory                = 8192
	DefaultEVESerial             = "31415926"
	NetDHCPID                    = "6822e35f-c1b8-43ca-b344-0bbc0ece8cf1"
	NetDHCPID2                   = "6822e35f-c1b8-43ca-b344-0bbc0ece8cf2"
	NetWiFiID                    = "6822e35f-c1b8-43ca-b344-0bbc0ece8cf3"
	NetSwitch                    = "6822e35f-c1b8-43ca-b344-0bbc0ece8cf4"
	DefaultTestProg              = "eden.escript.test"
	DefaultTestScenario          = ""
	DefaultRootFSVersionPattern  = `^.*-(xen|kvm|acrn|rpi|rpi-xen|rpi-kvm)-(amd64|arm64)$`
	DefaultControllerModePattern = `^(?P<Type>(file|proto|adam|zedcloud)):\/\/(?P<URL>.*)$`
	DefaultPodLinkPattern        = `^(?P<TYPE>(oci|docker|http[s]{0,1}|file|directory)):\/\/(?P<TAG>[^:]+):*(?P<VERSION>.*)$`
	DefaultRedisContainerName    = "eden_redis"
	DefaultAdamContainerName     = "eden_adam"
	DefaultRegistryContainerName = "eden_registry"
	DefaultEServerContainerName  = "eden_eserver"
	DefaultDockerNetworkName     = "eden_network"
	DefaultDockerNetIPv6Subnet   = "fd11:778b:03dd:1111::/64"
	DefaultLogLevelToPrint       = log.InfoLevel
	DefaultX509Country           = "RU"
	DefaultX509Company           = "lf-edge"
	DefaultAppsLogsRedisPrefix   = "APPS_EVE_"
	DefaultLogsRedisPrefix       = "LOGS_EVE_"
	DefaultInfoRedisPrefix       = "INFO_EVE_"
	DefaultMetricsRedisPrefix    = "METRICS_EVE_"
	DefaultRequestsRedisPrefix   = "REQUESTS_EVE_"
	DefaultFlowLogRedisPrefix    = "FLOW_MESSAGE_EVE_"

	DefaultEveLogLevel  = "info" // min level of logs saved in files on EVE device
	DefaultAdamLogLevel = "info" // min level of logs sent from EVE to Adam

	DefaultQemuAccelDarwin      = "-machine q35,accel=hvf -cpu kvm64,kvmclock=off "
	DefaultQemuAccelDarwinArm64 = "-machine virt,accel=hvf,usb=off,dump-guest-core=off -cpu host "
	DefaultQemuAccelLinuxAmd64  = "-machine q35,accel=kvm,dump-guest-core=off,kernel-irqchip=split -cpu host,invtsc=on,kvmclock=off -device intel-iommu,intremap=on,caching-mode=on,aw-bits=48 "
	DefaultQemuAmd64            = "-machine q35,smm=on --cpu SandyBridge "

	DefaultQemuAccelArm64 = "-machine virt,accel=kvm,usb=off,dump-guest-core=off -cpu host "
	DefaultQemuArm64      = "-machine virt,virtualization=true -cpu cortex-a57 "

	DefaultAppSubnet        = "10.11.12.0/24"
	DefaultHostOnlyNotation = "host-only-acl"

	DefaultQemuModel = "ZedVirtual-4G"

	DefaultRPIModel = "RPi4"

	DefaultGCPModel = "GCP"

	DefaultVBoxModel = "VBox"

	DefaultVBoxSubnet = "10.0.2.0/24"

	DefaultPortMapOffset = 10

	DefaultParallelsModel = "parallels"

	DefaultGeneralModel = "general"

	DefaultEVERemote = false

	DefaultEVEImageSize = 8192

	DefaultTPMEnabled = false

	DefaultAppMem = 1024000
	DefaultAppCPU = 1

	DefaultDummyExpect = "docker://image"

	DefaultVolumeSize = 200 * 1024 * 1024

	DefaultEmptyVolumeLinkDocker = "docker://hello-world"
	DefaultEmptyVolumeLinkQcow2  = "empty.qcow2"
	DefaultEmptyVolumeLinkRaw    = "empty.raw"
	DefaultEmptyVolumeLinkQcow   = "empty.qcow"
	DefaultEmptyVolumeLinkVMDK   = "empty.vmdk"
	DefaultEmptyVolumeLinkVHDX   = "empty.vhdx"

	//defaults for gcp

	DefaultGcpImageName    = "eden-gcp-test"
	DefaultGcpBucketName   = "eve-live"
	DefaultGcpProjectName  = "lf-edge-eve"
	DefaultGcpZone         = "us-west1-a"
	DefaultGcpMachineType  = "n1-standard-2" // 2 vCPU 7.5 GB RAM
	DefaultGcpRulePriority = 10

	//defaults for packet

	DefaultVMName            = "eden-packet-test"
	DefaultPacketProjectName = "Zededa"
	DefaultPacketZone        = "sjc1"
	DefaultPacketMachineType = "t1.small.x86"

	//default for VBox

	DefaultVBoxVMName = "eve_live"

	DefaultParallelsUUID = "{5fbaabe3-6958-40ff-92a7-860e329aab41}"

	DefaultPerfEVELocation       = "/persist/perf.data"
	DefaultPerfScriptEVELocation = "/persist/perf.script.out"
	DefaultHWEVELocation         = "/persist/lshw.out"

	//defaults for SDN
	DefaultSdnTelnetPort = 6623
	DefaultSdnSSHPort    = 6622
	DefaultSdnMgmtPort   = 6666
	DefaultSdnCpus       = 2
	DefaultSdnMemory     = 2048
	DefaultSdnIPv6Subnet = "fd59:9c46:bc86:2222::/64"
)

var (
	//DefaultQemuHostFwd represents port forward for ssh
	DefaultQemuHostFwd = map[string]string{strconv.Itoa(DefaultSSHPort): "22"}
)
