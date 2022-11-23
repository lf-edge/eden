package openevec

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/edensdn"
	"github.com/lf-edge/eden/pkg/eve"
	"github.com/lf-edge/eden/pkg/utils"
	sdnapi "github.com/lf-edge/eden/sdn/vm/api"
	"github.com/lf-edge/eve/api/go/info"
	log "github.com/sirupsen/logrus"
)

const SdnStartTimeout = 3 * time.Minute

func StartEve(vmName string, cfg *EdenSetupArgs) error {
	if cfg.Eve.Remote {
		return nil
	}

	if cfg.Eve.DevModel == defaults.DefaultParallelsModel {
		if err := eden.StartEVEParallels(vmName, cfg.Eve.ImageFile, cfg.Eve.QemuCpus, cfg.Eve.QemuMemory, cfg.Eve.HostFwd); err != nil {
			return fmt.Errorf("cannot start eve: %s", err)
		} else {
			log.Infof("EVE is starting in Parallels")
		}
	} else if cfg.Eve.DevModel == defaults.DefaultVBoxModel {
		if err := eden.StartEVEVBox(vmName, cfg.Eve.ImageFile, cfg.Eve.QemuCpus, cfg.Eve.QemuMemory, cfg.Eve.HostFwd); err != nil {
			return fmt.Errorf("cannot start eve: %s", err)
		} else {
			log.Infof("EVE is starting in Virtual Box")
		}
	} else {
		if err := StartEveQemu(cfg); err != nil {
			log.Fatal(err)
		}
	}
	return nil
}

