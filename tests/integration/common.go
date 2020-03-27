package integration

import (
	"errors"
	"fmt"
	"github.com/itmo-eve/eden/pkg/adam"
	"github.com/itmo-eve/eden/pkg/utils"
	uuid "github.com/satori/go.uuid"
	"os"
	"path"
	"path/filepath"
)

func adamPrepare() (adamCtx *adam.Ctx, id *uuid.UUID, err error) {
	currentPath, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	ip := os.Getenv("IP")
	if len(ip) == 0 {
		ip, err = utils.GetIPForDockerAccess()
		if err != nil {
			return nil, nil, err
		}
	}
	port := os.Getenv("ADAM_PORT")
	if len(port) == 0 {
		port = "3333"
	}
	adamDir := os.Getenv("ADAM_DIST")
	if len(adamDir) == 0 {
		adamDir = path.Join(filepath.Dir(filepath.Dir(currentPath)), "dist", "adam")
		if stat, err := os.Stat(adamDir); err != nil || !stat.IsDir() {
			return nil, nil, err
		}
	}
	ctx := adam.Ctx{
		Dir: adamDir,
		URL: fmt.Sprintf("https://%s:%s", ip, port),
	}
	cmdOut, err := ctx.DeviceList()
	if err != nil {
		return nil, nil, err
	}
	if len(cmdOut) > 0 {
		devUUID, err := uuid.FromString(cmdOut[0])
		if err != nil {
			return nil, nil, err
		}
		return &ctx, &devUUID, nil
	} else {
		return nil, nil, errors.New("no device found")
	}
}
