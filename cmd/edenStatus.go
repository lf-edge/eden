package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/info"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func eveStatusRPI() {
	log.Debugf("Will try to obtain info from ADAM")
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		log.Debugf("getControllerAndDev: %s", err)
		fmt.Println("EVE status: undefined (no onboarded EVE)")
	} else {
		var lastDInfo *info.ZInfoMsg
		var lastTime time.Time
		var handleInfo = func(im *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface, infoType einfo.ZInfoType) bool {
			lastTime = time.Unix(im.AtTimeStamp.GetSeconds(), 0)
			if im.GetZtype() == info.ZInfoTypes_ZiDevice {
				lastDInfo = im
			}
			return false
		}
		if err = ctrl.InfoLastCallback(dev.GetID(), map[string]string{"devId": dev.GetID().String()}, einfo.ZAll, handleInfo); err != nil {
			log.Fatalf("Fail in get InfoLastCallback: %s", err)
		}
		if lastDInfo != nil {
			var ips []string
			for _, nw := range lastDInfo.GetDinfo().Network {
				for _, addr := range nw.IPAddrs {
					ip, _, err := net.ParseCIDR(addr)
					if err != nil {
						log.Fatal(err)
					}
					ipv4 := ip.To4()
					if ipv4 != nil {
						ips = append(ips, ipv4.String())
					}
				}
			}
			fmt.Printf("EVE REMOTE IPs: %s\n", strings.Join(ips, "; "))
			fmt.Printf("\tLast info received time: %s\n", lastTime)
		} else {
			fmt.Printf("EVE REMOTE IPs: %s\n", "waiting for info...")
		}
	}
}

func eveStatusQEMU() {
	statusEVE, err := utils.StatusEVEQemu(evePidFile)
	if err != nil {
		log.Errorf("cannot obtain status of EVE process: %s", err)
	} else {
		fmt.Printf("EVE process status: %s\n", statusEVE)
		fmt.Printf("\tLogs for local EVE at: %s\n", utils.ResolveAbsPath("eve.log"))
	}
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "status of harness",
	Long:  `Status of harness.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
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
		statusAdam, err := utils.StatusAdam()
		if err != nil {
			log.Errorf("cannot obtain status of adam: %s", err)
		} else {
			fmt.Printf("Adam status: %s\n", statusAdam)
			fmt.Printf("\tAdam is expected at https://%s:%d\n", viper.GetString("adam.ip"), viper.GetInt("adam.port"))
			fmt.Printf("\tFor local Adam you can run 'docker logs %s' to see logs\n", defaults.DefaultAdamContainerName)
		}
		statusRedis, err := utils.StatusRedis()
		if err != nil {
			log.Errorf("cannot obtain status of redis: %s", err)
		} else {
			fmt.Printf("Redis status: %s\n", statusRedis)
			fmt.Printf("\tRedis is expected at %s\n", viper.GetString("adam.redis.eden"))
			fmt.Printf("\tFor local Redis you can run 'docker logs %s' to see logs\n", defaults.DefaultRedisContainerName)
		}
		statusEServer, err := utils.StatusEServer()
		if err != nil {
			log.Errorf("cannot obtain status of EServer process: %s", err)
		} else {
			fmt.Printf("EServer process status: %s\n", statusEServer)
			fmt.Printf("\tEServer is expected at http://%s:%d from EVE\n", viper.GetString("eden.eserver.ip"), viper.GetInt("eden.eserver.port"))
			fmt.Printf("\tFor local EServer you can run 'docker logs %s' to see logs\n", defaults.DefaultEServerContainerName)
		}
		if eveRemote {
			eveStatusRPI()
		} else {
			eveStatusQEMU()
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
