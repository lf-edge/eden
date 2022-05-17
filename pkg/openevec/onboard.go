package openevec

import (
	"fmt"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
)

func OnboardEve(eveUUID string) error {

	edenDir, err := utils.DefaultEdenDir()
	if err != nil {
		return fmt.Errorf("Error getting default eden dir %s", err)
	}
	if err = utils.TouchFile(filepath.Join(edenDir, fmt.Sprintf("state-%s.yml", eveUUID))); err != nil {
		return fmt.Errorf("Error getting file %s", err)
	}
	changer := &adamChanger{}
	ctrl, err := changer.getController()
	if err != nil {
		return fmt.Errorf("Error fetching controller %s", err)
	}
	vars := ctrl.GetVars()
	dev, err := ctrl.GetDeviceCurrent()
	if err != nil || dev == nil {
		//create new one if not exists
		dev = device.CreateEdgeNode()
		dev.SetSerial(vars.EveSerial)
		dev.SetOnboardKey(vars.EveCert)
		dev.SetDevModel(vars.DevModel)
		err = ctrl.OnBoardDev(dev)
		if err != nil {
			return fmt.Errorf("Error onboarding %s", err)
		}
	}
	if err = ctrl.StateUpdate(dev); err != nil {
		return fmt.Errorf("Error fetching state %s", err)
	}
	log.Info("onboarded")
	log.Info("device UUID: ", dev.GetID().String())

	return nil
}
