package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lf-edge/eden/pkg/eden"

	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/info"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	qemuARCH          string
	qemuOS            string
	qemuAccel         bool
	qemuSMBIOSSerial  string
	qemuConfigFile    string
	qemuForeground    bool
	eveSSHKey         string
	eveHost           string
	eveSSHPort        int
	eveTelnetPort     int
	eveRemoteAddr     string
	eveConfigFromFile bool
)

var eveCmd = &cobra.Command{
	Use: "eve",
}

var startEveCmd = &cobra.Command{
	Use:   "start",
	Short: "start eve",
	Long:  `Start eve.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			qemuARCH = viper.GetString("eve.arch")
			qemuOS = viper.GetString("eve.os")
			hostFwd = viper.GetStringMapString("eve.hostfwd")
			qemuAccel = viper.GetBool("eve.accel")
			qemuSMBIOSSerial = viper.GetString("eve.serial")
			qemuConfigFile = utils.ResolveAbsPath(viper.GetString("eve.qemu-config"))
			eveImageFile = utils.ResolveAbsPath(viper.GetString("eve.image-file"))
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
			eveLogFile = utils.ResolveAbsPath(viper.GetString("eve.log"))
			eveRemote = viper.GetBool("eve.remote")
			eveTelnetPort = viper.GetInt("eve.telnet-port")
			devModel = viper.GetString("eve.devmodel")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if eveRemote {
			return
		}

		if devModel == defaults.DefaultVBoxModel {
			if err := eden.StartEVEVBox(vmName, eveImageFile, cpus, mem, hostFwd); err != nil {
				log.Errorf("cannot start eve: %s", err)
			} else {
				log.Infof("EVE is starting in Virtual Box")
			}
		} else if devModel == defaults.DefaultParallelsModel {
			if err := eden.StartEVEParallels(vmName, eveImageFile, cpus, mem, hostFwd); err != nil {
				log.Errorf("cannot start eve: %s", err)
			} else {
				log.Infof("EVE is starting in Parallels")
			}
		} else {
			if err := eden.StartEVEQemu(qemuARCH, qemuOS, eveImageFile, qemuSMBIOSSerial, eveTelnetPort, hostFwd, qemuAccel, qemuConfigFile, eveLogFile, evePidFile, false); err != nil {
				log.Errorf("cannot start eve: %s", err)
			} else {
				log.Infof("EVE is starting")
			}
		}
	},
}

var stopEveCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop eve",
	Long:  `Stop eve.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
			eveRemote = viper.GetBool("eve.remote")
			devModel = viper.GetString("eve.devmodel")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if eveRemote {
			log.Debug("Cannot stop remote EVE")
			return
		}
		if devModel == defaults.DefaultVBoxModel {
			if err := eden.StopEVEVBox(vmName); err != nil {
				log.Errorf("cannot stop eve: %s", err)
			} else {
				log.Infof("EVE is stopping in Virtual Box")
			}
		} else if devModel == defaults.DefaultParallelsModel {
			if err := eden.StopEVEParallels(vmName); err != nil {
				log.Errorf("cannot stop eve: %s", err)
			} else {
				log.Infof("EVE is stopping in Virtual Box")
			}
		} else {
			if err := eden.StopEVEQemu(evePidFile); err != nil {
				log.Errorf("cannot stop eve: %s", err)
			} else {
				log.Infof("EVE is stopping")
			}
		}
	},
}

var versionEveCmd = &cobra.Command{
	Use:   "version",
	Short: "version of eve",
	Long:  `Version of eve.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
			eveRemote = viper.GetBool("eve.remote")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		log.Debugf("Will try to obtain info from ADAM")
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Debugf("getControllerAndDev: %s", err)
			fmt.Println("EVE status: undefined (no onboarded EVE)")
		} else {
			var lastDInfo *info.ZInfoMsg
			var handleInfo = func(im *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface) bool {
				if im.GetZtype() == info.ZInfoTypes_ZiDevice {
					lastDInfo = im
				}
				return false
			}
			if err = ctrl.InfoLastCallback(dev.GetID(), map[string]string{"devId": dev.GetID().String()}, handleInfo); err != nil {
				log.Fatalf("Fail in get InfoLastCallback: %s", err)
			}
			if lastDInfo == nil {
				log.Info("no info messages")
			} else {
				fmt.Println(lastDInfo.GetDinfo().SwList[0].ShortVersion)
			}
		}
	},
}

var statusEveCmd = &cobra.Command{
	Use:   "status",
	Short: "status of eve",
	Long:  `Status of eve.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
			eveRemote = viper.GetBool("eve.remote")
			devModel = viper.GetString("eve.devmodel")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		statusAdam, err := eden.StatusAdam()
		if err == nil && statusAdam != "container doesn't exist" {
			eveStatusRemote()
		}
		if !eveRemote {
			if devModel == defaults.DefaultVBoxModel {
				eveStatusVBox()
			} else if devModel == defaults.DefaultParallelsModel {
				eveStatusParallels()
			} else {
				eveStatusQEMU()
			}
		}
		if err == nil && statusAdam != "container doesn't exist" {
			eveRequestsAdam()
		}
	},
}

