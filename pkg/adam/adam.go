package adam

import (
	"fmt"
	"github.com/itmo-eve/eden/pkg/utils"
	uuid "github.com/satori/go.uuid"
	"log"
	"path"
	"strings"
)

//AdamCtx is struct for use with adam
type AdamCtx struct {
	Url string
	Dir string
}

//GetLogsDir return logs directory for devUUID
func (adam *AdamCtx) GetLogsDir(devUUID *uuid.UUID) (dir string) {
	return path.Join(adam.Dir, "run", "adam", "device", devUUID.String(), "logs")
}

//GetInfoDir return info directory for devUUID
func (adam *AdamCtx) GetInfoDir(devUUID *uuid.UUID) (dir string) {
	return path.Join(adam.Dir, "run", "adam", "device", devUUID.String(), "info")
}

//Register device in adam
func (adam *AdamCtx) Register(eveCert string, eveSerial string) error {
	adamOnboardCmd, adamOnboardArgs := adamOnboardAddPattern(adam.Dir, adam.Url, eveCert, eveSerial)
	cmdOut, cmdErr, err := utils.RunCommandAndWait(adamOnboardCmd, adamOnboardArgs...)
	if err != nil {
		log.Print(cmdOut)
		log.Print(cmdErr)
		return err
	} else {
		return nil
	}
}

//OnBoardList return onboard list
func (adam *AdamCtx) OnBoardList() (out []string, err error) {
	adamOnboardCmd, adamOnboardArgs := adamOnboardListPattern(adam.Dir, adam.Url)
	cmdOut, cmdErr, err := utils.RunCommandAndWait(adamOnboardCmd, adamOnboardArgs...)
	if err != nil {
		log.Print(cmdOut)
		log.Print(cmdErr)
		return strings.Fields(cmdOut), err
	} else {
		return strings.Fields(cmdOut), nil
	}
}

//DeviceList return device list
func (adam *AdamCtx) DeviceList() (out []string, err error) {
	adamOnboardCmd, adamOnboardArgs := adamDevicesListPattern(adam.Dir, adam.Url)
	cmdOut, cmdErr, err := utils.RunCommandAndWait(adamOnboardCmd, adamOnboardArgs...)
	if err != nil {
		log.Print(cmdOut)
		log.Print(cmdErr)
		return strings.Fields(cmdOut), err
	} else {
		return strings.Fields(cmdOut), nil
	}
}

//ConfigSet set config for devID
func (adam *AdamCtx) ConfigSet(devID string, config string) (out string, err error) {
	adamConfigSetCmd, adamConfigSetArgs := adamConfigSetPattern(adam.Dir, adam.Url, devID)
	cmdOut, cmdErr, err := utils.RunCommandWithSTDINAndWait(adamConfigSetCmd, config, adamConfigSetArgs...)
	if err != nil {
		log.Print(cmdOut)
		log.Print(cmdErr)
		return strings.TrimSpace(cmdOut), err
	} else {
		return strings.TrimSpace(cmdOut), nil
	}
}

func adamOnboardAddPattern(dir string, url string, cert string, serial string) (cmd string, args []string) {
	return "docker", strings.Split(fmt.Sprintf("run -v %s/run:/adam/run lfedge/adam admin --server %s onboard add --path %s --serial %s", dir, url, cert, serial), " ")
}

func adamOnboardListPattern(dir string, url string) (cmd string, args []string) {
	return "docker", strings.Split(fmt.Sprintf("run -v %s/run:/adam/run lfedge/adam admin --server %s onboard list", dir, url), " ")
}

func adamDevicesListPattern(dir string, url string) (cmd string, args []string) {
	return "docker", strings.Split(fmt.Sprintf("run -v %s/run:/adam/run lfedge/adam admin --server %s device list", dir, url), " ")
}

func adamConfigSetPattern(dir string, url string, id string) (cmd string, args []string) {
	return "docker", strings.Split(fmt.Sprintf("run -i -v %s/run:/adam/run lfedge/adam admin --server %s device config set --uuid %s --config-path -", dir, url, id), " ")
}
