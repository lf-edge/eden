package cmd

import (
	"fmt"
	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/controller/emetric"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/api/go/metrics"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
)

var (
	podName     string
	podMetadata string
	portPublish []string
	podNetworks []string
	qemuPorts   map[string]string
	vncDisplay  uint32
	vncPassword string
	appCpus     uint32
	appMemory   string
	diskSize    string

	outputTail   uint
	outputFields []string
)

var podCmd = &cobra.Command{
	Use: "pod",
}

//podDeployCmd is command for deploy application on EVE
var podDeployCmd = &cobra.Command{
	Use:   "deploy (docker|http(s)|file)://(<TAG>[:<VERSION>] | <URL for qcow2 image> | <path to qcow2 image>)",
	Short: "Deploy app in pod",
	Long:  `Deploy app in pod.`,
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		ssid = viper.GetString("eve.ssid")
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		appLink := args[0]
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev: %s", err)
		}
		var opts []expect.ExpectationOption
		opts = append(opts, expect.WithMetadata(podMetadata))
		opts = append(opts, expect.WithVnc(vncDisplay))
		opts = append(opts, expect.WithVncPassword(vncPassword))
		if len(podNetworks) > 0 {
			for i, el := range podNetworks {
				if i == 0 {
					//allocate ports on first network
					opts = append(opts, expect.AddNetInstanceNameAndPortPublish(el, portPublish))
				} else {
					opts = append(opts, expect.AddNetInstanceNameAndPortPublish(el, nil))
				}
			}
		} else {
			opts = append(opts, expect.WithPortsPublish(portPublish))
		}
		diskSizeParsed, err := humanize.ParseBytes(diskSize)
		if err != nil {
			log.Fatal(err)
		}
		opts = append(opts, expect.WithDiskSize(int64(diskSizeParsed)))
		appMemoryParsed, err := humanize.ParseBytes(appMemory)
		if err != nil {
			log.Fatal(err)
		}
		opts = append(opts, expect.WithResources(appCpus, uint32(appMemoryParsed/1000)))
		expectation := expect.AppExpectationFromUrl(ctrl, appLink, podName, opts...)
		appInstanceConfig := expectation.Application()
		dev.SetApplicationInstanceConfig(append(dev.GetApplicationInstances(), appInstanceConfig.Uuidandversion.Uuid))
		if err = changer.setControllerAndDev(ctrl, dev); err != nil {
			log.Fatalf("setControllerAndDev: %s", err)
		}
		log.Infof("deploy pod %s with %s request sent", appInstanceConfig.Displayname, appLink)
	},
}

type appState struct {
	name      string
	image     string
	adamState string
	eveState  string
	intIp     string
	extIp     string
	intPort   string
	extPort   string
	deleted   bool
}

func appStateHeader() string {
	return "NAME\tIMAGE\tINTERNAL\tEXTERNAL\tSTATE(ADAM)\tLAST_STATE(EVE)"
}

func (appStateObj *appState) toString() string {
	internal := fmt.Sprintf("%s:%s", appStateObj.intIp, appStateObj.intPort)
	if appStateObj.intPort == "" {
		internal = appStateObj.intIp
	}
	external := fmt.Sprintf("%s:%s", appStateObj.extIp, appStateObj.extPort)
	if appStateObj.extPort == "" {
		external = "-"
	}
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s", appStateObj.name, appStateObj.image,
		internal,
		external,
		appStateObj.adamState, appStateObj.eveState)
}

func getPortMapping(appConfig *config.AppInstanceConfig, qemuPorts map[string]string) (intports, extports string) {
	iports := []string{}
	eports := []string{}
	for _, intf := range appConfig.Interfaces {
		fromPort := ""
		toPort := ""
		for _, acl := range intf.Acls {
			for _, match := range acl.Matches {
				if match.Type == "lport" {
					fromPort = match.Value
				}
			}
			for _, action := range acl.Actions {
				if action.Portmap {
					toPort = strconv.Itoa(int(action.AppPort))
				}
			}
			if fromPort != "" && toPort != "" {
				if qemuPorts != nil && len(qemuPorts) > 0 {
					for p1, p2 := range qemuPorts {
						if p2 == fromPort {
							fromPort = p1
							break
						}
					}
				}
				iports = append(iports, toPort)
				eports = append(eports, fromPort)
			}
		}
	}
	return strings.Join(iports, ","), strings.Join(eports, ",")
}

