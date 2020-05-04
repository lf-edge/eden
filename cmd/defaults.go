package cmd

const (
	defaultDomain          = "mydomain.adam"
	defaultIP              = "192.168.0.1"
	defaultEVEIP           = "192.168.1.2"
	defaultUUID            = "1"
	defaultAdamTag         = "0.0.26"
	defaultEveTag          = "5.1.11"
	defaultEvePrefixInTar  = "bits"
	defaultEveRepo         = "https://github.com/lf-edge/eve.git"
	defaultBaseEveTag      = "5.1.10"
	defaultLinuxKitVersion = "v0.7"
	defaultImageTag        = "eden-alpine"
	defaultFileToSave      = "./test.tar"
	defaultImage           = "library/alpine"
	defaultRegistry        = "docker.io"
	defaultIsLocal         = false
	defaultQemuFileToSave  = "qemu.conf"
	defaultQemuCpus        = 4
	defaultQemuMemory      = 4096
	defaultEserverPort     = "8888"
)

var (
	defaultQemuHostFwd = map[string]string{"2222": "22"}
	cobraToViper       = map[string]string{
		"adam.dist":   "adam-dist",
		"adam.port":   "adam-port",
		"adam.domain": "domain",
		"adam.ip":     "ip",
		"adam.eve-ip": "eve-ip",
		"adam.force":  "adam-force",

		"eve.arch":        "eve-arch",
		"eve.os":          "eve-os",
		"eve.accel":       "eve-accel",
		"eve.hv":          "hv",
		"eve.serial":      "eve-serial",
		"eve.pid":         "eve-pid",
		"eve.log":         "eve-log",
		"eve.firmware":    "eve-firmware",
		"eve.repo":        "eve-repo",
		"eve.tag":         "eve-tag",
		"eve.base-tag":    "eve-base-tag",
		"eve.hostfwd":     "eve-hostfwd",
		"eve.dist":        "eve-dist",
		"eve.base-dist":   "eve-base-dist",
		"eve.qemu-config": "qemu-config",
		"eve.uuid":        "uuid",
		"eve.image-file":  "image-file",
		"eve.dtb-part":    "dtb-part",
		"eve.config-part": "config-part",

		"eden.images.dist":   "image-dist",
		"eden.images.docker": "docker-yml",
		"eden.images.vm":     "vm-yml",
		"eden.download":      "download",
		"eden.eserver.port":  "eserver-port",
		"eden.eserver.pid":   "eserver-pid",
		"eden.eserver.log":   "eserver-log",
		"eden.certs-dist":    "certs-dist",
		"eden.bin-dist":      "bin-dist",
		"eden.ssh-key":       "ssh-key",
	}
)
