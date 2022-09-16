package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/edensdn"
	"github.com/lf-edge/eden/pkg/eve"
	"github.com/lf-edge/eden/pkg/utils"
	sdnapi "github.com/lf-edge/eden/sdn/api"
	"github.com/lf-edge/eve/api/go/info"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const sdnStartTimeout = time.Minute

var (
	qemuARCH             string
	qemuOS               string
	qemuAccel            bool
	qemuSMBIOSSerial     string
	qemuConfigFile       string
	qemuForeground       bool
	qemuMonitorPort      int
	qemuNetdevSocketPort int
	eveSSHKey            string
	eveHost              string
	eveSSHPort           int
	eveTelnetPort        int
	eveRemoteAddr        string
	eveConfigFromFile    bool
	eveInterfaceName     string
	tapInterface         string
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
			qemuMonitorPort = viper.GetInt("eve.qemu.monitor-port")
			qemuNetdevSocketPort = viper.GetInt("eve.qemu.netdev-socket-port")
			qemuFirmware = viper.GetStringSlice("eve.firmware")
			eveImageFile = utils.ResolveAbsPath(viper.GetString("eve.image-file"))
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
			eveLogFile = utils.ResolveAbsPath(viper.GetString("eve.log"))
			eveRemote = viper.GetBool("eve.remote")
			eveUsbNetConfFile = viper.GetString("eve.usbnetconf-file")
			eveTelnetPort = viper.GetInt("eve.telnet-port")
			devModel = viper.GetString("eve.devmodel")
			gcpvTPM = viper.GetBool("eve.tpm")
			adamPort = viper.GetInt("adam.port")
			loadSdnOptsFromViper()
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
			startEveQemu()
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
			gcpvTPM = viper.GetBool("eve.tpm")
			eveImageFile = utils.ResolveAbsPath(viper.GetString("eve.image-file"))
			loadSdnOptsFromViper()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if eveRemote {
			log.Debug("Cannot stop remote EVE")
			return
		}
		eden.StopEve(evePidFile, swtpmPidFile(), sdnPidFile, devModel, vmName)
	},
}

func swtpmPidFile() string {
	if gcpvTPM {
		command := "swtpm"
		return filepath.Join(filepath.Join(filepath.Dir(eveImageFile), command),
			fmt.Sprintf("%s.pid", command))
	}
	return ""
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
			var handleInfo = func(im *info.ZInfoMsg) bool {
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
				sdnStatus()
			}
		}
		if err == nil && statusAdam != "container doesn't exist" {
			eveRequestsAdam()
		}
	},
}

func getEVEIP(ifName string) string {
	if isSdnEnabled() {
		// EVE VM is behind SDN VM.
		if ifName == "" {
			ifName = "eth0"
		}
		client := &edensdn.SdnClient{
			SSHPort:    uint16(sdnSSHPort),
			SSHKeyPath: sdnSSHKeyPath(),
			MgmtPort:   uint16(sdnMgmtPort),
		}
		ip, err := client.GetEveIfIP(ifName)
		if err != nil {
			log.Errorf("Failed to get EVE IP address: %v", err)
			return ""
		}
		return ip
	}
	// XXX ifName argument is not supported below
	if runtime.GOOS == "darwin" {
		if !eveRemote {
			return "127.0.0.1"
		}
		networks, err := getEveNetworkInfo()
		if err != nil {
			log.Error(err)
			return ""
		}
		var ips []string
		for _, nw := range networks {
			ips = append(ips, nw.IPAddrs...)
		}
		if len(ips) == 0 {
			return ""
		}
		return ips[0]
	}
	if ip, err := eveLastRequests(); err == nil && ip != "" {
		return ip
	}
	return ""
}

