package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/eve"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	allConfigs bool
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
	var lastRequest *types.APIRequest
	var handleRequest = func(request *types.APIRequest) bool {
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
		eveState := eve.Init(ctrl, dev)
		if err = ctrl.InfoLastCallback(dev.GetID(), nil, eveState.InfoCallback()); err != nil {
			log.Fatalf("Fail in get InfoLastCallback: %s", err)
		}
		if err = ctrl.MetricLastCallback(dev.GetID(), nil, eveState.MetricCallback()); err != nil {
			log.Fatalf("Fail in get InfoLastCallback: %s", err)
		}
		if lastDInfo := eveState.InfoAndMetrics().GetDinfo(); lastDInfo != nil {
			var ips []string
			for _, nw := range lastDInfo.Network {
				ips = append(ips, nw.IPAddrs...)
			}
			fmt.Printf("%s EVE REMOTE IPs: %s\n", statusOK(), strings.Join(ips, "; "))
			var lastseen = time.Unix(eveState.InfoAndMetrics().GetLastInfoTime().GetSeconds(), 0)
			var timenow = time.Now().Unix()
			fmt.Printf("\tLast info received time: %s\n", lastseen)
			if (timenow - lastseen.Unix()) > 600 {
				fmt.Printf("\t EVE MIGHT BE DOWN OR CONNECTIVITY BETWEEN EVE AND ADAM WAS LOST\n")
			}
		} else {
			fmt.Printf("%s EVE REMOTE IPs: %s\n", statusWarn(), "waiting for info...")
		}
		if lastDMetric := eveState.InfoAndMetrics().GetDeviceMetrics(); lastDMetric != nil {
			status := statusOK()
			if lastDMetric.Memory.GetUsedPercentage() >= 70 {
				status = statusWarn()
			}
			if lastDMetric.Memory.GetUsedPercentage() >= 90 {
				status = statusBad()
			}
			fmt.Printf("%s EVE memory: %s/%s\n", status,
				humanize.Bytes((uint64)(lastDMetric.Memory.GetUsedMem()*humanize.MByte)),
				humanize.Bytes((uint64)(lastDMetric.Memory.GetAvailMem()*humanize.MByte)))
		} else {
			fmt.Printf("%s EVE memory: %s\n", statusWarn(), "waiting for info...")
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

func eveStatusVBox() {
	statusEVE, err := eden.StatusEVEVBox(vmName)
	if err != nil {
		log.Errorf("%s cannot obtain status of EVE VBox process: %s", statusWarn(), err)
	} else {
		fmt.Printf("%s EVE on VBox status: %s\n", representProcessStatus(statusEVE), statusEVE)
	}
}

func eveStatusParallels() {
	statusEVE, err := eden.StatusEVEParallels(vmName)
	if err != nil {
		log.Errorf("%s cannot obtain status of EVE Parallels process: %s", statusWarn(), err)
	} else {
		fmt.Printf("%s EVE on Parallels status: %s\n", representProcessStatus(statusEVE), statusEVE)
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
			devModel = viper.GetString("eve.devmodel")
			loadSdnOptsFromViper()
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
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
		fmt.Println()
		context, err := utils.ContextLoad()
		if err != nil {
			log.Fatalf("Load context error: %s", err)
		}
		currentContext := context.Current
		contexts := context.ListContexts()
		for _, el := range contexts {
			if el == currentContext || allConfigs {
				fmt.Printf("--- context: %s ---\n", el)
				context.SetContext(el)
				configName = el
				evePidFile = utils.ResolveAbsPath(fmt.Sprintf("%s-eve.pid", el))
				_, err := utils.LoadConfigFileContext(context.GetCurrentConfig())
				if err != nil {
					log.Fatalf("error reading config: %s", err.Error())
				}
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
				if statusAdam != "container doesn't exist" {
					eveStatusRemote()
				}
				if !eveRemote {
					if devModel == defaults.DefaultVBoxModel {
						eveStatusVBox()
					} else if devModel == defaults.DefaultParallelsModel {
						eveStatusParallels()
					} else {
						eveStatusQEMU()
						sdnStatus()
					}
				}
				if statusAdam != "container doesn't exist" {
					eveRequestsAdam()
				}
				fmt.Println("------")
			}
		}
		context.SetContext(currentContext)
	},
}

func statusInit() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	statusCmd.Flags().StringVarP(&evePidFile, "eve-pid", "", filepath.Join(currentPath, defaults.DefaultDist, "eve.pid"), "file with EVE pid")
	statusCmd.Flags().BoolVar(&allConfigs, "all", true, "show status for all configs")
	statusCmd.Flags().StringVarP(&vmName, "vmname", "", defaults.DefaultVBoxVMName, "vbox vmname required to create vm")
	addSdnPidOpt(statusCmd)
	addSdnPortOpts(statusCmd)
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
