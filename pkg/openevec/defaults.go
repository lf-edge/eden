package openevec

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	uuid "github.com/satori/go.uuid"
)

type ConfigOption func(*EdenSetupArgsBuilder)

// Builder to propagate error
type EdenSetupArgsBuilder struct {
	Args *EdenSetupArgs
	Err  []error
}

func GetDefaultConfig(projectRootPath string) *EdenSetupArgsBuilder {
	res := &EdenSetupArgsBuilder{nil, make([]error, 0)}
	ipv4, ipv6, err := utils.GetIPForDockerAccess()
	if err != nil {
		res.Err = append(res.Err, err)
		return res
	}
	var ip string
	if ipv4 != nil {
		ip = ipv4.String()
	} else {
		ip = ipv6.String()
	}

	edenDir, err := utils.DefaultEdenDir()
	if err != nil {
		res.Err = append(res.Err, err)
		return res
	}

	id, err := uuid.NewV4()
	if err != nil {
		res.Err = append(res.Err, err)
		return res
	}

	imageDist := filepath.Join(projectRootPath, defaults.DefaultDist, fmt.Sprintf("%s-%s", defaults.DefaultContext, defaults.DefaultImageDist))
	certsDist := filepath.Join(projectRootPath, defaults.DefaultDist, fmt.Sprintf("%s-%s", defaults.DefaultContext, defaults.DefaultCertsDist))

	firmware := []string{filepath.Join(imageDist, "eve", "OVMF.fd")}
	if runtime.GOARCH == "amd64" {
		firmware = []string{
			filepath.Join(imageDist, "eve", "firmware", "OVMF_CODE.fd"),
			filepath.Join(imageDist, "eve", "firmware", "OVMF_VARS.fd")}
	}

	defaultEdenConfig := &EdenSetupArgs{
		Eden: EdenConfig{
			Root:         filepath.Join(projectRootPath, defaults.DefaultDist),
			Tests:        filepath.Join(projectRootPath, defaults.DefaultDist, "tests"),
			Download:     true,
			BinDir:       defaults.DefaultBinDist,
			SSHKey:       filepath.Join(projectRootPath, defaults.DefaultDist, fmt.Sprintf("%s-%s", defaults.DefaultContext, defaults.DefaultSSHKey)),
			CertsDir:     certsDist,
			TestBin:      defaults.DefaultTestProg,
			EdenBin:      "eden",
			TestScenario: defaults.DefaultTestScenario,
			EnableIPv6:   false,
			IPv6Subnet:   defaults.DefaultDockerNetIPv6Subnet,

			Images: ImagesConfig{
				EServerImageDist: defaults.DefaultEserverDist,
			},

			EServer: EServerConfig{
				IP:    ip,
				EVEIP: defaults.DefaultDomain,

				Port:  defaults.DefaultEserverPort,
				Force: true,
				Tag:   defaults.DefaultEServerTag,
			},

			EClient: EClientConfig{
				Tag:   defaults.DefaultEClientTag,
				Image: defaults.DefaultEClientContainerRef,
			},
		},

		Adam: AdamConfig{
			Tag:         defaults.DefaultAdamTag,
			Port:        defaults.DefaultAdamPort,
			Dist:        defaults.DefaultAdamDist,
			CertsDomain: defaults.DefaultDomain,
			CertsIP:     ip,
			CertsEVEIP:  ip,
			Force:       true,
			CA:          filepath.Join(fmt.Sprintf("%s-%s", defaults.DefaultContext, defaults.DefaultCertsDist), "root-certificate.pem"),
			APIv1:       false,

			Redis: RedisConfig{
				RemoteURL: fmt.Sprintf("%s:%d", defaults.DefaultRedisContainerName, defaults.DefaultRedisPort),
				Tag:       defaults.DefaultRedisTag,
				Port:      defaults.DefaultRedisPort,
				Eden:      net.JoinHostPort(ip, fmt.Sprintf("%d", defaults.DefaultRedisPort)),
			},

			Remote: RemoteConfig{
				Enabled: true,
				Redis:   true,
			},

			Caching: CachingConfig{
				Enabled: false,
				Redis:   false,
				Prefix:  "cache",
			},
		},

		Eve: EveConfig{
			Name:         strings.ToLower(defaults.DefaultContext),
			DevModel:     defaults.DefaultQemuModel,
			ModelFile:    "",
			Arch:         runtime.GOARCH,
			QemuOS:       runtime.GOOS,
			Accel:        true,
			HV:           defaults.DefaultEVEHV,
			CertsUUID:    id.String(),
			Cert:         filepath.Join(certsDist, "onboard.cert.pem"),
			DeviceCert:   filepath.Join(certsDist, "device.cert.pem"),
			QemuFirmware: firmware,
			Dist:         fmt.Sprintf("%s-%s", defaults.DefaultContext, defaults.DefaultEVEDist),
			Repo:         defaults.DefaultEveRepo,
			Registry:     defaults.DefaultEveRegistry,
			Tag:          defaults.DefaultEVETag,
			UefiTag:      defaults.DefaultEVETag,
			HostFwd: map[string]string{
				strconv.Itoa(defaults.DefaultSSHPort): "22",
				"2223":                                "2223",
				"2224":                                "2224",
				"5911":                                "5901",
				"5912":                                "5902",
				"8027":                                "8027",
				"8028":                                "8028",
				"8029":                                "8029",
				"8030":                                "8030",
				"8031":                                "8031",
			},
			QemuFileToSave: filepath.Join(edenDir, fmt.Sprintf("%s-%s", defaults.DefaultContext, defaults.DefaultQemuFileToSave)),
			QemuCpus:       defaults.DefaultCpus,
			QemuMemory:     defaults.DefaultMemory,
			ImageSizeMB:    defaults.DefaultEVEImageSize,
			Serial:         defaults.DefaultEVESerial,
			Pid:            filepath.Join(projectRootPath, defaults.DefaultDist, fmt.Sprintf("%s-eve.pid", strings.ToLower(defaults.DefaultContext))),
			Log:            filepath.Join(projectRootPath, defaults.DefaultDist, fmt.Sprintf("%s-eve.log", strings.ToLower(defaults.DefaultContext))),
			TelnetPort:     defaults.DefaultTelnetPort,
			TPM:            defaults.DefaultTPMEnabled,
			ImageFile:      filepath.Join(imageDist, "eve", "live.img"),
			QemuDTBPath:    "",
			QemuConfigPath: certsDist,
			Remote:         defaults.DefaultEVERemote,
			RemoteAddr:     defaults.DefaultEVEHost,
			LogLevel:       defaults.DefaultEveLogLevel,
			RemoteLogLevel: defaults.DefaultRemoteLogLevel,
			Ssid:           "",
			Disks:          defaults.DefaultAdditionalDisks,
			BootstrapFile:  "",
			UsbNetConfFile: "",
			Platform:       "none",

			CustomInstaller: CustomInstallerConfig{
				Path:   "",
				Format: "",
			},

			QemuConfig: QemuConfig{
				MonitorPort:      defaults.DefaultQemuMonitorPort,
				NetDevSocketPort: defaults.DefaultQemuNetdevSocketPort,
			},
		},

		Redis: RedisConfig{
			Tag:  defaults.DefaultRedisTag,
			Port: defaults.DefaultRedisPort,
			Dist: defaults.DefaultRedisDist,
		},

		Registry: RegistryConfig{
			Tag:  defaults.DefaultRegistryTag,
			Port: defaults.DefaultRegistryPort,
			IP:   ip,
			Dist: defaults.DefaultRegistryDist,
		},

		Sdn: SdnConfig{
			Version:        defaults.DefaultSDNVersion,
			RAM:            defaults.DefaultSdnMemory,
			CPU:            defaults.DefaultSdnCpus,
			ConsoleLogFile: filepath.Join(projectRootPath, defaults.DefaultDist, "sdn-console.log"),
			Disable:        true,
			TelnetPort:     defaults.DefaultSdnTelnetPort,
			MgmtPort:       defaults.DefaultSdnMgmtPort,
			PidFile:        filepath.Join(projectRootPath, defaults.DefaultDist, "sdn.pid"),
			SSHPort:        defaults.DefaultSdnSSHPort,
			SourceDir:      filepath.Join(projectRootPath, "sdn"),
			ConfigDir:      filepath.Join(edenDir, fmt.Sprintf("%s-sdn", "default")),
			ImageFile:      filepath.Join(imageDist, "eden", "eden-sdn.qcow2"),
			NetModelFile:   "",
			EnableIPv6:     false,
			IPv6Subnet:     defaults.DefaultSdnIPv6Subnet,
		},

		Gcp: GcpConfig{
			Key: "",
		},

		Packet: PacketConfig{
			Key: "",
		},

		ConfigName: defaults.DefaultContext,
		ConfigFile: utils.GetConfig(defaults.DefaultContext),
		EdenDir:    edenDir,
	}

	res.Args = defaultEdenConfig

	return res
}

