package eden

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
			log.Fatalf("Cannot create directory for redis (%s): %s", redisPath, err)
		}
	}
	if redisForce {
		_ = utils.StopContainer(defaults.DefaultRedisContainerName, true)
		if err := utils.CreateAndRunContainer(defaults.DefaultRedisContainerName, defaults.DefaultRedisContainerRef+":"+redisTag, portMap, volumeMap, redisServerCommand, nil); err != nil {
			return fmt.Errorf("error in create redis container: %s", err)
		}
	} else {
		state, err := utils.StateContainer(defaults.DefaultRedisContainerName)
		if err != nil {
			return fmt.Errorf("error in get state of redis container: %s", err)
		}
		if state == "" {
			if err := utils.CreateAndRunContainer(defaults.DefaultRedisContainerName, defaults.DefaultRedisContainerRef+":"+redisTag, portMap, volumeMap, redisServerCommand, nil); err != nil {
				return fmt.Errorf("error in create redis container: %s", err)
			}
		} else if !strings.Contains(state, "running") {
			if err := utils.StartContainer(defaults.DefaultRedisContainerName); err != nil {
				return fmt.Errorf("error in restart redis container: %s", err)
			}
		}
	}
	return nil
}

//StopRedis function stop redis container
func StopRedis(redisRm bool) (err error) {
	state, err := utils.StateContainer(defaults.DefaultRedisContainerName)
	if err != nil {
		return fmt.Errorf("error in get state of redis container: %s", err)
	}
	if !strings.Contains(state, "running") {
		if redisRm {
			if err := utils.StopContainer(defaults.DefaultRedisContainerName, true); err != nil {
				return fmt.Errorf("error in rm redis container: %s", err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if redisRm {
			if err := utils.StopContainer(defaults.DefaultRedisContainerName, false); err != nil {
				return fmt.Errorf("error in rm redis container: %s", err)
			}
		} else {
			if err := utils.StopContainer(defaults.DefaultRedisContainerName, true); err != nil {
				return fmt.Errorf("error in rm redis container: %s", err)
			}
		}
	}
	return nil
}

//StatusRedis function return status of redis
func StatusRedis() (status string, err error) {
	state, err := utils.StateContainer(defaults.DefaultRedisContainerName)
	if err != nil {
		return "", fmt.Errorf("error in get state of redis container: %s", err)
	}
	if state == "" {
		return "container doesn't exist", nil
	}
	return state, nil
}

//StartAdam function run adam in docker with mounted adamPath/run:/adam/run
//if adamForce is set, it recreates container
func StartAdam(adamPort int, adamPath string, adamForce bool, adamTag string, adamRemoteRedisURL string, opts ...string) (err error) {
	serverCertPath := filepath.Join(utils.ResolveAbsPath(defaults.DefaultCertsDist), "server.pem")
	serverKeyPath := filepath.Join(utils.ResolveAbsPath(defaults.DefaultCertsDist), "server-key.pem")
	cert, err := ioutil.ReadFile(serverCertPath)
	if err != nil {
		return fmt.Errorf("cannot load %s: %s", serverCertPath, err)
	}
	key, err := ioutil.ReadFile(serverKeyPath)
	if err != nil {
		return fmt.Errorf("cannot load %s: %s", serverKeyPath, err)
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
			return fmt.Errorf("error in create adam container: %s", err)
		}
	} else {
		state, err := utils.StateContainer(defaults.DefaultAdamContainerName)
		if err != nil {
			return fmt.Errorf("error in get state of adam container: %s", err)
		}
		if state == "" {
			if err := utils.CreateAndRunContainer(defaults.DefaultAdamContainerName, defaults.DefaultAdamContainerRef+":"+adamTag, portMap, volumeMap, adamServerCommand, envs); err != nil {
				return fmt.Errorf("error in create adam container: %s", err)
			}
		} else if !strings.Contains(state, "running") {
			if err := utils.StartContainer(defaults.DefaultAdamContainerName); err != nil {
				return fmt.Errorf("error in restart adam container: %s", err)
			}
		}
	}
	return nil
}

//StopAdam function stop adam container
func StopAdam(adamRm bool) (err error) {
	state, err := utils.StateContainer(defaults.DefaultAdamContainerName)
	if err != nil {
		return fmt.Errorf("error in get state of adam container: %s", err)
	}
	if !strings.Contains(state, "running") {
		if adamRm {
			if err := utils.StopContainer(defaults.DefaultAdamContainerName, true); err != nil {
				return fmt.Errorf("error in rm adam container: %s", err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if adamRm {
			if err := utils.StopContainer(defaults.DefaultAdamContainerName, false); err != nil {
				return fmt.Errorf("error in rm adam container: %s", err)
			}
		} else {
			if err := utils.StopContainer(defaults.DefaultAdamContainerName, true); err != nil {
				return fmt.Errorf("error in rm adam container: %s", err)
			}
		}
	}
	return nil
}

//StatusAdam function return status of adam
func StatusAdam() (status string, err error) {
	state, err := utils.StateContainer(defaults.DefaultAdamContainerName)
	if err != nil {
		return "", fmt.Errorf("error in get state of adam container: %s", err)
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
		return fmt.Errorf("error in get state of %s container: %s", serviceName, err)
	}
	if state == "" {
		if err := utils.CreateAndRunContainer(containerName, ref+":"+tag, portMap, volumeMap, cmd, nil); err != nil {
			return fmt.Errorf("error in create %s container: %s", serviceName, err)
		}
	} else if !strings.Contains(state, "running") {
		if err := utils.StartContainer(containerName); err != nil {
			return fmt.Errorf("error in restart %s container: %s", serviceName, err)
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
		return fmt.Errorf("error in get state of %s container: %s", serviceName, err)
	}
	if !strings.Contains(state, "running") {
		if rm {
			if err := utils.StopContainer(containerName, true); err != nil {
				return fmt.Errorf("error in rm %s container: %s", serviceName, err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if rm {
			if err := utils.StopContainer(containerName, false); err != nil {
				return fmt.Errorf("error in rm %s container: %s", serviceName, err)
			}
		} else {
			if err := utils.StopContainer(containerName, true); err != nil {
				return fmt.Errorf("error in rm %s container: %s", serviceName, err)
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
		return "", fmt.Errorf("error in get state of %s container: %s", serviceName, err)
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
		log.Fatalf("%s does not exist and can not be created", imageDist)
	}
	if eserverForce {
		_ = utils.StopContainer(defaults.DefaultEServerContainerName, true)
		if err := utils.CreateAndRunContainer(defaults.DefaultEServerContainerName, defaults.DefaultEServerContainerRef+":"+eserverTag, portMap, volumeMap, eserverServerCommand, nil); err != nil {
			return fmt.Errorf("error in create eserver container: %s", err)
		}
	} else {
		state, err := utils.StateContainer(defaults.DefaultEServerContainerName)
		if err != nil {
			return fmt.Errorf("error in get state of eserver container: %s", err)
		}
		if state == "" {
			if err := utils.CreateAndRunContainer(defaults.DefaultEServerContainerName, defaults.DefaultEServerContainerRef+":"+eserverTag, portMap, volumeMap, eserverServerCommand, nil); err != nil {
				return fmt.Errorf("error in create eserver container: %s", err)
			}
		} else if !strings.Contains(state, "running") {
			if err := utils.StartContainer(defaults.DefaultEServerContainerName); err != nil {
				return fmt.Errorf("error in restart eserver container: %s", err)
			}
		}
	}
	return nil
}

//StopEServer function stop eserver container
func StopEServer(eserverRm bool) (err error) {
	state, err := utils.StateContainer(defaults.DefaultEServerContainerName)
	if err != nil {
		return fmt.Errorf("error in get state of eserver container: %s", err)
	}
	if !strings.Contains(state, "running") {
		if eserverRm {
			if err := utils.StopContainer(defaults.DefaultEServerContainerName, true); err != nil {
				return fmt.Errorf("error in rm eserver container: %s", err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if eserverRm {
			if err := utils.StopContainer(defaults.DefaultEServerContainerName, false); err != nil {
				return fmt.Errorf("error in rm eserver container: %s", err)
			}
		} else {
			if err := utils.StopContainer(defaults.DefaultEServerContainerName, true); err != nil {
				return fmt.Errorf("error in rm eserver container: %s", err)
			}
		}
	}
	return nil
}

//StatusEServer function return eserver of adam
func StatusEServer() (status string, err error) {
	state, err := utils.StateContainer(defaults.DefaultEServerContainerName)
	if err != nil {
		return "", fmt.Errorf("error in get eserver of adam container: %s", err)
	}
	if state == "" {
		return "container doesn't exist", nil
	}
	return state, nil
}

//StartEVEQemu function run EVE in qemu
func StartEVEQemu(commandPath string, qemuARCH string, qemuOS string, eveImageFile string, qemuSMBIOSSerial string, qemuAccel bool, qemuConfigFilestring, logFile string, pidFile string) (err error) {
	commandArgsString := fmt.Sprintf("eve start --qemu-config=%s --eve-serial=%s --eve-accel=%t --eve-arch=%s --eve-os=%s --eve-log=%s --eve-pid=%s --image-file=%s -v %s",
		qemuConfigFilestring, qemuSMBIOSSerial, qemuAccel, qemuARCH, qemuOS, logFile, pidFile, eveImageFile, log.GetLevel())
	log.Infof("StartEVEQemu run: %s %s", commandPath, commandArgsString)
	return utils.RunCommandWithLogAndWait(commandPath, defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
}

//StopEVEQemu function stop EVE
func StopEVEQemu(pidFile string) (err error) {
	return utils.StopCommandWithPid(pidFile)
}

//StatusEVEQemu function get status of EVE
func StatusEVEQemu(pidFile string) (status string, err error) {
	return utils.StatusCommandWithPid(pidFile)
}

//GenerateEveCerts function generates certs for EVE
func GenerateEveCerts(commandPath string, certsDir string, domain string, ip string, eveIP string, uuid string, ssid string, password string) (err error) {
	if _, err := os.Stat(certsDir); os.IsNotExist(err) {
		if err = os.MkdirAll(certsDir, 0755); err != nil {
			return err
		}
	}
	commandArgsString := fmt.Sprintf(
		"utils certs --certs-dist=%s --domain=%s --ip=%s --eve-ip=%s --uuid=%s --ssid=%s --password=%s -v %s",
		certsDir, domain, ip, eveIP, uuid, ssid, password, log.GetLevel())
	log.Infof("GenerateEveCerts run: %s %s", commandPath, commandArgsString)
	return utils.RunCommandWithLogAndWait(commandPath, defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
}

//GenerateEVEConfig function copy certs to EVE config folder
func GenerateEVEConfig(eveConfig string, domain string, ip string, port int, apiV1 bool) (err error) {
	if _, err = os.Stat(eveConfig); os.IsNotExist(err) {
		if err = os.MkdirAll(eveConfig, 0755); err != nil {
			return err
		}
	}
	if _, err = os.Stat(filepath.Join(eveConfig, "hosts")); os.IsNotExist(err) {
		if err = ioutil.WriteFile(filepath.Join(eveConfig, "hosts"), []byte(fmt.Sprintf("%s %s\n", ip, domain)), 0666); err != nil {
			return err
		}
	}
	if apiV1 {
		if _, err = os.Stat(filepath.Join(eveConfig, "Force-API-V1")); os.IsNotExist(err) {
			if err := utils.TouchFile(filepath.Join(eveConfig, "Force-API-V1")); err != nil {
				log.Fatal(err)
			}
		}
	}
	if _, err = os.Stat(filepath.Join(eveConfig, "server")); os.IsNotExist(err) {
		if err = ioutil.WriteFile(filepath.Join(eveConfig, "server"), []byte(fmt.Sprintf("%s:%d\n", domain, port)), 0666); err != nil {
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
	return utils.RunCommandWithLogAndWait("git", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
}

//MakeEveInRepo build live image of EVE
func MakeEveInRepo(distEve string, configPath string, arch string, hv string, imageFormat string, rootFSOnly bool) (image, additional string, err error) {
	if _, err := os.Stat(distEve); os.IsNotExist(err) {
		return "", "", fmt.Errorf("directory not exists: %s", distEve)
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err = os.MkdirAll(configPath, 0755); err != nil {
			return "", "", err
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
		if err = utils.RunCommandWithLogAndWait("make", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err!=nil{
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
			return "", "", fmt.Errorf("unsupported arch %s", arch)
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
	imageConfigTmp := filepath.Join(distImageDir, fmt.Sprintf("%s-bios.img", utils.FileNameWithoutExtension(filepath.Base(distImage))))
	commandArgsString := fmt.Sprintf("build -format raw-bios -dir %s %s",
		distImageDir, imageConfig)
	log.Infof("BuildVM run: %s %s", linuxKitPath, commandArgsString)
	if err = utils.RunCommandWithLogAndWait(linuxKitPath, defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		return fmt.Errorf("error in linuxkit: %s", err)
	}
	commandArgsString = fmt.Sprintf("convert -c -f raw -O qcow2 %s %s",
		imageConfigTmp, distImage)
	log.Infof("BuildVM run: %s %s", "qemu-img", commandArgsString)
	if err = utils.RunCommandWithLogAndWait("qemu-img", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		return fmt.Errorf("error in qemu-img: %s", err)
	}
	return os.Remove(imageConfigTmp)
}

//CleanContext cleanup only context data
func CleanContext(commandPath, eveDist, certsDist, imagesDist, evePID, eveUUID string, configSaved string) (err error) {
	edenDir, err := utils.DefaultEdenDir()
	if err != nil {
		return err
	}
	eveStatusFile := filepath.Join(edenDir, fmt.Sprintf("state-%s.yml", eveUUID))
	if _, err = os.Stat(eveStatusFile); !os.IsNotExist(err) {
		ctrl, err := controller.CloudPrepare()
		if err != nil {
			return fmt.Errorf("error in CloudPrepare: %s", err)
		}
		log.Debugf("Get devUUID for onboardUUID %s", eveUUID)
		devUUID, err := ctrl.DeviceGetByOnboardUUID(eveUUID)
		if err != nil {
			return err
		}
		log.Debugf("Deleting devUUID %s", devUUID)
		if err := ctrl.DeviceRemove(devUUID); err != nil {
			return err
		}
		log.Debugf("Deleting onboardUUID %s", eveUUID)
		if err := ctrl.OnboardRemove(eveUUID); err != nil {
			return err
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
			return fmt.Errorf("error in %s delete: %s", eveStatusFile, err)
		}
	}
	commandArgsString := fmt.Sprintf("stop --eve-pid=%s --adam-rm=false --redis-rm=false --eserver-rm=false", evePID)
	log.Infof("CleanContext run: %s %s", commandPath, commandArgsString)
	_, _, err = utils.RunCommandAndWait(commandPath, strings.Fields(commandArgsString)...)
	if err != nil {
		return fmt.Errorf("error in eden stop: %s", err)
	}
	if _, err = os.Stat(eveDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(eveDist); err != nil {
			return fmt.Errorf("error in %s delete: %s", eveDist, err)
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
	if _, err = os.Stat(configSaved); !os.IsNotExist(err) {
		if err = os.RemoveAll(configSaved); err != nil {
			return fmt.Errorf("error in %s delete: %s", configSaved, err)
		}
	}
	return nil
}

//CleanEden teardown Eden and cleanup
func CleanEden(commandPath, eveDist, adamDist, certsDist, imagesDist, eserverDist, redisDist, registryDist, configDir, evePID string, configSaved string) (err error) {
	commandArgsString := fmt.Sprintf("stop --eve-pid=%s --adam-rm=true --redis-rm=true --eserver-rm=true --registry-rm=true", evePID)
	log.Infof("CleanEden run: %s %s", commandPath, commandArgsString)
	_, _, err = utils.RunCommandAndWait(commandPath, strings.Fields(commandArgsString)...)
	if err != nil {
		return fmt.Errorf("error in eden stop: %s", err)
	}
	if _, err = os.Stat(eveDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(eveDist); err != nil {
			return fmt.Errorf("error in %s delete: %s", eveDist, err)
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
	if _, err = os.Stat(eserverDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(eserverDist); err != nil {
			return fmt.Errorf("error in %s delete: %s", eserverDist, err)
		}
	}
	if _, err = os.Stat(adamDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(adamDist); err != nil {
			return fmt.Errorf("error in %s delete: %s", adamDist, err)
		}
	}
	if _, err = os.Stat(redisDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(redisDist); err != nil {
			return fmt.Errorf("error in %s delete: %s", redisDist, err)
		}
	}
	if _, err = os.Stat(registryDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(registryDist); err != nil {
			return fmt.Errorf("error in %s delete: %s", registryDist, err)
		}
	}
	if _, err = os.Stat(configDir); !os.IsNotExist(err) {
		if err = os.RemoveAll(configDir); err != nil {
			return fmt.Errorf("error in %s delete: %s", configDir, err)
		}
	}
	if _, err = os.Stat(configSaved); !os.IsNotExist(err) {
		if err = os.RemoveAll(configSaved); err != nil {
			return fmt.Errorf("error in %s delete: %s", configSaved, err)
		}
	}
	if err = utils.RemoveGeneratedVolumeOfContainer(defaults.DefaultEServerContainerName); err != nil {
		return fmt.Errorf("RemoveGeneratedVolumeOfContainer for %s: %s", defaults.DefaultEServerContainerName, err)
	}
	if err = utils.RemoveGeneratedVolumeOfContainer(defaults.DefaultRedisContainerName); err != nil {
		return fmt.Errorf("RemoveGeneratedVolumeOfContainer for %s: %s", defaults.DefaultRedisContainerName, err)
	}
	if err = utils.RemoveGeneratedVolumeOfContainer(defaults.DefaultAdamContainerName); err != nil {
		return fmt.Errorf("RemoveGeneratedVolumeOfContainer for %s: %s", defaults.DefaultAdamContainerName, err)
	}
	if err = utils.RemoveGeneratedVolumeOfContainer(defaults.DefaultRegistryContainerName); err != nil {
		return fmt.Errorf("RemoveGeneratedVolumeOfContainer for %s: %s", defaults.DefaultRegistryContainerName, err)
	}
	return nil
}

//EServer for connection to eserver
type EServer struct {
	EServerIP   string
	EserverPort string
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
	u, err := utils.ResolveURL(fmt.Sprintf("http://%s:%s", server.EServerIP, server.EserverPort), "admin/add-from-url")
	if err != nil {
		log.Fatalf("error constructing URL: %v", err)
	}
	client := server.getHTTPClient(defaults.DefaultRepeatTimeout)
	objToSend := api.URLArg{
		URL: url,
	}
	body, err := json.Marshal(objToSend)
	if err != nil {
		log.Fatalf("error encoding json: %v", err)
	}
	req, err := http.NewRequest("POST", u, bytes.NewBuffer(body))
	if err != nil {
		log.Fatalf("unable to create new http request: %v", err)
	}

	response, err := utils.RepeatableAttempt(client, req)
	if err != nil {
		log.Fatalf("unable to send request: %v", err)
	}
	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("unable to read data from URL %s: %v", u, err)
	}
	return string(buf)
}

//EServerCheckStatus checks status of image in eserver
func (server *EServer) EServerCheckStatus(name string) (fileInfo *api.FileInfo) {
	u, err := utils.ResolveURL(fmt.Sprintf("http://%s:%s", server.EServerIP, server.EserverPort), fmt.Sprintf("admin/status/%s", name))
	if err != nil {
		log.Fatalf("error constructing URL: %v", err)
	}
	client := server.getHTTPClient(defaults.DefaultRepeatTimeout * defaults.DefaultRepeatCount)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		log.Fatalf("unable to create new http request: %v", err)
	}

	response, err := utils.RepeatableAttempt(client, req)
	if err != nil {
		log.Fatalf("unable to send request: %v", err)
	}
	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("unable to read data from URL %s: %v", u, err)
	}
	if err := json.Unmarshal(buf, &fileInfo); err != nil {
		log.Fatal(err)
	}
	return
}

//EServerAddFile send file with image into eserver
func (server *EServer) EServerAddFile(filepath string) (fileInfo *api.FileInfo) {
	u, err := utils.ResolveURL(fmt.Sprintf("http://%s:%s", server.EServerIP, server.EserverPort), "admin/add-from-file")
	if err != nil {
		log.Fatalf("error constructing URL: %v", err)
	}
	client := server.getHTTPClient(0)
	response, err := utils.UploadFile(client, u, filepath)
	if err != nil {
		log.Fatal(err)
	}
	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("unable to read data from URL %s: %v", u, err)
	}
	if err := json.Unmarshal(buf, &fileInfo); err != nil {
		log.Fatal(err)
	}
	return
}

//ReadFileInSquashFS returns the content of a single file (filePath) inside squashfs (squashFSPath)
func ReadFileInSquashFS(squashFSPath, filePath string) (content []byte, err error) {
	tmpdir, err := ioutil.TempDir("", "squashfs-unpack")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpdir)
	dirToUnpack := filepath.Join(tmpdir, "temp")
	if output, err := exec.Command("unsquashfs", "-n", "-i", "-d", dirToUnpack, squashFSPath, filePath).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("unsquashfs (%s): %v", output, err)
	}
	content, err = ioutil.ReadFile(filepath.Join(dirToUnpack, filePath))
	if err != nil {
		return nil, err
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
		return nil, err
	}
	if _, err := os.Stat(rootfsDev); !os.IsNotExist(err) {
		eveRelease, err := ReadFileInSquashFS(rootfsDev, "/etc/eve-release")
		if err != nil {
			return nil, err
		}
		eveInfo.EVERelease = eveRelease
	}
	if _, err := os.Stat(persistDev); !os.IsNotExist(err) {
		fsPersist, err := gexto.NewFileSystem(persistDev)
		if err != nil {
			return nil, err
		}
		g, err := fsPersist.Open("/rsyslog/syslog.txt")
		if err != nil {
			return nil, err
		}
		syslog, err := ioutil.ReadAll(g)
		if err != nil {
			return nil, err
		}
		eveInfo.Syslog = syslog
	}
	return eveInfo, nil
}