//podPsCmd is a command to list deployed apps
var podPsCmd = &cobra.Command{
	Use:   "ps",
	Short: "List pods",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		devModel = viper.GetString("eve.devmodel")
		qemuPorts = viper.GetStringMapString("eve.hostfwd")
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev: %s", err)
		}
		appStates := make(map[string]*appState)
		for _, el := range dev.GetApplicationInstances() {
			app, err := ctrl.GetApplicationInstanceConfig(el)
			if err != nil {
				log.Fatalf("no app in cloud %s: %s", el, err)
			}
			imageName := ""
			if len(app.Drives) > 0 {
				imageName = app.Drives[0].Image.Name
			}
			intPort, extPort := getPortMapping(app, qemuPorts)
			appStateObj := &appState{name: app.Displayname, image: imageName, adamState: "IN_CONFIG",
				eveState: "UNKNOWN", intIp: "-", extIp: "-", intPort: intPort, extPort: extPort}
			appStates[app.Uuidandversion.Uuid] = appStateObj
		}
		var handleInfo = func(im *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface) bool {
			switch im.GetZtype() {
			case info.ZInfoTypes_ZiApp:
				appStateObj, ok := appStates[im.GetAinfo().AppID]
				if !ok {
					imageName := ""
					if len(im.GetAinfo().GetSoftwareList()) > 0 {
						imageName = im.GetAinfo().GetSoftwareList()[0].ImageName
					}
					appStateObj = &appState{name: im.GetAinfo().AppName, image: imageName, adamState: "NOT_IN_CONFIG"}
					appStates[im.GetAinfo().AppID] = appStateObj
				}
				appStateObj.eveState = im.GetAinfo().State.String()
				if len(im.GetAinfo().Network) > 0 && len(im.GetAinfo().Network[0].IPAddrs) > 0 {
					appStateObj.intIp = im.GetAinfo().Network[0].IPAddrs[0]
				} else {
					appStateObj.intIp = "-"
				}
			case info.ZInfoTypes_ZiDevice:
				for _, appStateObj := range appStates {
					if devModel == defaults.DefaultRPIModel {
						for _, nw := range im.GetDinfo().Network {
							for _, addr := range nw.IPAddrs {
								ip, _, err := net.ParseCIDR(addr)
								if err != nil {
									log.Fatal(err)
								}
								ipv4 := ip.To4()
								if ipv4 != nil {
									appStateObj.extIp = ipv4.String()
								}
							}
						}
					} else {
						appStateObj.extIp = "127.0.0.1"
					}
				}
			}
			return false
		}
		if err = ctrl.InfoLastCallback(dev.GetID(), map[string]string{"devId": dev.GetID().String()}, handleInfo); err != nil {
			log.Fatalf("Fail in get InfoLastCallback: %s", err)
		}
		var handleInfoDevice = func(im *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface) bool {
			for _, appStateObj := range appStates {
				if appStateObj.adamState == "NOT_IN_CONFIG" {
					appStateObj.deleted = true
					if im.GetZtype() == info.ZInfoTypes_ZiDevice {
						for _, el := range im.GetDinfo().AppInstances {
							if appStateObj.name == el.Name {
								appStateObj.deleted = false
							}
						}
					}
				}
			}
			return false
		}
		if err = ctrl.InfoLastCallback(dev.GetID(), map[string]string{"devId": dev.GetID().String()}, handleInfoDevice); err != nil {
			log.Fatalf("Fail in get InfoLastCallback: %s", err)
		}
		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 0, 8, 1, '\t', 0)
		if _, err = fmt.Fprintln(w, appStateHeader()); err != nil {
			log.Fatal(err)
		}
		appStatesSlice := make([]*appState, 0, len(appStates))
		for _, k := range appStates {
			appStatesSlice = append(appStatesSlice, k)
		}
		sort.SliceStable(appStatesSlice, func(i, j int) bool {
			return appStatesSlice[i].name < appStatesSlice[j].name
		})
		for _, el := range appStatesSlice {
			if el.deleted == false {
				if _, err = fmt.Fprintln(w, el.toString()); err != nil {
					log.Fatal(err)
				}
			}
		}
		if err = w.Flush(); err != nil {
			log.Fatal(err)
		}
	},
}

//podStopCmd is a command to stop app
var podStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop pod",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		appName := args[0]
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev: %s", err)
		}
		for _, el := range dev.GetApplicationInstances() {
			app, err := ctrl.GetApplicationInstanceConfig(el)
			if err != nil {
				log.Fatalf("no app in cloud %s: %s", el, err)
			}
			if app.Displayname == appName {
				app.Activate = false
				if err = changer.setControllerAndDev(ctrl, dev); err != nil {
					log.Fatalf("setControllerAndDev: %s", err)
				}
				log.Infof("app %s stop done", appName)
				return
			}
		}
		log.Infof("not found app with name %s", appName)
	},
}

//podStopCmd is a command to start app
var podStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start pod",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		appName := args[0]
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev: %s", err)
		}
		for _, el := range dev.GetApplicationInstances() {
			app, err := ctrl.GetApplicationInstanceConfig(el)
			if err != nil {
				log.Fatalf("no app in cloud %s: %s", el, err)
			}
			if app.Displayname == appName {
				app.Activate = true
				if err = changer.setControllerAndDev(ctrl, dev); err != nil {
					log.Fatalf("setControllerAndDev: %s", err)
				}
				log.Infof("app %s start done", appName)
				return
			}
		}
		log.Infof("not found app with name %s", appName)
	},
}

