package openevec

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/projects"
	"github.com/lf-edge/eve/api/go/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type configChanger interface {
	getControllerAndDev() (controller.Cloud, *device.Ctx, error)
	setControllerAndDev(controller.Cloud, *device.Ctx) error
}

type fileChanger struct {
	fileConfig string
	oldHash    [32]byte
}

func changerByControllerMode(controllerMode string) (configChanger, error) {
	if controllerMode == "" {
		return &adamChanger{}, nil
	}
	modeType, modeURL, err := projects.GetControllerMode(controllerMode)
	if err != nil {
		return nil, err
	}
	log.Debugf("Mode type: %s", modeType)
	log.Debugf("Mode url: %s", modeURL)
	var changer configChanger
	switch modeType {
	case "file":
		changer = &fileChanger{fileConfig: modeURL}
	case "adam":
		changer = &adamChanger{adamURL: modeURL}

	default:
		return nil, fmt.Errorf("not implemented type: %s", modeType)
	}
	return changer, nil
}

func (ctx *fileChanger) getControllerAndDev() (controller.Cloud, *device.Ctx, error) {
	if ctx.fileConfig == "" {
		return nil, nil, fmt.Errorf("cannot use empty url for file")
	}
	if _, err := os.Lstat(ctx.fileConfig); os.IsNotExist(err) {
		return nil, nil, err
	}
	ctrl, err := controller.CloudPrepare()
	if err != nil {
		return nil, nil, err
	}
	data, err := ioutil.ReadFile(ctx.fileConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("file reading error: %w", err)
	}
	var deviceConfig config.EdgeDevConfig
	err = json.Unmarshal(data, &deviceConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("unmarshal error: %w", err)
	}
	dev, err := ctrl.ConfigParse(&deviceConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("configParse error: %w", err)
	}
	res, err := ctrl.GetConfigBytes(dev, false)
	if err != nil {
		return nil, nil, fmt.Errorf("GetConfigBytes error: %w", err)
	}
	ctx.oldHash = sha256.Sum256(res)
	return ctrl, dev, nil
}

func (ctx *fileChanger) setControllerAndDev(ctrl controller.Cloud, dev *device.Ctx) error {
	res, err := ctrl.GetConfigBytes(dev, false)
	if err != nil {
		return fmt.Errorf("GetConfigBytes error: %w", err)
	}
	if ctx.oldHash == sha256.Sum256(res) {
		log.Debug("config not modified")
		return nil
	}
	if res, err = controller.VersionIncrement(res); err != nil {
		return fmt.Errorf("VersionIncrement error: %w", err)
	}
	if err = ioutil.WriteFile(ctx.fileConfig, res, 0755); err != nil {
		return fmt.Errorf("WriteFile error: %w", err)
	}
	log.Debug("config modification done")
	return nil
}

type adamChanger struct {
	adamURL string
}

func (ctx *adamChanger) getController() (controller.Cloud, error) {
	if ctx.adamURL != "" { // overwrite config only if url defined
		ipPort := strings.Split(ctx.adamURL, ":")
		ip := ipPort[0]
		if ip == "" {
			return nil, fmt.Errorf("cannot get ip/hostname from %s", ctx.adamURL)
		}
		port := "80"
		if len(ipPort) > 1 {
			port = ipPort[1]
		}
		viper.Set("adam.ip", ip)
		viper.Set("adam.port", port)
	}
	ctrl, err := controller.CloudPrepare()
	if err != nil {
		return nil, fmt.Errorf("CloudPrepare error: %w", err)
	}
	return ctrl, nil
}

func (ctx *adamChanger) getControllerAndDev() (controller.Cloud, *device.Ctx, error) {
	ctrl, err := ctx.getController()
	if err != nil {
		return nil, nil, fmt.Errorf("getController error: %w", err)
	}
	devFirst, err := ctrl.GetDeviceCurrent()
	if err != nil {
		return nil, nil, fmt.Errorf("GetDeviceCurrent error: %w", err)
	}
	return ctrl, devFirst, nil
}

func (ctx *adamChanger) setControllerAndDev(ctrl controller.Cloud, dev *device.Ctx) error {
	if err := ctrl.ConfigSync(dev); err != nil {
		return fmt.Errorf("configSync error: %w", err)
	}
	return nil
}
