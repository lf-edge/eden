package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/openevec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newEveCmd(configName, verbosity *string) *cobra.Command {
	cfg := &openevec.EdenSetupArgs{}
	var eveCmd = &cobra.Command{
		Use:               "eve",
		PersistentPreRunE: preRunViperLoadFunction(cfg, configName, verbosity),
	}
	groups := CommandGroups{
		{
			Message: "Control commands",
			Commands: []*cobra.Command{
				newStartEveCmd(cfg),
				newStopEveCmd(cfg),
				newStatusEveCmd(cfg),
				newIpEveCmd(),
				newSshEveCmd(cfg),
				newConsoleEveCmd(cfg),
				newOnboardEveCmd(cfg),
				newResetEveCmd(),
				newVersionEveCmd(),
				newEpochEveCmd(),
				newLinkEveCmd(cfg),
			},
		},
	}

	groups.AddTo(eveCmd)

	return eveCmd
}

func swtpmPidFile(cfg *openevec.EdenSetupArgs) string {
	if cfg.Eve.TPM {
		command := "swtpm"
		return filepath.Join(filepath.Join(filepath.Dir(cfg.Eve.ImageFile), command),
			fmt.Sprintf("%s.pid", command))
	}
	return ""
}

func newStartEveCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var vmName, tapInterface string

	var startEveCmd = &cobra.Command{
		Use:   "start",
		Short: "start eve",
		Long:  `Start eve.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.StartEve(vmName, tapInterface); err != nil {
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
	startEveCmd.Flags().IntVarP(&cfg.Eve.QemuConfig.MonitorPort, "qemu-monitor-port", "", defaults.DefaultQemuMonitorPort, "Port for access to QEMU monitor")
	startEveCmd.Flags().IntVarP(&cfg.Eve.QemuConfig.NetDevSocketPort, "qemu-netdev-socket-port", "", defaults.DefaultQemuNetdevSocketPort, "Base port for socket-based ethernet interfaces used in QEMU")
	startEveCmd.Flags().IntVarP(&cfg.Eve.TelnetPort, "eve-telnet-port", "", defaults.DefaultTelnetPort, "Port for telnet access")
	startEveCmd.Flags().IntVarP(&cfg.Eve.QemuCpus, "cpus", "", defaults.DefaultCpus, "vbox cpus")
	startEveCmd.Flags().IntVarP(&cfg.Eve.QemuMemory, "memory", "", defaults.DefaultMemory, "vbox memory size (MB)")
	startEveCmd.Flags().StringVarP(&tapInterface, "with-tap", "", "", "use tap interface in QEMU as the third")

	return startEveCmd
}

func newStopEveCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var vmName string

	var stopEveCmd = &cobra.Command{
		Use:   "stop",
		Short: "stop eve",
		Long:  `Stop eve.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.StopEve(vmName); err != nil {
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
			if err := openEVEC.VersionEve(); err != nil {
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
			if err := openEVEC.StatusEve(vmName); err != nil {
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

func newIpEveCmd() *cobra.Command {
	var ipEveCmd = &cobra.Command{
		Use:   "ip",
		Short: "ip of eve",
		Long:  `Get IP of eve.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(openEVEC.GetEveIP("eth0"))
		},
	}

	return ipEveCmd
}

func newConsoleEveCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var host string

	var consoleEveCmd = &cobra.Command{
		Use:   "console",
		Short: "telnet into eve",
		Long:  `Telnet into eve.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.ConsoleEve(host); err != nil {
				log.Fatal(err)
			}
		},
	}

	consoleEveCmd.Flags().StringVarP(&host, "eve-host", "", defaults.DefaultEVEHost, "IP of eve")
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
			if err := openEVEC.SSHEve(commandToRun); err != nil {
				log.Fatal(err)
			}
		},
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	sshEveCmd.Flags().StringVarP(&cfg.Eden.SSHKey, "ssh-key", "", filepath.Join(currentPath, defaults.DefaultCertsDist, "id_rsa"), "file to use for ssh access")

	return sshEveCmd
}

func newOnboardEveCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var onboardEveCmd = &cobra.Command{
		Use:   "onboard",
		Short: "OnBoard EVE in Adam",
		Long:  `Adding an EVE onboarding certificate to Adam and waiting for EVE to register.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.OnboardEve(cfg.Eve.CertsUUID); err != nil {
				log.Fatalf("Eve onboard failed: %s", err)
			}
		},
	}

	return onboardEveCmd
}

func newResetEveCmd() *cobra.Command {
	var resetEveCmd = &cobra.Command{
		Use:   "reset",
		Short: "Reset EVE to initial config",
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.ResetEve(); err != nil {
				log.Fatalf("EVE reset failed: %s", err)
			}
		},
	}

	return resetEveCmd
}

func newEpochEveCmd() *cobra.Command {
	var eveConfigFromFile bool

	var epochEveCmd = &cobra.Command{
		Use:   "epoch",
		Short: "Set new epoch of EVE",
		Run: func(cmd *cobra.Command, args []string) {
			if err := openEVEC.NewEpochEve(eveConfigFromFile); err != nil {
				log.Fatalf("EVE new epoch failed: %s", err)
			}
		},
	}

	epochEveCmd.Flags().BoolVar(&eveConfigFromFile, "use-config-file", false, "Load config of EVE from file")

	return epochEveCmd
}

func newLinkEveCmd(cfg *openevec.EdenSetupArgs) *cobra.Command {
	var eveInterfaceName, vmName string

	var linkEveCmd = &cobra.Command{
		Use:   "link up|down|status",
		Short: "manage EVE interface link state",
		Long:  `Manage EVE interface link state. Supported for QEMU and VirtualBox.`,
		Run: func(cmd *cobra.Command, args []string) {
			command := "status"
			if len(args) > 0 {
				command = args[0]
			}
			if err := openEVEC.NewLinkEve(command, eveInterfaceName, vmName); err != nil {
				log.Fatalf("EVE new link failed: %s", err)
			}
		},
	}

	linkEveCmd.Flags().IntVarP(&cfg.Eve.QemuConfig.MonitorPort, "qemu-monitor-port", "", defaults.DefaultQemuMonitorPort, "Port for access to QEMU monitor")
	linkEveCmd.Flags().StringVarP(&vmName, "vmname", "", defaults.DefaultVBoxVMName, "name of the EVE VBox VM")
	linkEveCmd.Flags().StringVarP(&eveInterfaceName, "interface-name", "i", "", "EVE interface to get/change the link state of")

	return linkEveCmd
}
