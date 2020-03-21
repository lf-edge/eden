package adam

import (
	"fmt"
	"github.com/itmo-eve/eden/pkg/utils"
	"log"
	"strings"
)

const eveCert = "/adam/run/config/onboard.cert.pem"

type AdamCtx struct {
	Url string
	Dir string
}

func (adam *AdamCtx) OnBoardAdd(eveSerial string) error {
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
