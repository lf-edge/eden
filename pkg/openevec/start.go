package openevec

import (
	"fmt"
	"os"

	"github.com/lf-edge/eden/pkg/eden"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func StartAdam(cfg EdenSetupArgs) error {
	command, err := os.Executable()
	if err != nil {
		return fmt.Errorf("startAdam: cannot obtain executable path: %w", err)
	}
	log.Infof("Executable path: %s", command)

	if !cfg.Adam.Remote.Redis {
		cfg.Adam.Redis.RemoteURL = ""
	}

	if err := eden.StartAdam(cfg.Adam.Port, cfg.Adam.Dist, cfg.Adam.Force, cfg.Adam.Tag, cfg.Adam.Redis.RemoteURL, cfg.Adam.APIv1); err != nil {
		return fmt.Errorf("cannot start adam: %w", err)
	}
	log.Infof("Adam is runnig and accessible on port %d", cfg.Adam.Port)
	return nil
}

func stopAdam(_ string) error {
	adamRm := viper.GetBool("adam-rm")

	if err := eden.StopAdam(adamRm); err != nil {
		return fmt.Errorf("cannot stop adam: %w", err)
	}
	return nil
}

func GetAdamStatus() (string, error) {
	statusAdam, err := eden.StatusAdam()
	if err != nil {
		return "", fmt.Errorf("cannot obtain status of adam: %w", err)
	} else {
		return statusAdam, nil
	}
}

func StartRedis(cfg EdenSetupArgs) error {
	if err := eden.StartRedis(cfg.Redis.Port, cfg.Adam.Redis.Dist, cfg.Redis.Force, cfg.Redis.Tag); err != nil {
		return fmt.Errorf("cannot start redis: %w", err)
	}
	log.Infof("Redis is running and accessible on port %d", cfg.Redis.Port)
	return nil
}

func StartRegistry(cfg EdenSetupArgs) error {
	if err := eden.StartRegistry(cfg.Registry.Port, cfg.Registry.Tag, cfg.Registry.Dist); err != nil {
		return fmt.Errorf("cannot start registry: %w", err)
	}
	log.Infof("registry is running and accesible on port %d", cfg.Registry.Port)
	return nil
}

func StartEServer(cfg EdenSetupArgs) error {
	if err := eden.StartEServer(cfg.Eden.EServer.Port, cfg.Eden.Images.EServerImageDist, cfg.Eden.EServer.Force, cfg.Eden.EServer.Tag); err != nil {
		return fmt.Errorf("cannot start eserver: %w", err)
	}
	log.Infof("Eserver is running and accesible on port %d", cfg.Eden.EServer.Port)
	return nil
}

func StartEden(cfg *EdenSetupArgs, vmName string) error {

	if err := StartRedis(*cfg); err != nil {
		return fmt.Errorf("cannot start redis %w", err)
	}

	if err := StartAdam(*cfg); err != nil {
		return fmt.Errorf("cannot start adam %w", err)
	}

	if err := StartRegistry(*cfg); err != nil {
		return fmt.Errorf("cannot start registry %w", err)
	}

	if err := StartEServer(*cfg); err != nil {
		return fmt.Errorf("cannot start adam %w", err)
	}

	if cfg.Eve.Remote {
		return nil
	}

	if err := StartEve(vmName, cfg); err != nil {
		return fmt.Errorf("cannot start eve %w", err)
	}
	log.Infof("EVE is starting")
	return nil
}
