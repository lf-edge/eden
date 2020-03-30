package integration

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/controller"
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

func controllerPrepare() (ctx *controller.Ctx, err error) {
	err = envRead()
	if err != nil {
		return ctx, err
	}
	ctx = &controller.Ctx{
		Dir:         adamDir,
		URL:         fmt.Sprintf("https://%s:%s", adamIP, adamPort),
		InsecureTLS: true,
	}
	if len(adamCA) != 0 {
		ctx.ServerCA = adamCA
		ctx.InsecureTLS = false
	}
	devices, err := ctx.DeviceList()
	if err != nil {
		return ctx, err
	}
	for _, dev := range devices {
		devUUID, err := uuid.FromString(dev)
		if err != nil {
			return ctx, err
		}
		err = ctx.AddDevice(&devUUID)
		if err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}
