package defaults

import (
	log "github.com/sirupsen/logrus"
	"strconv"
	"time"
)

const (
	//directories and files
	DefaultDist             = "dist"             //root directory
	DefaultImageDist        = "images"           //directory for images inside dist
	DefaultRedisDist        = "redis"            //directory for volume of redis inside dist
	DefaultAdamDist         = "adam"             //directory for volume of adam inside dist
	DefaultEVEDist          = "eve"              //directory for build EVE inside dist
	DefaultCertsDist        = "certs"            //directory for certs inside dist
	DefaultBinDist          = "bin"              //directory for binaries inside dist
	DefaultEdenHomeDir      = ".eden"            //directory inside HOME directory for configs
	DefaultCurrentDirConfig = "config.yml"       //file for search config in current directory
	DefaultContextFile      = "context.yml"      //file for saving current context inside DefaultEdenHomeDir
	DefaultContextDirectory = "contexts"         //directory for saving contexts inside DefaultEdenHomeDir
	DefaultQemuFileToSave   = "qemu.conf"        //qemu config file inside DefaultEdenHomeDir
	DefaultFilename         = "rootfs.img"       //EVE`s rootfs file
	DefaultSSHKey           = "certs/id_rsa.pub" //file for save ssh key
	DefaultConfigHidden     = ".config.yml"      //file to save config get --all

	DefaultContext = "default" //default context name

	//domains, ips, ports
	DefaultDomain      = "mydomain.adam"
	DefaultIP          = "192.168.0.1"
	DefaultEVEIP       = "192.168.1.2"
	DefaultEserverPort = 8888
	DefaultTelnetPort  = 7777
	DefaultSSHPort     = 2222
	DefaultEVEHost     = "127.0.0.1"
	DefaultRedisHost   = "localhost"
	DefaultRedisPort   = 6379
	DefaultAdamPort    = 3333

	//tags, versions, repos
	DefaultEVETag            = "5ee6043906449f7fa3447c96fd38dc9a536c5693"        //DefaultEVETag tag for EVE image
	DefaultBaseOSTag         = "571d94a11fa19d79805a0465030175b7257d343b"        //DefaultBaseOSTag for uploadable rootfs
	DefaultBaseOSVersion     = "0.0.0-snapshot-master-93c2dbb2-2020-05-08.13.15" //DefaultBaseOSVersion for uploadable rootfs
	DefaultAdamTag           = "0.0.44"
	DefaultEveTag            = "5.1.11"
	DefaultRedisTag          = "6"
	DefaultLinuxKitVersion   = "v0.7"
	DefaultImage             = "library/alpine"
	DefaultAdamContainerRef  = "lfedge/adam"
	DefaultRedisContainerRef = "redis"
	DefaultImageTag          = "eden-alpine"
	DefaultEveRepo           = "https://github.com/lf-edge/eve.git"
	DefaultRegistry          = "docker.io"

	//DefaultRepeatCount is repeat count for requests
	DefaultRepeatCount = 20
	//DefaultRepeatTimeout is time wait for next attempt
	DefaultRepeatTimeout         = 5 * time.Second
	DefaultUUID                  = "1"
	DefaultEvePrefixInTar        = "bits"
	DefaultFileToSave            = "./test.tar"
	DefaultIsLocal               = false
	DefaultEVEHV                 = "kvm"
	DefaultQemuCpus              = 4
	DefaultQemuMemory            = 4096
	DefaultEVESerial             = "31415926"
	DefaultImageID               = "1ab8761b-5f89-4e0b-b757-4b87a9fa93ec"
	DefaultDataStoreID           = "eab8761b-5f89-4e0b-b757-4b87a9fa93ec"
	DefaultBaseID                = "22b8761b-5f89-4e0b-b757-4b87a9fa93ec"
	NetDHCPID                    = "6822e35f-c1b8-43ca-b344-0bbc0ece8cf1"
	NetNoDHCPID                  = "6822e35f-c1b8-43ca-b344-0bbc0ece8cf2"
	DefaultTestProg              = "eden.integration.test"
	DefaultTestScript            = "eden.integration.tests.txt"
	DefaultRootFSVersionPattern  = `^(\d+\.*){2,3}.*-(xen|kvm|acrn)-(amd64|arm64)$`
	DefaultControllerModePattern = `^(?P<Type>(file|proto|adam|zedcloud)):\/\/(?P<URL>.*)$`
	DefaultRedisContainerName    = "eden_redis"
	DefaultAdamContainerName     = "eden_adam"
	DefaultDockerNetworkName     = "eden_network"
	DefaultLogLevelToPrint       = log.InfoLevel
	DefaultX509Country           = "RU"
	DefaultX509Company           = "Itmo"
	DefaultLogsRedisPrefix       = "LOGS_EVE_"
	DefaultInfoRedisPrefix       = "INFO_EVE_"
)

var (
	DefaultQemuHostFwd  = map[string]string{strconv.Itoa(DefaultSSHPort): "22"}
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

		"eve.arch":         "eve-arch",
		"eve.os":           "eve-os",
		"eve.accel":        "eve-accel",
		"eve.hv":           "hv",
		"eve.serial":       "eve-serial",
		"eve.pid":          "eve-pid",
		"eve.log":          "eve-log",
		"eve.firmware":     "eve-firmware",
		"eve.repo":         "eve-repo",
		"eve.tag":          "eve-tag",
		"eve.base-tag":     "eve-base-tag",
		"eve.hostfwd":      "eve-hostfwd",
		"eve.dist":         "eve-dist",
		"eve.base-dist":    "eve-base-dist",
		"eve.qemu-config":  "qemu-config",
		"eve.uuid":         "uuid",
		"eve.image-file":   "image-file",
		"eve.dtb-part":     "dtb-part",
		"eve.config-part":  "config-part",
		"eve.base-version": "os-version",

		"eden.images.dist":   "image-dist",
		"eden.images.docker": "docker-yml",
		"eden.images.vm":     "vm-yml",
		"eden.download":      "download",
		"eden.eserver.ip":    "eserver-ip",
		"eden.eserver.port":  "eserver-port",
		"eden.eserver.pid":   "eserver-pid",
		"eden.eserver.log":   "eserver-log",
		"eden.certs-dist":    "certs-dist",
		"eden.bin-dist":      "bin-dist",
		"eden.ssh-key":       "ssh-key",
		"eden.test-bin":      "eden.integration.test",
		"test-script":        "eden.integration.tests.txt",
	}
)