func hasVirtSupport() (bool, error) {
	cmd := exec.Command("lscpu")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return false, fmt.Errorf("Failed to run lscpu: %v", err)
	}
	return strings.Contains(out.String(), "vmx") || strings.Contains(out.String(), "svm"), nil
}

// Enabling Acclelerator requires you to specify firmware
func WithAccelerator(enabled bool, firmware []string) ConfigOption {
	return func(builder *EdenSetupArgsBuilder) {
		if enabled {
			virtSupport, err := hasVirtSupport()
			if err != nil {
				builder.Err = append(builder.Err, err)
			} else if !virtSupport {
				builder.Err = append(builder.Err, fmt.Errorf("Missing required HW-assisted virtualization support"))
			}
			builder.Args.Eve.Accel = true
		} else {
			builder.Args.Eve.Accel = false
			if len(firmware) > 0 {
				builder.Args.Eve.QemuFirmware = firmware
			}
		}
	}
}

// ParseDockerImage parses a Docker image reference into registry and tag.
// If no registry is given (e.g. "eve" or "eve:1.0"), it defaults to defaults.DefaultEveRegistry.
// If no tag is provided, it defaults to "latest".
func ParseDockerImage(image string) (registry, tag string) {
	image = strings.TrimSpace(image)

	if image == "" {
		return defaults.DefaultEveRegistry, defaults.DefaultEVETag
	}

	// Split by colon to separate tag (but handle registry:port/image:tag correctly)
	lastColon := strings.LastIndex(image, ":")
	lastSlash := strings.LastIndex(image, "/")

	if lastColon > lastSlash {
		// There's a tag
		registry = image[:lastColon]
		tag = image[lastColon+1:]
	} else {
		// No tag provided
		registry = image
		tag = "latest"
	}

	// If the image doesn’t contain a slash, assume it’s just a name and prepend the default registry
	if !strings.Contains(registry, "/") {
		registry = defaults.DefaultEveRegistry
	}

	return
}

