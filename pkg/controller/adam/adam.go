package adam

import (
	"encoding/json"
	"github.com/lf-edge/adam/pkg/server"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	uuid "github.com/satori/go.uuid"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"
)

type Ctx struct {
	Dir         string
	URL         string
	ServerCA    string
	InsecureTLS bool
}

//GetDir return Dir
func (adam *Ctx) GetDir() (dir string) {
	return adam.Dir
}

//getLogsDir return logs directory for devUUID
func (adam *Ctx) getLogsDir(devUUID *uuid.UUID) (dir string) {
	return path.Join(adam.Dir, "run", "adam", "device", devUUID.String(), "logs")
}

//getInfoDir return info directory for devUUID
func (adam *Ctx) getInfoDir(devUUID *uuid.UUID) (dir string) {
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

//ConfigSet get config for devID
func (adam *Ctx) ConfigSet(devUUID *uuid.UUID, devConfig []byte) (err error) {
	return adam.putObj(path.Join("/admin/device", devUUID.String(), "config"), devConfig)
}

//ConfigGet get config for devID
func (adam *Ctx) ConfigGet(devUUID *uuid.UUID) (out string, err error) {
	return adam.getObj(path.Join("/admin/device", devUUID.String(), "config"))
}

//LogChecker check logs by pattern from existence files with LogLast and use LogWatchWithTimeout with timeout for observe new files
func (adam *Ctx) LogChecker(devUUID *uuid.UUID, q map[string]string, timeout time.Duration) (err error) {
	return elog.LogChecker(adam.getLogsDir(devUUID), q, timeout)
}

//InfoChecker check info by pattern from existence files with InfoLast and use InfoWatchWithTimeout with timeout for observe new files
func (adam *Ctx) InfoChecker(devUUID *uuid.UUID, q map[string]string, infoType einfo.ZInfoType, timeout time.Duration) (err error) {
	return einfo.InfoChecker(adam.getInfoDir(devUUID), q, infoType, timeout)
}
