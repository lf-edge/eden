package openevec

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
)

func (cfg EdenSetupArgs) WithContext(name string) EdenSetupArgs {
	cfg.Eden.SSHKey = fmt.Sprintf("%s-%s", name, defaults.DefaultSSHKey)
	cfg.Eve.Name = strings.ToLower(name)
	cfg.Eve.Dist = fmt.Sprintf("%s-%s", name, defaults.DefaultEVEDist)
	cfg.Eve.QemuFileToSave = filepath.Join(cfg.EdenDir, fmt.Sprintf("%s-%s", name, defaults.DefaultQemuFileToSave))
	cfg.Eve.Pid = fmt.Sprintf("%s-eve.pid", strings.ToLower(name))
	cfg.Eve.Log = fmt.Sprintf("%s-eve.log", strings.ToLower(name))
	cfg.ConfigName = name
	cfg.ConfigFile = utils.GetConfig(name)

	return cfg
}
