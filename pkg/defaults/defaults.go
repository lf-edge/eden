package defaults

import (
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
)

//directories and files
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
	DefaultCurrentDirConfig = "eden-config.yml"  //file for search config in current directory
	DefaultContextFile      = "context.yml"      //file for saving current context inside DefaultEdenHomeDir
	DefaultContextDirectory = "contexts"         //directory for saving contexts inside DefaultEdenHomeDir
	DefaultQemuFileToSave   = "qemu.conf"        //qemu config file inside DefaultEdenHomeDir
	DefaultSSHKey           = "certs/id_rsa.pub" //file for save ssh key
	DefaultConfigHidden     = ".eden-config.yml" //file to save config get --all
	DefaultConfigSaved      = "config_saved.yml" //file to save config during 'eden setup'

	DefaultContext = "default" //default context name
)

//domains, ips, ports
const (
	DefaultDomain       = "mydomain.adam"
	DefaultIP           = "192.168.0.1"
	DefaultEVEIP        = "192.168.1.2"
	DefaultEserverPort  = 8888
	DefaultTelnetPort   = 7777
	DefaultSSHPort      = 2222
	DefaultEVEHost      = "127.0.0.1"
	DefaultRedisHost    = "localhost"
	DefaultRedisPort    = 6379
	DefaultAdamPort     = 3333
	DefaultRegistryPort = 5000

	//tags, versions, repos
	DefaultEVETag               = "0.0.0-master-2053a0b1" //DefaultEVETag tag for EVE image
	DefaultAdamTag              = "df3aa59442b25ecc9fe0c0032dc84eb0808125a4"
	DefaultRedisTag             = "6"
	DefaultRegistryTag          = "2.7"
	DefaultProcTag              = "1.2"
	DefaultImage                = "library/alpine"
	DefaultAdamContainerRef     = "lfedge/adam"
	DefaultRedisContainerRef    = "redis"
	DefaultRegistryContainerRef = "library/registry"
	DefaultProcContainerRef     = "itmoeve/eden-processing"
	DefaultEveRepo              = "https://github.com/lf-edge/eve.git"
	DefaultEveRegistry          = "lfedge"
	DefaultRegistry             = "docker.io"

	DefaultEServerTag          = "1.2"
	DefaultEServerContainerRef = "lfedge/eden-http-server"

	//DefaultRepeatCount is repeat count for requests
	DefaultRepeatCount = 20
	//DefaultRepeatTimeout is time wait for next attempt
	DefaultRepeatTimeout         = 5 * time.Second
	DefaultUUID                  = "1"
	DefaultFileToSave            = "./test.tar"
	DefaultIsLocal               = false
	DefaultEVEHV                 = "kvm"
	DefaultQemuCpus              = 4
	DefaultQemuMemory            = 4096
	DefaultEVESerial             = "31415926"
	NetDHCPID                    = "6822e35f-c1b8-43ca-b344-0bbc0ece8cf1"
	NetNoDHCPID                  = "6822e35f-c1b8-43ca-b344-0bbc0ece8cf2"
	NetWiFiID                    = "6822e35f-c1b8-43ca-b344-0bbc0ece8cf3"
	DefaultTestProg              = ""
	DefaultTestScenario          = ""
	DefaultRootFSVersionPattern  = `^(\d+\.*){2,3}.*-(xen|kvm|acrn|rpi|rpi-xen|rpi-kvm)-(amd64|arm64)$`
	DefaultControllerModePattern = `^(?P<Type>(file|proto|adam|zedcloud)):\/\/(?P<URL>.*)$`
	DefaultPodLinkPattern        = `^(?P<TYPE>(oci|docker|http[s]{0,1}|file)):\/\/(?P<TAG>[^:]+):*(?P<VERSION>.*)$`
	DefaultRedisContainerName    = "eden_redis"
	DefaultAdamContainerName     = "eden_adam"
	DefaultRegistryContainerName = "eden_registry"
	DefaultEServerContainerName  = "eden_eserver"
	DefaultDockerNetworkName     = "eden_network"
	DefaultLogLevelToPrint       = log.InfoLevel
	DefaultX509Country           = "RU"
	DefaultX509Company           = "Itmo"
	DefaultAppsLogsRedisPrefix   = "APPS_EVE_"
	DefaultLogsRedisPrefix       = "LOGS_EVE_"
	DefaultInfoRedisPrefix       = "INFO_EVE_"
	DefaultMetricsRedisPrefix    = "METRICS_EVE_"
	DefaultRequestsRedisPrefix   = "REQUESTS_EVE_"

	DefaultEveLogLevel  = "info"    //min level of logs saved in files on EVE device
	DefaultAdamLogLevel = "warning" //min level of logs sent from EVE to Adam

	DefaultQemuAccelDarwin = "-machine q35,accel=hvf -cpu kvm64,kvmclock=off "
	DefaultQemuAccelLinux  = "-machine q35,accel=kvm,dump-guest-core=off -cpu host,invtsc=on,kvmclock=off -machine kernel-irqchip=split -device intel-iommu,intremap=on,caching-mode=on,aw-bits=48 "

	DefaultAppSubnet = "10.1.0.0/24"

	DefaultEVEModel = "ZedVirtual-4G"

	DefaultRPIModel = "RPi4"

	DefaultGCPModel = "GCP"

	DefaultEVERemote = false

	DefaultEVEImageSize = 8192

	DefaultAppMem = 1024000
	DefaultAppCPU = 1

	DefaultDummyExpect = "docker://image"

	DefaultVolumeSize = 2 * 1024 * 1024 * 1024

	DefaultEmptyVolumeLinkDocker = "docker://hello-world"
	DefaultEmptyVolumeLinkQcow2  = "empty.qcow2"
	DefaultEmptyVolumeLinkRaw    = "empty.raw"

	//defaults for gcp

	DefaultGcpImageName   = "eden-gcp-test"
	DefaultGcpBucketName  = "eve-live"
	DefaultGcpProjectName = "lf-edge-eve"
	DefaultGcpZone        = "us-west1-a"
	DefaultGcpMachineType = "n1-standard-1"

	DefaultPerfEVELocation       = "/persist/perf.data"
	DefaultPerfScriptEVELocation = "/persist/perf.script.out"
)

