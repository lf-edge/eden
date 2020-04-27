package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	qemuLogFile      string
	qemuPidFile      string
)

var qemuRunCmd = &cobra.Command{
	Use:   "qemurun",
	Short: "run qemu-system with eve",
	Long:  `Run qemu-system with eve.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(config)
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
		qemuOptions := "-display none -serial mon:stdio -nodefaults -no-user-config "
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
		if qemuForeground {
			if err := utils.RunCommandForeground(qemuCommand, strings.Fields(qemuOptions)...); err != nil {
				log.Fatal(err)
			}
		} else {
			if err := utils.RunCommandNohup(qemuCommand, qemuLogFile, qemuPidFile, strings.Fields(qemuOptions)...); err != nil {
				log.Fatal(err)
			}
		}
	},
}

func qemuRunInit() {
	qemuRunCmd.Flags().StringVarP(&qemuARCH, "eve-arch", "", "", "arch of system")
	qemuRunCmd.Flags().StringVarP(&qemuOS, "eve-os", "", "", "os to run on")
	qemuRunCmd.Flags().BoolVarP(&qemuAccel, "eve-accel", "", true, "use acceleration")
	qemuRunCmd.Flags().StringVarP(&qemuSMBIOSSerial, "eve-serial", "", "", "SMBIOS serial")
	qemuRunCmd.Flags().StringVarP(&qemuConfigFile, "qemu-config", "", "", "config file to use")
	qemuRunCmd.Flags().BoolVarP(&qemuForeground, "foreground", "", false, "run in foreground")
	qemuRunCmd.Flags().StringVarP(&qemuLogFile, "eve-log", "", "", "file to save logs (for background variant)")
	qemuRunCmd.Flags().StringVarP(&qemuPidFile, "eve-pid", "", "", "file to save pid of qemu (for background variant)")
	qemuRunCmd.Flags().StringVarP(&eveImageFile, "image-file", "", "", "path for image drive (required)")
}