func getEVEIP() string {
	if !eveRemote && runtime.GOOS == "darwin" {
		return "127.0.0.1"
	}
	if ip, err := eveLastRequests(); err == nil && ip != "" {
		return ip
	}
	return ""
}

var ipEveCmd = &cobra.Command{
	Use:   "ip",
	Short: "ip of eve",
	Long:  `Get IP of eve.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			eveRemote = viper.GetBool("eve.remote")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(getEVEIP())
	},
}

var consoleEveCmd = &cobra.Command{
	Use:   "console",
	Short: "telnet into eve",
	Long:  `Telnet into eve.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		eveRemote = viper.GetBool("eve.remote")
		eveTelnetPort = viper.GetInt("eve.telnet-port")
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if eveRemote {
			log.Info("Cannot telnet to remote EVE")
			return
		}
		log.Infof("Try to telnet %s:%d", eveHost, eveTelnetPort)
		if err := utils.RunCommandForeground("telnet", strings.Fields(fmt.Sprintf("%s %d", eveHost, eveTelnetPort))...); err != nil {
			log.Fatalf("telnet error: %s", err)
		}
	},
}

var sshEveCmd = &cobra.Command{
	Use:     "ssh [command]",
	Short:   "ssh into eve",
	Long:    `SSH into eve.`,
	PreRunE: initSSHVariables,
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat(eveSSHKey); !os.IsNotExist(err) {
			changer := &adamChanger{}
			ctrl, dev, err := changer.getControllerAndDev()
			if err != nil {
				log.Fatalf("Cannot get controller or dev, please start them and onboard: %s", err)
			}
			b, err := ioutil.ReadFile(ctrl.GetVars().SSHKey)
			switch {
			case err != nil:
				log.Fatalf("error reading sshKey file %s: %v", ctrl.GetVars().SSHKey, err)
			}
			dev.SetConfigItem("debug.enable.ssh", string(b))
			if err = ctrl.ConfigSync(dev); err != nil {
				log.Fatal(err)
			}
			commandToRun := ""
			if len(args) > 0 {
				commandToRun = strings.Join(args, " ")
			}
			if !cmd.Flags().Changed("eve-host") {
				eveHost = getEVEIP()
			}
			arguments := fmt.Sprintf("-o ConnectTimeout=5 -oStrictHostKeyChecking=no -i %s -p %d root@%s %s", eveSSHKey, eveSSHPort, eveHost, commandToRun)
			log.Debugf("Try to ssh %s:%d with key %s and command %s", eveHost, eveSSHPort, eveSSHKey, arguments)
			if err := utils.RunCommandForeground("ssh", strings.Fields(arguments)...); err != nil {
				log.Fatalf("ssh error: %s", err)
			}
		} else {
			log.Fatalf("SSH key problem: %s", err)
		}
	},
}

