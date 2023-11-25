package openevec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eden"
	"github.com/lf-edge/eden/pkg/eve"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
)

const (
	warnmark = "?" // because some OSes are missing the code for the warnmark ⚠
	okmark   = "✔"
	xmark    = "✘"
)

func (openEVEC *OpenEVEC) Status(vmName string, allConfigs bool) error {
	cfg := openEVEC.cfg
	statusAdam, err := eden.StatusAdam()
	if err != nil {
		return fmt.Errorf("%s cannot obtain status of adam: %w", statusWarn(), err)
	} else {
		fmt.Printf("%s Adam status: %s\n", representContainerStatus(lastWord(statusAdam)), statusAdam)
		fmt.Printf("\tAdam is expected at https://%s:%d\n", cfg.Adam.CertsIP, cfg.Adam.Port)
		fmt.Printf("\tFor local Adam you can run 'docker logs %s' to see logs\n", defaults.DefaultAdamContainerName)
	}
	statusRegistry, err := eden.StatusRegistry()
	if err != nil {
		return fmt.Errorf("%s cannot obtain status of registry: %w", statusWarn(), err)
	} else {
		fmt.Printf("%s Registry status: %s\n", representContainerStatus(lastWord(statusRegistry)), statusRegistry)
		fmt.Printf("\tRegistry is expected at https://%s:%d\n", cfg.Registry.IP, cfg.Registry.Port)
		fmt.Printf("\tFor local registry you can run 'docker logs %s' to see logs\n", defaults.DefaultRegistryContainerName)
	}
	statusRedis, err := eden.StatusRedis()
	if err != nil {
		return fmt.Errorf("%s cannot obtain status of redis: %w", statusWarn(), err)
	} else {
		fmt.Printf("%s Redis status: %s\n", representContainerStatus(lastWord(statusRedis)), statusRedis)
		fmt.Printf("\tRedis is expected at %s\n", cfg.Adam.Redis.Eden)
		fmt.Printf("\tFor local Redis you can run 'docker logs %s' to see logs\n", defaults.DefaultRedisContainerName)
	}
	statusEServer, err := eden.StatusEServer()
	if err != nil {
		return fmt.Errorf("%s cannot obtain status of redis: %s", statusWarn(), err)
	} else {
		fmt.Printf("%s EServer process status: %s\n", representContainerStatus(lastWord(statusEServer)), statusEServer)
		fmt.Printf("\tEServer is expected at http://%s:%d from EVE\n", cfg.Eden.EServer.IP, cfg.Eden.EServer.Port)
		fmt.Printf("\tFor local EServer you can run 'docker logs %s' to see logs\n", defaults.DefaultEServerContainerName)
	}
	fmt.Println()
	context, err := utils.ContextLoad()
	if err != nil {
		return fmt.Errorf("load context error: %w", err)
	}
	currentContext := context.Current
	contexts := context.ListContexts()
	for _, el := range contexts {
		if el == currentContext || allConfigs {
			fmt.Printf("--- context: %s ---\n", el)
			context.SetContext(el)
			configName := el
			localCfg, err := LoadConfig(context.GetCurrentConfig())
			if err != nil {
				return err
			}
			localOpenEVEC := CreateOpenEVEC(localCfg)
			eveUUID := localCfg.Eve.CertsUUID
			edenDir, err := utils.DefaultEdenDir()
			if err != nil {
				return err
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
				if err := localOpenEVEC.eveStatusRemote(); err != nil {
					return err
				}
			}
			if !localCfg.Eve.Remote {
				switch {
				case localCfg.Eve.DevModel == defaults.DefaultVBoxModel:
					localOpenEVEC.eveStatusVBox(vmName)
				case localCfg.Eve.DevModel == defaults.DefaultParallelsModel:
					localOpenEVEC.eveStatusParallels(vmName)
				default:
					localOpenEVEC.eveStatusQEMU(configName, cfg.Eve.Pid)
				}
			}
			if statusAdam != "container doesn't exist" {
				localOpenEVEC.eveRequestsAdam()
			}
			fmt.Println("------")
		}
	}
	context.SetContext(currentContext)
	return nil
}

