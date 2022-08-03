package eden

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/edensdn"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
)

const natNetworkName = "natnet1"

//StartEVEVBox function runs EVE in VirtualBox
func StartEVEVBox(vmName, eveImageFile string, cpus int, mem int, hostFwd map[string]string) (err error) {
	vmStatus, err := getEveVMStatusVbox(vmName)
	if err != nil {
		log.Info("No VMs with eve_live name", err)
		commandArgsString := fmt.Sprintf("createvm --name %s --register", vmName)
		if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
			log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
		}
		commandArgsString = fmt.Sprintf("modifyvm %s --cpus %d --memory %d --vram 16 --nested-hw-virt on --ostype Ubuntu_64  --mouse usbtablet --graphicscontroller vmsvga --boot1 disk --boot2 net", vmName, cpus, mem)
		if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
			log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
		}
		commandArgsString = fmt.Sprintf("storagectl %s --name \"SATA\" --add sata --bootable on --hostiocache on", vmName)
		if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
			log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
		}
		commandArgsString = fmt.Sprintf("storageattach %s  --storagectl \"SATA\" --port 0 --device 0 --type hdd --medium %s", vmName, eveImageFile)
		if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
			log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
		}
		if err := createNATNetworkVBox(vmName); err != nil {
			log.Fatal(err)
		}
		commandArgsString = fmt.Sprintf("startvm  %s", vmName)
		if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
			log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
		}
		if err := configurePortFwdVBox(vmName, hostFwd); err != nil {
			log.Fatal(err)
		}
		return nil
	}

	log.Info("EVE VM already exists")
	netEnabled, err := isNATNetworkEnabledVBox()
	if err != nil {
		log.Fatal(err)
	}
	if !netEnabled {
		log.Info("NAT Network is not created/enabled")
		if err := createNATNetworkVBox(vmName); err != nil {
			log.Fatal(err)
		}
	}
	if vmStatus != "running" {
		commandArgsString := fmt.Sprintf("startvm  %s", vmName)
		if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
			log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
		}
	}
	if err := configurePortFwdVBox(vmName, hostFwd); err != nil {
		log.Fatal(err)
	}
	return err
}

// getEveVMStatusVbox retrieves the status of EVE VM or non-nil error if VM is not created.
func getEveVMStatusVbox(vmName string) (status string, err error) {
	var out string
	out, _, err = utils.RunCommandAndWait("VBoxManage",
		strings.Fields(fmt.Sprintf("showvminfo %s --machinereadable", vmName))...)
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(bytes.NewReader([]byte(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "VMState=") {
			return strings.Split(line, "\"")[1], nil
		}
	}
	return "unknown", nil
}

// isNATNetworkEnabledVBox checks if the internal NAT network is created and enabled.
func isNATNetworkEnabledVBox() (isEnabled bool, err error) {
	output := ""
	if output, _, err = utils.RunCommandAndWait("VBoxManage", strings.Fields(fmt.Sprintf("natnetwork list  %s", natNetworkName))...); err != nil {
		err = fmt.Errorf("VBoxManage error for command natnetwork list %s", err)
		return
	}
	scanner := bufio.NewScanner(bytes.NewReader([]byte(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Enabled") {
			enabled := strings.Split(line, ":")[1]
			enabled = strings.TrimSpace(enabled)
			if enabled == "Yes" {
				isEnabled = true
				return
			}
		}
	}
	return
}

// createNATNetworkVBox creates internal NAT network for the EVE VM.
func createNATNetworkVBox(vmName string) (err error) {
	commandArgsString := fmt.Sprintf("natnetwork add --netname %s --network %s --enable --dhcp on",
		natNetworkName, "10.0.2.0/24")
	if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		return fmt.Errorf("VBoxManage error for command %s %s", commandArgsString, err)
	}
	commandArgsString = fmt.Sprintf("natnetwork start --netname %s", natNetworkName)
	if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		return fmt.Errorf("VBoxManage error for command %s %s", commandArgsString, err)
	}

	commandArgsString = fmt.Sprintf("modifyvm %s --nic1 natnetwork --nat-network1 %s --cableconnected1 on",
		vmName, natNetworkName)
	if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		return fmt.Errorf("VBoxManage error for command %s %s", commandArgsString, err)
	}

	commandArgsString = fmt.Sprintf("modifyvm %s --nic2 natnetwork --nat-network2 %s --cableconnected2 on",
		vmName, natNetworkName)
	if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		return fmt.Errorf("VBoxManage error for command %s %s", commandArgsString, err)
	}
	return nil
}

