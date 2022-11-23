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
		return fmt.Errorf("startAdam: cannot obtain executable path: %s", err)
	}
	log.Infof("Executable path: %s", command)

	if !cfg.Adam.Remote.Redis {
		cfg.Adam.Redis.RemoteURL = ""
	}

	if err := eden.StartAdam(cfg.Adam.Port, cfg.Adam.Dist, cfg.Adam.Force, cfg.Adam.Tag, cfg.Adam.Redis.RemoteURL, cfg.Adam.ApiV1); err != nil {
		return fmt.Errorf("cannot start adam: %s", err)
	} else {
		log.Infof("Adam is runnig and accessible on port %d", cfg.Adam.Port)
	}
	return nil
}

func stopAdam(configFile string) error {
	adamRm := viper.GetBool("adam-rm")

	if err := eden.StopAdam(adamRm); err != nil {
		return fmt.Errorf("Cannot stop adam: %s", err)
	}
	return nil
}

func GetAdamStatus() (string, error) {
	statusAdam, err := eden.StatusAdam()
	if err != nil {
		return "", fmt.Errorf("Cannot obtain status of adam: %s", err)
	} else {
		return statusAdam, nil
	}
}

func StartRedis(cfg EdenSetupArgs) error {
	if err := eden.StartRedis(cfg.Redis.Port, cfg.Adam.Redis.Dist, cfg.Redis.Force, cfg.Redis.Tag); err != nil {
		return fmt.Errorf("cannot start redis: %s", err)
	} else {
		log.Infof("Redis is running and accessible on port %d", cfg.Redis.Port)
		return nil
	}
}

func StartRegistry(cfg EdenSetupArgs) error {
	if err := eden.StartRegistry(cfg.Registry.Port, cfg.Registry.Tag, cfg.Registry.Dist); err != nil {
		log.Errorf("cannot start registry: %s", err)
	} else {
		log.Infof("registry is running and accesible on port %d", cfg.Registry.Port)
	}
	return nil
}

func StartEServer(cfg EdenSetupArgs) error {
	if err := eden.StartEServer(cfg.Eden.Eserver.Port, cfg.Eden.Images.EserverImageDist, cfg.Eden.Eserver.Force, cfg.Eden.Eserver.Tag); err != nil {
		return fmt.Errorf("cannot start eserver: %s", err)
	} else {
		log.Infof("Eserver is running and accesible on port %d", cfg.Eden.Eserver.Port)
		return nil
	}
}

func StartEden(cfg *EdenSetupArgs, vmName string) error {

	if err := StartRedis(*cfg); err != nil {
		return fmt.Errorf("Cannot start redis %s", err)
	}

	if err := StartAdam(*cfg); err != nil {
		return fmt.Errorf("Cannot start adam %s", err)
	}

	if err := StartRegistry(*cfg); err != nil {
		return fmt.Errorf("Cannot start registry %s", err)
	} else {
		log.Infof("Registry is running and accesible on port %d", cfg.Registry.Port)
	}

	if err := StartEServer(*cfg); err != nil {
		return fmt.Errorf("Cannot start adam %s", err)
	}

	if cfg.Eve.Remote {
		return nil
	}

	if err := StartEve(vmName, cfg); err != nil {
		return fmt.Errorf("Cannot start eve %s", err)
	} else {
		log.Infof("EVE is starting")
	}
	return nil
}
