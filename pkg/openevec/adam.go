package openevec

import (
	"fmt"
	"os"

	"github.com/lf-edge/eden/pkg/eden"
	log "github.com/sirupsen/logrus"
)

func AdamStart(cfg *EdenSetupArgs) error {
	command, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot obtain executable path: %w", err)
	}
	log.Infof("Executable path: %s", command)
	// TODO: This should be removed to processing config
	if !cfg.Adam.Remote.Redis {
		cfg.Adam.Redis.RemoteURL = ""
	}
	if err := eden.StartAdam(cfg.Adam.Port, cfg.Adam.Dist, cfg.Adam.Force, cfg.Adam.Tag, cfg.Adam.Redis.RemoteURL, cfg.Adam.APIv1); err != nil {
		log.Errorf("cannot start adam: %s", err.Error())
	} else {
		log.Infof("Adam is running and accessible on port %d", cfg.Adam.Port)
	}
	return nil
}
