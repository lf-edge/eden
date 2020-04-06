package integration

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/adam"
	"github.com/lf-edge/eden/pkg/utils"
	uuid "github.com/satori/go.uuid"
	"os"
	"path"
	"path/filepath"
)

var (
	adamIP   string
	adamPort string
	adamDir  string
	adamCA   string
)

//envRead use environment variables for init controller
//environment variable ADAM_IP - IP of adam
//environment variable ADAM_PORT - PORT of adam
//environment variable ADAM_DIST - directory of adam (absolute path)
//environment variable ADAM_CA - CA of adam for https
func envRead() error {
	currentPath, err := os.Getwd()
	adamIP = os.Getenv("ADAM_IP")
	if len(adamIP) == 0 {
		adamIP, err = utils.GetIPForDockerAccess()
		if err != nil {
			return err
		}
	}
	adamPort = os.Getenv("ADAM_PORT")
	if len(adamPort) == 0 {
		adamPort = "3333"
	}
	adamDir = os.Getenv("ADAM_DIST")
	if len(adamDir) == 0 {
		adamDir = path.Join(filepath.Dir(filepath.Dir(currentPath)), "dist", "adam")
		if stat, err := os.Stat(adamDir); err != nil || !stat.IsDir() {
			return err
		}
	}

	adamCA = os.Getenv("ADAM_CA")
	return nil
}

//controllerPrepare is for init controller connection and obtain device list
func controllerPrepare() (ctx controller.Cloud, err error) {
	err = envRead()
	if err != nil {
		return ctx, err
	}
	var ctrl controller.Cloud = &controller.CloudCtx{Controller: &adam.Ctx{
		Dir:         adamDir,
		URL:         fmt.Sprintf("https://%s:%s", adamIP, adamPort),
		InsecureTLS: true,
	}}
	if len(adamCA) != 0 {
		ctrl = &controller.CloudCtx{Controller: &adam.Ctx{
			Dir:         adamDir,
			URL:         fmt.Sprintf("https://%s:%s", adamIP, adamPort),
			InsecureTLS: false,
			ServerCA:    adamCA,
		}}
	}
	devices, err := ctrl.DeviceList()
	if err != nil {
		return ctrl, err
	}
	for _, dev := range devices {
		devUUID, err := uuid.FromString(dev)
		if err != nil {
			return ctrl, err
		}
		err = ctrl.AddDevice(&devUUID)
		if err != nil {
			return ctrl, err
		}
	}
	return ctrl, nil
}
