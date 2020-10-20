package cmd

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/controller/eapps"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/controller/emetric"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/api/go/metrics"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	podName     string
	podMetadata string
	portPublish []string
	podNetworks []string
	appAdapters []string
	aclOnlyHost bool
	noHyper     bool
	pv          bool
	qemuPorts   map[string]string
	vncDisplay  uint32
	vncPassword string
	appCpus     uint32
	appMemory   string
	diskSize    string
	imageFormat string
	volumeType  string

	outputTail   uint
	outputFields []string

	logAppsFormat eapps.LogFormat
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
		opts = append(opts, expect.WithAppAdapters(appAdapters))
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
		opts = append(opts, expect.WithVolumeType(expect.VolumeTypeByName(volumeType)))
		opts = append(opts, expect.WithResources(appCpus, uint32(appMemoryParsed/1000)))
		opts = append(opts, expect.WithImageFormat(imageFormat))
		opts = append(opts, expect.WithACL(aclOnlyHost))
		registryToUse := registry
		switch registry {
		case "local":
			registryToUse = fmt.Sprintf("%s:%d", viper.GetString("registry.ip"), viper.GetInt("registry.port"))
		case "remote":
			registryToUse = ""
		}
		opts = append(opts, expect.WithRegistry(registryToUse))
		if noHyper {
			opts = append(opts, expect.WithVirtualizationMode(config.VmMode_NOHYPER))
		}
		if pv {
			opts = append(opts, expect.WithVirtualizationMode(config.VmMode_PV))
		}
		expectation := expect.AppExpectationFromURL(ctrl, dev, appLink, podName, opts...)
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
	uuid      string
	image     string
	adamState string
	eveState  string
	intIP     []string
	macs      []string
	volumes   map[string]uint32
	extIP     string
	intPort   string
	extPort   string
	deleted   bool
}

func appStateHeader() string {
	return "NAME\tIMAGE\tUUID\tINTERNAL\tEXTERNAL\tSTATE(ADAM)\tLAST_STATE(EVE)"
}

