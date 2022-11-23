package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/lf-edge/eden/pkg/openevec"

	"github.com/lf-edge/eden/pkg/defaults"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newEveCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var eveCmd = &cobra.Command{
		Use: "eve",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			viper_cfg, err := openevec.FromViper(*configName, *verbosity)
			if err != nil {
				return err
			}
			openevec.Merge(reflect.ValueOf(viper_cfg).Elem(), reflect.ValueOf(*cfg), cmd.Flags())
			*cfg = *viper_cfg
			return nil
		},
	}
	groups := CommandGroups{
		{
			Message: "Control commands",
			Commands: []*cobra.Command{
				newStartEveCmd(cfg),
				newStopEveCmd(cfg),
				newStatusEveCmd(cfg),
				newIpEveCmd(cfg),
				newSshEveCmd(cfg),
				newConsoleEveCmd(cfg),
				newOnboardEveCmd(cfg),
				newResetEveCmd(cfg),
				newVersionEveCmd(),
				newEpochEveCmd(cfg),
				newLinkEveCmd(cfg),
			},
		},
	}

	groups.AddTo(eveCmd)

	return eveCmd
}

func swtpmPidFile(cfg *openevec.EdenSetupArgs) string {
	if cfg.Eve.GcpvTPM {
		command := "swtpm"
		return filepath.Join(filepath.Join(filepath.Dir(cfg.Eve.ImageFile), command),
			fmt.Sprintf("%s.pid", command))
	}
	return ""
}

func newStartEveCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var vmName string

	var startEveCmd = &cobra.Command{
		Use:   "start",
		Short: "start eve",
		Long:  `Start eve.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.StartEve(vmName, cfg); err != nil {
				log.Fatal(err)
			}
		},
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	startEveCmd.Flags().StringVarP(&vmName, "vmname", "", defaults.DefaultVBoxVMName, "vbox vmname required to create vm")

	startEveCmd.Flags().StringVarP(&cfg.Eve.ImageFile, "image-file", "", "", "path for image drive (required)")
	startEveCmd.Flags().StringVarP(&cfg.Eve.Arch, "eve-arch", "", runtime.GOARCH, "arch of system")
	startEveCmd.Flags().StringVarP(&cfg.Eve.QemuOS, "eve-os", "", runtime.GOOS, "os to run on")
	startEveCmd.Flags().BoolVarP(&cfg.Eve.Accel, "eve-accel", "", cfg.Eve.Accel, "use acceleration")
	startEveCmd.Flags().StringVarP(&cfg.Eve.Serial, "eve-serial", "", cfg.Eve.Serial, "SMBIOS serial")
	startEveCmd.Flags().StringVarP(&cfg.Eve.QemuConfigPath, "qemu-config", "", filepath.Join(currentPath, defaults.DefaultDist, "qemu.conf"), "config file to use")
	startEveCmd.Flags().StringVarP(&cfg.Eve.Pid, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file for save EVE pid")
	startEveCmd.Flags().StringVarP(&cfg.Eve.Log, "eve-log", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.log"), "file for save EVE log")
	startEveCmd.Flags().BoolVarP(&cfg.Eve.QemuForeground, "foreground", "", false, "run in foreground")
	startEveCmd.Flags().IntVarP(&cfg.Eve.QemuConfig.MonitorPort, "qemu-monitor-port", "", defaults.DefaultQemuMonitorPort, "Port for access to QEMU monitor")
	startEveCmd.Flags().IntVarP(&cfg.Eve.QemuConfig.NetdevSocketPort, "qemu-netdev-socket-port", "", defaults.DefaultQemuNetdevSocketPort, "Base port for socket-based ethernet interfaces used in QEMU")
	startEveCmd.Flags().IntVarP(&cfg.Eve.TelnetPort, "eve-telnet-port", "", defaults.DefaultTelnetPort, "Port for telnet access")
	startEveCmd.Flags().IntVarP(&cfg.Eve.QemuCpus, "cpus", "", defaults.DefaultCpus, "vbox cpus")
	startEveCmd.Flags().IntVarP(&cfg.Eve.QemuMemory, "memory", "", defaults.DefaultMemory, "vbox memory size (MB)")
	startEveCmd.Flags().StringVarP(&cfg.Eve.TapInterface, "with-tap", "", "", "use tap interface in QEMU as the third")
	startEveCmd.Flags().IntVarP(&cfg.Eve.EthLoops, "with-eth-loops", "", 0, "add one or more ethernet loops (requires custom device model)")

	return startEveCmd
}

func newStopEveCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var vmName string

	var stopEveCmd = &cobra.Command{
		Use:   "stop",
		Short: "stop eve",
		Long:  `Stop eve.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.StopEve(vmName, cfg); err != nil {
				log.Fatal(err)
			}
		},
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	stopEveCmd.Flags().StringVarP(&cfg.Eve.Pid, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file for save EVE pid")
	stopEveCmd.Flags().StringVarP(&vmName, "vmname", "", defaults.DefaultVBoxVMName, "vbox vmname required to create vm")

	return stopEveCmd
}

