package eden

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
)

//StartPostgres function run postgres in docker with mounted postgresPath:/data
//if postgresForce is set, it recreates container
func StartPostgres(postgresPort int, postgresPath string, postgresForce bool, postgresTag string) (err error) {
	portMap := map[string]string{"5432": strconv.Itoa(postgresPort)}
	volumeMap := map[string]string{"/var/lib/postgresql/data": postgresPath}
	envMap := []string{"POSTGRES_PASSWORD=postgres"}
	var postgresServerCommand []string
	if postgresPath != "" {
		if err = os.MkdirAll(postgresPath, 0755); err != nil {
			return fmt.Errorf("StartPostgres: Cannot create directory for postgres (%s): %s", postgresPath, err)
		}
	}
	if postgresForce {
		_ = utils.StopContainer(defaults.DefaultPostgresContainerName, true)
		if err := utils.CreateAndRunContainer(defaults.DefaultPostgresContainerName, defaults.DefaultPostgresContainerRef+":"+postgresTag, portMap, volumeMap, postgresServerCommand, envMap); err != nil {
			return fmt.Errorf("StartPostgres: error in create postgres container: %s", err)
		}
	} else {
		state, err := utils.StateContainer(defaults.DefaultPostgresContainerName)
		if err != nil {
			return fmt.Errorf("StartPostgres: error in get state of postgres container: %s", err)
		}
		if state == "" {
			if err := utils.CreateAndRunContainer(defaults.DefaultPostgresContainerName, defaults.DefaultPostgresContainerRef+":"+postgresTag, portMap, volumeMap, postgresServerCommand, envMap); err != nil {
				return fmt.Errorf("StartPostgres: error in create postgres container: %s", err)
			}
		} else if !strings.Contains(state, "running") {
			if err := utils.StartContainer(defaults.DefaultPostgresContainerName); err != nil {
				return fmt.Errorf("StartPostgres: error in restart postgres container: %s", err)
			}
		}
	}
	return nil
}

//StopPostgres function stop postgres container
func StopPostgres(postgresRm bool) (err error) {
	state, err := utils.StateContainer(defaults.DefaultPostgresContainerName)
	if err != nil {
		return fmt.Errorf("StopPostgres: error in get state of postgres container: %s", err)
	}
	if !strings.Contains(state, "running") {
		if postgresRm {
			if err := utils.StopContainer(defaults.DefaultPostgresContainerName, true); err != nil {
				return fmt.Errorf("StopPostgres: error in rm postgres container: %s", err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if postgresRm {
			if err := utils.StopContainer(defaults.DefaultPostgresContainerName, false); err != nil {
				return fmt.Errorf("StopPostgres: error in rm postgres container: %s", err)
			}
		} else {
			if err := utils.StopContainer(defaults.DefaultPostgresContainerName, true); err != nil {
				return fmt.Errorf("StopPostgres: error in rm postgres container: %s", err)
			}
		}
	}
	return nil
}

//StatusPostgres function return status of postgres
func StatusPostgres() (status string, err error) {
	state, err := utils.StateContainer(defaults.DefaultPostgresContainerName)
	if err != nil {
		return "", fmt.Errorf("StatusPostgres: error in get state of postgres container: %s", err)
	}
	if state == "" {
		return "container doesn't exist", nil
	}
	return state, nil
}