func (openEVEC *OpenEVEC) eveRequestsAdam() {
	if ip, err := openEVEC.eveLastRequests(); err != nil {
		fmt.Printf("%s EVE Request IP: error: %s\n", statusBad(), err)
	} else {
		if ip == "" {
			fmt.Printf("%s EVE Request IP: not found\n", statusWarn())
		}
		fmt.Printf("%s EVE Request IP: %s\n", statusOK(), ip)
	}
}

func (openEVEC *OpenEVEC) eveStatusRemote() error {
	log.Debugf("Will try to obtain info from ADAM")
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDevFromConfig(openEVEC.cfg)
	if err != nil {
		log.Debugf("getControllerAndDev: %s", err)
		fmt.Printf("%s EVE status: undefined (no onboarded EVE)\n", statusWarn())
		return nil
	}

	eveState := eve.Init(ctrl, dev)
	if err = ctrl.InfoLastCallback(dev.GetID(), nil, eveState.InfoCallback()); err != nil {
		return fmt.Errorf("fail in get InfoLastCallback: %w", err)
	}
	if err = ctrl.MetricLastCallback(dev.GetID(), nil, eveState.MetricCallback()); err != nil {
		return fmt.Errorf("fail in get InfoLastCallback: %w", err)
	}
	if eveState.NodeState().LastSeen.Unix() == 0 {
		fmt.Printf("%s EVE REMOTE IPs: %s\n", statusWarn(), "waiting for info...")
		fmt.Printf("%s EVE memory: %s\n", statusWarn(), "waiting for info...")
	} else {
		var ips []string
		for _, v := range eveState.NodeState().RemoteIPs {
			ips = append(ips, v...)
		}
		fmt.Printf("%s EVE REMOTE IPs: %s\n", statusOK(), strings.Join(ips, "; "))
		var lastseen = eveState.NodeState().LastSeen
		var timenow = time.Now().Unix()
		fmt.Printf("\tLast info received time: %s\n", lastseen)
		if (timenow - lastseen.Unix()) > 600 {
			fmt.Printf("\t EVE MIGHT BE DOWN OR CONNECTIVITY BETWEEN EVE AND ADAM WAS LOST\n")
		}
		status := statusOK()
		if eveState.NodeState().UsedPercentageMem >= 70 {
			status = statusWarn()
		}
		if eveState.NodeState().UsedPercentageMem >= 90 {
			status = statusBad()
		}
		fmt.Printf("%s EVE memory: %s/%s\n", status,
			humanize.Bytes((uint64)(eveState.NodeState().UsedMem*humanize.MByte)),
			humanize.Bytes((uint64)(eveState.NodeState().AvailMem*humanize.MByte)))
	}
	return nil
}

func (openEVEC *OpenEVEC) eveStatusQEMU(configName, evePidFile string) {
	statusEVE, err := eden.StatusEVEQemu(evePidFile)
	if err != nil {
		log.Errorf("%s cannot obtain status of EVE Qemu process: %s", statusWarn(), err)
		return
	}
	fmt.Printf("%s EVE on Qemu status: %s\n", representProcessStatus(statusEVE), statusEVE)
	fmt.Printf("\tLogs for local EVE at: %s\n", utils.ResolveAbsPath(configName+"-"+"eve.log"))
}

func (openEVEC *OpenEVEC) eveStatusVBox(vmName string) {
	statusEVE, err := eden.StatusEVEVBox(vmName)
	if err != nil {
		log.Errorf("%s cannot obtain status of EVE VBox process: %s", statusWarn(), err)
		return
	}
	fmt.Printf("%s EVE on VBox status: %s\n", representProcessStatus(statusEVE), statusEVE)
}

func (openEVEC *OpenEVEC) eveStatusParallels(vmName string) {
	statusEVE, err := eden.StatusEVEParallels(vmName)
	if err != nil {
		log.Errorf("%s cannot obtain status of EVE Parallels process: %s", statusWarn(), err)
		return
	}
	fmt.Printf("%s EVE on Parallels status: %s\n", representProcessStatus(statusEVE), statusEVE)
}

// lastWord get last work in string
func lastWord(in string) string {
	if ss := strings.Fields(in); len(ss) > 0 {
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
