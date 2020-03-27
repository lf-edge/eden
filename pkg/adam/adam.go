package adam

import (
	"fmt"
	"github.com/itmo-eve/eden/pkg/utils"
	uuid "github.com/satori/go.uuid"
	"log"
	"path"
	"strings"
)

//Ctx is struct for use with adam
type Ctx struct {
	URL string
	Dir string
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
	adamOnboardCmd, adamOnboardArgs := adamOnboardAddPattern(adam.Dir, adam.URL, eveCert, eveSerial)
	cmdOut, cmdErr, err := utils.RunCommandAndWait(adamOnboardCmd, adamOnboardArgs...)
	if err != nil {
		log.Print(cmdOut)
		log.Print(cmdErr)
		return err
	}
	return nil
}

//OnBoardList return onboard list
func (adam *Ctx) OnBoardList() (out []string, err error) {
	adamOnboardCmd, adamOnboardArgs := adamOnboardListPattern(adam.Dir, adam.URL)
	cmdOut, cmdErr, err := utils.RunCommandAndWait(adamOnboardCmd, adamOnboardArgs...)
	if err != nil {
		log.Print(cmdOut)
		log.Print(cmdErr)
		return strings.Fields(cmdOut), err
	}
	return strings.Fields(cmdOut), nil
}

//DeviceList return device list
func (adam *Ctx) DeviceList() (out []string, err error) {
	adamOnboardCmd, adamOnboardArgs := adamDevicesListPattern(adam.Dir, adam.URL)
	cmdOut, cmdErr, err := utils.RunCommandAndWait(adamOnboardCmd, adamOnboardArgs...)
	if err != nil {
		log.Print(cmdOut)
		log.Print(cmdErr)
		return strings.Fields(cmdOut), err
	}
	return strings.Fields(cmdOut), nil
}

//ConfigSet set config for devID
func (adam *Ctx) ConfigSet(devID string, config string) (out string, err error) {
	adamConfigSetCmd, adamConfigSetArgs := adamConfigSetPattern(adam.Dir, adam.URL, devID)
	cmdOut, cmdErr, err := utils.RunCommandWithSTDINAndWait(adamConfigSetCmd, config, adamConfigSetArgs...)
	if err != nil {
		log.Print(cmdOut)
		log.Print(cmdErr)
		return strings.TrimSpace(cmdOut), err
	}
	return strings.TrimSpace(cmdOut), nil
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
