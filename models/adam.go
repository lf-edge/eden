package models

import (
	"../utils"
	"fmt"
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

func (adam *AdamCtx) OnBoardList() (out string, err error) {
	adamOnboardCmd, adamOnboardArgs := adamOnboardListPattern(adam.Dir, adam.Url)
	cmdOut, cmdErr, err := utils.RunCommandAndWait(adamOnboardCmd, adamOnboardArgs...)
	if err != nil {
		log.Print(cmdOut)
		log.Print(cmdErr)
		return strings.TrimSpace(cmdOut), err
	} else {
		return strings.TrimSpace(cmdOut), nil
	}
}

func (adam *AdamCtx) DeviceList() (out string, err error) {
	adamOnboardCmd, adamOnboardArgs := adamDevicesListPattern(adam.Dir, adam.Url)
	cmdOut, cmdErr, err := utils.RunCommandAndWait(adamOnboardCmd, adamOnboardArgs...)
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
