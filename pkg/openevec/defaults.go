package openevec

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
)

func GetDefaultConfig() *EdenSetupArgs {

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

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
