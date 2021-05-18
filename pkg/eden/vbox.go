package eden

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
)

//StartEVEVBox function run EVE in VirtualBox
func StartEVEVBox(vmName, eveImageFile string, cpus int, mem int, hostFwd map[string]string, ipMap map[string]net.IP) (err error) {
	poweroff := false
	if out, _, err := utils.RunCommandAndWait("VBoxManage", strings.Fields(fmt.Sprintf("showvminfo %s --machinereadable", vmName))...); err != nil {
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

		commandArgsString = fmt.Sprintf("natnetwork add --netname %s --network %s --enable --dhcp on",
										"natnet1", defaults.DefaultVBoxSubnet)
		if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
			log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
		}
		commandArgsString = fmt.Sprintf("natnetwork start --netname %s", "natnet1")
		if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
			log.Errorf("VBoxManage error for command %s %s", commandArgsString, err)
		}

		commandArgsString = fmt.Sprintf("modifyvm %s --nic1 natnetwork --nat-network1 %s --cableconnected1 on",
										vmName, "natnet1")
		if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
			log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
		}

		eth0IP := ipMap["eth0"]
		for k, v := range hostFwd {
			commandArgsString = fmt.Sprintf("natnetwork  modify --netname %s --port-forward-4 :tcp:[]:%s:[%s]:%s",
											"natnet1", k, eth0IP.String(), v)
			if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
				log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
			}
		}

		commandArgsString = fmt.Sprintf("modifyvm %s  --nic2 natnetwork --nat-network2 %s --cableconnected2 on",
										vmName, "natnet1")
		if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
			log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
		}
		eth1IP := ipMap["eth1"]
		for k, v := range hostFwd {
			hostPort, err := strconv.Atoi(k)
			if err != nil {
				log.Errorf("Parsing %s to Integer value failed", k)
				break
			}
			hostPort += 10
			guestPort, err := strconv.Atoi(v)
			if err != nil {
				log.Errorf("Parsing %s to Integer value failed", v)
				break
			}
			guestPort += 10
			commandArgsString = fmt.Sprintf("natnetwork  modify --netname %s --port-forward-4 :tcp:[]:%d:[%s]:%d",
											"natnet1", hostPort, eth1IP.String(), guestPort)
			if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
				log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
			}
		}

		commandArgsString = fmt.Sprintf("startvm  %s", vmName)
		if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
			log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
		}
	} else {
		scanner := bufio.NewScanner(bytes.NewReader([]byte(out)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.Contains(line, "VMState=\"poweroff\"") {
				poweroff = true
				break
			}
		}

		networkFound := false
		output := ""
		var err error
		if output, _, err = utils.RunCommandAndWait("VBoxManage", strings.Fields(fmt.Sprintf("natnetwork list  %s", "natnet1"))...); err != nil {
			log.Fatalf("VBoxManage error for command natnetwork list %s", err)
		}
		scanner = bufio.NewScanner(bytes.NewReader([]byte(output)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "Enabled") {
				enabled := strings.Split(line, ":")[1]
				enabled = strings.TrimSpace(enabled)
				if enabled == "Yes" {
					networkFound = true
				}
			}
		}
		if !networkFound {
			commandArgsString := fmt.Sprintf("natnetwork add --netname %s --network %s --enable --dhcp on",
			"natnet1", "10.0.2.0/24")
			if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
				log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
			}
			commandArgsString = fmt.Sprintf("natnetwork start --netname %s", "natnet1")
			if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
				log.Errorf("VBoxManage error for command %s %s", commandArgsString, err)
			}

			commandArgsString = fmt.Sprintf("modifyvm %s --nic1 natnetwork --nat-network1 %s --cableconnected1 on",
			vmName, "natnet1")
			if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
				log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
			}

			commandArgsString = fmt.Sprintf("modifyvm %s --nic2 natnetwork --nat-network2 %s --cableconnected2 on",
			vmName, "natnet1")
			if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
				log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
			}
		}

		eth0IP := ipMap["eth0"]
		for k, v := range hostFwd {
			commandArgsString := fmt.Sprintf("natnetwork  modify --netname %s --port-forward-4 :tcp:[]:%s:[%s]:%s",
											"natnet1", k, eth0IP.String(), v)
			if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
				if err.Error() == "exit status 2" {
					log.Infof("VBoxManage NAT rule: %s: already exists", commandArgsString)
				} else {
					log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
				}
			}
		}

		eth1IP := ipMap["eth1"]
		for k, v := range hostFwd {
			hostPort, err := strconv.Atoi(k)
			if err != nil {
				log.Errorf("Parsing %s to Integer value failed", k)
				break
			}
			hostPort += 10
			guestPort, err := strconv.Atoi(v)
			if err != nil {
				log.Errorf("Parsing %s to Integer value failed", v)
				break
			}
			guestPort += 10
			commandArgsString := fmt.Sprintf("natnetwork  modify --netname %s --port-forward-4 :tcp:[]:%d:[%s]:%d",
											"natnet1", hostPort, eth1IP.String(), guestPort)

			if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
				if err.Error() == "exit status 2" {
					log.Infof("VBoxManage NAT rule: %s: already exists", commandArgsString)
				} else {
					log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
				}
			}
		}
		if poweroff {
			commandArgsString := fmt.Sprintf("startvm  %s", vmName)
			if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
				log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
			}
		}
	}

	return err
}

//StopEVEVBox function stop EVE in VirtualBox
func StopEVEVBox(vmName string) (err error) {
	commandArgsString := fmt.Sprintf("natnetwork modify --netname %s --dhcp off", "natnet1")
	if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		log.Errorf("VBoxManage error for command %s %s", commandArgsString, err)
	}
	commandArgsString = fmt.Sprintf("natnetwork stop --netname %s", "natnet1")
	if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		log.Errorf("VBoxManage error for command %s %s", commandArgsString, err)
	}
	commandArgsString = fmt.Sprintf("natnetwork remove --netname %s", "natnet1")
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

//DeleteEVEVBox function removes EVE from VirtualBox
func DeleteEVEVBox(vmName string) (err error) {
	commandArgsString := fmt.Sprintf("unregistervm %s --delete", vmName)
	if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		log.Errorf("VBoxManage error for command %s %s", commandArgsString, err)
	}
	return err
}

//StatusEVEVBox function get status of EVE
func StatusEVEVBox(vmName string) (status string, err error) {
	out, _, err := utils.RunCommandAndWait("VBoxManage", strings.Fields(fmt.Sprintf("showvminfo %s --machinereadable", vmName))...)
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(bytes.NewReader([]byte(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "VMState=") {
			return strings.Split(line, "\"")[1], nil
		}
	}
	return "process doesn''t exist", nil
}

//SetLinkStateVbox changes the link state of the given interface.
//If interface name is undefined, the function changes the link state of every uplink interface.
func SetLinkStateVbox(vmName, ifName string, up bool) error {
	if ifName == "" {
		if err := setLinkStateVbox(vmName, "eth0", up); err != nil {
			return err
		}
		if err := setLinkStateVbox(vmName, "eth1", up); err != nil {
			return err
		}
		return nil
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

//GetLinkStateVbox returns the link state of the interface.
//If interface name is undefined, link state of all interfaces is returned.
func GetLinkStateVbox(vmName, ifName string) (linkStates []LinkState, err error) {
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
			linkStates = append(linkStates, LinkState{
				InterfaceName: nicName,
				IsUP:          isUp,
			})
		}
	}
	return linkStates, nil
}
