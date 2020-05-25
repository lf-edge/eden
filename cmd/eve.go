package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	qemuARCH         string
	qemuOS           string
	qemuAccel        bool
	qemuSMBIOSSerial string
	qemuConfigFile   string
	qemuForeground   bool
	eveSSHKey        string
	eveHost          string
	eveSSHPort       int
	eveTelnetPort    int
)

var eveCmd = &cobra.Command{
	Use: "eve",
}

var startEveCmd = &cobra.Command{
	Use:   "start",
	Short: "start eve",
	Long:  `Start eve.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			qemuARCH = viper.GetString("eve.arch")
			qemuOS = viper.GetString("eve.os")
			qemuAccel = viper.GetBool("eve.accel")
			qemuSMBIOSSerial = viper.GetString("eve.serial")
			qemuConfigFile = utils.ResolveAbsPath(viper.GetString("eve.qemu-config"))
			eveImageFile = utils.ResolveAbsPath(viper.GetString("eve.image-file"))
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
			eveLogFile = utils.ResolveAbsPath(viper.GetString("eve.log"))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		qemuCommand := ""
		qemuOptions := fmt.Sprintf("-display none -serial telnet:localhost:%d,server,nowait -nodefaults -no-user-config ", eveTelnetPort)
		if qemuSMBIOSSerial != "" {
			qemuOptions += fmt.Sprintf("-smbios type=1,serial=%s ", qemuSMBIOSSerial)
		}
		if qemuOS == "" {
			qemuOS = runtime.GOOS
		} else {
			qemuOS = strings.ToLower(qemuOS)
		}
		if qemuOS != "linux" && qemuOS != "darwin" {
			log.Fatalf("OS not supported: %s", qemuOS)
		}
		if qemuARCH == "" {
			qemuARCH = runtime.GOARCH
		} else {
			qemuARCH = strings.ToLower(qemuARCH)
		}
		switch qemuARCH {
		case "amd64":
			qemuCommand = "qemu-system-x86_64"
			if qemuAccel {
				if qemuOS == "darwin" {
					qemuOptions += "-M accel=hvf --cpu host "
				} else {
					qemuOptions += "-enable-kvm --cpu host "
				}
			} else {
				qemuOptions += "--cpu SandyBridge "
			}
		case "arm64":
			qemuCommand = "qemu-system-aarch64"
			qemuOptions += "-machine virt,gic_version=3 -machine virtualization=true -cpu cortex-a57 -machine type=virt "
		default:
			log.Fatalf("Arch not supported: %s", runtime.GOARCH)
		}
		qemuOptions += fmt.Sprintf("-drive file=%s,format=qcow2 ", eveImageFile)
		if qemuConfigFile != "" {
			qemuOptions += fmt.Sprintf("-readconfig %s ", qemuConfigFile)
		}
		log.Infof("Start EVE: %s %s", qemuCommand, qemuOptions)
		if qemuForeground {
			if err := utils.RunCommandForeground(qemuCommand, strings.Fields(qemuOptions)...); err != nil {
				log.Fatal(err)
			}
		} else {
			log.Infof("With pid: %s ; log: %s", evePidFile, eveLogFile)
			if err := utils.RunCommandNohup(qemuCommand, eveLogFile, evePidFile, strings.Fields(qemuOptions)...); err != nil {
				log.Fatal(err)
			}
		}
	},
}

var stopEveCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop eve",
	Long:  `Stop eve.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if err := utils.StopEVEQemu(evePidFile); err != nil {
			log.Errorf("cannot stop EVE: %s", err)
		}
	},
}

var statusEveCmd = &cobra.Command{
	Use:   "status",
	Short: "status of eve",
	Long:  `Status of eve.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		statusEVE, err := utils.StatusEVEQemu(evePidFile)
		if err != nil {
			log.Errorf("cannot obtain status of EVE: %s", err)
		} else {
			fmt.Printf("EVE status: %s\n", statusEVE)
		}
	},
}

var consoleEveCmd = &cobra.Command{
	Use:   "console",
	Short: "telnet into eve",
	Long:  `Telnet into eve.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Infof("Try to telnet %s:%d", eveHost, eveTelnetPort)
		if err := utils.RunCommandForeground("telnet", strings.Fields(fmt.Sprintf("%s %d", eveHost, eveTelnetPort))...); err != nil {
			log.Fatalf("telnet error: %s", err)
		}
	},
}