// configurePortFwdVBox configures port forwarding between the host and the EVE guest VM.
func configurePortFwdVBox(vmName string, hostFwd map[string]string) (err error) {
	ipAddrs, err := waitForGuestIPsVBox(vmName, 3*time.Minute)
	if err != nil {
		return err
	}
	// for eth0:
	for k, v := range hostFwd {
		commandArgsString := fmt.Sprintf("natnetwork  modify --netname %s --port-forward-4 :tcp:[]:%s:[%s]:%s",
			natNetworkName, k, ipAddrs[0], v)
		if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
			return fmt.Errorf("VBoxManage error for command %s %s", commandArgsString, err)
		}
	}
	// for eth1:
	for k, v := range hostFwd {
		hostPort, err := strconv.Atoi(k)
		if err != nil {
			log.Errorf("Parsing %s to Integer value failed", k)
			continue
		}
		hostPort += 10
		guestPort, err := strconv.Atoi(v)
		if err != nil {
			log.Errorf("Parsing %s to Integer value failed", v)
			continue
		}
		guestPort += 10
		commandArgsString := fmt.Sprintf("natnetwork  modify --netname %s --port-forward-4 :tcp:[]:%d:[%s]:%d",
			natNetworkName, hostPort, ipAddrs[1], guestPort)
		if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
			return fmt.Errorf("VBoxManage error for command %s %s", commandArgsString, err)
		}
	}
	return nil
}

// waitForGuestIPsVBox waits until VBox DHCP server assigns IP addresses to the EVE VM.
func waitForGuestIPsVBox(vmName string, timeout time.Duration) (ipAddrs [2]string, err error) {
	fmt.Print("Waiting for DHCP leases...")
	defer func() {
		if err == nil {
			fmt.Println(" [DONE]")
		} else {
			fmt.Println(" [TIMEOUT]")
		}
	}()
	for start := time.Now(); time.Since(start) < timeout; {
		ipAddrs, err = getGuestIPsVBox(vmName)
		if err == nil {
			break
		}
		time.Sleep(defaults.DefaultRepeatTimeout)
	}
	return
}

// getGuestIPsVBox returns IP addresses allocated to the EVE VM by the VBox DHCP server.
func getGuestIPsVBox(vmName string) (ipAddrs [2]string, err error) {
	var output string
	// First get MAC addresses assigned to EVE's uplink interfaces.
	output, _, err = utils.RunCommandAndWait("VBoxManage",
		strings.Fields(fmt.Sprintf("showvminfo %s", vmName))...)
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(bytes.NewReader([]byte(output)))
	nicInfoReg := regexp.MustCompile(`NIC ([0-9]+):.* MAC: ([0-9A-F]+),`)
	var macAddrs [2]string
	for scanner.Scan() {
		match := nicInfoReg.FindStringSubmatch(scanner.Text())
		if len(match) == 3 {
			nicIdx, err := strconv.Atoi(match[1])
			if err != nil {
				continue
			}
			if nicIdx == 1 || nicIdx == 2 {
				macAddrs[nicIdx-1] = match[2]
			}
		}
	}
	// Get IP address(es) from DHCP leases associated with EVE's MAC addresses.
	for i, macAddr := range macAddrs {
		if macAddr == "" {
			err = fmt.Errorf("failed to get MAC address of eth%d", i)
			return
		}
		output, _, err = utils.RunCommandAndWait("VBoxManage",
			strings.Fields(fmt.Sprintf("dhcpserver findlease --network %s --mac-address %s",
				natNetworkName, macAddr))...)
		if err != nil {
			return
		}
		scanner := bufio.NewScanner(bytes.NewReader([]byte(output)))
		const ipAddrPrefix = "IP Address:"
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, ipAddrPrefix) {
				ipAddrs[i] = strings.TrimPrefix(line, ipAddrPrefix)
				ipAddrs[i] = strings.TrimSpace(ipAddrs[i])
				break
			}
		}
	}
	for i, ipAddr := range ipAddrs {
		if ipAddr == "" {
			err = fmt.Errorf("failed to get IP address of eth%d", i)
			return
		}
	}
	return
}