func WithEVEImage(image string) ConfigOption {
	return func(builder *EdenSetupArgsBuilder) {
		registry, tag := ParseDockerImage(image)
		builder.Args.Eve.Registry = registry
		builder.Args.Eve.Tag = tag
	}
}

func WithLogLevel(level string) ConfigOption {
	return func(builder *EdenSetupArgsBuilder) {
		if level == "" {
			builder.Args.Eve.LogLevel = defaults.DefaultEveLogLevel
		} else {
			builder.Args.Eve.LogLevel = level
		}
	}
}

func WithFilesystem(fs string) ConfigOption {
	return func(builder *EdenSetupArgsBuilder) {
		switch fs {
		case "zfs":
			builder.Args.Eve.Disks = 4
			builder.Args.Eve.ImageSizeMB = 4096
			builder.Args.Eve.GrubOptions = []string{
				"set_global dom0_extra_args \"$dom0_extra_args eve_install_zfs_with_raid_level \"",
			}
		default:
			// assuming ext4, no need to setup anything extra
		}
	}
}

func WithHypervisor(hv string) ConfigOption {
	return func(builder *EdenSetupArgsBuilder) {
		if hv == "" {
			builder.Args.Eve.HV = defaults.DefaultEVEHV
		} else {
			builder.Args.Eve.HV = hv
		}
	}
}

func GetDefaultPodConfig() *PodConfig {
	dpc := &PodConfig{
		AppMemory:         humanize.Bytes(defaults.DefaultAppMem * 1024),
		DiskSize:          humanize.Bytes(0),
		VolumeType:        "qcow2",
		AppCpus:           defaults.DefaultAppCPU,
		ACLOnlyHost:       false,
		NoHyper:           false,
		Registry:          "remote",
		DirectLoad:        true,
		SftpLoad:          false,
		VolumeSize:        humanize.IBytes(defaults.DefaultVolumeSize),
		OpenStackMetadata: false,
		PinCpus:           false,
	}

	return dpc
}
