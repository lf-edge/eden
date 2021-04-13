package eden

import (
	"fmt"
	"strings"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
)

//DeleteEVEParallels function removes EVE from parallels
func DeleteEVEParallels(vmName string) (err error) {
	commandArgsString := fmt.Sprintf("delete %s", vmName)
	if err = utils.RunCommandWithLogAndWait("prlctl", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		log.Errorf("prlctl error for command %s %s", commandArgsString, err)
	}
	return err
}

//StartEVEParallels function run EVE in parallels
func StartEVEParallels(vmName, eveImageFile string, parallelsCpus int, parallelsMem int, hostFwd map[string]string) (err error) {
	status, err := StatusEVEParallels(vmName)
	if err != nil {
		log.Fatal(err)
	}
	if strings.Contains(status, "running") {
		return nil
	}
	_ = StopEVEParallels(vmName)

	commandArgsString := fmt.Sprintf("create %s --distribution ubuntu --no-hdd", vmName)
	if err = utils.RunCommandWithLogAndWait("prlctl", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		log.Fatalf("prlctl error for command %s %s", commandArgsString, err)
	}
	commandArgsString = fmt.Sprintf("set %s --device-del net0 --cpus %d --memsize %d --nested-virt on --adaptive-hypervisor on --hypervisor-type parallels", vmName, parallelsCpus, parallelsMem)
	if err = utils.RunCommandWithLogAndWait("prlctl", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		log.Fatalf("prlctl error for command %s %s", commandArgsString, err)
	}
	dirForParallels := strings.TrimRight(eveImageFile, filepath.Ext(eveImageFile))
	commandArgsString = fmt.Sprintf("set %s --device-add hdd --image %s", vmName, dirForParallels)
	if err = utils.RunCommandWithLogAndWait("prlctl", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		log.Fatalf("prlctl error for command %s %s", commandArgsString, err)
	}
	commandArgsString = fmt.Sprintf("set %s --device-add net --type shared --adapter-type virtio --ipadd 192.168.1.0/24 --dhcp yes", vmName)
	if err = utils.RunCommandWithLogAndWait("prlctl", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		log.Fatalf("prlctl error for command %s %s", commandArgsString, err)
	}
	commandArgsString = fmt.Sprintf("set %s --device-add net --type shared --adapter-type virtio --ipadd 192.168.2.0/24 --dhcp yes", vmName)
	if err = utils.RunCommandWithLogAndWait("prlctl", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		log.Fatalf("prlctl error for command %s %s", commandArgsString, err)
	}
	for k, v := range hostFwd {
		commandArgsString = fmt.Sprintf("net set Shared --nat-tcp-add %s_%s,%s,%s,%s", k, v, k, vmName, v)
		if err = utils.RunCommandWithLogAndWait("prlsrvctl", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
			log.Fatalf("prlsrvctl error for command %s %s", commandArgsString, err)
		}
	}
	commandArgsString = fmt.Sprintf("start %s", vmName)
	return utils.RunCommandWithLogAndWait("prlctl", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
}

//StopEVEParallels function stop EVE and delete parallels VM
func StopEVEParallels(vmName string) (err error) {
	commandArgsString := fmt.Sprintf("stop %s --kill", vmName)
	if err = utils.RunCommandWithLogAndWait("prlctl", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		log.Errorf("prlctl error for command %s %s", commandArgsString, err)
	}
	return DeleteEVEParallels(vmName)
}

//StatusEVEParallels function get status of EVE
func StatusEVEParallels(vmName string) (status string, err error) {
	commandArgsString := fmt.Sprintf("status %s", vmName)
	statusEVE, _, err := utils.RunCommandAndWait("prlctl", strings.Fields(commandArgsString)...)
	if err != nil {
		return "process doesn''t exist", nil
	}
	statusEVE = strings.TrimLeft(statusEVE, fmt.Sprintf("VM %s exist ", vmName))
	return statusEVE, nil
}