// StopEVEVBox function stop EVE in VirtualBox
func StopEVEVBox(vmName string) (err error) {
	commandArgsString := fmt.Sprintf("natnetwork modify --netname %s --dhcp off", natNetworkName)
	if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		log.Errorf("VBoxManage error for command %s %s", commandArgsString, err)
	}
	commandArgsString = fmt.Sprintf("natnetwork stop --netname %s", natNetworkName)
	if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		log.Errorf("VBoxManage error for command %s %s", commandArgsString, err)
	}
	commandArgsString = fmt.Sprintf("natnetwork remove --netname %s", natNetworkName)
	if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		log.Errorf("VBoxManage error for command %s %s", commandArgsString, err)
	}
	commandArgsString = fmt.Sprintf("dhcpserver remove --netname %s", natNetworkName)
	if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		log.Errorf("VBoxManage error for command %s %s", commandArgsString, err)
	}
	commandArgsString = fmt.Sprintf("controlvm %s poweroff", vmName)
	if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		log.Errorf("VBoxManage error for command %s %s", commandArgsString, err)
	} else {
		for i := 0; i < 5; i++ {
			time.Sleep(defaults.DefaultRepeatTimeout)
			status, err := StatusEVEVBox(vmName)
			if err != nil {
				return err
			}
			if strings.Contains(status, "poweroff") {
				return nil
			}
		}
	}
	return err
}

// DeleteEVEVBox function removes EVE from VirtualBox
func DeleteEVEVBox(vmName string) (err error) {
	commandArgsString := fmt.Sprintf("unregistervm %s --delete", vmName)
	if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		log.Errorf("VBoxManage error for command %s %s", commandArgsString, err)
	}
	return err
}

// StatusEVEVBox function get status of EVE
func StatusEVEVBox(vmName string) (status string, err error) {
	return getEveVMStatusVbox(vmName)
}

// SetLinkStateVbox changes the link state of the given interface.
// If interface name is undefined, the function changes the link state of every uplink interface.
func SetLinkStateVbox(vmName, ifName string, up bool) error {
	if ifName == "" {
		if err := setLinkStateVbox(vmName, "eth0", up); err != nil {
			return err
		}
		return setLinkStateVbox(vmName, "eth1", up)
	}
	return setLinkStateVbox(vmName, ifName, up)
}

func setLinkStateVbox(vmName, ifName string, up bool) error {
	var ifIdx int
	switch ifName {
	case "eth0":
		ifIdx = 1
	case "eth1":
		ifIdx = 2
	default:
		return errors.New("no such device")
	}
	linkState := "on"
	if !up {
		linkState = "off"
	}
	_, _, err := utils.RunCommandAndWait("VBoxManage",
		strings.Fields(fmt.Sprintf("controlvm %s setlinkstate%d %s", vmName, ifIdx, linkState))...)
	return err
}

// GetLinkStateVbox returns the link state of the interface.
// If interface name is undefined, link state of all interfaces is returned.
func GetLinkStateVbox(vmName, ifName string) (linkStates []edensdn.LinkState, err error) {
	out, _, err := utils.RunCommandAndWait("VBoxManage", strings.Fields(fmt.Sprintf("showvminfo %s", vmName))...)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader([]byte(out)))
	nicInfoReg := regexp.MustCompile("NIC ([0-9]+):.*Cable connected: (on|off)")
	for scanner.Scan() {
		match := nicInfoReg.FindStringSubmatch(scanner.Text())
		if len(match) == 3 {
			nicIdx, err := strconv.Atoi(match[1])
			if err != nil {
				continue
			}
			nicName := fmt.Sprintf("eth%d", nicIdx-1)
			if ifName != "" && ifName != nicName {
				continue
			}
			isUp := match[2] == "on"
			linkStates = append(linkStates, edensdn.LinkState{
				EveIfName: nicName,
				IsUP:      isUp,
			})
		}
	}
	return linkStates, nil
}
