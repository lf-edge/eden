package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/lf-edge/eden/eserver/api"
	"github.com/lf-edge/eden/pkg/defaults"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//StartRedis function run redis in docker with mounted redisPath:/data
//if redisForce is set, it recreates container
func StartRedis(redisPort int, redisPath string, redisForce bool, redisTag string) (err error) {
	portMap := map[string]string{"6379": strconv.Itoa(redisPort)}
	volumeMap := map[string]string{"/data": fmt.Sprintf("%s", redisPath)}
	redisServerCommand := strings.Fields("redis-server")
	if redisForce {
		_ = StopContainer(defaults.DefaultRedisContainerName, true)
		if err := CreateAndRunContainer(defaults.DefaultRedisContainerName, defaults.DefaultRedisContainerRef+":"+redisTag, portMap, volumeMap, redisServerCommand); err != nil {
			return fmt.Errorf("error in create redis container: %s", err)
		}
	} else {
		state, err := StateContainer(defaults.DefaultRedisContainerName)
		if err != nil {
			return fmt.Errorf("error in get state of redis container: %s", err)
		}
		if state == "" {
			if err := CreateAndRunContainer(defaults.DefaultRedisContainerName, defaults.DefaultRedisContainerRef+":"+redisTag, portMap, volumeMap, redisServerCommand); err != nil {
				return fmt.Errorf("error in create redis container: %s", err)
			}
		} else if !strings.Contains(state, "running") {
			if err := StartContainer(defaults.DefaultRedisContainerName); err != nil {
				return fmt.Errorf("error in restart redis container: %s", err)
			}
		}
	}
	return nil
}

