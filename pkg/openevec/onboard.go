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
		return fmt.Errorf("error getting default eden dir %w", err)
	}
	if err = utils.TouchFile(filepath.Join(edenDir, fmt.Sprintf("state-%s.yml", eveUUID))); err != nil {
		return fmt.Errorf("error getting file %w", err)
	}
	changer := &adamChanger{}
	ctrl, err := changer.getController()
	if err != nil {
		return fmt.Errorf("error fetching controller %w", err)
	}
	vars := ctrl.GetVars()
	dev, err := ctrl.GetDeviceCurrent()
	if err != nil || dev == nil {
		// create new one if not exists
		dev = device.CreateEdgeNode()
		dev.SetSerial(vars.EveSerial)
		dev.SetOnboardKey(vars.EveCert)
		dev.SetDevModel(vars.DevModel)
		err = ctrl.OnBoardDev(dev)
		if err != nil {
			return fmt.Errorf("error onboarding %w", err)
		}
	}
	if err = ctrl.StateUpdate(dev); err != nil {
		return fmt.Errorf("error fetching state %w", err)
	}
	log.Info("onboarded")
	log.Info("device UUID: ", dev.GetID().String())

	return nil
}
