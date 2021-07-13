package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/controller/eapps"
	"github.com/lf-edge/eden/pkg/controller/eflowlog"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/controller/emetric"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eve"
	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
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
	qemuPorts   map[string]string
	vncDisplay  uint32
	vncPassword string
	appCpus     uint32
	appMemory   string
	diskSize    string
	mount       []string
	volumeSize  string
	imageFormat string
	volumeType  string
	acl         []string
	profiles    []string
	vlans       []string

	deleteVolumes bool

	outputTail   uint
	outputFields []string

	logAppsFormat eapps.LogFormat

	directLoad bool
	sftpLoad   bool

	openStackMetadata bool
)

var podCmd = &cobra.Command{
	Use: "pod",
}

func processAcls(acls []string) expect.ACLs {
	m := expect.ACLs{}
	for _, el := range acls {
		parsed := strings.SplitN(el, ":", 3)
		ni := parsed[0]
		var ep string
		if len(parsed) > 1 {
			ep = strings.TrimSpace(parsed[1])
		}
		if ep == "" {
			m[ni] = []expect.ACE{}
		} else {
			drop := false
			if len(parsed) == 3 {
				drop = parsed[2] == "drop"
			}
			m[ni] = append(m[ni], expect.ACE{Endpoint: ep, Drop: drop})
		}
	}
	return m
}

func processVLANs(vlans []string) (map[string]int, error) {
	m := map[string]int{}
	for _, el := range vlans {
		parsed := strings.SplitN(el, ":", 2)
		if len(parsed) < 2 {
			return nil, errors.New("missing VLAN ID")
		}
		vid, err := strconv.Atoi(parsed[1])
		if err != nil {
			return nil, fmt.Errorf("invalid VLAN ID: %w", err)
		}
		m[parsed[0]] = vid
	}
	return m, nil
}

