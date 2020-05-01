package adam

import (
	"encoding/json"
	"fmt"
	"github.com/lf-edge/adam/pkg/server"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/utils"
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

//EnvRead use variables from viper for init controller
func (adam *Ctx) InitWithVars(vars *utils.ConfigVars) error {
	adam.Dir = vars.AdamDir
	adam.URL = fmt.Sprintf("https://%s:%s", vars.AdamIP, vars.AdamPort)
	adam.InsecureTLS = len(vars.AdamCA) == 0
	adam.ServerCA = vars.AdamCA
	return nil
}

//GetDir return Dir
func (adam *Ctx) GetDir() (dir string) {
	return adam.Dir
}

//getLogsDir return logs directory for devUUID
func (adam *Ctx) getLogsDir(devUUID uuid.UUID) (dir string) {
	return path.Join(adam.Dir, "run", "adam", "device", devUUID.String(), "logs")
}

//getInfoDir return info directory for devUUID
func (adam *Ctx) getInfoDir(devUUID uuid.UUID) (dir string) {
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
func (adam *Ctx) ConfigSet(devUUID uuid.UUID, devConfig []byte) (err error) {
	return adam.putObj(path.Join("/admin/device", devUUID.String(), "config"), devConfig)
}

//ConfigGet get config for devID
func (adam *Ctx) ConfigGet(devUUID uuid.UUID) (out string, err error) {
	return adam.getObj(path.Join("/admin/device", devUUID.String(), "config"))
}

//LogChecker check logs by pattern from existence files with LogLast and use LogWatchWithTimeout with timeout for observe new files
func (adam *Ctx) LogChecker(devUUID uuid.UUID, q map[string]string, timeout time.Duration) (err error) {
	return elog.LogChecker(adam.getLogsDir(devUUID), q, timeout)
}

//LogLastCallback check logs by pattern from existence files with callback
func (adam *Ctx) LogLastCallback(devUUID uuid.UUID, q map[string]string, handler elog.HandlerFunc) (err error) {
	return elog.LogLast(adam.getInfoDir(devUUID), q, handler)
}

//InfoChecker checks the information in the regular expression pattern 'query' and processes the info.ZInfoMsg found by the function 'handler' from existing files (mode=einfo.InfoExist), new files (mode=einfo.InfoNew) or any of them (mode=einfo.InfoAny) with timeout.
func (adam *Ctx) InfoChecker(devUUID uuid.UUID, q map[string]string, infoType einfo.ZInfoType, handler einfo.HandlerFunc, mode einfo.InfoCheckerMode, timeout time.Duration) (err error) {
	return einfo.InfoChecker(adam.getInfoDir(devUUID), q, infoType, handler, mode, timeout)
}

//InfoLastCallback check info by pattern from existence files with callback
func (adam *Ctx) InfoLastCallback(devUUID uuid.UUID, q map[string]string, infoType einfo.ZInfoType, handler einfo.HandlerFunc) (err error) {
	return einfo.InfoLast(adam.getInfoDir(devUUID), q, einfo.ZInfoFind, handler, infoType)
}
