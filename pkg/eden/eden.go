package eden

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/lf-edge/eden/eserver/api"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/models"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/nerd2/gexto"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

//StartRedis function run redis in docker with mounted redisPath:/data
//if redisForce is set, it recreates container
func StartRedis(redisPort int, redisPath string, redisForce bool, redisTag string) (err error) {
	portMap := map[string]string{"6379": strconv.Itoa(redisPort)}
	volumeMap := map[string]string{"/data": redisPath}
	redisServerCommand := strings.Fields("redis-server --appendonly yes")
	if redisPath != "" {
		if err = os.MkdirAll(redisPath, 0755); err != nil {
			return fmt.Errorf("StartRedis: Cannot create directory for redis (%s): %s", redisPath, err)
		}
	}
	if redisForce {
		_ = utils.StopContainer(defaults.DefaultRedisContainerName, true)
		if err := utils.CreateAndRunContainer(defaults.DefaultRedisContainerName, defaults.DefaultRedisContainerRef+":"+redisTag, portMap, volumeMap, redisServerCommand, nil); err != nil {
			return fmt.Errorf("StartRedis: error in create redis container: %s", err)
		}
	} else {
		state, err := utils.StateContainer(defaults.DefaultRedisContainerName)
		if err != nil {
			return fmt.Errorf("StartRedis: error in get state of redis container: %s", err)
		}
		if state == "" {
			if err := utils.CreateAndRunContainer(defaults.DefaultRedisContainerName, defaults.DefaultRedisContainerRef+":"+redisTag, portMap, volumeMap, redisServerCommand, nil); err != nil {
				return fmt.Errorf("StartRedis: error in create redis container: %s", err)
			}
		} else if !strings.Contains(state, "running") {
			if err := utils.StartContainer(defaults.DefaultRedisContainerName); err != nil {
				return fmt.Errorf("StartRedis: error in restart redis container: %s", err)
			}
		}
	}
	return nil
}

