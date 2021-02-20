package eden

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
)

//StartRegistry function run registry in docker
func StartRegistry(port int, tag, registryPath string, opts ...string) (err error) {
	containerName := defaults.DefaultRegistryContainerName
	ref := defaults.DefaultRegistryContainerRef
	serviceName := "registry"
	portMap := map[string]string{"5000": strconv.Itoa(port)}
	cmd := []string{}
	cmd = append(cmd, opts...)
	volumeMap := map[string]string{"/var/lib/registry": registryPath}
	state, err := utils.StateContainer(containerName)
	if err != nil {
		return fmt.Errorf("StartRegistry: error in get state of %s container: %s", serviceName, err)
	}
	if state == "" {
		if err := utils.CreateAndRunContainer(containerName, ref+":"+tag, portMap, volumeMap, cmd, nil); err != nil {
			return fmt.Errorf("StartRegistry: error in create %s container: %s", serviceName, err)
		}
	} else if !strings.Contains(state, "running") {
		if err := utils.StartContainer(containerName); err != nil {
			return fmt.Errorf("StartRegistry: error in restart %s container: %s", serviceName, err)
		}
	}
	return nil
}

// StopRegistry function stop registry container
func StopRegistry(rm bool) (err error) {
	containerName := defaults.DefaultRegistryContainerName
	serviceName := "registry"
	state, err := utils.StateContainer(containerName)
	if err != nil {
		return fmt.Errorf("StopRegistry: error in get state of %s container: %s", serviceName, err)
	}
	if !strings.Contains(state, "running") {
		if rm {
			if err := utils.StopContainer(containerName, true); err != nil {
				return fmt.Errorf("StopRegistry: error in rm %s container: %s", serviceName, err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if rm {
			if err := utils.StopContainer(containerName, false); err != nil {
				return fmt.Errorf("StopRegistry: error in rm %s container: %s", serviceName, err)
			}
		} else {
			if err := utils.StopContainer(containerName, true); err != nil {
				return fmt.Errorf("StopRegistry: error in rm %s container: %s", serviceName, err)
			}
		}
	}
	return nil
}

// StatusRegistry function return status of registry
func StatusRegistry() (status string, err error) {
	containerName := defaults.DefaultRegistryContainerName
	serviceName := "registry"
	state, err := utils.StateContainer(containerName)
	if err != nil {
		return "", fmt.Errorf("StatusRegistry: error in get state of %s container: %s", serviceName, err)
	}
	if state == "" {
		return "container doesn't exist", nil
	}
	return state, nil
}
