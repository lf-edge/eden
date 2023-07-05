package openevec

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
)

func AdamStart(cfg *EdenSetupArgs) error {
	command, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot obtain executable path: %w", err)
	}
	log.Infof("Executable path: %s", command)
	if !cfg.Adam.Remote.Redis {
		cfg.Adam.Redis.RemoteURL = ""
	}
	if err := StartAdam(cfg.Eden.CertsDir, cfg.Adam); err != nil {
		log.Errorf("cannot start adam: %s", err.Error())
	} else {
		log.Infof("Adam is running and accessible on port %d", cfg.Adam.Port)
	}
	return nil
}

// StartAdam function run adam in docker with mounted adamPath/run:/adam/run
// if adamForce is set, it recreates container
func StartAdam(certsDir string, cfg AdamConfig, opts ...string) (err error) {
	globalCertsDir := filepath.Join(certsDir, defaults.DefaultCertsDist)
	serverCertPath := filepath.Join(globalCertsDir, "server.pem")
	serverKeyPath := filepath.Join(globalCertsDir, "server-key.pem")
	cert, err := ioutil.ReadFile(serverCertPath)
	if err != nil {
		return fmt.Errorf("StartAdam: cannot load %s: %s", serverCertPath, err)
	}
	key, err := ioutil.ReadFile(serverKeyPath)
	if err != nil {
		return fmt.Errorf("StartAdam: cannot load %s: %s", serverKeyPath, err)
	}
	envs := []string{
		fmt.Sprintf("SERVER_CERT=%s", cert),
		fmt.Sprintf("SERVER_KEY=%s", key),
	}
	if !cfg.APIv1 {
		signingCertPath := filepath.Join(globalCertsDir, "signing.pem")
		signingKeyPath := filepath.Join(globalCertsDir, "signing-key.pem")
		signingCert, err := ioutil.ReadFile(signingCertPath)
		if err != nil {
			return fmt.Errorf("StartAdam: cannot load %s: %s", signingCertPath, err)
		}
		signingKey, err := ioutil.ReadFile(signingKeyPath)
		if err != nil {
			return fmt.Errorf("StartAdam: cannot load %s: %s", signingKeyPath, err)
		}
		envs = append(envs, fmt.Sprintf("SIGNING_CERT=%s", signingCert))
		envs = append(envs, fmt.Sprintf("SIGNING_KEY=%s", signingKey))

		encryptCertPath := filepath.Join(globalCertsDir, "encrypt.pem")
		encryptKeyPath := filepath.Join(globalCertsDir, "encrypt-key.pem")
		encryptCert, err := ioutil.ReadFile(encryptCertPath)
		if err != nil {
			return fmt.Errorf("StartAdam: cannot load %s: %s", encryptCertPath, err)
		}
		encryptKey, err := ioutil.ReadFile(encryptKeyPath)
		if err != nil {
			return fmt.Errorf("StartAdam: cannot load %s: %s", encryptKeyPath, err)
		}
		envs = append(envs, fmt.Sprintf("ENCRYPT_CERT=%s", encryptCert))
		envs = append(envs, fmt.Sprintf("ENCRYPT_KEY=%s", encryptKey))
	}
	portMap := map[string]string{"8080": strconv.Itoa(cfg.Port)}
	volumeMap := map[string]string{"/adam/run": fmt.Sprintf("%s/run", cfg.Dist)}
	adamServerCommand := strings.Fields("server --conf-dir ./run/conf")
	if cfg.Dist == "" {
		volumeMap = map[string]string{"/adam/run": ""}
		adamServerCommand = strings.Fields("server")
	}
	if cfg.Redis.RemoteURL != "" {
		redisPasswordFile := filepath.Join(globalCertsDir, defaults.DefaultRedisPasswordFile)
		pwd, err := ioutil.ReadFile(redisPasswordFile)
		if err == nil {
			cfg.Redis.RemoteURL = fmt.Sprintf("redis://%s:%s@%s", string(pwd), string(pwd), cfg.Redis.RemoteURL)
		} else {
			log.Errorf("cannot read redis password: %v", err)
			cfg.Redis.RemoteURL = fmt.Sprintf("redis://%s", cfg.Redis.RemoteURL)
		}
		adamServerCommand = append(adamServerCommand, strings.Fields(fmt.Sprintf("--db-url %s", cfg.Redis.RemoteURL))...)
	}
	adamServerCommand = append(adamServerCommand, opts...)
	if cfg.Force {
		_ = utils.StopContainer(defaults.DefaultAdamContainerName, true)
		if err := utils.CreateAndRunContainer(defaults.DefaultAdamContainerName, defaults.DefaultAdamContainerRef+":"+cfg.Tag, portMap, volumeMap, adamServerCommand, envs); err != nil {
			return fmt.Errorf("StartAdam: error in create adam container: %s", err)
		}
	} else {
		state, err := utils.StateContainer(defaults.DefaultAdamContainerName)
		if err != nil {
			return fmt.Errorf("StartAdam: error in get state of adam container: %s", err)
		}
		if state == "" {
			if err := utils.CreateAndRunContainer(defaults.DefaultAdamContainerName, defaults.DefaultAdamContainerRef+":"+cfg.Tag, portMap, volumeMap, adamServerCommand, envs); err != nil {
				return fmt.Errorf("StartAdam: error in create adam container: %s", err)
			}
		} else if !strings.Contains(state, "running") {
			if err := utils.StartContainer(defaults.DefaultAdamContainerName); err != nil {
				return fmt.Errorf("StartAdam: error in restart adam container: %s", err)
			}
		}
	}
	return nil
}
