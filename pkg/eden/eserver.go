package eden

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/lf-edge/eden/eserver/api"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
)

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
