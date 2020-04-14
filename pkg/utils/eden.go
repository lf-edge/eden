package utils

import (
	"fmt"
	"strings"
)

const (
	adamContainerName = "eden_adam"
	adamContainerRef  = "lfedge/adam"
)

//StartAdam function run adam in docker with mounted adamPath/run:/adam/run
//if adamForce is set, it recreates container
func StartAdam(adamPort string, adamPath string, adamForce bool) (err error) {
	if adamForce {
		_, _, err = RunCommandAndWait("docker", strings.Fields(fmt.Sprintf("rm -f %s", adamContainerName))...)
		if err != nil {
			return fmt.Errorf("error in rm adam container: %s", err)
		}
		stringArgs := fmt.Sprintf("run --name %s -d -v %s/run:/adam/run -p %s:8080 %s server --conf-dir /tmp", adamContainerName, adamPath, adamPort, adamContainerRef)
		_, _, err = RunCommandAndWait("docker", strings.Fields(stringArgs)...)
		if err != nil {
			return fmt.Errorf("error in create adam container: %s", err)
		}
	} else {
		_, stderr, err := RunCommandAndWait("docker", strings.Fields(fmt.Sprintf("top %s", adamContainerName))...)
		if strings.Contains(stderr, "is not running") {
			_, _, err = RunCommandAndWait("docker", strings.Fields(fmt.Sprintf("start %s", adamContainerName))...)
			if err != nil {
				return fmt.Errorf("error in restart adam container: %s", err)
			}
		} else if strings.Contains(stderr, "No such container") {
			stringArgs := fmt.Sprintf("run --name %s -d -v %s/run:/adam/run -p %s:8080 %s server --conf-dir /tmp", adamContainerName, adamPath, adamPort, adamContainerRef)
			_, _, err = RunCommandAndWait("docker", strings.Fields(stringArgs)...)
			if err != nil {
				return fmt.Errorf("error in create adam container: %s", err)
			}
		}
	}
	return nil
}

//StopAdam function stop adam container
func StopAdam(adamRm bool) (err error) {
	_, stderr, err := RunCommandAndWait("docker", strings.Fields(fmt.Sprintf("top %s", adamContainerName))...)
	if strings.Contains(stderr, "is not running") {
		if adamRm {
			_, _, err = RunCommandAndWait("docker", strings.Fields(fmt.Sprintf("rm %s", adamContainerName))...)
			if err != nil {
				return fmt.Errorf("error in rm adam container: %s", err)
			}
		}
	} else if strings.Contains(stderr, "No such container") {
		return nil
	} else {
		if adamRm {
			_, _, err = RunCommandAndWait("docker", strings.Fields(fmt.Sprintf("stop %s", adamContainerName))...)
			if err != nil {
				return fmt.Errorf("error in rm adam container: %s", err)
			}
		} else {
			_, _, err = RunCommandAndWait("docker", strings.Fields(fmt.Sprintf("rm -f %s", adamContainerName))...)
			if err != nil {
				return fmt.Errorf("error in rm adam container: %s", err)
			}
		}
	}
	return nil
}

//StatusAdam function return status of adam
func StatusAdam() (status string, err error) {
	_, stderr, err := RunCommandAndWait("docker", strings.Fields(fmt.Sprintf("top %s", adamContainerName))...)
	if strings.Contains(stderr, "is not running") {
		return "adam container not running", nil
	} else if strings.Contains(stderr, "No such container") {
		return "adam container not exists", nil
	}
	if err == nil {
		return "adam container is running", nil
	} else {
		return "", fmt.Errorf("error in top adam container: %s", err)
	}
}

//StartEServer function run eserver to serve images
func StartEServer(commandPath string, serverPort string, imageDist string, logFile string, pidFile string) (err error) {
	return RunCommandNohup(commandPath, logFile, pidFile, strings.Fields(fmt.Sprintf("server -p %s -d %s", serverPort, imageDist))...)
}

//StopEServer function stop eserver
func StopEServer(pidFile string) (err error) {
	return StopCommandWithPid(pidFile)
}

//StatusEServer function get status of eserver
func StatusEServer(pidFile string) (status string, err error) {
	return StatusCommandWithPid(pidFile)
}

//StartEVEQemu function run EVE in qemu
func StartEVEQemu(commandPath string, qemuARCH string, qemuOS string, qemuSMBIOSSerial string, qemuAccel bool, qemuConfigFilestring, logFile string, pidFile string) (err error) {
	_, _, err = RunCommandAndWait(commandPath, strings.Fields(fmt.Sprintf("qemurun --config=%s --serial=%s --accel=%t --arch=%s --os=%s --qemu-log=%s --qemu-pid=%s", qemuConfigFilestring, qemuSMBIOSSerial, qemuAccel, qemuARCH, qemuOS, logFile, pidFile))...)
	return
}

//StopEVEQemu function stop EVE
func StopEVEQemu(pidFile string) (err error) {
	return StopCommandWithPid(pidFile)
}

//StatusEVEQemu function get status of EVE
func StatusEVEQemu(pidFile string) (status string, err error) {
	return StatusCommandWithPid(pidFile)
}
