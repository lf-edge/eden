package openevec

import (
	"fmt"
	"os"

	"github.com/lf-edge/eden/pkg/eden"
	log "github.com/sirupsen/logrus"
)

func (openEVEC *OpenEVEC) StartAdam() error {
	cfg := openEVEC.cfg
	command, err := os.Executable()
	if err != nil {
		return fmt.Errorf("startAdam: cannot obtain executable path: %w", err)
	}
	log.Infof("Executable path: %s", command)

	if !cfg.Adam.Remote.Redis {
		cfg.Adam.Redis.RemoteURL = ""
	}

	if err := eden.StartAdam(cfg.Adam.Port, cfg.Adam.Dist, cfg.Adam.Force, cfg.Adam.Tag,
		cfg.Adam.Redis.RemoteURL, cfg.Adam.APIv1, cfg.Eden.EnableIPv6, cfg.Eden.IPv6Subnet); err != nil {
		return fmt.Errorf("cannot start adam: %w", err)
	}
	log.Infof("Adam is runnig and accessible on port %d", cfg.Adam.Port)
	return nil
}

func (openEVEC *OpenEVEC) GetAdamStatus() (string, error) {
	statusAdam, err := eden.StatusAdam()
	if err != nil {
		return "", fmt.Errorf("cannot obtain status of adam: %w", err)
	} else {
		return statusAdam, nil
	}
}

func (openEVEC *OpenEVEC) StartRedis() error {
	cfg := openEVEC.cfg
	if err := eden.StartRedis(cfg.Redis.Port, cfg.Adam.Redis.Dist, cfg.Redis.Force, cfg.Redis.Tag,
		cfg.Eden.EnableIPv6, cfg.Eden.IPv6Subnet); err != nil {
		return fmt.Errorf("cannot start redis: %w", err)
	}
	log.Infof("Redis is running and accessible on port %d", cfg.Redis.Port)
	return nil
}

func (openEVEC *OpenEVEC) StartRegistry() error {
	cfg := openEVEC.cfg
	if err := eden.StartRegistry(cfg.Registry.Port, cfg.Registry.Tag, cfg.Registry.Dist,
		cfg.Eden.EnableIPv6, cfg.Eden.IPv6Subnet); err != nil {
		return fmt.Errorf("cannot start registry: %w", err)
	}
	log.Infof("registry is running and accesible on port %d", cfg.Registry.Port)
	return nil
}

func (openEVEC *OpenEVEC) StartEServer() error {
	cfg := openEVEC.cfg
	if err := eden.StartEServer(cfg.Eden.EServer.Port, cfg.Eden.Images.EServerImageDist,
		cfg.Eden.EServer.Force, cfg.Eden.EServer.Tag, cfg.Eden.EnableIPv6, cfg.Eden.IPv6Subnet); err != nil {
		return fmt.Errorf("cannot start eserver: %w", err)
	}
	log.Infof("Eserver is running and accesible on port %d", cfg.Eden.EServer.Port)
	return nil
}

func (openEVEC *OpenEVEC) StartEden(vmName, zedControlURL, tapInterface string) error {
	cfg := openEVEC.cfg
	// Note that custom installer only works with zedcloud controller.
	useZedcloud := cfg.Eve.CustomInstaller.Path != "" || zedControlURL != ""

	if !useZedcloud {
		if err := openEVEC.StartRedis(); err != nil {
			return fmt.Errorf("cannot start redis %w", err)
		}

		if err := openEVEC.StartAdam(); err != nil {
			return fmt.Errorf("cannot start adam %w", err)
		}

		if err := openEVEC.StartRegistry(); err != nil {
			return fmt.Errorf("cannot start registry %w", err)
		}

		if err := openEVEC.StartEServer(); err != nil {
			return fmt.Errorf("cannot start adam %w", err)
		}
	}

	if cfg.Eve.Remote {
		return nil
	}

	if err := openEVEC.StartEve(vmName, tapInterface); err != nil {
		return fmt.Errorf("cannot start eve %w", err)
	}
	log.Infof("EVE is starting")
	return nil
}