//StopRedis function stop redis container
func StopRedis(redisRm bool) (err error) {
	state, err := utils.StateContainer(defaults.DefaultRedisContainerName)
	if err != nil {
		return fmt.Errorf("StopRedis: error in get state of redis container: %s", err)
	}
	if !strings.Contains(state, "running") {
		if redisRm {
			if err := utils.StopContainer(defaults.DefaultRedisContainerName, true); err != nil {
				return fmt.Errorf("StopRedis: error in rm redis container: %s", err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if redisRm {
			if err := utils.StopContainer(defaults.DefaultRedisContainerName, false); err != nil {
				return fmt.Errorf("StopRedis: error in rm redis container: %s", err)
			}
		} else {
			if err := utils.StopContainer(defaults.DefaultRedisContainerName, true); err != nil {
				return fmt.Errorf("StopRedis: error in rm redis container: %s", err)
			}
		}
	}
	return nil
}

//StatusRedis function return status of redis
func StatusRedis() (status string, err error) {
	state, err := utils.StateContainer(defaults.DefaultRedisContainerName)
	if err != nil {
		return "", fmt.Errorf("StatusRedis: error in get state of redis container: %s", err)
	}
	if state == "" {
		return "container doesn't exist", nil
	}
	return state, nil
}

//StartAdam function run adam in docker with mounted adamPath/run:/adam/run
//if adamForce is set, it recreates container
func StartAdam(adamPort int, adamPath string, adamForce bool, adamTag string, adamRemoteRedisURL string, opts ...string) (err error) {
	edenHome, err := utils.DefaultEdenDir()
	if err != nil {
		return err
	}
	globalCertsDir := filepath.Join(edenHome, defaults.DefaultCertsDist)
	serverCertPath := filepath.Join(globalCertsDir, "server.pem")
	serverKeyPath := filepath.Join(globalCertsDir, "server-key.pem")
	cert, err := ioutil.ReadFile(serverCertPath)
	if err != nil {
		return fmt.Errorf("StartAdam: cannot load %s: %s", serverCertPath, err)
	}
	key, err := ioutil.ReadFile(serverKeyPath)
	if err != nil {
		return fmt.Errorf("StartAdam: cannot load %s: %s", serverKeyPath, err)
	}
	envs := []string{
		fmt.Sprintf("SERVER_CERT=%s", cert),
		fmt.Sprintf("SERVER_KEY=%s", key),
	}
	portMap := map[string]string{"8080": strconv.Itoa(adamPort)}
	volumeMap := map[string]string{"/adam/run": fmt.Sprintf("%s/run", adamPath)}
	adamServerCommand := strings.Fields("server --conf-dir ./run/conf")
	if adamPath == "" {
		volumeMap = map[string]string{"/adam/run": ""}
		adamServerCommand = strings.Fields("server")
	}
	if adamRemoteRedisURL != "" {
		adamServerCommand = append(adamServerCommand, strings.Fields(fmt.Sprintf("--db-url %s", adamRemoteRedisURL))...)
	}
	adamServerCommand = append(adamServerCommand, opts...)
	if adamForce {
		_ = utils.StopContainer(defaults.DefaultAdamContainerName, true)
		if err := utils.CreateAndRunContainer(defaults.DefaultAdamContainerName, defaults.DefaultAdamContainerRef+":"+adamTag, portMap, volumeMap, adamServerCommand, envs); err != nil {
			return fmt.Errorf("StartAdam: error in create adam container: %s", err)
		}
	} else {
		state, err := utils.StateContainer(defaults.DefaultAdamContainerName)
		if err != nil {
			return fmt.Errorf("StartAdam: error in get state of adam container: %s", err)
		}
		if state == "" {
			if err := utils.CreateAndRunContainer(defaults.DefaultAdamContainerName, defaults.DefaultAdamContainerRef+":"+adamTag, portMap, volumeMap, adamServerCommand, envs); err != nil {
				return fmt.Errorf("StartAdam: error in create adam container: %s", err)
			}
		} else if !strings.Contains(state, "running") {
			if err := utils.StartContainer(defaults.DefaultAdamContainerName); err != nil {
				return fmt.Errorf("StartAdam: error in restart adam container: %s", err)
			}
		}
	}
	return nil
}

//StopAdam function stop adam container
func StopAdam(adamRm bool) (err error) {
	state, err := utils.StateContainer(defaults.DefaultAdamContainerName)
	if err != nil {
		return fmt.Errorf("StopAdam: error in get state of adam container: %s", err)
	}
	if !strings.Contains(state, "running") {
		if adamRm {
			if err := utils.StopContainer(defaults.DefaultAdamContainerName, true); err != nil {
				return fmt.Errorf("StopAdam: error in rm adam container: %s", err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if adamRm {
			if err := utils.StopContainer(defaults.DefaultAdamContainerName, false); err != nil {
				return fmt.Errorf("StopAdam: error in rm adam container: %s", err)
			}
		} else {
			if err := utils.StopContainer(defaults.DefaultAdamContainerName, true); err != nil {
				return fmt.Errorf("StopAdam: error in rm adam container: %s", err)
			}
		}
	}
	return nil
}

//StatusAdam function return status of adam
func StatusAdam() (status string, err error) {
	state, err := utils.StateContainer(defaults.DefaultAdamContainerName)
	if err != nil {
		return "", fmt.Errorf("StatusAdam: error in get state of adam container: %s", err)
	}
	if state == "" {
		return "container doesn't exist", nil
	}
	return state, nil
}

//StartRegistry function run registry in docker
func StartRegistry(port int, tag, registryPath string, opts ...string) (err error) {
	containerName := defaults.DefaultRegistryContainerName
	ref := defaults.DefaultRegistryContainerRef
	serviceName := "registry"
	portMap := map[string]string{"5000": strconv.Itoa(port)}
	cmd := []string{}
	cmd = append(cmd, opts...)
	volumeMap := map[string]string{"/var/lib/registry": registryPath}
	state, err := utils.StateContainer(containerName)
	if err != nil {
		return fmt.Errorf("StartRegistry: error in get state of %s container: %s", serviceName, err)
	}
	if state == "" {
		if err := utils.CreateAndRunContainer(containerName, ref+":"+tag, portMap, volumeMap, cmd, nil); err != nil {
			return fmt.Errorf("StartRegistry: error in create %s container: %s", serviceName, err)
		}
	} else if !strings.Contains(state, "running") {
		if err := utils.StartContainer(containerName); err != nil {
			return fmt.Errorf("StartRegistry: error in restart %s container: %s", serviceName, err)
		}
	}
	return nil
}

// StopRegistry function stop registry container
func StopRegistry(rm bool) (err error) {
	containerName := defaults.DefaultRegistryContainerName
	serviceName := "registry"
	state, err := utils.StateContainer(containerName)
	if err != nil {
		return fmt.Errorf("StopRegistry: error in get state of %s container: %s", serviceName, err)
	}
	if !strings.Contains(state, "running") {
		if rm {
			if err := utils.StopContainer(containerName, true); err != nil {
				return fmt.Errorf("StopRegistry: error in rm %s container: %s", serviceName, err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if rm {
			if err := utils.StopContainer(containerName, false); err != nil {
				return fmt.Errorf("StopRegistry: error in rm %s container: %s", serviceName, err)
			}
		} else {
			if err := utils.StopContainer(containerName, true); err != nil {
				return fmt.Errorf("StopRegistry: error in rm %s container: %s", serviceName, err)
			}
		}
	}
	return nil
}

// StatusRegistry function return status of registry
func StatusRegistry() (status string, err error) {
	containerName := defaults.DefaultRegistryContainerName
	serviceName := "registry"
	state, err := utils.StateContainer(containerName)
	if err != nil {
		return "", fmt.Errorf("StatusRegistry: error in get state of %s container: %s", serviceName, err)
	}
	if state == "" {
		return "container doesn't exist", nil
	}
	return state, nil
}

//StartEServer function run eserver in docker
//if eserverForce is set, it recreates container
func StartEServer(serverPort int, imageDist string, eserverForce bool, eserverTag string) (err error) {
	portMap := map[string]string{"8888": strconv.Itoa(serverPort)}
	volumeMap := map[string]string{"/eserver/run/eserver/": imageDist}
	eserverServerCommand := strings.Fields("server")
	// lets make sure eserverImageDist exists
	if imageDist != "" && os.MkdirAll(imageDist, os.ModePerm) != nil {
		return fmt.Errorf("StartEServer: %s does not exist and can not be created", imageDist)
	}
	if eserverForce {
		_ = utils.StopContainer(defaults.DefaultEServerContainerName, true)
		if err := utils.CreateAndRunContainer(defaults.DefaultEServerContainerName, defaults.DefaultEServerContainerRef+":"+eserverTag, portMap, volumeMap, eserverServerCommand, nil); err != nil {
			return fmt.Errorf("StartEServer: error in create eserver container: %s", err)
		}
	} else {
		state, err := utils.StateContainer(defaults.DefaultEServerContainerName)
		if err != nil {
			return fmt.Errorf("StartEServer: error in get state of eserver container: %s", err)
		}
		if state == "" {
			if err := utils.CreateAndRunContainer(defaults.DefaultEServerContainerName, defaults.DefaultEServerContainerRef+":"+eserverTag, portMap, volumeMap, eserverServerCommand, nil); err != nil {
				return fmt.Errorf("StartEServer: error in create eserver container: %s", err)
			}
		} else if !strings.Contains(state, "running") {
			if err := utils.StartContainer(defaults.DefaultEServerContainerName); err != nil {
				return fmt.Errorf("StartEServer: error in restart eserver container: %s", err)
			}
		}
	}
	return nil
}

//StopEServer function stop eserver container
func StopEServer(eserverRm bool) (err error) {
	state, err := utils.StateContainer(defaults.DefaultEServerContainerName)
	if err != nil {
		return fmt.Errorf("StopEServer: error in get state of eserver container: %s", err)
	}
	if !strings.Contains(state, "running") {
		if eserverRm {
			if err := utils.StopContainer(defaults.DefaultEServerContainerName, true); err != nil {
				return fmt.Errorf("StopEServer: error in rm eserver container: %s", err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if eserverRm {
			if err := utils.StopContainer(defaults.DefaultEServerContainerName, false); err != nil {
				return fmt.Errorf("StopEServer: error in rm eserver container: %s", err)
			}
		} else {
			if err := utils.StopContainer(defaults.DefaultEServerContainerName, true); err != nil {
				return fmt.Errorf("StopEServer: error in rm eserver container: %s", err)
			}
		}
	}
	return nil
}

//StatusEServer function return eserver of adam
func StatusEServer() (status string, err error) {
	state, err := utils.StateContainer(defaults.DefaultEServerContainerName)
	if err != nil {
		return "", fmt.Errorf("StatusEServer: error in get eserver of adam container: %s", err)
	}
	if state == "" {
		return "container doesn't exist", nil
	}
	return state, nil
}

//StartEVEVBox function run EVE in VirtualBox
func StartEVEVBox(vmName, eveImageFile string, cpus int, mem int, hostFwd map[string]string) (err error) {
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

		commandArgsString = fmt.Sprintf("modifyvm %s  --nic1 nat --cableconnected1 on", vmName)
		if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
			log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
		}

		for k, v := range hostFwd {
			commandArgsString = fmt.Sprintf("modifyvm %s --nic1 nat --cableconnected1 on --natpf1 ,tcp,,%s,,%s", vmName, k, v)
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
				continue
			}
			if !strings.HasPrefix(line, "Forwarding") {
				continue
			}
			if i := strings.IndexRune(line, '='); i != -1 {
				line = line[i+1:]
			}
			if s, err := strconv.Unquote(line); err == nil {
				line = s
			}
			// forwarding rule is in format "tcp_2222_22,tcp,,2222,,22", where
			v := strings.Split(line, ",")
			commandArgsString := fmt.Sprintf("modifyvm %s --natpf1 delete %s", vmName, v[0])
			if !poweroff {
				commandArgsString = fmt.Sprintf("controlvm %s natpf1 delete %s", vmName, v[0])
			}
			if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
				log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
			}
		}

		for k, v := range hostFwd {
			commandArgsString := fmt.Sprintf("modifyvm %s --nic1 nat --cableconnected1 on --natpf1 ,tcp,,%s,,%s", vmName, k, v)
			if !poweroff {
				commandArgsString = fmt.Sprintf("controlvm %s nic1 nat natpf1 ,tcp,,%s,,%s", vmName, k, v)
			}
			if err = utils.RunCommandWithLogAndWait("VBoxManage", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
				log.Fatalf("VBoxManage error for command %s %s", commandArgsString, err)
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

//StartEVEQemu function run EVE in qemu
func StartEVEQemu(qemuARCH, qemuOS, eveImageFile, qemuSMBIOSSerial string, eveTelnetPort int, qemuHostFwd map[string]string, qemuAccel bool, qemuConfigFile, logFile string, pidFile string, foregroud bool) (err error) {
	qemuCommand := ""
	qemuOptions := fmt.Sprintf("-display none -serial telnet:localhost:%d,server,nowait -nodefaults -no-user-config ", eveTelnetPort)
	if qemuSMBIOSSerial != "" {
		qemuOptions += fmt.Sprintf("-smbios type=1,serial=%s ", qemuSMBIOSSerial)
	}
	nets, err := utils.GetSubnetsNotUsed(2)
	if err != nil {
		return fmt.Errorf("StartEVEQemu: %s", err)
	}
	for ind, n := range nets {
		qemuOptions += fmt.Sprintf("-netdev user,id=eth%d,net=%s,dhcpstart=%s", ind, n.Subnet, n.FirstAddress)
		if ind == 0 {
			for k, v := range qemuHostFwd {
				qemuOptions += fmt.Sprintf(",hostfwd=tcp::%s-:%s", k, v)
			}
		}
		qemuOptions += fmt.Sprintf(" -device virtio-net-pci,netdev=eth%d ", ind)
	}
	if qemuOS == "" {
		qemuOS = runtime.GOOS
	} else {
		qemuOS = strings.ToLower(qemuOS)
	}
	if qemuOS != "linux" && qemuOS != "darwin" {
		return fmt.Errorf("StartEVEQemu: OS not supported: %s", qemuOS)
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
				qemuOptions += defaults.DefaultQemuAccelDarwin
			} else {
				qemuOptions += defaults.DefaultQemuAccelLinux
			}
		} else {
			qemuOptions += "--cpu SandyBridge "
		}
	case "arm64":
		qemuCommand = "qemu-system-aarch64"
		qemuOptions += "-machine virt,gic_version=3 -machine virtualization=true -cpu cortex-a57 -machine type=virt "
	default:
		return fmt.Errorf("StartEVEQemu: Arch not supported: %s", qemuARCH)
	}
	qemuOptions += fmt.Sprintf("-drive file=%s,format=qcow2 ", eveImageFile)
	if qemuConfigFile != "" {
		qemuOptions += fmt.Sprintf("-readconfig %s ", qemuConfigFile)
	}
	log.Infof("Start EVE: %s %s", qemuCommand, qemuOptions)
	if foregroud {
		if err := utils.RunCommandForeground(qemuCommand, strings.Fields(qemuOptions)...); err != nil {
			return fmt.Errorf("StartEVEQemu: %s", err)
		}
	} else {
		log.Infof("With pid: %s ; log: %s", pidFile, logFile)
		if err := utils.RunCommandNohup(qemuCommand, logFile, pidFile, strings.Fields(qemuOptions)...); err != nil {
			return fmt.Errorf("StartEVEQemu: %s", err)
		}
	}
	return nil
}

//StopEVEQemu function stop EVE
func StopEVEQemu(pidFile string) (err error) {
	return utils.StopCommandWithPid(pidFile)
}

//StopEVEVBox function stop EVE in VirtualBox
func StopEVEVBox(vmName string) (err error) {
	commandArgsString := fmt.Sprintf("controlvm %s poweroff", vmName)
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

//DeleteEVEParallels function removes EVE from parallels
func DeleteEVEParallels(vmName string) (err error) {
	commandArgsString := fmt.Sprintf("delete %s", vmName)
	if err = utils.RunCommandWithLogAndWait("prlctl", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		log.Errorf("prlctl error for command %s %s", commandArgsString, err)
	}
	return err
}

//StatusEVEQemu function get status of EVE
func StatusEVEQemu(pidFile string) (status string, err error) {
	return utils.StatusCommandWithPid(pidFile)
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

//GenerateEveCerts function generates certs for EVE
func GenerateEveCerts(certsDir, domain, ip, eveIP, uuid, devModel, ssid, password string) (err error) {
	model, err := models.GetDevModelByName(devModel)
	if err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	if _, err := os.Stat(certsDir); os.IsNotExist(err) {
		if err = os.MkdirAll(certsDir, 0755); err != nil {
			return fmt.Errorf("GenerateEveCerts: %s", err)
		}
	}
	edenHome, err := utils.DefaultEdenDir()
	if err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	globalCertsDir := filepath.Join(edenHome, defaults.DefaultCertsDist)
	if _, err := os.Stat(globalCertsDir); os.IsNotExist(err) {
		if err = os.MkdirAll(globalCertsDir, 0755); err != nil {
			return fmt.Errorf("GenerateEveCerts: %s", err)
		}
	}
	log.Debug("generating CA")
	caCertPath := filepath.Join(globalCertsDir, "root-certificate.pem")
	caKeyPath := filepath.Join(globalCertsDir, "root-certificate-key.pem")
	rootCert, rootKey := utils.GenCARoot()
	if _, err := tls.LoadX509KeyPair(caCertPath, caKeyPath); err == nil { //existing certs looks ok
		log.Info("Use existing certs")
		rootCert, err = utils.ParseCertificate(caCertPath)
		if err != nil {
			return fmt.Errorf("GenerateEveCerts: cannot parse certificate from %s: %s", caCertPath, err)
		}
		rootKey, err = utils.ParsePrivateKey(caKeyPath)
		if err != nil {
			return fmt.Errorf("GenerateEveCerts: cannot parse key from %s: %s", caKeyPath, err)
		}
	}
	if err := utils.WriteToFiles(rootCert, rootKey, caCertPath, caKeyPath); err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	serverCertPath := filepath.Join(globalCertsDir, "server.pem")
	serverKeyPath := filepath.Join(globalCertsDir, "server-key.pem")
	if _, err := tls.LoadX509KeyPair(serverCertPath, serverKeyPath); err != nil {
		log.Debug("generating Adam cert and key")
		ips := []net.IP{net.ParseIP(ip), net.ParseIP(eveIP), net.ParseIP("127.0.0.1")}
		ServerCert, ServerKey := utils.GenServerCert(rootCert, rootKey, big.NewInt(1), ips, []string{domain}, domain)
		if err := utils.WriteToFiles(ServerCert, ServerKey, serverCertPath, serverKeyPath); err != nil {
			return fmt.Errorf("GenerateEveCerts: %s", err)
		}
	}
	log.Debug("generating EVE cert and key")
	if err := utils.CopyFile(caCertPath, filepath.Join(certsDir, "root-certificate.pem")); err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	ClientCert, ClientKey := utils.GenServerCert(rootCert, rootKey, big.NewInt(2), nil, nil, uuid)
	log.Debug("saving files")
	if err := utils.WriteToFiles(ClientCert, ClientKey, filepath.Join(certsDir, "onboard.cert.pem"), filepath.Join(certsDir, "onboard.key.pem")); err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	log.Debug("generating ssh pair")
	if err := utils.GenerateSSHKeyPair(filepath.Join(certsDir, "id_rsa"), filepath.Join(certsDir, "id_rsa.pub")); err != nil {
		return fmt.Errorf("GenerateEveCerts: %s", err)
	}
	if ssid != "" && password != "" {
		log.Debug("generating DevicePortConfig")
		if portConfig := model.GetPortConfig(ssid, password); portConfig != "" {
			if _, err := os.Stat(filepath.Join(certsDir, "DevicePortConfig", "override.json")); os.IsNotExist(err) {
				if err := os.MkdirAll(filepath.Join(certsDir, "DevicePortConfig"), 0755); err != nil {
					return fmt.Errorf("GenerateEveCerts: %s", err)
				}
				if err := ioutil.WriteFile(filepath.Join(certsDir, "DevicePortConfig", "override.json"), []byte(portConfig), 0666); err != nil {
					return fmt.Errorf("GenerateEveCerts: %s", err)
				}
			}
		}
	}
	if _, err := os.Stat(certsDir); os.IsNotExist(err) {
		if err = os.MkdirAll(certsDir, 0755); err != nil {
			return err
		}
	}
	return nil
}

//GenerateEVEConfig function copy certs to EVE config folder
func GenerateEVEConfig(eveConfig string, domain string, ip string, port int, apiV1 bool) (err error) {
	if _, err = os.Stat(eveConfig); os.IsNotExist(err) {
		if err = os.MkdirAll(eveConfig, 0755); err != nil {
			return fmt.Errorf("GenerateEVEConfig: %s", err)
		}
	}
	if _, err = os.Stat(filepath.Join(eveConfig, "hosts")); os.IsNotExist(err) {
		if err = ioutil.WriteFile(filepath.Join(eveConfig, "hosts"), []byte(fmt.Sprintf("%s %s\n", ip, domain)), 0666); err != nil {
			return fmt.Errorf("GenerateEVEConfig: %s", err)
		}
	}
	if apiV1 {
		if _, err = os.Stat(filepath.Join(eveConfig, "Force-API-V1")); os.IsNotExist(err) {
			if err := utils.TouchFile(filepath.Join(eveConfig, "Force-API-V1")); err != nil {
				return fmt.Errorf("GenerateEVEConfig: %s", err)
			}
		}
	}
	if _, err = os.Stat(filepath.Join(eveConfig, "server")); os.IsNotExist(err) {
		if err = ioutil.WriteFile(filepath.Join(eveConfig, "server"), []byte(fmt.Sprintf("%s:%d\n", domain, port)), 0666); err != nil {
			return fmt.Errorf("GenerateEVEConfig: %s", err)
		}
	}
	return nil
}

//CloneFromGit function clone from git into dist
func CloneFromGit(dist string, gitRepo string, tag string) (err error) {
	if _, err := os.Stat(dist); !os.IsNotExist(err) {
		return fmt.Errorf("CloneFromGit: directory already exists: %s", dist)
	}
	if tag == "" {
		tag = "master"
	}
	commandArgsString := fmt.Sprintf("clone --branch %s --single-branch %s %s", tag, gitRepo, dist)
	log.Infof("CloneFromGit run: %s %s", "git", commandArgsString)
	return utils.RunCommandWithLogAndWait("git", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
}

//MakeEveInRepo build live image of EVE
func MakeEveInRepo(distEve string, configPath string, arch string, hv string, imageFormat string, rootFSOnly bool) (image, additional string, err error) {
	if _, err := os.Stat(distEve); os.IsNotExist(err) {
		return "", "", fmt.Errorf("MakeEveInRepo: directory not exists: %s", distEve)
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err = os.MkdirAll(configPath, 0755); err != nil {
			return "", "", fmt.Errorf("MakeEveInRepo: %s", err)
		}
	}
	if rootFSOnly {
		commandArgsString := fmt.Sprintf("-C %s ZARCH=%s HV=%s CONF_DIR=%s rootfs",
			distEve, arch, hv, configPath)
		log.Infof("MakeEveInRepo run: %s %s", "make", commandArgsString)
		err = utils.RunCommandWithLogAndWait("make", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
		image = filepath.Join(distEve, "dist", arch, "installer", fmt.Sprintf("live.%s", imageFormat))
	} else {
		image = filepath.Join(distEve, "dist", arch, fmt.Sprintf("live.%s", imageFormat))
		if imageFormat == "gcp" {
			image = filepath.Join(distEve, "dist", arch, "live.img.tar.gz")
		}
		commandArgsString := fmt.Sprintf("-C %s ZARCH=%s HV=%s CONF_DIR=%s IMG_FORMAT=%s live",
			distEve, arch, hv, configPath, imageFormat)
		log.Infof("MakeEveInRepo run: %s %s", "make", commandArgsString)
		if err = utils.RunCommandWithLogAndWait("make", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
			log.Info(err)
		}
		switch arch {
		case "amd64":
			biosPath1 := filepath.Join(distEve, "dist", arch, "OVMF.fd")
			biosPath2 := filepath.Join(distEve, "dist", arch, "OVMF_CODE.fd")
			biosPath3 := filepath.Join(distEve, "dist", arch, "OVMF_VARS.fd")
			commandArgsString = fmt.Sprintf("-C %s ZARCH=%s HV=%s %s %s %s",
				distEve, arch, hv, biosPath1, biosPath2, biosPath3)
			log.Infof("MakeEveInRepo run: %s %s", "make", commandArgsString)
			err = utils.RunCommandWithLogAndWait("make", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
			additional = strings.Join([]string{biosPath1, biosPath2, biosPath3}, ",")
		case "arm64":
			dtbPath := filepath.Join(distEve, "dist", arch, "dtb", "eve.dtb")
			commandArgsString = fmt.Sprintf("-C %s ZARCH=%s HV=%s %s",
				distEve, arch, hv, dtbPath)
			log.Infof("MakeEveInRepo run: %s %s", "make", commandArgsString)
			err = utils.RunCommandWithLogAndWait("make", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
			additional = dtbPath
		default:
			return "", "", fmt.Errorf("MakeEveInRepo: unsupported arch %s", arch)
		}
	}
	return
}

//CleanContext cleanup only context data
func CleanContext(eveDist, certsDist, imagesDist, evePID, eveUUID, vmName string, configSaved string, remote bool) (err error) {
	edenDir, err := utils.DefaultEdenDir()
	if err != nil {
		return fmt.Errorf("CleanContext: %s", err)
	}
	eveStatusFile := filepath.Join(edenDir, fmt.Sprintf("state-%s.yml", eveUUID))
	if _, err = os.Stat(eveStatusFile); !os.IsNotExist(err) {
		ctrl, err := controller.CloudPrepare()
		if err != nil {
			return fmt.Errorf("CleanContext: error in CloudPrepare: %s", err)
		}
		log.Debugf("Get devUUID for onboardUUID %s", eveUUID)
		devUUID, err := ctrl.DeviceGetByOnboardUUID(eveUUID)
		if err != nil {
			return fmt.Errorf("CleanContext: %s", err)
		}
		log.Debugf("Deleting devUUID %s", devUUID)
		if err := ctrl.DeviceRemove(devUUID); err != nil {
			return fmt.Errorf("CleanContext: %s", err)
		}
		log.Debugf("Deleting onboardUUID %s", eveUUID)
		if err := ctrl.OnboardRemove(eveUUID); err != nil {
			return fmt.Errorf("CleanContext: %s", err)
		}
		localViper := viper.New()
		localViper.SetConfigFile(eveStatusFile)
		if err := localViper.ReadInConfig(); err != nil {
			log.Debug(err)
		} else {
			eveConfigFile := localViper.GetString("eve-config")
			if _, err = os.Stat(eveConfigFile); !os.IsNotExist(err) {
				if err := os.Remove(eveConfigFile); err != nil {
					log.Debug(err)
				}
			}
		}
		if err = os.RemoveAll(eveStatusFile); err != nil {
			return fmt.Errorf("CleanContext: error in %s delete: %s", eveStatusFile, err)
		}
	}
	if !remote {
		if viper.GetString("eve.devModel") == defaults.DefaultVBoxModel {
			if err := StopEVEVBox(vmName); err != nil {
				log.Infof("cannot stop EVE: %s", err)
			} else {
				log.Infof("EVE stopped")
			}
			if err := DeleteEVEVBox(vmName); err != nil {
				log.Infof("cannot delete EVE: %s", err)
			}
		} else if viper.GetString("eve.devModel") == defaults.DefaultParallelsModel {
			if err := StopEVEParallels(vmName); err != nil {
				log.Infof("cannot stop EVE: %s", err)
			} else {
				log.Infof("EVE stopped")
			}
			if err := DeleteEVEParallels(vmName); err != nil {
				log.Infof("cannot delete EVE: %s", err)
			}
		} else {
			if err := StopEVEQemu(evePID); err != nil {
				log.Infof("cannot stop EVE: %s", err)
			} else {
				log.Infof("EVE stopped")
			}
		}
	}
	if _, err = os.Stat(eveDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(eveDist); err != nil {
			return fmt.Errorf("CleanContext: error in %s delete: %s", eveDist, err)
		}
	}
	if _, err = os.Stat(certsDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(certsDist); err != nil {
			return fmt.Errorf("CleanContext: error in %s delete: %s", certsDist, err)
		}
	}
	if _, err = os.Stat(imagesDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(imagesDist); err != nil {
			return fmt.Errorf("CleanContext: error in %s delete: %s", imagesDist, err)
		}
	}
	if _, err = os.Stat(configSaved); !os.IsNotExist(err) {
		if err = os.RemoveAll(configSaved); err != nil {
			return fmt.Errorf("CleanContext: error in %s delete: %s", configSaved, err)
		}
	}
	return nil
}

//StopEden teardown Eden
func StopEden(adamRm, redisRm, registryRm, eserverRm, eveRemote bool, evePidFile string, devModel string, vmName string) {
	if err := StopAdam(adamRm); err != nil {
		log.Infof("cannot stop adam: %s", err)
	} else {
		log.Infof("adam stopped")
	}
	if err := StopRedis(redisRm); err != nil {
		log.Infof("cannot stop redis: %s", err)
	} else {
		log.Infof("redis stopped")
	}
	if err := StopRegistry(registryRm); err != nil {
		log.Infof("cannot stop registry: %s", err)
	} else {
		log.Infof("registry stopped")
	}
	if err := StopEServer(eserverRm); err != nil {
		log.Infof("cannot stop eserver: %s", err)
	} else {
		log.Infof("eserver stopped")
	}
	if !eveRemote {
		if devModel == defaults.DefaultVBoxModel {
			if err := StopEVEVBox(vmName); err != nil {
				log.Infof("cannot stop EVE: %s", err)
			} else {
				log.Infof("EVE stopped")
			}
		} else if devModel == defaults.DefaultParallelsModel {
			if err := StopEVEParallels(vmName); err != nil {
				log.Infof("cannot stop EVE: %s", err)
			} else {
				log.Infof("EVE stopped")
			}
		} else {
			if err := StopEVEQemu(evePidFile); err != nil {
				log.Infof("cannot stop EVE: %s", err)
			} else {
				log.Infof("EVE stopped")
			}
		}
	}
}

//CleanEden teardown Eden and cleanup
func CleanEden(eveDist, adamDist, certsDist, imagesDist, eserverDist, redisDist, registryDist, configDir, evePID string, configSaved string, remote bool, devModel string, vmName string) (err error) {
	StopEden(true, true, true, true, remote, evePID, devModel, vmName)
	if _, err = os.Stat(eveDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(eveDist); err != nil {
			return fmt.Errorf("CleanEden: error in %s delete: %s", eveDist, err)
		}
	}
	if _, err = os.Stat(certsDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(certsDist); err != nil {
			return fmt.Errorf("CleanEden: error in %s delete: %s", certsDist, err)
		}
	}
	if _, err = os.Stat(imagesDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(imagesDist); err != nil {
			return fmt.Errorf("CleanEden: error in %s delete: %s", imagesDist, err)
		}
	}
	if _, err = os.Stat(eserverDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(eserverDist); err != nil {
			return fmt.Errorf("CleanEden: error in %s delete: %s", eserverDist, err)
		}
	}
	if _, err = os.Stat(adamDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(adamDist); err != nil {
			return fmt.Errorf("CleanEden: error in %s delete: %s", adamDist, err)
		}
	}
	if _, err = os.Stat(redisDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(redisDist); err != nil {
			return fmt.Errorf("CleanEden: error in %s delete: %s", redisDist, err)
		}
	}
	if _, err = os.Stat(registryDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(registryDist); err != nil {
			return fmt.Errorf("CleanEden: error in %s delete: %s", registryDist, err)
		}
	}
	if _, err = os.Stat(configDir); !os.IsNotExist(err) {
		if err = os.RemoveAll(configDir); err != nil {
			return fmt.Errorf("CleanEden: error in %s delete: %s", configDir, err)
		}
	}
	if _, err = os.Stat(configSaved); !os.IsNotExist(err) {
		if err = os.RemoveAll(configSaved); err != nil {
			return fmt.Errorf("CleanEden: error in %s delete: %s", configSaved, err)
		}
	}
	if err = utils.RemoveGeneratedVolumeOfContainer(defaults.DefaultEServerContainerName); err != nil {
		return fmt.Errorf("CleanEden: RemoveGeneratedVolumeOfContainer for %s: %s", defaults.DefaultEServerContainerName, err)
	}
	if err = utils.RemoveGeneratedVolumeOfContainer(defaults.DefaultRedisContainerName); err != nil {
		return fmt.Errorf("CleanEden: RemoveGeneratedVolumeOfContainer for %s: %s", defaults.DefaultRedisContainerName, err)
	}
	if err = utils.RemoveGeneratedVolumeOfContainer(defaults.DefaultAdamContainerName); err != nil {
		return fmt.Errorf("CleanEden: RemoveGeneratedVolumeOfContainer for %s: %s", defaults.DefaultAdamContainerName, err)
	}
	if err = utils.RemoveGeneratedVolumeOfContainer(defaults.DefaultRegistryContainerName); err != nil {
		return fmt.Errorf("CleanEden: RemoveGeneratedVolumeOfContainer for %s: %s", defaults.DefaultRegistryContainerName, err)
	}
	if devModel == defaults.DefaultVBoxModel {
		if err := DeleteEVEVBox(vmName); err != nil {
			log.Infof("cannot delete EVE: %s", err)
		}
	} else if devModel == defaults.DefaultParallelsModel {
		if err := DeleteEVEParallels(vmName); err != nil {
			log.Infof("cannot delete EVE: %s", err)
		}
	}
	return nil
}

//EServer for connection to eserver
type EServer struct {
	EServerIP   string
	EServerPort string
}

func (server *EServer) getHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			ResponseHeaderTimeout: defaults.DefaultRepeatTimeout * defaults.DefaultRepeatCount,
		},
	}
}

//EServerAddFileURL send url to download image into eserver
func (server *EServer) EServerAddFileURL(url string) (name string) {
	u, err := utils.ResolveURL(fmt.Sprintf("http://%s:%s", server.EServerIP, server.EServerPort), "admin/add-from-url")
	if err != nil {
		log.Fatalf("error constructing URL: %v", err)
	}
	client := server.getHTTPClient(defaults.DefaultRepeatTimeout)
	objToSend := api.URLArg{
		URL: url,
	}
	body, err := json.Marshal(objToSend)
	if err != nil {
		log.Fatalf("EServerAddFileURL: error encoding json: %v", err)
	}
	req, err := http.NewRequest("POST", u, bytes.NewBuffer(body))
	if err != nil {
		log.Fatalf("EServerAddFileURL: unable to create new http request: %v", err)
	}

	response, err := utils.RepeatableAttempt(client, req)
	if err != nil {
		log.Fatalf("EServerAddFileURL: unable to send request: %v", err)
	}
	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("EServerAddFileURL: unable to read data from URL %s: %v", u, err)
	}
	return string(buf)
}

//EServerCheckStatus checks status of image in eserver
func (server *EServer) EServerCheckStatus(name string) (fileInfo *api.FileInfo) {
	u, err := utils.ResolveURL(fmt.Sprintf("http://%s:%s", server.EServerIP, server.EServerPort), fmt.Sprintf("admin/status/%s", name))
	if err != nil {
		log.Fatalf("EServerAddFileURL: error constructing URL: %v", err)
	}
	client := server.getHTTPClient(defaults.DefaultRepeatTimeout * defaults.DefaultRepeatCount)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		log.Fatalf("EServerAddFileURL: unable to create new http request: %v", err)
	}

	response, err := utils.RepeatableAttempt(client, req)
	if err != nil {
		log.Fatalf("EServerAddFileURL: unable to send request: %v", err)
	}
	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("EServerAddFileURL: unable to read data from URL %s: %v", u, err)
	}
	if err := json.Unmarshal(buf, &fileInfo); err != nil {
		log.Fatalf("EServerAddFileURL: %s", err)
	}
	return
}

//EServerAddFile send file with image into eserver
func (server *EServer) EServerAddFile(filepath string) (fileInfo *api.FileInfo) {
	u, err := utils.ResolveURL(fmt.Sprintf("http://%s:%s", server.EServerIP, server.EServerPort), "admin/add-from-file")
	if err != nil {
		log.Fatalf("EServerAddFile: error constructing URL: %v", err)
	}
	client := server.getHTTPClient(0)
	response, err := utils.UploadFile(client, u, filepath)
	if err != nil {
		log.Fatalf("EServerAddFile: %s", err)
	}
	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("EServerAddFile: unable to read data from URL %s: %v", u, err)
	}
	if err := json.Unmarshal(buf, &fileInfo); err != nil {
		log.Fatalf("EServerAddFile: %s", err)
	}
	return
}

//ReadFileInSquashFS returns the content of a single file (filePath) inside squashfs (squashFSPath)
func ReadFileInSquashFS(squashFSPath, filePath string) (content []byte, err error) {
	tmpdir, err := ioutil.TempDir("", "squashfs-unpack")
	if err != nil {
		return nil, fmt.Errorf("ReadFileInSquashFS: %s", err)
	}
	defer os.RemoveAll(tmpdir)
	dirToUnpack := filepath.Join(tmpdir, "temp")
	if output, err := exec.Command("unsquashfs", "-n", "-i", "-d", dirToUnpack, squashFSPath, filePath).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("ReadFileInSquashFS: unsquashfs (%s): %v", output, err)
	}
	content, err = ioutil.ReadFile(filepath.Join(dirToUnpack, filePath))
	if err != nil {
		return nil, fmt.Errorf("ReadFileInSquashFS: %s", err)
	}
	return content, nil
}

//EVEInfo contains info from SD card
type EVEInfo struct {
	EVERelease []byte //EVERelease is /etc/eve-release from rootfs
	Syslog     []byte //Syslog is /rsyslog/syslog.txt from persist volume
}

//GetInfoFromSDCard obtain info from SD card with EVE
// devicePath is /dev/sdX or /dev/diskX
func GetInfoFromSDCard(devicePath string) (eveInfo *EVEInfo, err error) {
	eveInfo = &EVEInfo{}
	// /dev/sdXN notation by default
	rootfsDev := fmt.Sprintf("%s2", devicePath)
	persistDev := fmt.Sprintf("%s9", devicePath)
	// /dev/diskXsN notation for MacOS
	if runtime.GOOS == `darwin` {
		rootfsDev = fmt.Sprintf("%ss2", devicePath)
		persistDev = fmt.Sprintf("%ss9", devicePath)
	}
	if _, err := os.Stat(devicePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("GetInfoFromSDCard: %s", err)
	}
	if _, err := os.Stat(rootfsDev); !os.IsNotExist(err) {
		eveRelease, err := ReadFileInSquashFS(rootfsDev, "/etc/eve-release")
		if err != nil {
			return nil, fmt.Errorf("GetInfoFromSDCard: %s", err)
		}
		eveInfo.EVERelease = eveRelease
	}
	if _, err := os.Stat(persistDev); !os.IsNotExist(err) {
		fsPersist, err := gexto.NewFileSystem(persistDev)
		if err != nil {
			return nil, fmt.Errorf("GetInfoFromSDCard: %s", err)
		}
		g, err := fsPersist.Open("/rsyslog/syslog.txt")
		if err != nil {
			return nil, fmt.Errorf("GetInfoFromSDCard: %s", err)
		}
		syslog, err := ioutil.ReadAll(g)
		if err != nil {
			return nil, fmt.Errorf("GetInfoFromSDCard: %s", err)
		}
		eveInfo.Syslog = syslog
	}
	return eveInfo, nil
}

//AddFileIntoEServer puts file into eserver
func AddFileIntoEServer(server *EServer, filePath string) (*api.FileInfo, error) {
	status := server.EServerCheckStatus(filepath.Base(filePath))
	if !status.ISReady || status.Size != utils.GetFileSize(filePath) {
		log.Infof("Start uploading into eserver of %s", filePath)
		status = server.EServerAddFile(filePath)
		if status.Error != "" {
			return nil, fmt.Errorf("AddFileIntoEServer: %s", status.Error)
		}
	}
	return status, nil
}