func getEveNetworkInfo() (networks []*info.ZInfoNetwork, err error) {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return nil, fmt.Errorf("getControllerAndDev failed: %s", err)
	}
	eveState := eve.Init(ctrl, dev)
	if err = ctrl.InfoLastCallback(dev.GetID(), nil, eveState.InfoCallback()); err != nil {
		return nil, fmt.Errorf("InfoLastCallback failed: %s", err)
	}
	if err = ctrl.MetricLastCallback(dev.GetID(), nil, eveState.MetricCallback()); err != nil {
		return nil, fmt.Errorf("MetricLastCallback failed: %s", err)
	}
	if lastDInfo := eveState.InfoAndMetrics().GetDinfo(); lastDInfo != nil {
		for _, nw := range lastDInfo.Network {
			networks = append(networks, nw)
		}
	}
	return networks, nil
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
			devModel = viper.GetString("eve.devmodel")
			loadSdnOptsFromViper()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(getEVEIP("eth0"))
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
			if eveRemote {
				if !cmd.Flags().Changed("eve-host") {
					eveHost = getEVEIP("eth0")
				}
				arguments := fmt.Sprintf("-o ConnectTimeout=5 -oStrictHostKeyChecking=no -i %s -p %d root@%s %s",
					eveSSHKey, eveSSHPort, eveHost, commandToRun)
				log.Debugf("Try to ssh %s:%d with key %s and command %s", eveHost, eveSSHPort, eveSSHKey, arguments)
				if err := utils.RunCommandForeground("ssh", strings.Fields(arguments)...); err != nil {
					log.Fatalf("ssh error: %s", err)
				}
			} else {
				sdnForwardSSHToEve(commandToRun)
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
		dev, err := ctrl.GetDeviceCurrent()
		if err != nil || dev == nil {
			//create new one if not exists
			dev = device.CreateEdgeNode()
			dev.SetSerial(vars.EveSerial)
			dev.SetOnboardKey(vars.EveCert)
			dev.SetDevModel(vars.DevModel)
			err = ctrl.OnBoardDev(dev)
			if err != nil {
				log.Fatal(err)
			}
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
		dev.SetGlobalProfile("")
		dev.SetLocalProfileServer("")
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

var linkEveCmd = &cobra.Command{
	Use:   "link up|down|status",
	Short: "manage EVE interface link state",
	Long:  `Manage EVE interface link state. Supported for QEMU and VirtualBox.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			eveRemote = viper.GetBool("eve.remote")
			qemuMonitorPort = viper.GetInt("eve.qemu.monitor-port")
			devModel = viper.GetString("eve.devmodel")
			loadSdnOptsFromViper()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		if eveRemote {
			log.Fatal("Cannot change interface link of a remote EVE")
		}
		command := "status"
		if len(args) > 0 {
			command = args[0]
		}
		// Get the set of interfaces to get/set the link state of.
		var eveIfNames []string
		if eveInterfaceName != "" {
			eveIfNames = append(eveIfNames, eveInterfaceName)
		} else {
			if isSdnEnabled() {
				client := &edensdn.SdnClient{
					SSHPort:    uint16(sdnSSHPort),
					SSHKeyPath: sdnSSHKeyPath(),
					MgmtPort:   uint16(sdnMgmtPort),
				}
				netModel, err := client.GetNetworkModel()
				if err != nil {
					log.Fatalf("Failed to get network model: %v", err)
				}
				for i := range netModel.Ports {
					eveIfNames = append(eveIfNames, fmt.Sprintf("eth%d", i))
				}
			} else {
				eveIfNames = []string{"eth0", "eth1"}
			}
		}
		if command == "up" || command == "down" {
			bringUp := command == "up"
			switch devModel {
			case defaults.DefaultVBoxModel:
				for _, ifName := range eveIfNames {
					err = eden.SetLinkStateVbox(vmName, ifName, bringUp)
				}
			case defaults.DefaultQemuModel:
				for _, ifName := range eveIfNames {
					err = eden.SetLinkStateQemu(qemuMonitorPort, ifName, bringUp)
				}
			default:
				log.Fatalf("Link operations are not supported for devmodel '%s'", devModel)
			}
			if err != nil {
				log.Fatal(err)
			}
			// continue to print the new link state of every interface after the update
			log.Info("Link state of EVE interfaces after update:")
			eveInterfaceName = ""
		}

		var linkStates []edensdn.LinkState
		switch devModel {
		case defaults.DefaultVBoxModel:
			linkStates, err = eden.GetLinkStatesVbox(vmName, eveIfNames)
		case defaults.DefaultQemuModel:
			linkStates, err = eden.GetLinkStatesQemu(qemuMonitorPort, eveIfNames)
		default:
			log.Fatalf("Link operations are not supported for devmodel '%s'", devModel)
		}
		if err != nil {
			log.Fatal(err)
		}

		// print table with link states into stdout
		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 0, 8, 1, '\t', 0)
		if _, err = fmt.Fprintln(w, "INTERFACE\tLINK"); err != nil {
			log.Fatal(err)
		}
		sort.SliceStable(linkStates, func(i, j int) bool {
			return linkStates[i].EveIfName < linkStates[j].EveIfName
		})
		for _, linkState := range linkStates {
			state := "UP"
			if !linkState.IsUP {
				state = "DOWN"
			}
			if _, err := fmt.Fprintln(w, linkState.EveIfName+"\t"+state); err != nil {
				log.Fatal(err)
			}
		}
		if err = w.Flush(); err != nil {
			log.Fatal(err)
		}
	},
}

func startEveQemu() {
	// Load network model and prepare SDN config.
	var err error
	var netModel sdnapi.NetworkModel
	if !isSdnEnabled() || sdnNetModelFile == "" {
		netModel, err = edensdn.GetDefaultNetModel()
		if err != nil {
			log.Fatal(err)
		}
	} else {
		netModel, err = edensdn.LoadNetModeFromFile(sdnNetModelFile)
		if err != nil {
			log.Fatalf("Failed to load network model from file '%s': %v",
				sdnNetModelFile, err)
		}
	}
	netModel.Host.ControllerPort = uint16(adamPort)
	if isSdnEnabled() {
		nets, err := utils.GetSubnetsNotUsed(1)
		if err != nil {
			log.Fatalf("Failed to get unused IP subnet: %s", err)
		}
		// Reuse firmware installed for EVE VM.
		var firmware []string
		for _, line := range qemuFirmware {
			for _, el := range strings.Split(line, " ") {
				firmware = append(firmware, utils.ResolveAbsPath(el))
			}
		}
		sdnConfig := edensdn.SdnVMConfig{
			Architecture: qemuARCH,
			Acceleration: qemuAccel,
			HostOS:       qemuOS,
			ImagePath:    sdnImageFile,
			ConfigDir:    sdnConfigDir,
			CPU:          sdnCPU,
			RAM:          sdnRAM,
			Firmware:     firmware,
			NetModel:     netModel,
			TelnetPort:   uint16(sdnTelnetPort),
			SSHPort:      uint16(sdnSSHPort),
			SSHKeyPath:   sdnSSHKeyPath(),
			MgmtPort:     uint16(sdnMgmtPort),
			MgmtSubnet: edensdn.SdnMgmtSubnet{
				IPNet:     nets[0].Subnet,
				DHCPStart: nets[0].FirstAddress,
			},
			NetDevBasePort: uint16(qemuNetdevSocketPort),
			PidFile:        sdnPidFile,
			ConsoleLogFile: sdnConsoleLogFile,
		}
		sdnVmRunner, err := edensdn.GetSdnVMRunner(devModel, sdnConfig)
		if err != nil {
			log.Fatalf("failed to get SDN VM runner: %v", err)
		}
		// Start SDN.
		err = sdnVmRunner.Start()
		if err != nil {
			log.Fatalf("Cannot start SDN: %v", err)
		} else {
			log.Infof("SDN is starting")
		}
		// Wait for SDN to start and apply network model.
		startTime := time.Now()
		client := &edensdn.SdnClient{
			SSHPort:  uint16(sdnSSHPort),
			MgmtPort: uint16(sdnMgmtPort),
		}
		for time.Since(startTime) < sdnStartTimeout {
			time.Sleep(2 * time.Second)
			if _, err = client.GetSdnStatus(); err == nil {
				break
			}
		}
		if err != nil {
			log.Fatalf("Timeout waiting for SDN to start: %v", err)
		}
		err = client.ApplyNetworkModel(netModel)
		if err != nil {
			log.Fatalf("Failed to apply network model: %v", err)
		}
		log.Infof("SDN started, network model was submitted.")
	}
	// Create USB network config override image if requested.
	var usbImagePath string
	if eveUsbNetConfFile != "" {
		currentPath, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		usbImagePath = filepath.Join(currentPath, defaults.DefaultDist, "usb.img")
		err = utils.CreateUsbNetConfImg(eveUsbNetConfFile, usbImagePath)
		if err != nil {
			log.Fatal(err)
		}
	}
	// Start vTPM.
	if gcpvTPM {
		err = eden.StartSWTPM(filepath.Join(filepath.Dir(eveImageFile), "swtpm"))
		if err != nil {
			log.Errorf("cannot start swtpm: %s", err)
		} else {
			log.Infof("swtpm is starting")
		}
	}
	// Start EVE VM.
	if err = eden.StartEVEQemu(qemuARCH, qemuOS, eveImageFile, qemuSMBIOSSerial, eveTelnetPort,
		qemuMonitorPort, qemuNetdevSocketPort, hostFwd, qemuAccel, qemuConfigFile, eveLogFile,
		evePidFile, netModel, isSdnEnabled(), tapInterface, usbImagePath, gcpvTPM, false); err != nil {
		log.Errorf("cannot start eve: %s", err)
	} else {
		log.Infof("EVE is starting")
	}
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
	eveCmd.AddCommand(linkEveCmd)
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
}
