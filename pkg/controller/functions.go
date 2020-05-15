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
		return nil, fmt.Errorf("utils.InitVars: %s", err)
	}
	ctx := &CloudCtx{vars: vars, Controller: &adam.Ctx{}}
	if err := ctx.InitWithVars(vars); err != nil {
		return nil, fmt.Errorf("cloud.InitWithVars: %s", err)
	}
	devices, err := ctx.DeviceList()
	if err != nil {
		return ctx, fmt.Errorf("DeviceList.GetDevModelByName: %s", err)
	}
	for _, devID := range devices {
		if err = ctx.devInit(devID); err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

func (cloud *CloudCtx) GetVars() *utils.ConfigVars {
	return cloud.vars
}

func (cloud *CloudCtx) devInit(devID string) error {
	deviceModel, err := cloud.GetDevModelByName(cloud.vars.DevModel)
	if err != nil {
		return fmt.Errorf("cloud.GetDevModelByName: %s", err)
	}
	devUUID, err := uuid.FromString(devID)
	if err != nil {
		return fmt.Errorf("uuid.FromString(%s): %s", devID, err)
	}
	dev, err := cloud.GetDeviceUUID(devUUID)
	if err != nil {
		dev, err = cloud.AddDevice(devUUID)
		if err != nil {
			return fmt.Errorf("cloud.AddDevice(%s): %s", devUUID, err)
		}
	}
	if cloud.vars.SshKey != "" {
		b, err := ioutil.ReadFile(cloud.vars.SshKey)
		switch {
		case err != nil && os.IsNotExist(err):
			return fmt.Errorf("sshKey file %s does not exist", cloud.vars.SshKey)
		case err != nil:
			return fmt.Errorf("error reading sshKey file %s: %v", cloud.vars.SshKey, err)
		}
		dev.SetConfigItem("debug.enable.ssh", string(b))
	}
	dev.SetConfigItem("app.allow.vnc", "true")
	if err = cloud.ApplyDevModel(dev, deviceModel); err != nil {
		return fmt.Errorf("fail in ApplyDevModel: %s", err)
	}
	return nil
}

//OnBoard in controller
func (cloud *CloudCtx) OnBoard() error {
	devUUID, err := cloud.GetDeviceFirst()
	if devUUID == nil {
		log.Info("Try to add onboarding")
		err = cloud.Register(cloud.vars.EveCert, cloud.vars.EveSerial)
		if err != nil {
			return fmt.Errorf("ctx.register: %s", err)
		}
		res, err := cloud.OnBoardList()
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
			cmdOut, err := cloud.DeviceList()
			if err != nil {
				return fmt.Errorf("ctx.DeviceList: %s", err)
			}
			if len(cmdOut) > 0 {
				log.Info("Done onboarding in adam!")
				log.Infof("Device uuid: %s", cmdOut[0])
				return cloud.devInit(cmdOut[0])
			}
			log.Infof("Attempt to list devices (%d) of (%d)", i, maxRepeat)
			time.Sleep(delayTime)
		}
		return fmt.Errorf("onboarding timeout")
	}
	return nil
}
