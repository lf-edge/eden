package adam

import (
	"encoding/json"
	"fmt"
	"github.com/lf-edge/adam/pkg/server"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/controller/loaders"
	"github.com/lf-edge/eden/pkg/utils"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
	"time"
)

type Ctx struct {
	dir         string
	url         string
	serverCA    string
	insecureTLS bool
	AdamRemote  bool
}

func (adam *Ctx) getLoader() loaders.Loader {
	if adam.AdamRemote {
		log.Info("will use remote adam loader")
		return loaders.RemoteLoader(adam.getHTTPClient, adam.getLogsUrl, adam.getInfoUrl)
	}
	log.Info("will use local adam loader")
	return loaders.FileLoader(adam.getLogsDir, adam.getInfoDir)
}

//EnvRead use variables from viper for init controller
func (adam *Ctx) InitWithVars(vars *utils.ConfigVars) error {
	adam.dir = vars.AdamDir
	adam.url = fmt.Sprintf("https://%s:%s", vars.AdamIP, vars.AdamPort)
	adam.insecureTLS = len(vars.AdamCA) == 0
	adam.serverCA = vars.AdamCA
	adam.AdamRemote = vars.AdamRemote
	return nil
}

//GetDir return dir
func (adam *Ctx) GetDir() (dir string) {
	return adam.dir
}

//getLogsDir return logs directory for devUUID
func (adam *Ctx) getLogsDir(devUUID uuid.UUID) (dir string) {
	return path.Join(adam.dir, "run", "adam", "device", devUUID.String(), "logs")
}

//getInfoDir return info directory for devUUID
func (adam *Ctx) getInfoDir(devUUID uuid.UUID) (dir string) {
	return path.Join(adam.dir, "run", "adam", "device", devUUID.String(), "info")
}

//getLogsUrl return logs url for devUUID
func (adam *Ctx) getLogsUrl(devUUID uuid.UUID) string {
	resUrl, err := utils.ResolveURL(adam.url, path.Join("/admin/device", devUUID.String(), "logs"))
	if err != nil {
		log.Fatalf("ResolveURL: %s", err)
	}
	return resUrl
}

//getLogsUrl return info url for devUUID
func (adam *Ctx) getInfoUrl(devUUID uuid.UUID) string {
	resUrl, err := utils.ResolveURL(adam.url, path.Join("/admin/device", devUUID.String(), "info"))
	if err != nil {
		log.Fatalf("ResolveURL: %s", err)
	}
	return resUrl
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
func (adam *Ctx) LogChecker(devUUID uuid.UUID, q map[string]string, handler elog.HandlerFunc, mode elog.LogCheckerMode, timeout time.Duration) (err error) {
	return elog.LogChecker(adam.getLoader(), devUUID, q, handler, mode, timeout)
}

//LogLastCallback check logs by pattern from existence files with callback
func (adam *Ctx) LogLastCallback(devUUID uuid.UUID, q map[string]string, handler elog.HandlerFunc) (err error) {
	var loader = adam.getLoader()
	loader.SetUUID(devUUID)
	return elog.LogLast(loader, q, handler)
}

//InfoChecker checks the information in the regular expression pattern 'query' and processes the info.ZInfoMsg found by the function 'handler' from existing files (mode=einfo.InfoExist), new files (mode=einfo.InfoNew) or any of them (mode=einfo.InfoAny) with timeout.
func (adam *Ctx) InfoChecker(devUUID uuid.UUID, q map[string]string, infoType einfo.ZInfoType, handler einfo.HandlerFunc, mode einfo.InfoCheckerMode, timeout time.Duration) (err error) {
	return einfo.InfoChecker(adam.getLoader(), devUUID, q, infoType, handler, mode, timeout)
}

//InfoLastCallback check info by pattern from existence files with callback
func (adam *Ctx) InfoLastCallback(devUUID uuid.UUID, q map[string]string, infoType einfo.ZInfoType, handler einfo.HandlerFunc) (err error) {
	var loader = adam.getLoader()
	loader.SetUUID(devUUID)
	return einfo.InfoLast(loader, q, einfo.ZInfoFind, handler, infoType)
}
