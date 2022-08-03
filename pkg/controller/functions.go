package controller

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lf-edge/eden/pkg/controller/adam"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/models"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/term"
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

//GetVars returns variables of controller
func (cloud *CloudCtx) GetVars() *utils.ConfigVars {
	return cloud.vars
}

//SetVars sets variables of controller
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
		cert, err := utils.ParseFirstCertFromBlock(b)
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
			node.SetRemote(cloud.vars.EveRemote)
			node.SetRemoteAddr(cloud.vars.EveRemoteAddr)
			if !alreadyRegistered { //new node
				node.SetConfigItem("timer.config.interval", "10")
				node.SetConfigItem("timer.location.app.interval", "10")
				node.SetConfigItem("timer.location.cloud.interval", "300")
				node.SetConfigItem("app.allow.vnc", "true")
				node.SetConfigItem("newlog.allow.fastupload", "true")
				node.SetConfigItem("timer.download.retry", "60")
				// TODO: allow to enable/disable:
				//node.SetConfigItem("network.fallback.any.eth", "disabled")
				log.Debugf("will apply devModel %s", node.GetDevModel())
				deviceModel, err := models.GetDevModelByName(node.GetDevModel())
				if err != nil {
					log.Fatalf("fail to get dev model %s: %s", node.GetDevModel(), err)
				}
				if cloud.vars.EveSSID != "" {
					ssid := cloud.vars.EveSSID
					fmt.Printf("Enter password for wifi %s: ", ssid)
					pass, _ := term.ReadPassword(0)
					wifiPSK := strings.ToLower(hex.EncodeToString(pbkdf2.Key(pass, []byte(ssid), 4096, 32, sha1.New)))
					fmt.Println()
					deviceModel.SetWiFiParams(cloud.vars.EveSSID, wifiPSK)
				}
				if cloud.vars.AdamLogLevel != "" {
					node.SetConfigItem("debug.default.remote.loglevel", cloud.vars.AdamLogLevel)
				}
				if cloud.vars.LogLevel != "" {
					node.SetConfigItem("debug.default.loglevel", cloud.vars.LogLevel)
				}
				if cloud.vars.SSHKey != "" {
					b, err := ioutil.ReadFile(cloud.vars.SSHKey)
					switch {
					case err != nil && os.IsNotExist(err):
						return fmt.Errorf("sshKey file %s does not exist", cloud.vars.SSHKey)
					case err != nil:
						return fmt.Errorf("error reading sshKey file %s: %v", cloud.vars.SSHKey, err)
					}
					node.SetConfigItem("debug.enable.ssh", string(b))
				}
				if err = cloud.ApplyDevModel(node, deviceModel); err != nil {
					return fmt.Errorf("fail in ApplyDevModel: %s", err)
				}
				if err = cloud.ConfigSync(node); err != nil {
					log.Fatal(err)
				}
				//wait for certs
				if _, err = cloud.CertsGet(node.GetID()); err != nil {
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
	existingID := deviceConfig.Id
	oldVersion := 0
	newVersion, versionError := strconv.Atoi(existingID.Version)
	if versionError == nil {
		oldVersion = newVersion
		newVersion++
	}
	if deviceConfig.Id == nil {
		if versionError != nil {
			return nil, fmt.Errorf("cannot automatically non-number bump version %s", existingID.Version)
		}
		deviceConfig.Id = &config.UUIDandVersion{
			Uuid:    existingID.Uuid,
			Version: strconv.Itoa(newVersion),
		}
	} else {
		if deviceConfig.Id.Version == "" {
			if versionError != nil {
				return nil, fmt.Errorf("cannot automatically non-number bump version %s", existingID.Version)
			}
			deviceConfig.Id.Version = strconv.Itoa(newVersion)
		} else {
			deviceConfig.Id.Version = strconv.Itoa(newVersion)
		}
	}
	log.Debugf("VersionIncrement %d->%s", oldVersion, deviceConfig.Id.Version)
	return json.Marshal(&deviceConfig)
}
