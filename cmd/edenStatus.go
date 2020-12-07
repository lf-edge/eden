package cmd

import (
	"fmt"
	"github.com/lf-edge/adam/pkg/driver/common"
	"github.com/lf-edge/eden/pkg/eden"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/info"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	warnmark = "?" // because some OSes are missing the code for the warnmark ⚠
	okmark   = "✔"
	xmark    = "✘"
)

func eveLastRequests() (string, error) {
	log.Debugf("Will try to obtain info from ADAM")
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return "", err
	}
	var lastRequest *common.ApiRequest
	var handleRequest = func(request *common.ApiRequest) bool {
		if request.ClientIP != "" {
			lastRequest = request
		}
		return false
	}
	if err := ctrl.RequestLastCallback(dev.GetID(), map[string]string{"UUID": dev.GetID().String()}, handleRequest); err != nil {
		return "", err
	}
	if lastRequest == nil {
		return "", nil
	}
	return strings.Split(lastRequest.ClientIP, ":")[0], nil
}
func eveRequestsAdam() {
	if ip, err := eveLastRequests(); err != nil {
		fmt.Printf("%s EVE Request IP: error: %s\n", statusBad(), err)
	} else {
		if ip == "" {
			fmt.Printf("%s EVE Request IP: not found\n", statusWarn())
		}
		fmt.Printf("%s EVE Request IP: %s\n", statusOK(), ip)
	}
}

func eveStatusRemote() {
	log.Debugf("Will try to obtain info from ADAM")
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		log.Debugf("getControllerAndDev: %s", err)
		fmt.Printf("%s EVE status: undefined (no onboarded EVE)\n", statusWarn())
	} else {
		var lastDInfo *info.ZInfoMsg
		var lastTime time.Time
		var handleInfo = func(im *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface) bool {
			lastTime = time.Unix(im.AtTimeStamp.GetSeconds(), 0)
			if im.GetZtype() == info.ZInfoTypes_ZiDevice {
				lastDInfo = im
			}
			return false
		}
		if err = ctrl.InfoLastCallback(dev.GetID(), map[string]string{"devId": dev.GetID().String()}, handleInfo); err != nil {
			log.Fatalf("Fail in get InfoLastCallback: %s", err)
		}
		if lastDInfo != nil {
			var ips []string
			for _, nw := range lastDInfo.GetDinfo().Network {
				ips = append(ips, nw.IPAddrs...)
			}
			fmt.Printf("%s EVE REMOTE IPs: %s\n", statusOK(), strings.Join(ips, "; "))
			fmt.Printf("\tLast info received time: %s\n", lastTime)
		} else {
			fmt.Printf("%s EVE REMOTE IPs: %s\n", statusWarn(), "waiting for info...")
		}
	}
}

func eveStatusQEMU() {
	statusEVE, err := eden.StatusEVEQemu(evePidFile)
	if err != nil {
		log.Errorf("%s cannot obtain status of EVE Qemu process: %s", statusWarn(), err)
	} else {
		fmt.Printf("%s EVE on Qemu status: %s\n", representProcessStatus(statusEVE), statusEVE)
		fmt.Printf("\tLogs for local EVE at: %s\n", utils.ResolveAbsPath(configName+"-"+"eve.log"))
	}
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "status of harness",
	Long:  `Status of harness.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		viperLoaded, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		if viperLoaded {
			evePidFile = utils.ResolveAbsPath(viper.GetString("eve.pid"))
			eveRemote = viper.GetBool("eve.remote")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		eveUUID := viper.GetString("eve.uuid")
		edenDir, err := utils.DefaultEdenDir()
		if err != nil {
			log.Fatal(err)
		}
		fi, err := os.Stat(filepath.Join(edenDir, fmt.Sprintf("state-%s.yml", eveUUID)))
		if err != nil {
			fmt.Printf("EVE state: not onboarded\n")
		} else {
			size := fi.Size()
			if size > 0 {
				fmt.Printf("EVE state: registered\n")
			} else {
				fmt.Printf("EVE state: onboarding\n")
			}
		}
		fmt.Println()
		statusAdam, err := eden.StatusAdam()
		if err != nil {
			log.Errorf("%s cannot obtain status of adam: %s", statusWarn(), err)
		} else {
			fmt.Printf("%s Adam status: %s\n", representContainerStatus(lastWord(statusAdam)), statusAdam)
			fmt.Printf("\tAdam is expected at https://%s:%d\n", viper.GetString("adam.ip"), viper.GetInt("adam.port"))
			fmt.Printf("\tFor local Adam you can run 'docker logs %s' to see logs\n", defaults.DefaultAdamContainerName)
		}
		statusRegistry, err := eden.StatusRegistry()
		if err != nil {
			log.Errorf("%s cannot obtain status of registry: %s", statusWarn(), err)
		} else {
			fmt.Printf("%s Registry status: %s\n", representContainerStatus(lastWord(statusRegistry)), statusRegistry)
			fmt.Printf("\tRegistry is expected at https://%s:%d\n", viper.GetString("registry.ip"), viper.GetInt("registry.port"))
			fmt.Printf("\tFor local registry you can run 'docker logs %s' to see logs\n", defaults.DefaultRegistryContainerName)
		}
		statusRedis, err := eden.StatusRedis()
		if err != nil {
			log.Errorf("%s cannot obtain status of redis: %s", statusWarn(), err)
		} else {
			fmt.Printf("%s Redis status: %s\n", representContainerStatus(lastWord(statusRedis)), statusRedis)
			fmt.Printf("\tRedis is expected at %s\n", viper.GetString("adam.redis.eden"))
			fmt.Printf("\tFor local Redis you can run 'docker logs %s' to see logs\n", defaults.DefaultRedisContainerName)
		}
		statusEServer, err := eden.StatusEServer()
		if err != nil {
			log.Errorf("%s cannot obtain status of EServer process: %s", statusWarn(), err)
		} else {
			fmt.Printf("%s EServer process status: %s\n", representContainerStatus(lastWord(statusEServer)), statusEServer)
			fmt.Printf("\tEServer is expected at http://%s:%d from EVE\n", viper.GetString("eden.eserver.ip"), viper.GetInt("eden.eserver.port"))
			fmt.Printf("\tFor local EServer you can run 'docker logs %s' to see logs\n", defaults.DefaultEServerContainerName)
		}
		if statusAdam != "container doesn't exist" {
			eveStatusRemote()
		}
		if !eveRemote {
			eveStatusQEMU()
		}
		if statusAdam != "container doesn't exist" {
			eveRequestsAdam()
		}
	},
}

func statusInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	statusCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file with EVE pid")
}

// lastWord get last work in string
func lastWord(in string) string {
	ss := strings.Fields(in)
	if len(ss) > 0 {
		return ss[len(ss)-1]
	}
	return ""
}

// representContainerStatus convert one of the known container states into a colorized character
func representContainerStatus(status string) string {
	switch status {
	case "created":
		return statusWarn()
	case "restarting":
		return statusWarn()
	case "running":
		return statusOK()
	case "paused":
		return statusWarn()
	case "exited":
		return statusBad()
	case "dead":
		return statusBad()
	default:
		return statusWarn()
	}
}

// representProcessStatus convert one of the response messages from utils.StatusCommandWithPid into a colorized character
func representProcessStatus(status string) string {
	if strings.HasPrefix(status, "running") {
		return statusOK()
	}
	return statusBad()
}

func statusWarn() string {
	return color.YellowString(warnmark)
}
func statusOK() string {
	return color.GreenString(okmark)
}
func statusBad() string {
	return color.RedString(xmark)
}
