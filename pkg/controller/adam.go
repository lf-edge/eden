package controller

import (
	"encoding/json"
	"github.com/lf-edge/adam/pkg/server"
	uuid "github.com/satori/go.uuid"
	"io/ioutil"
	"log"
	"os"
	"path"
)

//Ctx is struct for use with adam
type Ctx struct {
	Dir         string
	URL         string
	ServerCA    string
	InsecureTLS bool
}

//GetLogsDir return logs directory for devUUID
func (adam *Ctx) GetLogsDir(devUUID *uuid.UUID) (dir string) {
	return path.Join(adam.Dir, "run", "adam", "device", devUUID.String(), "logs")
}

//GetInfoDir return info directory for devUUID
func (adam *Ctx) GetInfoDir(devUUID *uuid.UUID) (dir string) {
	return path.Join(adam.Dir, "run", "adam", "device", devUUID.String(), "info")
}

//Register device in adam
func (adam *Ctx) Register(eveCert string, eveSerial string) error {
	b, err := ioutil.ReadFile(eveCert)
	switch {
	case err != nil && os.IsNotExist(err):
		log.Printf("cert file %s does not exist", eveCert)
		return err
	case err != nil:
		log.Printf("error reading cert file %s: %v", eveCert, err)
		return err
	}

	objToSend := server.OnboardCert{
		Cert:   b,
		Serial: eveSerial,
	}
	body, err := json.Marshal(objToSend)
	if err != nil {
		log.Printf("error encoding json: %v", err)
		return err
	}
	return adam.postObj("/admin/onboard", body)
}

//OnBoardList return onboard list
func (adam *Ctx) OnBoardList() (out []string, err error) {
	return adam.getList("/admin/onboard")
}

//DeviceList return device list
func (adam *Ctx) DeviceList() (out []string, err error) {
	return adam.getList("/admin/device")
}

//ConfigSet set config for devID
func (adam *Ctx) ConfigSet(devID string, config string) (err error) {
	return adam.putObj(path.Join("/admin/device", devID, "config"), []byte(config))
}

//ConfigGet get config for devID
func (adam *Ctx) ConfigGet(devID string) (out string, err error) {
	return adam.getObj(path.Join("/admin/device", devID, "config"))
}