var (
	//DefaultQemuHostFwd represents port forward for ssh
	DefaultQemuHostFwd = map[string]string{strconv.Itoa(DefaultSSHPort): "22"}
	//DefaultCobraToViper represents mapping values between cobra (cli) and viper (yml)
	DefaultCobraToViper = map[string]string{
		"redis.dist":  "redis-dist",
		"redis.tag":   "redis-tag",
		"redis.port":  "redis-port",
		"redis.force": "redis-force",

		"adam.dist":         "adam-dist",
		"adam.tag":          "adam-tag",
		"adam.port":         "adam-port",
		"adam.domain":       "domain",
		"adam.ip":           "ip",
		"adam.eve-ip":       "eve-ip",
		"adam.force":        "adam-force",
		"adam.v1":           "api-v1",
		"adam.redis.adam":   "adam-redis-url",
		"adam.remote.redis": "adam-redis",

		"registry.tag":  "registry-tag",
		"registry.port": "registry-port",
		"registry.dist": "registry-dist",

		"eve.arch":         "eve-arch",
		"eve.os":           "eve-os",
		"eve.accel":        "eve-accel",
		"eve.hv":           "eve-hv",
		"eve.serial":       "eve-serial",
		"eve.pid":          "eve-pid",
		"eve.log":          "eve-log",
		"eve.firmware":     "eve-firmware",
		"eve.repo":         "eve-repo",
		"eve.registry":     "eve-registry",
		"eve.tag":          "eve-tag",
		"eve.uefi-tag":     "eve-uefi-tag",
		"eve.hostfwd":      "eve-hostfwd",
		"eve.dist":         "eve-dist",
		"eve.base-dist":    "eve-base-dist",
		"eve.qemu-config":  "qemu-config",
		"eve.uuid":         "uuid",
		"eve.image-file":   "image-file",
		"eve.dtb-part":     "dtb-part",
		"eve.config-part":  "config-part",
		"eve.base-version": "os-version",
		"eve.devmodel":     "devmodel",
		"eve.telnet-port":  "eve-telnet-port",

		"eden.images.dist":   "image-dist",
		"eden.images.docker": "docker-yml",
		"eden.images.vm":     "vm-yml",
		"eden.download":      "download",
		"eden.eserver.ip":    "eserver-ip",
		"eden.eserver.port":  "eserver-port",
		"eden.eserver.tag":   "eserver-tag",
		"eden.eserver.force": "eserver-force",
		"eden.certs-dist":    "certs-dist",
		"eden.bin-dist":      "bin-dist",
		"eden.ssh-key":       "ssh-key",
		"eden.test-bin":      "prog",
		"eden.test-scenario": "scenario",

		"gcp.key": "key",
	}
)