//podStopCmd is a command to delete app from EVE
var podDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete pod",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		appName := args[0]
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev: %s", err)
		}
		for id, el := range dev.GetApplicationInstances() {
			app, err := ctrl.GetApplicationInstanceConfig(el)
			if err != nil {
				log.Fatalf("no app in cloud %s: %s", el, err)
			}
			if app.Displayname == appName {
				configs := dev.GetApplicationInstances()
				utils.DelEleInSlice(&configs, id)
				dev.SetApplicationInstanceConfig(configs)
				if err = changer.setControllerAndDev(ctrl, dev); err != nil {
					log.Fatalf("setControllerAndDev: %s", err)
				}
				log.Infof("app %s delete done", appName)
				return
			}
		}
		log.Infof("not found app with name %s", appName)
	},
}

//podStopCmd is a command to delete app from EVE
var podLogsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "Logs of pod",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		appName := args[0]
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev: %s", err)
		}
		for _, el := range dev.GetApplicationInstances() {
			app, err := ctrl.GetApplicationInstanceConfig(el)
			if err != nil {
				log.Fatalf("no app in cloud %s: %s", el, err)
			}
			if app.Displayname == appName {
				for _, el := range outputFields {
					switch el {
					case "log":
						//block for process logs
						fmt.Printf("Log list for app %s:\n", app.Uuidandversion.Uuid)
						//process only existing elements
						logType := elog.LogExist

						if outputTail > 0 {
							//process only outputTail elements from end
							logType = elog.LogTail(outputTail)
						}

						//logsQ for filtering logs by app
						logsQ := make(map[string]string)
						logsQ["msg"] = app.Uuidandversion.Uuid
						if err = ctrl.LogChecker(dev.GetID(), logsQ, elog.HandleAll, logType, 0); err != nil {
							log.Fatalf("LogChecker: %s", err)
						}
					case "info":
						//block for process info
						fmt.Printf("Info list for app %s:\n", app.Uuidandversion.Uuid)
						//process only existing elements
						infoType := einfo.InfoExist

						if outputTail > 0 {
							//process only outputTail elements from end
							infoType = einfo.InfoTail(outputTail)
						}

						//infoQ for filtering infos by app
						infoQ := make(map[string]string)
						infoQ["InfoContent.Ainfo.AppID"] = app.Uuidandversion.Uuid
						if err = ctrl.InfoChecker(dev.GetID(), infoQ, einfo.HandleAll, infoType, 0); err != nil {
							log.Fatalf("InfoChecker: %s", err)
						}
					case "metric":
						//block for process metrics
						fmt.Printf("Metric list for app %s:\n", app.Uuidandversion.Uuid)

						//process only existing elements
						metricType := emetric.MetricExist

						if outputTail > 0 {
							//process only outputTail elements from end
							metricType = emetric.MetricTail(outputTail)
						}
						handleMetric := func(le *metrics.ZMetricMsg) bool {
							for i, el := range le.Am {
								//filter metrics by application
								if el.AppID == app.Uuidandversion.Uuid {
									//we print only Am from obtained metric
									emetric.MetricItemPrint(le, []string{fmt.Sprintf("Am[%d]", i)}).Print()
								}
							}
							return false
						}

						//metricsQ for filtering metrics by app
						metricsQ := make(map[string]string)
						metricsQ["Am[].AppID"] = app.Uuidandversion.Uuid
						if err = ctrl.MetricChecker(dev.GetID(), metricsQ, handleMetric, metricType, 0); err != nil {
							log.Fatalf("MetricChecker: %s", err)
						}

					}
				}
				return
			}
		}
		log.Infof("not found app with name %s", appName)
	},
}

func podInit() {
	podCmd.AddCommand(podDeployCmd)
	podDeployCmd.Flags().StringSliceVarP(&portPublish, "publish", "p", nil, "Ports to publish in format EXTERNAL_PORT:INTERNAL_PORT")
	podDeployCmd.Flags().StringVarP(&podMetadata, "metadata", "", "", "metadata for pod")
	podDeployCmd.Flags().StringVarP(&podName, "name", "n", "", "name for pod")
	podDeployCmd.Flags().Uint32Var(&vncDisplay, "vnc-display", 0, "display number for VNC pod (0 - no VNC)")
	podDeployCmd.Flags().StringVar(&vncPassword, "vnc-password", "", "VNC password (empty - no password)")
	podDeployCmd.Flags().Uint32Var(&appCpus, "cpus", defaults.DefaultAppCpu, "cpu number for app")
	podDeployCmd.Flags().StringVar(&appMemory, "memory", humanize.Bytes(defaults.DefaultAppMem*1024), "memory for app")
	podDeployCmd.Flags().StringVar(&diskSize, "disk-size", humanize.Bytes(0), "disk size (empty or 0 - same as in image)")
	podDeployCmd.Flags().StringSliceVar(&podNetworks, "networks", nil, "Networks to connect to app (ports will be mapped to first network)")
	podCmd.AddCommand(podPsCmd)
	podCmd.AddCommand(podStopCmd)
	podCmd.AddCommand(podStartCmd)
	podCmd.AddCommand(podDeleteCmd)
	podCmd.AddCommand(podLogsCmd)
	podLogsCmd.Flags().UintVar(&outputTail, "tail", 0, "Show only last N lines")
	podLogsCmd.Flags().StringSliceVar(&outputFields, "fields", []string{"log", "info", "metric"}, "Show defined elements")
}
