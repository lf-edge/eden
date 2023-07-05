package openevec

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
)

func StartAdamCmd(cfg EdenSetupArgs) error {
	command, err := os.Executable()
	if err != nil {
		return fmt.Errorf("startAdam: cannot obtain executable path: %w", err)
	}
	log.Infof("Executable path: %s", command)

	if !cfg.Adam.Remote.Redis {
		cfg.Adam.Redis.RemoteURL = ""
	}

	if err := StartAdam(cfg.Eden.CertsDir, cfg.Adam); err != nil {
		return fmt.Errorf("cannot start adam: %w", err)
	}
	log.Infof("Adam is runnig and accessible on port %d", cfg.Adam.Port)
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
	portMap := map[string]string{"6379": strconv.Itoa(cfg.Redis.Port)}
	volumeMap := map[string]string{"/data": cfg.Redis.Dist}
	redisServerCommand := strings.Fields("redis-server --appendonly yes")
	edenHome, err := utils.DefaultEdenDir()
	if err != nil {
		return err
	}
	globalCertsDir := filepath.Join(edenHome, defaults.DefaultCertsDist)
	redisPasswordFile := filepath.Join(globalCertsDir, defaults.DefaultRedisPasswordFile)
	pwd, err := ioutil.ReadFile(redisPasswordFile)
	if err == nil {
		redisServerCommand = append(redisServerCommand, strings.Fields(fmt.Sprintf("--requirepass %s", string(pwd)))...)
	} else {
		log.Errorf("cannot read redis password: %v", err)
	}
	if cfg.Adam.Redis.Dist != "" {
		if err = os.MkdirAll(cfg.Adam.Redis.Dist, 0755); err != nil {
			return fmt.Errorf("StartRedis: Cannot create directory for redis (%s): %s", cfg.Adam.Redis.Dist, err)
		}
	}
	if cfg.Redis.Force {
		_ = utils.StopContainer(defaults.DefaultRedisContainerName, true)
		if err := utils.CreateAndRunContainer(defaults.DefaultRedisContainerName, defaults.DefaultRedisContainerRef+":"+cfg.Redis.Tag, portMap, volumeMap, redisServerCommand, nil); err != nil {
			return fmt.Errorf("StartRedis: error in create redis container: %s", err)
		}
	} else {
		state, err := utils.StateContainer(defaults.DefaultRedisContainerName)
		if err != nil {
			return fmt.Errorf("StartRedis: error in get state of redis container: %s", err)
		}
		if state == "" {
			if err := utils.CreateAndRunContainer(defaults.DefaultRedisContainerName, defaults.DefaultRedisContainerRef+":"+cfg.Redis.Tag, portMap, volumeMap, redisServerCommand, nil); err != nil {
				return fmt.Errorf("StartRedis: error in create redis container: %s", err)
			}
		} else if !strings.Contains(state, "running") {
			if err := utils.StartContainer(defaults.DefaultRedisContainerName); err != nil {
				return fmt.Errorf("StartRedis: error in restart redis container: %s", err)
			}
		}
	}
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

func StartEden(cfg *EdenSetupArgs, vmName, zedControlURL, tapInterface string) error {

	// Note that custom installer only works with zedcloud controller.
	useZedcloud := cfg.Eve.CustomInstaller.Path != "" || zedControlURL != ""

	if !useZedcloud {
		if err := StartRedis(*cfg); err != nil {
			return fmt.Errorf("cannot start redis %w", err)
		}

		if err := StartAdamCmd(*cfg); err != nil {
			return fmt.Errorf("cannot start adam %w", err)
		}

		if err := StartRegistry(*cfg); err != nil {
			return fmt.Errorf("cannot start registry %w", err)
		}

		if err := StartEServer(*cfg); err != nil {
			return fmt.Errorf("cannot start adam %w", err)
		}
	}

	if cfg.Eve.Remote {
		return nil
	}

	if err := StartEve(vmName, tapInterface, cfg); err != nil {
		return fmt.Errorf("cannot start eve %w", err)
	}
	log.Infof("EVE is starting")
	return nil
}