func StartEveQemu(cfg *EdenSetupArgs) error {
	// Load network model and prepare SDN config.
	var err error
	var netModel sdnapi.NetworkModel
	if !isSdnEnabled(cfg.Sdn.Disable, cfg.Eve.Remote, cfg.Eve.DevModel) || cfg.Sdn.NetModelFile == "" {
		netModel, err = edensdn.GetDefaultNetModel()
		if err != nil {
			return err
		}
	} else {
		netModel, err = edensdn.LoadNetModeFromFile(cfg.Sdn.NetModelFile)
		if err != nil {
			log.Fatalf("Failed to load network model from file '%s': %v",
				cfg.Sdn.NetModelFile, err)
		}
	}
	if cfg.Eve.CustomInstaller.Path == "" {
		netModel.Host.ControllerPort = uint16(cfg.Adam.Port)
	} else {
		// With custom EVE installer it is
		// assumed that controller other
		// than Adam is being used.
		netModel.Host.ControllerPort = 443
	}
	if isSdnEnabled(cfg.Sdn.Disable, cfg.Eve.Remote, cfg.Eve.DevModel) {
		nets, err := utils.GetSubnetsNotUsed(1)
		if err != nil {
			log.Fatalf("Failed to get unused IP subnet: %s", err)
		}
		imageDir := filepath.Dir(cfg.Sdn.ImageFile)
		firmware := []string{"OVMF_CODE.fd", "OVMF_VARS.fd"}
		for i := range firmware {
			firmware[i] = utils.ResolveAbsPath(filepath.Join(imageDir, firmware[i]))
		}
		sdnConfig := edensdn.SdnVMConfig{
			Architecture: cfg.Eve.Arch,
			Acceleration: cfg.Eve.Accel,
			HostOS:       cfg.Eve.QemuOS,
			ImagePath:    cfg.Sdn.ImageFile,
			ConfigDir:    cfg.Sdn.ConfigDir,
			CPU:          cfg.Sdn.CPU,
			RAM:          cfg.Sdn.RAM,
			Firmware:     firmware,
			NetModel:     netModel,
			TelnetPort:   uint16(cfg.Sdn.TelnetPort),
			SSHPort:      uint16(cfg.Sdn.SSHPort),
			SSHKeyPath:   sdnSSHKeyPath(cfg.Sdn.SourceDir),
			MgmtPort:     uint16(cfg.Sdn.MgmtPort),
			MgmtSubnet: edensdn.SdnMgmtSubnet{
				IPNet:     nets[0].Subnet,
				DHCPStart: nets[0].FirstAddress,
			},
			NetDevBasePort: uint16(cfg.Eve.QemuConfig.NetdevSocketPort),
			PidFile:        cfg.Sdn.PidFile,
			ConsoleLogFile: cfg.Sdn.ConsoleLogFile,
		}
		sdnVmRunner, err := edensdn.GetSdnVMRunner(cfg.Eve.DevModel, sdnConfig)
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
			SSHPort:  uint16(cfg.Sdn.SSHPort),
			MgmtPort: uint16(cfg.Sdn.MgmtPort),
		}
		for time.Since(startTime) < SdnStartTimeout {
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
	if cfg.Eve.UsbNetConfFile != "" {
		currentPath, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		usbImagePath = filepath.Join(currentPath, defaults.DefaultDist, "usb.img")
		err = utils.CreateUsbNetConfImg(cfg.Eve.UsbNetConfFile, usbImagePath)
		if err != nil {
			log.Fatal(err)
		}
	}
	// Prepare for EVE installation if requested.
	isInstaller := false
	imageFile := cfg.Eve.ImageFile
	imageFormat := "qcow2"
	if cfg.Eve.CustomInstaller.Path != "" {
		isInstaller = true
		imageFile = cfg.Eve.CustomInstaller.Path
		imageFormat = cfg.Eve.CustomInstaller.Format
	}
	// Start vTPM.
	if cfg.Eve.TPM {
		err = eden.StartSWTPM(filepath.Join(filepath.Dir(imageFile), "swtpm"))
		if err != nil {
			log.Errorf("cannot start swtpm: %s", err)
		} else {
			log.Infof("swtpm is starting")
		}
	}
	// Start EVE VM.
	if err = eden.StartEVEQemu(cfg.Eve.Arch, cfg.Eve.QemuOS, imageFile, imageFormat, isInstaller, cfg.Eve.Serial, cfg.Eve.TelnetPort,
		cfg.Eve.QemuConfig.MonitorPort, cfg.Eve.QemuConfig.NetdevSocketPort, cfg.Eve.HostFwd, cfg.Eve.Accel, cfg.Eve.QemuFileToSave, cfg.Eve.Log,
		cfg.Eve.Pid, netModel, isSdnEnabled(cfg.Sdn.Disable, cfg.Eve.Remote, cfg.Eve.DevModel), cfg.Eve.TapInterface, usbImagePath, cfg.Eve.TPM, false); err != nil {
		log.Errorf("cannot start eve: %s", err)
	} else {
		log.Infof("EVE is starting")
	}
	return nil
}

func StopEve(vmName string, cfg *EdenSetupArgs) error {
	if cfg.Eve.Remote {
		log.Debug("Cannot stop remote EVE")
		return nil
	}
	if cfg.Eve.DevModel == defaults.DefaultVBoxModel {
		if err := eden.StopEVEVBox(vmName); err != nil {
			log.Errorf("cannot stop eve: %s", err)
		} else {
			log.Infof("EVE is stopping in Virtual Box")
		}
	} else if cfg.Eve.DevModel == defaults.DefaultParallelsModel {
		if err := eden.StopEVEParallels(vmName); err != nil {
			log.Errorf("cannot stop eve: %s", err)
		} else {
			log.Infof("EVE is stopping in Virtual Box")
		}
	} else {
		if err := eden.StopEVEQemu(cfg.Eve.Pid); err != nil {
			log.Errorf("cannot stop eve: %s", err)
		} else {
			log.Infof("EVE is stopping")
		}
		if cfg.Eve.TPM {
			err := eden.StopSWTPM(filepath.Join(filepath.Dir(cfg.Eve.ImageFile), "swtpm"))
			if err != nil {
				log.Errorf("cannot stop swtpm: %s", err)
			} else {
				log.Infof("swtpm is stopping")
			}
		}
	}
	return nil
}

func VersionEve() error {
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
			return fmt.Errorf("Fail in get InfoLastCallback: %s", err)
		}
		if lastDInfo == nil {
			log.Info("no info messages")
		} else {
			fmt.Println(lastDInfo.GetDinfo().SwList[0].ShortVersion)
		}
	}
	return nil
}