var sshEveCmd = &cobra.Command{
	Use:   "ssh",
	Short: "ssh into eve",
	Long:  `SSH into eve.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			eveSSHKey = utils.ResolveAbsPath(viper.GetString("eden.ssh-key"))
			extension := filepath.Ext(eveSSHKey)
			eveSSHKey = strings.TrimRight(eveSSHKey, extension)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat(eveSSHKey); !os.IsNotExist(err) {
			log.Infof("Try to SHH %s:%d with key %s", eveHost, eveSSHPort, eveSSHKey)
			if err := utils.RunCommandForeground("ssh", strings.Fields(fmt.Sprintf("-o ConnectTimeout=3 -oStrictHostKeyChecking=no -i %s -p %d root@%s", eveSSHKey, eveSSHPort, eveHost))...); err != nil {
				log.Fatalf("ssh error: %s", err)
			}
		} else {
			log.Fatalf("SSH key problem: %s", err)
		}
	},
}

func eveInit() {
	eveCmd.AddCommand(confChangerCmd)
	confChangerInit()
	eveCmd.AddCommand(downloaderCmd)
	downloaderInit()
	eveCmd.AddCommand(startEveCmd)
	eveCmd.AddCommand(stopEveCmd)
	eveCmd.AddCommand(statusEveCmd)
	eveCmd.AddCommand(sshEveCmd)
	eveCmd.AddCommand(consoleEveCmd)
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	startEveCmd.Flags().StringVarP(&eveImageFile, "image-file", "", "", "path for image drive (required)")
	startEveCmd.Flags().StringVarP(&qemuARCH, "eve-arch", "", runtime.GOARCH, "arch of system")
	startEveCmd.Flags().StringVarP(&qemuOS, "eve-os", "", runtime.GOOS, "os to run on")
	startEveCmd.Flags().BoolVarP(&qemuAccel, "eve-accel", "", true, "use acceleration")
	startEveCmd.Flags().StringVarP(&qemuSMBIOSSerial, "eve-serial", "", "", "SMBIOS serial")
	startEveCmd.Flags().StringVarP(&qemuConfigFile, "qemu-config", "", filepath.Join(currentPath, defaults.DefaultDist, "qemu.conf"), "config file to use")
	startEveCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file for save EVE pid")
	startEveCmd.Flags().StringVarP(&eveLogFile, "eve-log", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.log"), "file for save EVE log")
	startEveCmd.Flags().BoolVarP(&qemuForeground, "foreground", "", false, "run in foreground")
	startEveCmd.Flags().IntVarP(&eveTelnetPort, "eve-telnet-port", "", defaults.DefaultTelnetPort, "Port for telnet access")
	stopEveCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file for save EVE pid")
	statusEveCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file for save EVE pid")
	sshEveCmd.Flags().StringVarP(&eveSSHKey, "ssh-key", "", filepath.Join(currentPath, defaults.DefaultCertsDist, "id_rsa"), "file to use for ssh access")
	sshEveCmd.Flags().StringVarP(&eveHost, "eve-host", "", defaults.DefaultEVEHost, "IP of eve")
	sshEveCmd.Flags().IntVarP(&eveSSHPort, "eve-ssh-port", "", defaults.DefaultSSHPort, "Port for ssh access")
	consoleEveCmd.Flags().StringVarP(&eveHost, "eve-host", "", defaults.DefaultEVEHost, "IP of eve")
	consoleEveCmd.Flags().IntVarP(&eveTelnetPort, "eve-telnet-port", "", defaults.DefaultTelnetPort, "Port for telnet access")
	eveCmd.PersistentFlags().StringVar(&configFile, "config", "", "path to config file")
}