//podDeployCmd is command for deploy application on EVE
var podDeployCmd = &cobra.Command{
	Use:   "deploy (docker|http(s)|file|directory)://(<TAG|PATH>[:<VERSION>] | <URL for qcow2 image> | <path to qcow2 image>)",
	Short: "Deploy app in pod",
	Long:  `Deploy app in pod.`,
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assignCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		certsDir = utils.ResolveAbsPath(viper.GetString("eden.certs-dist"))
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
		volumeSizeParsed, err := humanize.ParseBytes(volumeSize)
		if err != nil {
			log.Fatal(err)
		}
		opts = append(opts, expect.WithVolumeSize(int64(volumeSizeParsed)))
		appMemoryParsed, err := humanize.ParseBytes(appMemory)
		if err != nil {
			log.Fatal(err)
		}
		opts = append(opts, expect.WithVolumeType(expect.VolumeTypeByName(volumeType)))
		opts = append(opts, expect.WithResources(appCpus, uint32(appMemoryParsed/1000)))
		opts = append(opts, expect.WithImageFormat(imageFormat))
		if aclOnlyHost {
			opts = append(opts, expect.WithACL(map[string][]expect.ACE{
				"": {{Endpoint: defaults.DefaultHostOnlyNotation}},
			}))
		} else {
			opts = append(opts, expect.WithACL(processAcls(acl)))
		}
		vlansParsed, err := processVLANs(vlans)
		if err != nil {
			log.Fatal(err)
		}
		opts = append(opts, expect.WithVLANs(vlansParsed))
		opts = append(opts, expect.WithSFTPLoad(sftpLoad))
		if !sftpLoad {
			opts = append(opts, expect.WithHTTPDirectLoad(directLoad))
		}
		opts = append(opts, expect.WithAdditionalDisks(append(disks, mount...)))
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
		opts = append(opts, expect.WithOpenStackMetadata(openStackMetadata))
		opts = append(opts, expect.WithProfiles(profiles))
		expectation := expect.AppExpectationFromURL(ctrl, dev, appLink, podName, opts...)
		appInstanceConfig := expectation.Application()
		dev.SetApplicationInstanceConfig(append(dev.GetApplicationInstances(), appInstanceConfig.Uuidandversion.Uuid))
		if err = changer.setControllerAndDev(ctrl, dev); err != nil {
			log.Fatalf("setControllerAndDev: %s", err)
		}
		log.Infof("deploy pod %s with %s request sent", appInstanceConfig.Displayname, appLink)
	},
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
		eveRemote = viper.GetBool("eve.remote")
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev: %s", err)
		}
		state := eve.Init(ctrl, dev)
		if err := ctrl.InfoLastCallback(dev.GetID(), nil, state.InfoCallback()); err != nil {
			log.Fatalf("fail in get InfoLastCallback: %s", err)
		}
		if err := ctrl.MetricLastCallback(dev.GetID(), nil, state.MetricCallback()); err != nil {
			log.Fatalf("fail in get MetricLastCallback: %s", err)
		}
		if err := state.PodsList(); err != nil {
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
				if deleteVolumes {
					volumeIDs := dev.GetVolumes()
					utils.DelEleInSliceByFunction(&volumeIDs, func(i interface{}) bool {
						vol, err := ctrl.GetVolume(i.(string))
						if err != nil {
							log.Fatalf("no volume in cloud %s: %s", i.(string), err)
						}
						for _, volRef := range app.VolumeRefList {
							if vol.Uuid == volRef.Uuid {
								return true
							}
						}
						return false
					})
					dev.SetVolumeConfigs(volumeIDs)
				}
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
									emetric.MetricItemPrint(le, []string{fmt.Sprintf("am[%d]", i)}).Print()
								}
							}
							return false
						}

						//metricsQ for filtering metrics by app
						metricsQ := make(map[string]string)
						metricsQ["am[].AppID"] = app.Uuidandversion.Uuid
						if err = ctrl.MetricChecker(dev.GetID(), metricsQ, handleMetric, metricType, 0); err != nil {
							log.Fatalf("MetricChecker: %s", err)
						}
					case "netstat":
						//block for process FlowLog
						fmt.Printf("netstat list for app %s:\n", app.Uuidandversion.Uuid)
						//process only existing elements
						flowLogType := eflowlog.FlowLogExist

						if outputTail > 0 {
							//process only outputTail elements from end
							flowLogType = eflowlog.FlowLogTail(outputTail)
						}

						//logsQ for filtering logs by app
						logsQ := make(map[string]string)
						logsQ["scope.uuid"] = app.Uuidandversion.Uuid
						if err = ctrl.FlowLogChecker(dev.GetID(), logsQ, eflowlog.HandleFactory(false), flowLogType, 0); err != nil {
							log.Fatalf("FlowLogChecker: %s", err)
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
	podDeployCmd.Flags().StringVar(&volumeType, "volume-type", "qcow2", "volume type for empty volumes (qcow2, raw, qcow, vmdk, vhdx or oci); set it to none to not use volumes")
	podDeployCmd.Flags().StringSliceVar(&appAdapters, "adapters", nil, "adapters to assign to the application instance")
	podDeployCmd.Flags().StringSliceVar(&podNetworks, "networks", nil, "Networks to connect to app (ports will be mapped to first network). May have <name:[MAC address]> notation.")
	podDeployCmd.Flags().StringVar(&imageFormat, "format", "", "format for image, one of 'container','qcow2','raw','qcow','vmdk','vhdx'; if not provided, defaults to container image for docker and oci transports, qcow2 for file and http/s transports")
	podDeployCmd.Flags().BoolVar(&aclOnlyHost, "only-host", false, "Allow access only to host and external networks")
	podDeployCmd.Flags().BoolVar(&noHyper, "no-hyper", false, "Run pod without hypervisor")
	podDeployCmd.Flags().StringVar(&registry, "registry", "remote", "Select registry to use for containers (remote/local)")
	podDeployCmd.Flags().BoolVar(&directLoad, "direct", true, "Use direct download for image instead of eserver")
	podDeployCmd.Flags().BoolVar(&sftpLoad, "sftp", false, "Force use of sftp to load http/file image from eserver")
	podDeployCmd.Flags().StringSliceVar(&disks, "disks", nil, `Additional disks to use. You can write it in notation <link> or <mount point>:<link>. Deprecated. Please use volumes instead.`)
	podDeployCmd.Flags().StringArrayVar(&mount, "mount", nil, `Additional volumes to use. You can write it in notation src=<link>,dst=<mount point>.`)
	podDeployCmd.Flags().StringVar(&volumeSize, "volume-size", humanize.IBytes(defaults.DefaultVolumeSize), "volume size")
	podDeployCmd.Flags().StringSliceVar(&profiles, "profile", nil, "profile to set for app")
	podDeployCmd.Flags().StringSliceVar(&acl, "acl", nil, `Allow access only to defined hosts/ips/subnets.
Without explicitly configured ACLs, all traffic is allowed.
You can set ACL for a particular network in format '<network_name[:endpoint[:action]]>', where 'action' is either 'allow' (default) or 'drop'.
With ACLs configured, endpoints not matched by any rule are blocked.
To block all traffic define ACL with no endpoints: '<network_name>:'`)
	podDeployCmd.Flags().StringSliceVar(&vlans, "vlan", nil, `Connect application to the (switch) network over an access port assigned to the given VLAN.
You can set access VLAN ID (VID) for a particular network in the format '<network_name:VID>'`)
	podDeployCmd.Flags().BoolVar(&openStackMetadata, "openstack-metadata", false, "Use OpenStack metadata for VM")
	podCmd.AddCommand(podPsCmd)
	podCmd.AddCommand(podStopCmd)
	podCmd.AddCommand(podStartCmd)
	podCmd.AddCommand(podDeleteCmd)
	podDeleteCmd.Flags().BoolVar(&deleteVolumes, "with-volumes", true, "delete volumes of pod")
	podCmd.AddCommand(podLogsCmd)
	podLogsCmd.Flags().UintVar(&outputTail, "tail", 0, "Show only last N lines")
	podLogsCmd.Flags().StringSliceVar(&outputFields, "fields", []string{"log", "info", "metric", "netstat", "app"}, "Show defined elements")
	podLogsCmd.Flags().StringVarP(&logFormatName, "format", "", "lines", "Format to print logs, supports: lines, json")
	podModifyInit()
}
