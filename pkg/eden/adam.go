package eden

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
)

//StartAdam function run adam in docker with mounted adamPath/run:/adam/run
//if adamForce is set, it recreates container
func StartAdam(adamPort int, adamPath string, adamForce bool, adamTag string, adamRemoteURL string, opts ...string) (err error) {
	edenHome, err := utils.DefaultEdenDir()
	if err != nil {
		return err
	}
	globalCertsDir := filepath.Join(edenHome, defaults.DefaultCertsDist)
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
	portMap := map[string]string{"8080": strconv.Itoa(adamPort)}
	volumeMap := map[string]string{"/adam/run": fmt.Sprintf("%s/run", adamPath)}
	adamServerCommand := strings.Fields("server --conf-dir ./run/conf")
	if adamPath == "" {
		volumeMap = map[string]string{"/adam/run": ""}
		adamServerCommand = strings.Fields("server")
	}
	if adamRemoteURL != "" {
		adamServerCommand = append(adamServerCommand, strings.Fields(fmt.Sprintf("--db-url %s", adamRemoteURL))...)
	}
	adamServerCommand = append(adamServerCommand, opts...)
	if adamForce {
		_ = utils.StopContainer(defaults.DefaultAdamContainerName, true)
		if err := utils.CreateAndRunContainer(defaults.DefaultAdamContainerName, defaults.DefaultAdamContainerRef+":"+adamTag, portMap, volumeMap, adamServerCommand, envs); err != nil {
			return fmt.Errorf("StartAdam: error in create adam container: %s", err)
		}
	} else {
		state, err := utils.StateContainer(defaults.DefaultAdamContainerName)
		if err != nil {
			return fmt.Errorf("StartAdam: error in get state of adam container: %s", err)
		}
		if state == "" {
			if err := utils.CreateAndRunContainer(defaults.DefaultAdamContainerName, defaults.DefaultAdamContainerRef+":"+adamTag, portMap, volumeMap, adamServerCommand, envs); err != nil {
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

//StopAdam function stop adam container
func StopAdam(adamRm bool) (err error) {
	state, err := utils.StateContainer(defaults.DefaultAdamContainerName)
	if err != nil {
		return fmt.Errorf("StopAdam: error in get state of adam container: %s", err)
	}
	if !strings.Contains(state, "running") {
		if adamRm {
			if err := utils.StopContainer(defaults.DefaultAdamContainerName, true); err != nil {
				return fmt.Errorf("StopAdam: error in rm adam container: %s", err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if adamRm {
			if err := utils.StopContainer(defaults.DefaultAdamContainerName, false); err != nil {
				return fmt.Errorf("StopAdam: error in rm adam container: %s", err)
			}
		} else {
			if err := utils.StopContainer(defaults.DefaultAdamContainerName, true); err != nil {
				return fmt.Errorf("StopAdam: error in rm adam container: %s", err)
			}
		}
	}
	return nil
}

//StatusAdam function return status of adam
func StatusAdam() (status string, err error) {
	state, err := utils.StateContainer(defaults.DefaultAdamContainerName)
	if err != nil {
		return "", fmt.Errorf("StatusAdam: error in get state of adam container: %s", err)
	}
	if state == "" {
		return "container doesn't exist", nil
	}
	return state, nil
}