func StatusEve(vmName string, cfg *EdenSetupArgs) error {
	statusAdam, err := eden.StatusAdam()
	if err == nil && statusAdam != "container doesn't exist" {
		eveStatusRemote()
	}
	if !cfg.Eve.Remote {
		if cfg.Eve.DevModel == defaults.DefaultVBoxModel {
			eveStatusVBox(vmName)
		} else if cfg.Eve.DevModel == defaults.DefaultParallelsModel {
			eveStatusParallels(vmName)
		} else {
			eveStatusQEMU(cfg.ConfigName, cfg.Eve.Pid)
		}
	}
	if err == nil && statusAdam != "container doesn't exist" {
		eveRequestsAdam()
	}
	return nil
}

func GetEveIp(ifName string, cfg *EdenSetupArgs) string {
	if isSdnEnabled(cfg.Sdn.Disable, cfg.Eve.Remote, cfg.Eve.DevModel) {
		// EVE VM is behind SDN VM.
		if ifName == "" {
			ifName = "eth0"
		}
		client := &edensdn.SdnClient{
			SSHPort:    uint16(cfg.Sdn.SSHPort),
			SSHKeyPath: sdnSSHKeyPath(cfg.Sdn.SourceDir),
			MgmtPort:   uint16(cfg.Sdn.MgmtPort),
		}
		ip, err := client.GetEveIfIP(ifName)
		if err != nil {
			log.Errorf("Failed to get EVE IP address: %v", err)
			return ""
		}
		return ip
	}
	networks, err := getEveNetworkInfo()
	if err != nil {
		log.Error(err)
		return ""
	}
	for _, nw := range networks {
		if nw.LocalName == ifName {
			if len(nw.IPAddrs) == 0 {
				return ""
			}
			return nw.IPAddrs[0]
		}
	}
	return ""
}

func eveLastRequests() (string, error) {
	log.Debugf("Will try to obtain info from ADAM")
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return "", err
	}
	var lastRequest *types.APIRequest
	var handleRequest = func(request *types.APIRequest) bool {
		if request.ClientIP != "" {
			lastRequest = request
		}
		return false
	}
	if err := ctrl.RequestLastCallback(dev.GetID(), map[string]string{"UUID": dev.GetID().String()}, handleRequest); err != nil {
		return "", err
	}
	if lastRequest == nil {
		return "", nil
	}
	return strings.Split(lastRequest.ClientIP, ":")[0], nil
}

func ConsoleEve(cfg *EdenSetupArgs) error {
	if cfg.Eve.Remote {
		return fmt.Errorf("cannot telnet to remote EVE")
	}
	log.Infof("Try to telnet %s:%d", cfg.Eve.Host, cfg.Eve.TelnetPort)
	if err := utils.RunCommandForeground("telnet", strings.Fields(fmt.Sprintf("%s %d", cfg.Eve.Host, cfg.Eve.TelnetPort))...); err != nil {
		return fmt.Errorf("telnet error: %s", err)
	}
	return nil
}

func SshEve(commandToRun string, cfg *EdenSetupArgs) error {
	if _, err := os.Stat(cfg.Eden.SshKey); !os.IsNotExist(err) {
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			return fmt.Errorf("Cannot get controller or dev, please start them and onboard: %s", err)
		}
		b, err := ioutil.ReadFile(ctrl.GetVars().SSHKey)
		switch {
		case err != nil:
			return fmt.Errorf("error reading sshKey file %s: %v", ctrl.GetVars().SSHKey, err)
		}
		dev.SetConfigItem("debug.enable.ssh", string(b))
		if err = ctrl.ConfigSync(dev); err != nil {
			return err
		}
		if err = SdnForwardSSHToEve(commandToRun, cfg); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("SSH key problem: %s", err)
	}

	return nil
}