//StopRedis function stop redis container
func StopRedis(redisRm bool) (err error) {
	state, err := StateContainer(defaults.DefaultRedisContainerName)
	if err != nil {
		return fmt.Errorf("error in get state of redis container: %s", err)
	}
	if !strings.Contains(state, "running") {
		if redisRm {
			if err := StopContainer(defaults.DefaultRedisContainerName, true); err != nil {
				return fmt.Errorf("error in rm redis container: %s", err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if redisRm {
			if err := StopContainer(defaults.DefaultRedisContainerName, false); err != nil {
				return fmt.Errorf("error in rm redis container: %s", err)
			}
		} else {
			if err := StopContainer(defaults.DefaultRedisContainerName, true); err != nil {
				return fmt.Errorf("error in rm redis container: %s", err)
			}
		}
	}
	return nil
}

//StatusRedis function return status of redis
func StatusRedis() (status string, err error) {
	state, err := StateContainer(defaults.DefaultRedisContainerName)
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
func StartAdam(adamPort int, adamPath string, adamForce bool, adamTag string, adamRemoteRedisURL string) (err error) {
	portMap := map[string]string{"8080": strconv.Itoa(adamPort)}
	volumeMap := map[string]string{"/adam/run": fmt.Sprintf("%s/run", adamPath)}
	adamServerCommand := strings.Fields("server --conf-dir ./run/conf")
	if adamRemoteRedisURL != "" {
		adamServerCommand = append(adamServerCommand, strings.Fields(fmt.Sprintf("--db-url %s", adamRemoteRedisURL))...)
	}
	if adamForce {
		_ = StopContainer(defaults.DefaultAdamContainerName, true)
		if err := CreateAndRunContainer(defaults.DefaultAdamContainerName, defaults.DefaultAdamContainerRef+":"+adamTag, portMap, volumeMap, adamServerCommand); err != nil {
			return fmt.Errorf("error in create adam container: %s", err)
		}
	} else {
		state, err := StateContainer(defaults.DefaultAdamContainerName)
		if err != nil {
			return fmt.Errorf("error in get state of adam container: %s", err)
		}
		if state == "" {
			if err := CreateAndRunContainer(defaults.DefaultAdamContainerName, defaults.DefaultAdamContainerRef+":"+adamTag, portMap, volumeMap, adamServerCommand); err != nil {
				return fmt.Errorf("error in create adam container: %s", err)
			}
		} else if !strings.Contains(state, "running") {
			if err := StartContainer(defaults.DefaultAdamContainerName); err != nil {
				return fmt.Errorf("error in restart adam container: %s", err)
			}
		}
	}
	return nil
}

//StopAdam function stop adam container
func StopAdam(adamRm bool) (err error) {
	state, err := StateContainer(defaults.DefaultAdamContainerName)
	if err != nil {
		return fmt.Errorf("error in get state of adam container: %s", err)
	}
	if !strings.Contains(state, "running") {
		if adamRm {
			if err := StopContainer(defaults.DefaultAdamContainerName, true); err != nil {
				return fmt.Errorf("error in rm adam container: %s", err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if adamRm {
			if err := StopContainer(defaults.DefaultAdamContainerName, false); err != nil {
				return fmt.Errorf("error in rm adam container: %s", err)
			}
		} else {
			if err := StopContainer(defaults.DefaultAdamContainerName, true); err != nil {
				return fmt.Errorf("error in rm adam container: %s", err)
			}
		}
	}
	return nil
}

//StatusAdam function return status of adam
func StatusAdam() (status string, err error) {
	state, err := StateContainer(defaults.DefaultAdamContainerName)
	if err != nil {
		return "", fmt.Errorf("error in get state of adam container: %s", err)
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
	volumeMap := map[string]string{"/eserver/run/eserver/": fmt.Sprintf("%s", imageDist)}
	eserverServerCommand := strings.Fields("server")
	if eserverForce {
		_ = StopContainer(defaults.DefaultEServerContainerName, true)
		if err := CreateAndRunContainer(defaults.DefaultEServerContainerName, defaults.DefaultEServerContainerRef+":"+eserverTag, portMap, volumeMap, eserverServerCommand); err != nil {
			return fmt.Errorf("error in create eserver container: %s", err)
		}
	} else {
		state, err := StateContainer(defaults.DefaultEServerContainerName)
		if err != nil {
			return fmt.Errorf("error in get state of eserver container: %s", err)
		}
		if state == "" {
			if err := CreateAndRunContainer(defaults.DefaultEServerContainerName, defaults.DefaultEServerContainerRef+":"+eserverTag, portMap, volumeMap, eserverServerCommand); err != nil {
				return fmt.Errorf("error in create eserver container: %s", err)
			}
		} else if !strings.Contains(state, "running") {
			if err := StartContainer(defaults.DefaultEServerContainerName); err != nil {
				return fmt.Errorf("error in restart eserver container: %s", err)
			}
		}
	}
	return nil
}

//StopEServer function stop eserver container
func StopEServer(eserverRm bool) (err error) {
	state, err := StateContainer(defaults.DefaultEServerContainerName)
	if err != nil {
		return fmt.Errorf("error in get state of eserver container: %s", err)
	}
	if !strings.Contains(state, "running") {
		if eserverRm {
			if err := StopContainer(defaults.DefaultEServerContainerName, true); err != nil {
				return fmt.Errorf("error in rm eserver container: %s", err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if eserverRm {
			if err := StopContainer(defaults.DefaultEServerContainerName, false); err != nil {
				return fmt.Errorf("error in rm eserver container: %s", err)
			}
		} else {
			if err := StopContainer(defaults.DefaultEServerContainerName, true); err != nil {
				return fmt.Errorf("error in rm eserver container: %s", err)
			}
		}
	}
	return nil
}

//StatusEServer function return eserver of adam
func StatusEServer() (status string, err error) {
	state, err := StateContainer(defaults.DefaultEServerContainerName)
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
	return RunCommandWithLogAndWait(commandPath, defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
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
	return RunCommandWithLogAndWait(commandPath, defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
}

//CopyCertsToAdamConfig function copy certs to adam config
func CopyCertsToAdamConfig(certsDir string, domain string, ip string, port int, adamDist string, apiV1 bool) (err error) {
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
	/*if err = CopyFileNotExists(filepath.Join(certsDir, "id_rsa.pub"), filepath.Join(adamConfig, "authorized_keys")); err != nil {
		return err
	}*/
	if _, err = os.Stat(filepath.Join(adamConfig, "hosts")); os.IsNotExist(err) {
		if err = ioutil.WriteFile(filepath.Join(adamConfig, "hosts"), []byte(fmt.Sprintf("%s %s\n", ip, domain)), 0666); err != nil {
			return err
		}
	}
	if apiV1 {
		if _, err = os.Stat(filepath.Join(adamConfig, "Force-API-V1")); os.IsNotExist(err) {
			if err := TouchFile(filepath.Join(adamConfig, "Force-API-V1")); err != nil {
				log.Fatal(err)
			}
		}
	}
	if _, err = os.Stat(filepath.Join(adamConfig, "server")); os.IsNotExist(err) {
		if err = ioutil.WriteFile(filepath.Join(adamConfig, "server"), []byte(fmt.Sprintf("%s:%d\n", domain, port)), 0666); err != nil {
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
	return RunCommandWithLogAndWait("git", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
}

//DownloadEveFormDocker function clone EVE from docker
func DownloadEveFormDocker(commandPath string, dist string, arch string, hv string, tag string, newDownload bool) (err error) {
	if _, err := os.Stat(dist); !os.IsNotExist(err) {
		return fmt.Errorf("directory already exists: %s", dist)
	}
	if tag == "" {
		tag = "latest"
	}
	commandArgsString := fmt.Sprintf("eve download --eve-tag=%s --eve-arch=%s --eve-hv=%s --new-download=%t -d %s -v %s",
		tag, arch, hv, newDownload, filepath.Join(dist, "dist", arch), log.GetLevel())
	log.Infof("DownloadEveFormDocker run: %s %s", commandPath, commandArgsString)
	return RunCommandWithLogAndWait(commandPath, defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
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
	commandArgsString := fmt.Sprintf("eve confchanger --image-file=%s --config-part=%s --eve-hv=%s -v %s",
		imagePath, configPath, hv, log.GetLevel())
	log.Infof("ChangeConfigPartAndRootFs run: %s %s", commandPath, commandArgsString)
	return RunCommandWithLogAndWait(commandPath, defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
}

//MakeEveInRepo build live image of EVE
func MakeEveInRepo(distEve string, distAdam string, arch string, hv string, imageFormat string, rootFSOnly bool) (image, additional string, err error) {
	if _, err := os.Stat(distEve); os.IsNotExist(err) {
		return "", "", fmt.Errorf("directory not exists: %s", distEve)
	}
	configPath := filepath.Join(distAdam, "run", "config")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err = os.MkdirAll(configPath, 0755); err != nil {
			return "", "", err
		}
	}
	if rootFSOnly {
		commandArgsString := fmt.Sprintf("-C %s ZARCH=%s HV=%s CONF_DIR=%s rootfs",
			distEve, arch, hv, configPath)
		log.Infof("MakeEveInRepo run: %s %s", "make", commandArgsString)
		err = RunCommandWithLogAndWait("make", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
		image = filepath.Join(distEve, "dist", arch, "installer", fmt.Sprintf("live.%s", imageFormat))
	} else {
		image = filepath.Join(distEve, "dist", arch, fmt.Sprintf("live.%s", imageFormat))
		commandArgsString := fmt.Sprintf("-C %s ZARCH=%s HV=%s CONF_DIR=%s IMG_FORMAT=%s live",
			distEve, arch, hv, configPath, imageFormat)
		log.Infof("MakeEveInRepo run: %s %s", "make", commandArgsString)
		err = RunCommandWithLogAndWait("make", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
		switch arch {
		case "amd64":
			biosPath1 := filepath.Join(distEve, "dist", arch, "OVMF.fd")
			biosPath2 := filepath.Join(distEve, "dist", arch, "OVMF_CODE.fd")
			biosPath3 := filepath.Join(distEve, "dist", arch, "OVMF_VARS.fd")
			commandArgsString = fmt.Sprintf("-C %s ZARCH=%s HV=%s %s %s %s",
				distEve, arch, hv, biosPath1, biosPath2, biosPath3)
			log.Infof("MakeEveInRepo run: %s %s", "make", commandArgsString)
			err = RunCommandWithLogAndWait("make", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
			additional = strings.Join([]string{biosPath1, biosPath2, biosPath3}, ",")
		case "arm64":
			dtbPath := filepath.Join(distEve, "dist", arch, "dtb", "eve.dtb")
			commandArgsString = fmt.Sprintf("-C %s ZARCH=%s HV=%s %s",
				distEve, arch, hv, dtbPath)
			log.Infof("MakeEveInRepo run: %s %s", "make", commandArgsString)
			err = RunCommandWithLogAndWait("make", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...)
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
	imageConfigTmp := filepath.Join(distImageDir, fmt.Sprintf("%s-bios.img", fileNameWithoutExtension(filepath.Base(distImage))))
	commandArgsString := fmt.Sprintf("build -format raw-bios -dir %s %s",
		distImageDir, imageConfig)
	log.Infof("BuildVM run: %s %s", linuxKitPath, commandArgsString)
	if err = RunCommandWithLogAndWait(linuxKitPath, defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		return fmt.Errorf("error in linuxkit: %s", err)
	}
	commandArgsString = fmt.Sprintf("convert -c -f raw -O qcow2 %s %s",
		imageConfigTmp, distImage)
	log.Infof("BuildVM run: %s %s", "qemu-img", commandArgsString)
	if err = RunCommandWithLogAndWait("qemu-img", defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
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
	if err = RunCommandWithLogAndWait(commandPath, defaults.DefaultLogLevelToPrint, strings.Fields(commandArgsString)...); err != nil {
		return fmt.Errorf("error in qemuconf: %s", err)
	}
	return nil
}

//CleanEden teardown Eden and cleanup
func CleanEden(commandPath, eveDist, adamDist, certsDist, imagesDist, redisDist, configDir, evePID string) (err error) {
	commandArgsString := fmt.Sprintf("stop --eve-pid=%s --adam-rm=true",
		evePID)
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
	if _, err = os.Stat(redisDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(redisDist); err != nil {
			return fmt.Errorf("error in %s delete: %s", redisDist, err)
		}
	}
	if _, err = os.Stat(configDir); !os.IsNotExist(err) {
		if err = os.RemoveAll(configDir); err != nil {
			return fmt.Errorf("error in %s delete: %s", configDir, err)
		}
	}
	return nil
}

//EServer for connection to eserver
type EServer struct {
	EServerIP   string
	EserverPort string
}

func (server *EServer) getHTTPClient() *http.Client {
	var client = &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			ResponseHeaderTimeout: 10 * time.Second,
		},
	}
	return client
}

//EServerAddFileUrl send url to download image into eserver
func (server *EServer) EServerAddFileUrl(url string) (name string) {
	u, err := ResolveURL(fmt.Sprintf("http://%s:%s", server.EServerIP, server.EserverPort), "admin/add-from-url")
	if err != nil {
		log.Fatalf("error constructing URL: %v", err)
	}
	client := server.getHTTPClient()
	objToSend := api.UrlArg{
		Url: url,
	}
	body, err := json.Marshal(objToSend)
	if err != nil {
		log.Fatalf("error encoding json: %v", err)
	}
	req, err := http.NewRequest("POST", u, bytes.NewBuffer(body))
	if err != nil {
		log.Fatalf("unable to create new http request: %v", err)
	}

	response, err := RepeatableAttempt(client, req)
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
	u, err := ResolveURL(fmt.Sprintf("http://%s:%s", server.EServerIP, server.EserverPort), fmt.Sprintf("admin/status/%s", name))
	if err != nil {
		log.Fatalf("error constructing URL: %v", err)
	}
	client := server.getHTTPClient()
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		log.Fatalf("unable to create new http request: %v", err)
	}

	response, err := RepeatableAttempt(client, req)
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
	u, err := ResolveURL(fmt.Sprintf("http://%s:%s", server.EServerIP, server.EserverPort), "admin/add-from-file")
	if err != nil {
		log.Fatalf("error constructing URL: %v", err)
	}
	client := server.getHTTPClient()
	response, err := UploadFile(client, u, filepath)
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
