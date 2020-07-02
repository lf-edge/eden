package controller

import (
	"encoding/json"
	"fmt"
	"github.com/lf-edge/adam/pkg/x509"
	"github.com/lf-edge/eden/pkg/controller/adam"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
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
	ctx.GetAllNodes()
	return ctx, nil
}

func (cloud *CloudCtx) GetVars() *utils.ConfigVars {
	return cloud.vars
}

func (cloud *CloudCtx) SetVars(vars *utils.ConfigVars) {
	cloud.vars = vars
}

//OnBoardDev in controller
func (cloud *CloudCtx) OnBoardDev(node *device.Ctx) error {
	edenDir, err := utils.DefaultEdenDir()
	if err != nil {
		return err
	}
	alreadyRegistered := false
	oldDevUUID, _ := cloud.DeviceGetByOnboard(node.GetOnboardKey())
	if oldDevUUID != uuid.Nil {
		b, err := ioutil.ReadFile(node.GetOnboardKey())
		switch {
		case err != nil && os.IsNotExist(err):
			log.Printf("cert file %s does not exist", node.GetOnboardKey())
			return err
		case err != nil:
			log.Printf("error reading cert file %s: %v", node.GetOnboardKey(), err)
			return err
		}
		cert, err := x509.ParseCert(b)
		if err != nil {
			return err
		}
		uuidToFound, err := uuid.FromString(cert.Subject.CommonName)
		if err != nil {
			return err
		}
		fi, err := os.Stat(filepath.Join(edenDir, fmt.Sprintf("state-%s.yml", uuidToFound)))
		if err == nil {
			size := fi.Size()
			if size > 0 {
				alreadyRegistered = true
			}
		}
	}
	err = cloud.Register(node)
	if err != nil {
		return fmt.Errorf("register: %s", err)
	}

	maxRepeat := 20
	delayTime := 20 * time.Second

	for i := 0; i < maxRepeat; i++ {
		dev, err := cloud.DeviceGetByOnboard(node.GetOnboardKey())
		if err != nil {
			log.Debugf("DeviceGetByOnboard %s", err)
			log.Infof("Adam waiting for EVE registration (%d) of (%d)", i, maxRepeat)
			time.Sleep(delayTime)
		} else {
			log.Debug("Done onboarding in adam!")
			log.Infof("Device uuid: %s", dev.String())
			node.SetID(dev)
			node.SetState(device.Onboarded)
			if !alreadyRegistered { //new node
				node.SetConfigItem("app.allow.vnc", "true")
				log.Debugf("will apply devModel %s", node.GetDevModel())
				deviceModel, err := cloud.GetDevModelByName(node.GetDevModel())
				if err != nil {
					log.Fatalf("fail to get dev model %s: %s", node.GetDevModel(), err)
				}
				if cloud.vars.SshKey != "" {
					b, err := ioutil.ReadFile(cloud.vars.SshKey)
					switch {
					case err != nil && os.IsNotExist(err):
						return fmt.Errorf("sshKey file %s does not exist", cloud.vars.SshKey)
					case err != nil:
						return fmt.Errorf("error reading sshKey file %s: %v", cloud.vars.SshKey, err)
					}
					node.SetConfigItem("debug.enable.ssh", string(b))
				}
				if err = cloud.ApplyDevModel(node, deviceModel); err != nil {
					return fmt.Errorf("fail in ApplyDevModel: %s", err)
				}
				if err = cloud.ConfigSync(node); err != nil {
					log.Fatal(err)
				}
			}
			return nil
		}
	}
	return fmt.Errorf("onboarding timeout. You may try to run 'eden eve onboard' command again in several minutes. If not successful see logs of adam/eve")
}

//VersionIncrement use []byte with config.EdgeDevConfig and increment config version
func VersionIncrement(configOld []byte) ([]byte, error) {
	var deviceConfig config.EdgeDevConfig
	if err := json.Unmarshal(configOld, &deviceConfig); err != nil {
		return nil, fmt.Errorf("unmarshal error: %s", err)
	}
	existingId := deviceConfig.Id
	oldVersion := 0
	newVersion, versionError := strconv.Atoi(existingId.Version)
	if versionError == nil {
		oldVersion = newVersion
		newVersion++
	}
	if deviceConfig.Id == nil {
		if versionError != nil {
			return nil, fmt.Errorf("cannot automatically non-number bump version %s", existingId.Version)
		}
		deviceConfig.Id = &config.UUIDandVersion{
			Uuid:    existingId.Uuid,
			Version: strconv.Itoa(newVersion),
		}
	} else {
		if deviceConfig.Id.Version == "" {
			if versionError != nil {
				return nil, fmt.Errorf("cannot automatically non-number bump version %s", existingId.Version)
			}
			deviceConfig.Id.Version = strconv.Itoa(newVersion)
		} else {
			deviceConfig.Id.Version = strconv.Itoa(newVersion)
		}
	}
	log.Debugf("VersionIncrement %d->%s", oldVersion, deviceConfig.Id.Version)
	return json.Marshal(&deviceConfig)
}
