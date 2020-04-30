package controller

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/controller/adam"
	"github.com/lf-edge/eden/pkg/utils"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"
)

type configVars struct {
	adamIP     string
	adamPort   string
	adamDir    string
	adamCA     string
	eveBaseTag string
	eveHV      string
	sshKey     string
	checkLogs  bool
	eveCert    string
	eveSerial  string
	zArch      string
	devModel   string
}

//envRead use environment variables for init controller
//environment variable ADAM_IP - IP of adam
//environment variable ADAM_PORT - PORT of adam
//environment variable ADAM_DIST - directory of adam (absolute path)
//environment variable ADAM_CA - CA file of adam for https
//environment variable SSH_KEY - ssh public key for integrate into eve
//environment variable EVE_CERT - path to eve onboarding cert
//environment variable EVE_SERIAL - serial number of eve
//environment variable EVE_BASE_REF - version of eve image
//environment variable ZARCH - architecture of eve image
//environment variable HV - hypervisor of eve image
func envRead() (*configVars, error) {
	var params configVars
	configPath, err := utils.DefaultConfigPath()
	if err != nil {
		return nil, err
	}
	loaded, err := utils.LoadConfigFile(configPath)
	if err != nil {
		return nil, err
	}
	if !loaded {
		currentPath, err := os.Getwd()
		params.adamIP = os.Getenv("ADAM_IP")
		if len(params.adamIP) == 0 {
			params.adamIP, err = utils.GetIPForDockerAccess()
			if err != nil {
				return nil, err
			}
		}
		params.adamPort = os.Getenv("ADAM_PORT")
		if len(params.adamPort) == 0 {
			params.adamPort = "3333"
		}
		params.adamDir = os.Getenv("ADAM_DIST")
		if len(params.adamDir) == 0 {
			params.adamDir = path.Join(filepath.Dir(filepath.Dir(currentPath)), "dist", "adam")
			if stat, err := os.Stat(params.adamDir); err != nil || !stat.IsDir() {
				return nil, err
			}
		}

		params.adamCA = os.Getenv("ADAM_CA")
		params.sshKey = os.Getenv("SSH_KEY")
		params.checkLogs = os.Getenv("LOGS") != ""
		params.eveCert = os.Getenv("EVE_CERT")
		if len(params.eveCert) == 0 {
			params.eveCert = path.Join(params.adamDir, "run", "config", "onboard.cert.pem")
		}
		params.eveSerial = os.Getenv("EVE_SERIAL")
		if params.eveSerial == "" {
			params.eveSerial = "31415926"
		}
		params.eveBaseTag = os.Getenv("EVE_BASE_REF")
		if len(params.eveBaseTag) == 0 {
			params.eveBaseTag = "4.10.0"
		}
		params.zArch = os.Getenv("ZARCH")
		if len(params.eveBaseTag) == 0 {
			params.zArch = "amd64"
		}
		params.eveHV = os.Getenv("HV")
		if params.eveHV == "xen" {
			params.eveHV = ""
		}
	} else {
		params.adamIP = viper.GetString("adam.ip")
		params.adamPort = viper.GetString("adam.port")
		params.adamDir = utils.ResolveAbsPath(viper.GetString("adam.dist"))
		params.adamCA = utils.ResolveAbsPath(viper.GetString("adam.ca"))
		params.sshKey = utils.ResolveAbsPath(viper.GetString("eden.ssh-key"))
		params.checkLogs = viper.GetBool("eden.logs")
		params.eveCert = utils.ResolveAbsPath(viper.GetString("eve.cert"))
		params.eveSerial = viper.GetString("eve.serial")
		params.zArch = viper.GetString("eve.arch")
		params.eveHV = viper.GetString("eve.hv")
		params.eveBaseTag = fmt.Sprintf("%s-%s-%s", viper.GetString("eve.base-tag"), params.eveHV, params.zArch)
		params.devModel = viper.GetString("eve.devmodel")
	}
	return &params, nil
}

//controllerPrepare is for init controller connection and obtain device list
func controllerPrepare() (ctx Cloud, params *configVars, err error) {
	params, err = envRead()
	if err != nil {
		return nil, params, err
	}
	var ctrl Cloud = &CloudCtx{Controller: &adam.Ctx{
		Dir:         params.adamDir,
		URL:         fmt.Sprintf("https://%s:%s", params.adamIP, params.adamPort),
		InsecureTLS: true,
	}}
	if len(params.adamCA) != 0 {
		ctrl = &CloudCtx{Controller: &adam.Ctx{
			Dir:         params.adamDir,
			URL:         fmt.Sprintf("https://%s:%s", params.adamIP, params.adamPort),
			InsecureTLS: false,
			ServerCA:    params.adamCA,
		}}
	}
	deviceModel, err := ctrl.GetDevModelByName(params.devModel)
	if err != nil {
		return ctx, params, err
	}
	devices, err := ctrl.DeviceList()
	if err != nil {
		return ctrl, params, err
	}
	for _, devID := range devices {
		devUUID, err := uuid.FromString(devID)
		if err != nil {
			return ctrl, params, err
		}
		dev, err := ctrl.AddDevice(devUUID)
		if err != nil {
			return ctrl, params, err
		}
		if params.sshKey != "" {
			b, err := ioutil.ReadFile(params.sshKey)
			switch {
			case err != nil && os.IsNotExist(err):
				return nil, params, fmt.Errorf("sshKey file %s does not exist", params.sshKey)
			case err != nil:
				return nil, params, fmt.Errorf("error reading sshKey file %s: %v", params.sshKey, err)
			}
			dev.SetSSHKeys([]string{string(b)})
		}
		dev.SetVncAccess(true)
		dev.SetControllerLogLevel("info")
		err = ctrl.ApplyDevModel(dev, deviceModel)
		if err != nil {
			return ctrl, params, fmt.Errorf("fail in ApplyDevModel: %s", err)
		}
	}
	return ctrl, params, nil
}

//OnBoard in Adam
func OnBoard() {
	ctx, params, err := controllerPrepare()
	if ctx == nil {
		log.Fatalf("Fail in controller prepare: %s", err)
	}
	devUUID, err := ctx.GetDeviceFirst()
	if devUUID == nil {
		log.Info("Try to add onboarding")
		err = ctx.Register(params.eveCert, params.eveSerial)
		if err != nil {
			log.Fatalf("Register: %s", err)
		}
		res, err := ctx.OnBoardList()
		if err != nil {
			log.Fatalf("OnBoardList: %s", err)
		}
		if len(res) == 0 {
			log.Fatal("No onboard in list")
		}
		log.Info(res)

		maxRepeat := 20
		delayTime := 20 * time.Second

		for i := 0; i < maxRepeat; i++ {
			cmdOut, err := ctx.DeviceList()
			if err != nil {
				log.Fatalf("DeviceList: %s", err)
			}
			if len(cmdOut) > 0 {
				log.Info("Done onboarding in adam!")
				log.Infof("Device uuid: %s", cmdOut)
				return
			}
			log.Infof("Attempt to list devices (%d) of (%d)", i, maxRepeat)
			time.Sleep(delayTime)
		}
		log.Fatal("Onboarding timeout")
	}
}
