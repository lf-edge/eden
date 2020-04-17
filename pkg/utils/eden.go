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
		err = StopContainer(adamContainerName, true)
		if err != nil {
			return fmt.Errorf("error in rm adam container: %s", err)
		}
		err = CreateAndRunContainer(adamContainerName, adamContainerRef, map[string]string{"8080": adamPort}, map[string]string{"/adam/run": fmt.Sprintf("%s/run", adamPath)}, strings.Fields("server --conf-dir /tmp"))
		if err != nil {
			return fmt.Errorf("error in create adam container: %s", err)
		}
	} else {
		state, err := StateContainer(adamContainerName)
		if err != nil {
			return fmt.Errorf("error in get state of adam container: %s", err)
		}
		if state == "" {
			err = CreateAndRunContainer(adamContainerName, adamContainerRef, map[string]string{"8080": adamPort}, map[string]string{"/adam/run": fmt.Sprintf("%s/run", adamPath)}, strings.Fields("server --conf-dir /tmp"))
			if err != nil {
				return fmt.Errorf("error in create adam container: %s", err)
			}
		} else if state != "running" {
			err = StartContainer(adamContainerName)
			if err != nil {
				return fmt.Errorf("error in restart adam container: %s", err)
			}
		}
	}
	return nil
}

//StopAdam function stop adam container
func StopAdam(adamRm bool) (err error) {
	state, err := StateContainer(adamContainerName)
	if err != nil {
		return fmt.Errorf("error in get state of adam container: %s", err)
	}
	if state != "running" {
		if adamRm {
			err = StopContainer(adamContainerName, true)
			if err != nil {
				return fmt.Errorf("error in rm adam container: %s", err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if adamRm {
			err = StopContainer(adamContainerName, false)
			if err != nil {
				return fmt.Errorf("error in rm adam container: %s", err)
			}
		} else {
			err = StopContainer(adamContainerName, true)
			if err != nil {
				return fmt.Errorf("error in rm adam container: %s", err)
			}
		}
	}
	return nil
}

//StatusAdam function return status of adam
func StatusAdam() (status string, err error) {
	state, err := StateContainer(adamContainerName)
	if err != nil {
		return "", fmt.Errorf("error in get state of adam container: %s", err)
	}
	if state == "" {
		return "adam container not exists", nil
	}
	return state, nil
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
