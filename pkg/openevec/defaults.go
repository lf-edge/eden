package openevec

import (
	"path/filepath"
	"runtime"

	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
)

func GetDefaultConfig(currentPath string) *EdenSetupArgs {

	defaultEdenConfig := &EdenSetupArgs{
		Eden: EdenConfig{
			Download: true,
			BinDir:   filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultBinDist),
			SSHKey:   filepath.Join(currentPath, defaults.DefaultCertsDist, "id_rsa"),

			EServer: EServerConfig{
				Port:  defaults.DefaultEserverPort,
				Force: false,
				Tag:   defaults.DefaultEServerTag,
			},
		},

		Adam: AdamConfig{
			Tag:        defaults.DefaultAdamTag,
			Port:       defaults.DefaultAdamPort,
			CertsIP:    defaults.DefaultIP,
			CertsEVEIP: defaults.DefaultEVEIP,

			Redis: RedisConfig{
				Tag:  defaults.DefaultRedisTag,
				Port: defaults.DefaultRedisPort,
			},
		},

		Eve: EveConfig{
			QemuConfig: QemuConfig{
				MonitorPort:      defaults.DefaultQemuMonitorPort,
				NetDevSocketPort: defaults.DefaultQemuNetdevSocketPort,
			},
			CertsUUID:      defaults.DefaultUUID,
			Dist:           filepath.Join(currentPath, defaults.DefaultDist, defaults.DefaultEVEDist),
			Repo:           defaults.DefaultEveRepo,
			Registry:       defaults.DefaultEveRegistry,
			Tag:            defaults.DefaultEVETag,
			UefiTag:        defaults.DefaultEVETag,
			HV:             defaults.DefaultEVEHV,
			Arch:           runtime.GOARCH,
			HostFwd:        defaults.DefaultQemuHostFwd,
			QemuFileToSave: defaults.DefaultQemuFileToSave,
			QemuCpus:       defaults.DefaultCpus,
			QemuMemory:     defaults.DefaultMemory,
			ImageSizeMB:    defaults.DefaultEVEImageSize,
			DevModel:       defaults.DefaultQemuModel,
			Serial:         defaults.DefaultEVESerial,
			Pid:            filepath.Join(currentPath, defaults.DefaultDist),
			Log:            filepath.Join(currentPath, defaults.DefaultDist),
			TelnetPort:     defaults.DefaultTelnetPort,
			TPM:            defaults.DefaultTPMEnabled,
		},

		Redis: RedisConfig{
			Tag:  defaults.DefaultRedisTag,
			Port: defaults.DefaultRedisPort,
		},

		Registry: RegistryConfig{
			Tag:  defaults.DefaultRegistryTag,
			Port: defaults.DefaultRegistryPort,
		},

		Sdn: SdnConfig{
			RAM:            defaults.DefaultSdnMemory,
			CPU:            defaults.DefaultSdnCpus,
			ConsoleLogFile: filepath.Join(currentPath, defaults.DefaultDist, "sdn-console.log"),
			Disable:        false,
			TelnetPort:     defaults.DefaultSdnTelnetPort,
			MgmtPort:       defaults.DefaultSdnMgmtPort,
			PidFile:        filepath.Join(currentPath, defaults.DefaultDist, "sdn.pid"),
			SSHPort:        defaults.DefaultSdnSSHPort,
		},

		ConfigName: defaults.DefaultContext,
		ConfigFile: utils.GetConfig(defaults.DefaultContext),
	}

	return defaultEdenConfig
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