func (appStateObj *appState) toString() string {
	internal := "-"
	if len(appStateObj.intIP) == 1 {
		if appStateObj.intPort == "" {
			internal = appStateObj.intIP[0] //if one internal IP and not forward, display it
		} else {
			internal = fmt.Sprintf("%s:%s", appStateObj.intIP[0], appStateObj.intPort)
		}
	} else if len(appStateObj.intIP) > 1 {
		var els []string
		for i, el := range appStateObj.intIP {
			if i == 0 { //forward only on first network
				if appStateObj.intPort == "" {
					els = append(els, fmt.Sprintf("%s", el)) //if multiple internal IPs and not forward, display them
				} else {
					els = append(els, fmt.Sprintf("%s:%s", el, appStateObj.intPort))
				}
			} else {
				els = append(els, el)
			}
		}
		internal = strings.Join(els, "; ")
	}
	external := fmt.Sprintf("%s:%s", appStateObj.extIP, appStateObj.extPort)
	if appStateObj.extPort == "" {
		external = "-"
	}
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s", appStateObj.name, appStateObj.image, appStateObj.uuid,
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
			volumes := make(map[string]uint32)
			for _, el := range app.GetVolumeRefList() {
				volumes[el.Uuid] = 0
			}
			appStateObj := &appState{
				name:      app.Displayname,
				image:     imageName,
				adamState: "IN_CONFIG",
				eveState:  "UNKNOWN",
				intIP:     []string{"-"},
				extIP:     "-",
				intPort:   intPort,
				extPort:   extPort,
				volumes:   volumes,
				uuid:      app.Uuidandversion.Uuid,
			}
			appStates[app.Uuidandversion.Uuid] = appStateObj
		}
		var handleInfo = func(im *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface) bool {
			switch im.GetZtype() {
			case info.ZInfoTypes_ZiVolume:
				for _, app := range appStates {
					if len(app.volumes) == 0 {
						continue
					}
					var percent uint32 = 0
					for vol := range app.volumes {
						percent += app.volumes[vol] //we sum all percents of all volumes and will divide them by count
						if im.GetVinfo().Uuid == vol {
							app.volumes[vol] = im.GetVinfo().ProgressPercentage
							break
						}
					}
					if strings.HasPrefix(app.eveState, info.ZSwState_DOWNLOAD_STARTED.String()) {
						app.eveState = fmt.Sprintf("%s (%d%%)", info.ZSwState_DOWNLOAD_STARTED.String(), int(percent)/len(app.volumes))
					}
				}
			case info.ZInfoTypes_ZiApp:
				appStateObj, ok := appStates[im.GetAinfo().AppID]
				if !ok {
					appStateObj = &appState{
						name:      im.GetAinfo().AppName,
						image:     "-",
						adamState: "NOT_IN_CONFIG",
						uuid:      im.GetAinfo().AppID,
					}
					appStates[im.GetAinfo().AppID] = appStateObj
				}
				appStateObj.eveState = im.GetAinfo().State.String()
				if len(im.GetAinfo().AppErr) > 0 {
					//if AppErr, show them
					appStateObj.eveState = fmt.Sprintf("%s: %s", im.GetAinfo().State.String(), im.GetAinfo().AppErr)
				}
				if len(im.GetAinfo().Network) != 0 && len(im.GetAinfo().Network[0].IPAddrs) != 0 {
					if len(im.GetAinfo().Network) > 1 {
						appStateObj.intIP = []string{}
						appStateObj.macs = []string{}
						for _, el := range im.GetAinfo().Network {
							if len(im.GetAinfo().Network[0].IPAddrs) != 0 {
								appStateObj.intIP = append(appStateObj.intIP, el.IPAddrs[0])
								appStateObj.macs = append(appStateObj.macs, el.MacAddr)
							}
						}
					} else {
						if len(im.GetAinfo().Network[0].IPAddrs) != 0 {
							appStateObj.intIP = []string{im.GetAinfo().Network[0].IPAddrs[0]}
							appStateObj.macs = []string{im.GetAinfo().Network[0].MacAddr}
						}
					}
				} else {
					appStateObj.intIP = []string{"-"}
					appStateObj.macs = []string{}
				}
			case info.ZInfoTypes_ZiNetworkInstance: //try to find ips from NetworkInstances
				for _, el := range im.GetNiinfo().IpAssignments {
					for _, appStateObj := range appStates {
						for ind, mac := range appStateObj.macs {
							if mac == el.MacAddress {
								appStateObj.intIP[ind] = el.IpAddress[0]
							}
						}
					}
				}
			case info.ZInfoTypes_ZiDevice:
				for _, appStateObj := range appStates {
					seen := false
					for _, el := range im.GetDinfo().AppInstances {
						if appStateObj.uuid == el.Uuid {
							seen = true
							break
						}
					}
					if !seen {
						appStateObj.eveState = "UNKNOWN" //UNKNOWN if not found in recent AppInstances
					}
					if devModel == defaults.DefaultRPIModel || devModel == defaults.DefaultGCPModel {
						for _, nw := range im.GetDinfo().Network {
							for _, addr := range nw.IPAddrs {
								if addr != "" {
									s := strings.Split(addr, ";")
									for _, oneip := range s {
										if strings.Contains(oneip, ".") {
											appStateObj.extIP = oneip
										}
									}
								}
							}
						}
					} else {
						appStateObj.extIP = "127.0.0.1"
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
				//check appStateObj not defined in adam
				if appStateObj.adamState == "NOT_IN_CONFIG" {
					switch im.GetZtype() {
					case info.ZInfoTypes_ZiApp:
						if im.GetAinfo().AppID == appStateObj.uuid {
							appStateObj.deleted = false //if in recent ZInfoTypes_ZiApp, than not deleted
						}
					case info.ZInfoTypes_ZiDevice:
						appStateObj.deleted = true
						for _, el := range im.GetDinfo().AppInstances {
							if el.Uuid == appStateObj.uuid {
								appStateObj.deleted = false //if in recent ZInfoTypes_ZiDevice, than not deleted
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
		switch logFormatName {
		case "json":
			logFormat = elog.LogJSON
			logAppsFormat = eapps.LogJSON
		case "lines":
			logFormat = elog.LogLines
			logAppsFormat = eapps.LogLines
		default:
			return fmt.Errorf("unknown log format: %s", logFormatName)
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
						if err = ctrl.LogChecker(dev.GetID(), logsQ, elog.HandleFactory(logFormat, false), logType, 0); err != nil {
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
									//we print only AppMetrics from obtained metric
									emetric.MetricItemPrint(le, []string{fmt.Sprintf("AppMetrics[%d]", i)}).Print()
								}
							}
							return false
						}

						//metricsQ for filtering metrics by app
						metricsQ := make(map[string]string)
						metricsQ["AppMetrics[].AppID"] = app.Uuidandversion.Uuid
						if err = ctrl.MetricChecker(dev.GetID(), metricsQ, handleMetric, metricType, 0); err != nil {
							log.Fatalf("MetricChecker: %s", err)
						}
					case "app":
						//block for process app logs
						fmt.Printf("App logs list for app %s:\n", app.Uuidandversion.Uuid)

						//process only existing elements
						appLogType := eapps.LogExist

						if outputTail > 0 {
							//process only outputTail elements from end
							appLogType = eapps.LogTail(outputTail)
						}

						appID, err := uuid.FromString(app.Uuidandversion.Uuid)
						if err != nil {
							log.Fatal(err)
						}
						if err = ctrl.LogAppsChecker(dev.GetID(), appID, nil, eapps.HandleFactory(logAppsFormat, false), appLogType, 0); err != nil {
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
	podDeployCmd.Flags().Uint32Var(&appCpus, "cpus", defaults.DefaultAppCPU, "cpu number for app")
	podDeployCmd.Flags().StringVar(&appMemory, "memory", humanize.Bytes(defaults.DefaultAppMem*1024), "memory for app")
	podDeployCmd.Flags().StringVar(&diskSize, "disk-size", humanize.Bytes(0), "disk size (empty or 0 - same as in image)")
	podDeployCmd.Flags().StringVar(&volumeType, "volume-type", "qcow2", "volume type for empty volumes (qcow2 or oci)")
	podDeployCmd.Flags().StringSliceVar(&appAdapters, "adapters", nil, "adapters to assign to the application instance")
	podDeployCmd.Flags().StringSliceVar(&podNetworks, "networks", nil, "Networks to connect to app (ports will be mapped to first network)")
	podDeployCmd.Flags().StringVar(&imageFormat, "format", "", "format for image, one of 'container','qcow2'; if not provided, defaults to container image for docker and oci transports, qcow2 for file and http/s transports")
	podDeployCmd.Flags().BoolVar(&aclOnlyHost, "only-host", false, "Allow access only to host and external networks")
	podDeployCmd.Flags().BoolVar(&noHyper, "no-hyper", false, "Run pod without hypervisor")
	podDeployCmd.Flags().BoolVar(&pv, "pv", false, "Run pod in PV mode")
	podDeployCmd.Flags().StringVar(&registry, "registry", "remote", "Select registry to use for containers (remote/local)")
	podCmd.AddCommand(podPsCmd)
	podCmd.AddCommand(podStopCmd)
	podCmd.AddCommand(podStartCmd)
	podCmd.AddCommand(podDeleteCmd)
	podCmd.AddCommand(podLogsCmd)
	podLogsCmd.Flags().UintVar(&outputTail, "tail", 0, "Show only last N lines")
	podLogsCmd.Flags().StringSliceVar(&outputFields, "fields", []string{"log", "info", "metric", "app"}, "Show defined elements")
	podLogsCmd.Flags().StringVarP(&logFormatName, "format", "", "lines", "Format to print logs, supports: lines, json")
}