var onboardEveCmd = &cobra.Command{
	Use:   "onboard",
	Short: "OnBoard EVE in Adam",
	Long:  `Adding an EVE onboarding certificate to Adam and waiting for EVE to register.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		eveUUID := viper.GetString("eve.uuid")
		edenDir, err := utils.DefaultEdenDir()
		if err != nil {
			log.Fatal(err)
		}
		if err = utils.TouchFile(filepath.Join(edenDir, fmt.Sprintf("state-%s.yml", eveUUID))); err != nil {
			log.Fatal(err)
		}
		changer := &adamChanger{}
		ctrl, err := changer.getController()
		if err != nil {
			log.Fatal(err)
		}
		vars := ctrl.GetVars()
		dev := device.CreateEdgeNode()
		dev.SetSerial(vars.EveSerial)
		dev.SetOnboardKey(vars.EveCert)
		dev.SetDevModel(vars.DevModel)
		err = ctrl.OnBoardDev(dev)
		if err != nil {
			log.Fatal(err)
		}
		if err = ctrl.StateUpdate(dev); err != nil {
			log.Fatal(err)
		}
		log.Info("onboarded")
		log.Info("device UUID: ", dev.GetID().String())
	},
}

var resetEveCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset EVE to initial config",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		eveUUID := viper.GetString("eve.uuid")
		edenDir, err := utils.DefaultEdenDir()
		if err != nil {
			log.Fatal(err)
		}
		if err = os.Remove(filepath.Join(edenDir, fmt.Sprintf("state-%s.yml", eveUUID))); err != nil {
			log.Fatal(err)
		}
		if err = utils.TouchFile(filepath.Join(edenDir, fmt.Sprintf("state-%s.yml", eveUUID))); err != nil {
			log.Fatal(err)
		}
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatal(err)
		}
		vars := ctrl.GetVars()
		dev.SetApplicationInstanceConfig(nil)
		dev.SetBaseOSConfig(nil)
		dev.SetNetworkInstanceConfig(nil)
		dev.SetVolumeConfigs(nil)
		dev.SetSerial(vars.EveSerial)
		dev.SetOnboardKey(vars.EveCert)
		dev.SetDevModel(vars.DevModel)
		err = ctrl.OnBoardDev(dev)
		if err != nil {
			log.Fatal(err)
		}
		if err = ctrl.StateUpdate(dev); err != nil {
			log.Fatal(err)
		}
		log.Info("reset done")
		log.Info("device UUID: ", dev.GetID().String())
	},
}

var epochEveCmd = &cobra.Command{
	Use:   "epoch",
	Short: "Set new epoch of EVE",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatal(err)
		}
		if eveConfigFromFile {
			edenDir, err := utils.DefaultEdenDir()
			if err != nil {
				log.Fatal(err)
			}
			changer := &fileChanger{fileConfig: filepath.Join(edenDir, fmt.Sprintf("devUUID-%s.json", dev.GetID()))}
			_, devFromFile, err := changer.getControllerAndDev()
			if err != nil {
				log.Fatalf("getControllerAndDev: %s", err)
			}
			dev = devFromFile
		}
		dev.SetEpoch(dev.GetEpoch() + 1)
		if err = ctrl.ConfigSync(dev); err != nil {
			log.Fatal(err)
		}
		log.Infof("new epoch %d sent", dev.GetEpoch())
		log.Info("device UUID: ", dev.GetID().String())
	},
}

func eveInit() {
	eveCmd.AddCommand(startEveCmd)
	eveCmd.AddCommand(stopEveCmd)
	eveCmd.AddCommand(statusEveCmd)
	eveCmd.AddCommand(ipEveCmd)
	eveCmd.AddCommand(sshEveCmd)
	eveCmd.AddCommand(consoleEveCmd)
	eveCmd.AddCommand(onboardEveCmd)
	eveCmd.AddCommand(resetEveCmd)
	eveCmd.AddCommand(versionEveCmd)
	eveCmd.AddCommand(epochEveCmd)
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
	startEveCmd.Flags().StringVarP(&vmName, "vmname", "", defaults.DefaultVBoxVMName, "vbox vmname required to create vm")
	startEveCmd.Flags().IntVarP(&cpus, "cpus", "", defaults.DefaultCpus, "vbox cpus")
	startEveCmd.Flags().IntVarP(&mem, "memory", "", defaults.DefaultMemory, "vbox memory size (MB)")
	stopEveCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file for save EVE pid")
	stopEveCmd.Flags().StringVarP(&vmName, "vmname", "", defaults.DefaultVBoxVMName, "vbox vmname required to create vm")
	statusEveCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file for save EVE pid")
	statusEveCmd.Flags().StringVarP(&vmName, "vmname", "", defaults.DefaultVBoxVMName, "vbox vmname required to create vm")
	sshEveCmd.Flags().StringVarP(&eveSSHKey, "ssh-key", "", filepath.Join(currentPath, defaults.DefaultCertsDist, "id_rsa"), "file to use for ssh access")
	sshEveCmd.Flags().StringVarP(&eveHost, "eve-host", "", defaults.DefaultEVEHost, "IP of eve")
	sshEveCmd.Flags().IntVarP(&eveSSHPort, "eve-ssh-port", "", defaults.DefaultSSHPort, "Port for ssh access")
	consoleEveCmd.Flags().StringVarP(&eveHost, "eve-host", "", defaults.DefaultEVEHost, "IP of eve")
	consoleEveCmd.Flags().IntVarP(&eveTelnetPort, "eve-telnet-port", "", defaults.DefaultTelnetPort, "Port for telnet access")
	epochEveCmd.Flags().BoolVar(&eveConfigFromFile, "use-config-file", false, "Load config of EVE from file")
}