func ResetEve(certsUUID string) error {
	edenDir, err := utils.DefaultEdenDir()
	if err != nil {
		return err
	}
	if err = os.Remove(filepath.Join(edenDir, fmt.Sprintf("state-%s.yml", certsUUID))); err != nil {
		return err
	}
	if err = utils.TouchFile(filepath.Join(edenDir, fmt.Sprintf("state-%s.yml", certsUUID))); err != nil {
		return err
	}
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return err
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
		return err
	}
	if err = ctrl.StateUpdate(dev); err != nil {
		return err
	}
	log.Info("reset done")
	log.Info("device UUID: ", dev.GetID().String())

	return nil
}

func NewEpochEve(eveConfigFromFile bool) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return err
	}
	if eveConfigFromFile {
		edenDir, err := utils.DefaultEdenDir()
		if err != nil {
			return err
		}
		changer := &fileChanger{fileConfig: filepath.Join(edenDir, fmt.Sprintf("devUUID-%s.json", dev.GetID()))}
		_, devFromFile, err := changer.getControllerAndDev()
		if err != nil {
			return fmt.Errorf("getControllerAndDev: %s", err)
		}
		dev = devFromFile
	}
	dev.SetEpoch(dev.GetEpoch() + 1)
	if err = ctrl.ConfigSync(dev); err != nil {
		return err
	}
	log.Infof("new epoch %d sent", dev.GetEpoch())
	log.Info("device UUID: ", dev.GetID().String())

	return nil
}

func NewLinkEve(command, eveInterfaceName string, cfg *EdenSetupArgs) error {
	var err error
	if cfg.Eve.Remote {
		log.Fatal("Cannot change interface link of a remote EVE")
	}
	// Get the set of interfaces to get/set the link state of.
	var eveIfNames []string
	if eveInterfaceName != "" {
		eveIfNames = append(eveIfNames, eveInterfaceName)
	} else {
		if isSdnEnabled(cfg.Sdn.Disable, cfg.Eve.Remote, cfg.Eve.DevModel) {
			client := &edensdn.SdnClient{
				SSHPort:    uint16(cfg.Sdn.SSHPort),
				SSHKeyPath: sdnSSHKeyPath(cfg.Sdn.SourceDir),
				MgmtPort:   uint16(cfg.Sdn.MgmtPort),
			}
			netModel, err := client.GetNetworkModel()
			if err != nil {
				return fmt.Errorf("Failed to get network model: %v", err)
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
		switch cfg.Eve.DevModel {
		case defaults.DefaultVBoxModel:
			for _, ifName := range eveIfNames {
				err = eden.SetLinkStateVbox(cfg.Runtime.VmName, ifName, bringUp)
			}
		case defaults.DefaultQemuModel:
			for _, ifName := range eveIfNames {
				err = eden.SetLinkStateQemu(cfg.Eve.QemuConfig.MonitorPort, ifName, bringUp)
			}
		default:
			return fmt.Errorf("Link operations are not supported for devmodel '%s'", cfg.Eve.DevModel)
		}
		if err != nil {
			return err
		}
		// continue to print the new link state of every interface after the update
		log.Info("Link state of EVE interfaces after update:")
		eveInterfaceName = ""
	}

	var linkStates []edensdn.LinkState
	switch cfg.Eve.DevModel {
	case defaults.DefaultVBoxModel:
		linkStates, err = eden.GetLinkStatesVbox(cfg.Runtime.VmName, eveIfNames)
	case defaults.DefaultQemuModel:
		linkStates, err = eden.GetLinkStatesQemu(cfg.Eve.QemuConfig.MonitorPort, eveIfNames)
	default:
		return fmt.Errorf("Link operations are not supported for devmodel '%s'", cfg.Eve.DevModel)
	}
	if err != nil {
		return err
	}

	// print table with link states into stdout
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	if _, err = fmt.Fprintln(w, "INTERFACE\tLINK"); err != nil {
		return err
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
			return err
		}
	}
	if err = w.Flush(); err != nil {
		return err
	}
	return nil
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
