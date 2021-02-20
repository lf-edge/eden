package eden

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
)

//StartRedis function run redis in docker with mounted redisPath:/data
//if redisForce is set, it recreates container
func StartRedis(redisPort int, redisPath string, redisForce bool, redisTag string) (err error) {
	portMap := map[string]string{"6379": strconv.Itoa(redisPort)}
	volumeMap := map[string]string{"/data": redisPath}
	redisServerCommand := strings.Fields("redis-server --appendonly yes")
	if redisPath != "" {
		if err = os.MkdirAll(redisPath, 0755); err != nil {
			return fmt.Errorf("StartRedis: Cannot create directory for redis (%s): %s", redisPath, err)
		}
	}
	if redisForce {
		_ = utils.StopContainer(defaults.DefaultRedisContainerName, true)
		if err := utils.CreateAndRunContainer(defaults.DefaultRedisContainerName, defaults.DefaultRedisContainerRef+":"+redisTag, portMap, volumeMap, redisServerCommand, nil); err != nil {
			return fmt.Errorf("StartRedis: error in create redis container: %s", err)
		}
	} else {
		state, err := utils.StateContainer(defaults.DefaultRedisContainerName)
		if err != nil {
			return fmt.Errorf("StartRedis: error in get state of redis container: %s", err)
		}
		if state == "" {
			if err := utils.CreateAndRunContainer(defaults.DefaultRedisContainerName, defaults.DefaultRedisContainerRef+":"+redisTag, portMap, volumeMap, redisServerCommand, nil); err != nil {
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

//StopRedis function stop redis container
func StopRedis(redisRm bool) (err error) {
	state, err := utils.StateContainer(defaults.DefaultRedisContainerName)
	if err != nil {
		return fmt.Errorf("StopRedis: error in get state of redis container: %s", err)
	}
	if !strings.Contains(state, "running") {
		if redisRm {
			if err := utils.StopContainer(defaults.DefaultRedisContainerName, true); err != nil {
				return fmt.Errorf("StopRedis: error in rm redis container: %s", err)
			}
		}
	} else if state == "" {
		return nil
	} else {
		if redisRm {
			if err := utils.StopContainer(defaults.DefaultRedisContainerName, false); err != nil {
				return fmt.Errorf("StopRedis: error in rm redis container: %s", err)
			}
		} else {
			if err := utils.StopContainer(defaults.DefaultRedisContainerName, true); err != nil {
				return fmt.Errorf("StopRedis: error in rm redis container: %s", err)
			}
		}
	}
	return nil
}

//StatusRedis function return status of redis
func StatusRedis() (status string, err error) {
	state, err := utils.StateContainer(defaults.DefaultRedisContainerName)
	if err != nil {
		return "", fmt.Errorf("StatusRedis: error in get state of redis container: %s", err)
	}
	if state == "" {
		return "container doesn't exist", nil
	}
	return state, nil
}