func newVersionEveCmd() *cobra.Command {
	var versionEveCmd = &cobra.Command{
		Use:   "version",
		Short: "version of eve",
		Long:  `Version of eve.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.VersionEve(); err != nil {
				log.Fatal(err)
			}
		},
	}

	return versionEveCmd
}

func newStatusEveCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var vmName string

	var statusEveCmd = &cobra.Command{
		Use:   "status",
		Short: "status of eve",
		Long:  `Status of eve.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.StatusEve(vmName, cfg); err != nil {
				log.Fatal(err)
			}
		},
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	statusEveCmd.Flags().StringVarP(&cfg.Eve.Pid, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file for save EVE pid")
	statusEveCmd.Flags().StringVarP(&vmName, "vmname", "", defaults.DefaultVBoxVMName, "vbox vmname required to create vm")

	return statusEveCmd
}

func newIpEveCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var ipEveCmd = &cobra.Command{
		Use:   "ip",
		Short: "ip of eve",
		Long:  `Get IP of eve.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(openevec.GetEveIp("eth0", cfg))
		},
	}

	return ipEveCmd
}

func newConsoleEveCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var consoleEveCmd = &cobra.Command{
		Use:   "console",
		Short: "telnet into eve",
		Long:  `Telnet into eve.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.ConsoleEve(cfg); err != nil {
				log.Fatal(err)
			}
		},
	}

	consoleEveCmd.Flags().StringVarP(&cfg.Eve.Host, "eve-host", "", defaults.DefaultEVEHost, "IP of eve")
	consoleEveCmd.Flags().IntVarP(&cfg.Eve.TelnetPort, "eve-telnet-port", "", defaults.DefaultTelnetPort, "Port for telnet access")

	return consoleEveCmd
}

func newSshEveCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var sshEveCmd = &cobra.Command{
		Use:   "ssh [command]",
		Short: "ssh into eve",
		Long:  `SSH into eve.`,
		Run: func(cmd *cobra.Command, args []string) {
			commandToRun := ""
			if len(args) > 0 {
				commandToRun = strings.Join(args, " ")
			}
			if err := openevec.SshEve(commandToRun, cfg); err != nil {
				log.Fatal(err)
			}
		},
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	sshEveCmd.Flags().StringVarP(&cfg.Eden.SshKey, "ssh-key", "", filepath.Join(currentPath, defaults.DefaultCertsDist, "id_rsa"), "file to use for ssh access")
	sshEveCmd.Flags().StringVarP(&cfg.Eve.Host, "eve-host", "", defaults.DefaultEVEHost, "IP of eve")
	sshEveCmd.Flags().IntVarP(&cfg.Eve.SshPort, "eve-ssh-port", "", defaults.DefaultSSHPort, "Port for ssh access")

	return sshEveCmd
}

func newOnboardEveCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var onboardEveCmd = &cobra.Command{
		Use:   "onboard",
		Short: "OnBoard EVE in Adam",
		Long:  `Adding an EVE onboarding certificate to Adam and waiting for EVE to register.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.OnboardEve(cfg.Eve.CertsUUID); err != nil {
				log.Fatalf("Eve onboard failed: %s", err)
			}
		},
	}

	return onboardEveCmd
}

func newResetEveCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var resetEveCmd = &cobra.Command{
		Use:   "reset",
		Short: "Reset EVE to initial config",
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.ResetEve(cfg.Eve.CertsUUID); err != nil {
				log.Fatalf("Eve reset failed: %s", err)
			}
		},
	}

	return resetEveCmd
}

func newEpochEveCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var epochEveCmd = &cobra.Command{
		Use:   "epoch",
		Short: "Set new epoch of EVE",
		Run: func(cmd *cobra.Command, args []string) {
			if err := openevec.NewEpochEve(cfg.Runtime.EveConfigFromFile); err != nil {
				log.Fatalf("Eve new epoch failed: %s", err)
			}
		},
	}

	epochEveCmd.Flags().BoolVar(&cfg.Runtime.EveConfigFromFile, "use-config-file", false, "Load config of EVE from file")

	return epochEveCmd
}

/*
	startEveCmd.Flags().StringVarP(&eveImageFile, "image-file", "", "", "path for image drive (required)")
	startEveCmd.Flags().StringVarP(&qemuARCH, "eve-arch", "", runtime.GOARCH, "arch of system")
	startEveCmd.Flags().StringVarP(&qemuOS, "eve-os", "", runtime.GOOS, "os to run on")
	startEveCmd.Flags().BoolVarP(&qemuAccel, "eve-accel", "", true, "use acceleration")
	startEveCmd.Flags().StringVarP(&qemuSMBIOSSerial, "eve-serial", "", "", "SMBIOS serial")
	startEveCmd.Flags().StringVarP(&qemuConfigFile, "qemu-config", "", filepath.Join(currentPath, defaults.DefaultDist, "qemu.conf"), "config file to use")
	startEveCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file for save EVE pid")
	startEveCmd.Flags().StringVarP(&eveLogFile, "eve-log", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.log"), "file for save EVE log")
	startEveCmd.Flags().BoolVarP(&qemuForeground, "foreground", "", false, "run in foreground")
	startEveCmd.Flags().IntVarP(&qemuMonitorPort, "qemu-monitor-port", "", defaults.DefaultQemuMonitorPort, "Port for access to QEMU monitor")
	startEveCmd.Flags().IntVarP(&qemuNetdevSocketPort, "qemu-netdev-socket-port", "", defaults.DefaultQemuNetdevSocketPort, "Base port for socket-based ethernet interfaces used in QEMU")
	startEveCmd.Flags().IntVarP(&eveTelnetPort, "eve-telnet-port", "", defaults.DefaultTelnetPort, "Port for telnet access")
	startEveCmd.Flags().StringVarP(&vmName, "vmname", "", defaults.DefaultVBoxVMName, "vbox vmname required to create vm")
	startEveCmd.Flags().IntVarP(&cpus, "cpus", "", defaults.DefaultCpus, "vbox cpus")
	startEveCmd.Flags().IntVarP(&mem, "memory", "", defaults.DefaultMemory, "vbox memory size (MB)")
	startEveCmd.Flags().StringVarP(&tapInterface, "with-tap", "", "", "use tap interface in QEMU as the third")
	startEveCmd.Flags().StringVarP(&eveUsbNetConfFile, "eve-usbnetconf-file", "", "", "path to device network config (aka usb.json) applied in runtime using a USB stick")
	addSdnStartOpts(startEveCmd)
	stopEveCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file for save EVE pid")
	stopEveCmd.Flags().StringVarP(&vmName, "vmname", "", defaults.DefaultVBoxVMName, "vbox vmname required to create vm")
	addSdnPidOpt(stopEveCmd)
	statusEveCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file for save EVE pid")
	statusEveCmd.Flags().StringVarP(&vmName, "vmname", "", defaults.DefaultVBoxVMName, "vbox vmname required to create vm")
	addSdnPidOpt(statusEveCmd)
	addSdnPortOpts(statusEveCmd)
	sshEveCmd.Flags().StringVarP(&eveSSHKey, "ssh-key", "", filepath.Join(currentPath, defaults.DefaultCertsDist, "id_rsa"), "file to use for ssh access")
	sshEveCmd.Flags().StringVarP(&eveHost, "eve-host", "", defaults.DefaultEVEHost, "IP of eve")
	sshEveCmd.Flags().IntVarP(&eveSSHPort, "eve-ssh-port", "", defaults.DefaultSSHPort, "Port for ssh access")
	addSdnPortOpts(sshEveCmd)
	consoleEveCmd.Flags().StringVarP(&eveHost, "eve-host", "", defaults.DefaultEVEHost, "IP of eve")
	consoleEveCmd.Flags().IntVarP(&eveTelnetPort, "eve-telnet-port", "", defaults.DefaultTelnetPort, "Port for telnet access")
	epochEveCmd.Flags().BoolVar(&eveConfigFromFile, "use-config-file", false, "Load config of EVE from file")
	linkEveCmd.Flags().IntVarP(&qemuMonitorPort, "qemu-monitor-port", "", defaults.DefaultQemuMonitorPort, "Port for access to QEMU monitor")
	linkEveCmd.Flags().StringVarP(&vmName, "vmname", "", defaults.DefaultVBoxVMName, "name of the EVE VBox VM")
	linkEveCmd.Flags().StringVarP(&eveInterfaceName, "interface-name", "i", "", "EVE interface to get/change the link state of")
	addSdnPortOpts(linkEveCmd)
*/
func newLinkEveCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var eveInterfaceName string

	var linkEveCmd = &cobra.Command{
		Use:   "link up|down|status",
		Short: "manage EVE interface link state",
		Long:  `Manage EVE interface link state. Supported for QEMU and VirtualBox.`,
		Run: func(cmd *cobra.Command, args []string) {
			command := "status"
			if len(args) > 0 {
				command = args[0]
			}
			if err := openevec.NewLinkEve(command, eveInterfaceName, cfg); err != nil {
				log.Fatalf("Eve new link failed: %s", err)
			}
		},
	}

	linkEveCmd.Flags().IntVarP(&cfg.Eve.QemuConfig.MonitorPort, "qemu-monitor-port", "", defaults.DefaultQemuMonitorPort, "Port for access to QEMU monitor")
	linkEveCmd.Flags().StringVarP(&cfg.Runtime.VmName, "vmname", "", defaults.DefaultVBoxVMName, "name of the EVE VBox VM")
	linkEveCmd.Flags().StringVarP(&eveInterfaceName, "interface-name", "i", "", "EVE interface to get/change the link state of")

	return linkEveCmd
}
