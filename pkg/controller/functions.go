package controller

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/controller/adam"
	"github.com/lf-edge/eden/pkg/utils"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"time"
)

//CloudPrepare is for init controller connection and obtain device list
func CloudPrepare() (Cloud, error) {
	vars, err := utils.InitVars()
	if err != nil {
		return nil, err
	}
	cloud := &CloudCtx{vars: vars, Controller: &adam.Ctx{}}
	if err := cloud.InitWithVars(vars); err != nil {
		return nil, err
	}
	deviceModel, err := cloud.GetDevModelByName(vars.DevModel)
	if err != nil {
		return cloud, err
	}
	devices, err := cloud.DeviceList()
	if err != nil {
		return cloud, err
	}
	for _, devID := range devices {
		devUUID, err := uuid.FromString(devID)
		if err != nil {
			return cloud, err
		}
		dev, err := cloud.AddDevice(devUUID)
		if err != nil {
			return cloud, err
		}
		if vars.SshKey != "" {
			b, err := ioutil.ReadFile(vars.SshKey)
			switch {
			case err != nil && os.IsNotExist(err):
				return nil, fmt.Errorf("sshKey file %s does not exist", vars.SshKey)
			case err != nil:
				return nil, fmt.Errorf("error reading sshKey file %s: %v", vars.SshKey, err)
			}
			dev.SetSSHKeys([]string{string(b)})
		}
		dev.SetVncAccess(true)
		dev.SetControllerLogLevel("info")
		err = cloud.ApplyDevModel(dev, deviceModel)
		if err != nil {
			return cloud, fmt.Errorf("fail in ApplyDevModel: %s", err)
		}
	}
	return cloud, nil
}

//OnBoard in controller
func (ctx *CloudCtx) OnBoard() error {
	devUUID, err := ctx.GetDeviceFirst()
	if devUUID == nil {
		log.Info("Try to add onboarding")
		err = ctx.Register(ctx.vars.EveCert, ctx.vars.EveSerial)
		if err != nil {
			return fmt.Errorf("ctx.register: %s", err)
		}
		res, err := ctx.OnBoardList()
		if err != nil {
			return fmt.Errorf("ctx.OnBoardList: %s", err)
		}
		if len(res) == 0 {
			return fmt.Errorf("no onboard in list")
		}
		log.Info(res)

		maxRepeat := 20
		delayTime := 20 * time.Second

		for i := 0; i < maxRepeat; i++ {
			cmdOut, err := ctx.DeviceList()
			if err != nil {
				return fmt.Errorf("ctx.DeviceList: %s", err)
			}
			if len(cmdOut) > 0 {
				log.Info("Done onboarding in adam!")
				log.Infof("Device uuid: %s", cmdOut)
				return nil
			}
			log.Infof("Attempt to list devices (%d) of (%d)", i, maxRepeat)
			time.Sleep(delayTime)
		}
		return fmt.Errorf("onboarding timeout")
	}
	return nil
}
