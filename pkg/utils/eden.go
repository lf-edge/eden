package utils

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	adamContainerName = "eden_adam"
	adamContainerRef  = "lfedge/adam"
	logLevelToPrint   = log.InfoLevel
)

//StartAdam function run adam in docker with mounted adamPath/run:/adam/run
//if adamForce is set, it recreates container
func StartAdam(adamPort string, adamPath string, adamForce bool) (err error) {
	portMap := map[string]string{"8080": adamPort}
	volumeMap := map[string]string{"/adam/run": fmt.Sprintf("%s/run", adamPath)}
	adamServerCommand := strings.Fields("server --conf-dir ./run/conf")
	if adamForce {
		_ = StopContainer(adamContainerName, true)
		if err := CreateAndRunContainer(adamContainerName, adamContainerRef, portMap, volumeMap, adamServerCommand); err != nil {
			return fmt.Errorf("error in create adam container: %s", err)
		}
	} else {
		state, err := StateContainer(adamContainerName)
		if err != nil {
			return fmt.Errorf("error in get state of adam container: %s", err)
		}
		if state == "" {
			if err := CreateAndRunContainer(adamContainerName, adamContainerRef, portMap, volumeMap, adamServerCommand); err != nil {
				return fmt.Errorf("error in create adam container: %s", err)
			}
		} else if state != "running" {
			if err := StartContainer(adamContainerName); err != nil {
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
			if err := StopContainer(adamContainerName, true); err != nil {
				return fmt.Errorf("error in rm adam container: %s", err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if adamRm {
			if err := StopContainer(adamContainerName, false); err != nil {
				return fmt.Errorf("error in rm adam container: %s", err)
			}
		} else {
			if err := StopContainer(adamContainerName, true); err != nil {
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
	commandArgsString := fmt.Sprintf("server -p %s -d %s -v %s", serverPort, imageDist, log.GetLevel())
	log.Infof("StartEServer run: %s %s", commandPath, commandArgsString)
	return RunCommandNohup(commandPath, logFile, pidFile, strings.Fields(commandArgsString)...)
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
func StartEVEQemu(commandPath string, qemuARCH string, qemuOS string, eveImageFile string, qemuSMBIOSSerial string, qemuAccel bool, qemuConfigFilestring, logFile string, pidFile string) (err error) {
	commandArgsString := fmt.Sprintf("eve start --qemu-config=%s --eve-serial=%s --eve-accel=%t --eve-arch=%s --eve-os=%s --eve-log=%s --eve-pid=%s --image-file=%s -v %s",
		qemuConfigFilestring, qemuSMBIOSSerial, qemuAccel, qemuARCH, qemuOS, logFile, pidFile, eveImageFile, log.GetLevel())
	log.Infof("StartEVEQemu run: %s %s", commandPath, commandArgsString)
	return RunCommandWithLogAndWait(commandPath, logLevelToPrint, strings.Fields(commandArgsString)...)
}

//StopEVEQemu function stop EVE
func StopEVEQemu(pidFile string) (err error) {
	return StopCommandWithPid(pidFile)
}

//StatusEVEQemu function get status of EVE
func StatusEVEQemu(pidFile string) (status string, err error) {
	return StatusCommandWithPid(pidFile)
}

//GenerateEveCerts function generates certs for EVE
func GenerateEveCerts(commandPath string, certsDir string, domain string, ip string, eveIP string, uuid string) (err error) {
	if _, err := os.Stat(certsDir); os.IsNotExist(err) {
		if err = os.MkdirAll(certsDir, 0755); err != nil {
			return err
		}
	}
	commandArgsString := fmt.Sprintf("certs --certs-dist=%s --domain=%s --ip=%s --eve-ip=%s --uuid=%s -v %s", certsDir, domain, ip, eveIP, uuid, log.GetLevel())
	log.Infof("GenerateEveCerts run: %s %s", commandPath, commandArgsString)
	return RunCommandWithLogAndWait(commandPath, logLevelToPrint, strings.Fields(commandArgsString)...)
}

//CopyCertsToAdamConfig function copy certs to adam config
func CopyCertsToAdamConfig(certsDir string, domain string, ip string, port string, adamDist string) (err error) {
	adamConfig := filepath.Join(adamDist, "run", "config")
	adamServer := filepath.Join(adamDist, "run", "adam")
	if _, err = os.Stat(filepath.Join(certsDir, "server.pem")); os.IsNotExist(err) {
		return err
	}
	if _, err = os.Stat(adamConfig); os.IsNotExist(err) {
		if err = os.MkdirAll(adamConfig, 0755); err != nil {
			return err
		}
	}
	if _, err = os.Stat(adamServer); os.IsNotExist(err) {
		if err = os.MkdirAll(adamServer, 0755); err != nil {
			return err
		}
	}
	if err = CopyFileNotExists(filepath.Join(certsDir, "server.pem"), filepath.Join(adamServer, "server.pem")); err != nil {
		return err
	}
	if err = CopyFileNotExists(filepath.Join(certsDir, "server-key.pem"), filepath.Join(adamServer, "server-key.pem")); err != nil {
		return err
	}
	if err = CopyFileNotExists(filepath.Join(certsDir, "root-certificate.pem"), filepath.Join(adamConfig, "root-certificate.pem")); err != nil {
		return err
	}
	if err = CopyFileNotExists(filepath.Join(certsDir, "onboard.cert.pem"), filepath.Join(adamConfig, "onboard.cert.pem")); err != nil {
		return err
	}
	if err = CopyFileNotExists(filepath.Join(certsDir, "onboard.key.pem"), filepath.Join(adamConfig, "onboard.key.pem")); err != nil {
		return err
	}
	if err = CopyFileNotExists(filepath.Join(certsDir, "id_rsa.pub"), filepath.Join(adamConfig, "authorized_keys")); err != nil {
		return err
	}
	if _, err = os.Stat(filepath.Join(adamConfig, "hosts")); os.IsNotExist(err) {
		if err = ioutil.WriteFile(filepath.Join(adamConfig, "hosts"), []byte(fmt.Sprintf("%s %s\n", ip, domain)), 0666); err != nil {
			return err
		}
	}
	if _, err = os.Stat(filepath.Join(adamConfig, "server")); os.IsNotExist(err) {
		if err = ioutil.WriteFile(filepath.Join(adamConfig, "server"), []byte(fmt.Sprintf("%s:%s\n", domain, port)), 0666); err != nil {
			return err
		}
	}
	return nil
}

//CloneFromGit function clone from git into dist
func CloneFromGit(dist string, gitRepo string, tag string) (err error) {
	if _, err := os.Stat(dist); !os.IsNotExist(err) {
		return fmt.Errorf("directory already exists: %s", dist)
	}
	if tag == "" {
		tag = "master"
	}
	commandArgsString := fmt.Sprintf("clone --branch %s --single-branch %s %s", tag, gitRepo, dist)
	log.Infof("CloneFromGit run: %s %s", "git", commandArgsString)
	return RunCommandWithLogAndWait("git", logLevelToPrint, strings.Fields(commandArgsString)...)
}

//DownloadEveFormDocker function clone EVE from docker
func DownloadEveFormDocker(commandPath string, dist string, arch string, tag string, baseOs bool) (err error) {
	if _, err := os.Stat(dist); !os.IsNotExist(err) {
		return fmt.Errorf("directory already exists: %s", dist)
	}
	if tag == "" {
		tag = "latest"
	}
	commandArgsString := fmt.Sprintf("eve download --eve-tag=%s --eve-arch=%s -d %s --baseos=%t -v %s",
		tag, arch, filepath.Join(dist, "dist", arch), baseOs, log.GetLevel())
	log.Infof("DownloadEveFormDocker run: %s %s", commandPath, commandArgsString)
	return RunCommandWithLogAndWait(commandPath, logLevelToPrint, strings.Fields(commandArgsString)...)
}

//ChangeConfigPartAndRootFs replace config and rootfs part in EVE live image
func ChangeConfigPartAndRootFs(commandPath string, distEve string, distAdam string, arch string, hv string) (err error) {
	imagePath := filepath.Join(distEve, "dist", arch, "live.qcow2")
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return fmt.Errorf("file not exists: %s", imagePath)
	}
	configPath := filepath.Join(distAdam, "run", "config")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("directory not exists: %s", configPath)
	}
	commandArgsString := fmt.Sprintf("eve confchanger --image-file=%s --config-part=%s --hv=%s -v %s",
		imagePath, configPath, hv, log.GetLevel())
	log.Infof("ChangeConfigPartAndRootFs run: %s %s", commandPath, commandArgsString)
	return RunCommandWithLogAndWait(commandPath, logLevelToPrint, strings.Fields(commandArgsString)...)
}

//MakeEveInRepo build live image of EVE
func MakeEveInRepo(distEve string, distAdam string, arch string, hv string, rootFSOnly bool) (err error) {
	if _, err := os.Stat(distEve); os.IsNotExist(err) {
		return fmt.Errorf("directory not exists: %s", distEve)
	}
	configPath := filepath.Join(distAdam, "run", "config")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err = os.MkdirAll(configPath, 0755); err != nil {
			return err
		}
	}
	if rootFSOnly {
		commandArgsString := fmt.Sprintf("-C %s HV=%s CONF_DIR=%s rootfs",
			distEve, hv, configPath)
		log.Infof("MakeEveInRepo run: %s %s", "make", commandArgsString)
		err = RunCommandWithLogAndWait("make", logLevelToPrint, strings.Fields(commandArgsString)...)
	} else {
		commandArgsString := fmt.Sprintf("-C %s HV=%s CONF_DIR=%s IMG_FORMAT=qcow2 live",
			distEve, hv, configPath)
		log.Infof("MakeEveInRepo run: %s %s", "make", commandArgsString)
		err = RunCommandWithLogAndWait("make", logLevelToPrint, strings.Fields(commandArgsString)...)
		biosPath := filepath.Join(distEve, "dist", arch, "OVMF.fd")
		commandArgsString = fmt.Sprintf("-C %s HV=%s %s",
			distEve, hv, biosPath)
		log.Infof("MakeEveInRepo run: %s %s", "make", commandArgsString)
		err = RunCommandWithLogAndWait("make", logLevelToPrint, strings.Fields(commandArgsString)...)
		if arch == "arm64" {
			dtbPath := filepath.Join(distEve, "dist", "eve.dtb")
			commandArgsString = fmt.Sprintf("-C %s HV=%s %s",
				distEve, hv, dtbPath)
			log.Infof("MakeEveInRepo run: %s %s", "make", commandArgsString)
			err = RunCommandWithLogAndWait("make", logLevelToPrint, strings.Fields(commandArgsString)...)
		}
	}
	return
}

//BuildVM build VM image with linuxkit
func BuildVM(linuxKitPath string, imageConfig string, distImage string) (err error) {
	distImageDir := filepath.Dir(distImage)
	if _, err := os.Stat(distImageDir); os.IsNotExist(err) {
		if err = os.MkdirAll(distImageDir, 0755); err != nil {
			return err
		}
	}
	imageConfigTmp := filepath.Join(distImageDir, fmt.Sprintf("%s-bios.img", fileNameWithoutExtension(filepath.Base(distImage))))
	commandArgsString := fmt.Sprintf("build -format raw-bios -dir %s %s",
		distImageDir, imageConfig)
	log.Infof("BuildVM run: %s %s", linuxKitPath, commandArgsString)
	if err = RunCommandWithLogAndWait(linuxKitPath, logLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		return fmt.Errorf("error in linuxkit: %s", err)
	}
	commandArgsString = fmt.Sprintf("convert -c -f raw -O qcow2 %s %s",
		imageConfigTmp, distImage)
	log.Infof("BuildVM run: %s %s", "qemu-img", commandArgsString)
	if err = RunCommandWithLogAndWait("qemu-img", logLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		return fmt.Errorf("error in qemu-img: %s", err)
	}
	return os.Remove(imageConfigTmp)
}

//PrepareQEMUConfig create config file for QEMU
func PrepareQEMUConfig(commandPath string, qemuConfigFile string, firmwareFile []string, configPart string, dtbPath string, eveHostFWD string) (err error) {
	firmwares := strings.Join(firmwareFile, ",")
	commandArgsString := fmt.Sprintf("eve qemuconf --qemu-config=%s --eve-firmware=%s --config-part=%s --eve-hostfwd=\"%s\" --dtb-part=%s -v %s",
		qemuConfigFile, firmwares, configPart, eveHostFWD, dtbPath, log.GetLevel())
	log.Infof("PrepareQEMUConfig run: %s %s", commandPath, commandArgsString)
	if err = RunCommandWithLogAndWait(commandPath, logLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		return fmt.Errorf("error in qemuconf: %s", err)
	}
	return nil
}

//CleanEden teardown Eden and cleanup
func CleanEden(commandPath, eveDist, eveBaseDist, adamDist, certsDist, imagesDist, binDir, eserverPID, evePID string) (err error) {
	commandArgsString := fmt.Sprintf("stop --eserver-pid=%s --eve-pid=%s --adam-rm=true",
		eserverPID, evePID)
	log.Infof("CleanEden run: %s %s", commandPath, commandArgsString)
	_, _, err = RunCommandAndWait(commandPath, strings.Fields(commandArgsString)...)
	if err != nil {
		return fmt.Errorf("error in eden stop: %s", err)
	}
	if _, err = os.Stat(eveDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(eveDist); err != nil {
			return fmt.Errorf("error in %s delete: %s", eveDist, err)
		}
	}
	if _, err = os.Stat(eveBaseDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(eveBaseDist); err != nil {
			return fmt.Errorf("error in %s delete: %s", eveBaseDist, err)
		}
	}
	if _, err = os.Stat(certsDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(certsDist); err != nil {
			return fmt.Errorf("error in %s delete: %s", certsDist, err)
		}
	}
	if _, err = os.Stat(imagesDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(imagesDist); err != nil {
			return fmt.Errorf("error in %s delete: %s", imagesDist, err)
		}
	}
	if _, err = os.Stat(adamDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(adamDist); err != nil {
			return fmt.Errorf("error in %s delete: %s", adamDist, err)
		}
	}
	if _, err = os.Stat(binDir); !os.IsNotExist(err) {
		if err = os.RemoveAll(binDir); err != nil {
			return fmt.Errorf("error in %s delete: %s", binDir, err)
		}
	}
	return nil
}
