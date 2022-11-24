package openevec

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/controller/eapps"
	"github.com/lf-edge/eden/pkg/controller/eflowlog"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/controller/emetric"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eve"
	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eden/pkg/utils"
	edgeRegistry "github.com/lf-edge/edge-containers/pkg/registry"
	"github.com/lf-edge/edge-containers/pkg/resolver"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/api/go/metrics"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

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

func PodDeploy(appLink string, cfg *EdenSetupArgs) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %s", err)
	}
	var opts []expect.ExpectationOption
	opts = append(opts, expect.WithMetadata(cfg.Runtime.PodMetadata))
	opts = append(opts, expect.WithVnc(cfg.Runtime.VncDisplay))
	opts = append(opts, expect.WithVncPassword(cfg.Runtime.VncPassword))
	opts = append(opts, expect.WithAppAdapters(cfg.Runtime.AppAdapters))
	if len(cfg.Runtime.PodNetworks) > 0 {
		for i, el := range cfg.Runtime.PodNetworks {
			if i == 0 {
				//allocate ports on first network
				opts = append(opts, expect.AddNetInstanceNameAndPortPublish(el, cfg.Runtime.PortPublish))
			} else {
				opts = append(opts, expect.AddNetInstanceNameAndPortPublish(el, nil))
			}
		}
	} else {
		opts = append(opts, expect.WithPortsPublish(cfg.Runtime.PortPublish))
	}
	diskSizeParsed, err := humanize.ParseBytes(cfg.Runtime.DiskSize)
	if err != nil {
		return err
	}
	opts = append(opts, expect.WithDiskSize(int64(diskSizeParsed)))
	volumeSizeParsed, err := humanize.ParseBytes(cfg.Runtime.VolumeSize)
	if err != nil {
		return err
	}
	opts = append(opts, expect.WithVolumeSize(int64(volumeSizeParsed)))
	appMemoryParsed, err := humanize.ParseBytes(cfg.Runtime.AppMemory)
	if err != nil {
		return err
	}
	opts = append(opts, expect.WithVolumeType(expect.VolumeTypeByName(cfg.Runtime.VolumeType)))
	opts = append(opts, expect.WithResources(cfg.Runtime.AppCpus, uint32(appMemoryParsed/1000)))
	opts = append(opts, expect.WithImageFormat(cfg.Runtime.ImageFormat))
	if cfg.Runtime.AclOnlyHost {
		opts = append(opts, expect.WithACL(map[string][]expect.ACE{
			"": {{Endpoint: defaults.DefaultHostOnlyNotation}},
		}))
	} else {
		opts = append(opts, expect.WithACL(processAcls(cfg.Runtime.Acl)))
	}
	vlansParsed, err := processVLANs(cfg.Runtime.Vlans)
	if err != nil {
		return err
	}
	opts = append(opts, expect.WithVLANs(vlansParsed))
	opts = append(opts, expect.WithSFTPLoad(cfg.Runtime.SftpLoad))
	if !cfg.Runtime.SftpLoad {
		opts = append(opts, expect.WithHTTPDirectLoad(cfg.Runtime.DirectLoad))
	}
	opts = append(opts, expect.WithAdditionalDisks(append(cfg.Runtime.Disks, cfg.Runtime.Mount...)))
	registryToUse := cfg.Runtime.Registry
	switch cfg.Runtime.Registry {
	case "local":
		registryToUse = fmt.Sprintf("%s:%d", cfg.Registry.Ip, cfg.Registry.Port)
	case "remote":
		registryToUse = ""
	}
	opts = append(opts, expect.WithRegistry(registryToUse))
	if cfg.Runtime.NoHyper {
		opts = append(opts, expect.WithVirtualizationMode(config.VmMode_NOHYPER))
	}
	opts = append(opts, expect.WithOpenStackMetadata(cfg.Runtime.OpenStackMetadata))
	opts = append(opts, expect.WithProfiles(cfg.Runtime.Profiles))
	opts = append(opts, expect.WithDatastoreOverride(cfg.Runtime.DatastoreOverride))
	opts = append(opts, expect.WithStartDelay(cfg.Runtime.StartDelay))
	expectation := expect.AppExpectationFromURL(ctrl, dev, appLink, cfg.Runtime.PodName, opts...)
	appInstanceConfig := expectation.Application()
	dev.SetApplicationInstanceConfig(append(dev.GetApplicationInstances(), appInstanceConfig.Uuidandversion.Uuid))
	if err = changer.setControllerAndDev(ctrl, dev); err != nil {
		return fmt.Errorf("setControllerAndDev: %s", err)
	}
	log.Infof("deploy pod %s with %s request sent", appInstanceConfig.Displayname, appLink)
	return nil
}

func PodPs(cfg *EdenSetupArgs) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %s", err)
	}
	state := eve.Init(ctrl, dev)
	if err := ctrl.InfoLastCallback(dev.GetID(), nil, state.InfoCallback()); err != nil {
		return fmt.Errorf("fail in get InfoLastCallback: %s", err)
	}
	if err := ctrl.MetricLastCallback(dev.GetID(), nil, state.MetricCallback()); err != nil {
		return fmt.Errorf("fail in get MetricLastCallback: %s", err)
	}
	if err := state.PodsList(); err != nil {
		return err
	}
	return nil
}

func PodStop(appName string) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %s", err)
	}
	for _, el := range dev.GetApplicationInstances() {
		app, err := ctrl.GetApplicationInstanceConfig(el)
		if err != nil {
			return fmt.Errorf("no app in cloud %s: %s", el, err)
		}
		if app.Displayname == appName {
			app.Activate = false
			if err = changer.setControllerAndDev(ctrl, dev); err != nil {
				return fmt.Errorf("setControllerAndDev: %s", err)
			}
			log.Infof("app %s stop done", appName)
			return nil
		}
	}
	log.Infof("not found app with name %s", appName)
	return nil
}

func PodPurge(volumesToPurge []string, appName string, explicitVolumes bool) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %s", err)
	}
	for _, el := range dev.GetApplicationInstances() {
		app, err := ctrl.GetApplicationInstanceConfig(el)
		if err != nil {
			return fmt.Errorf("no app in cloud %s: %s", el, err)
		}
		if app.Displayname == appName {
			if app.Purge == nil {
				app.Purge = &config.InstanceOpsCmd{Counter: 0}
			}
			app.Purge.Counter++
			volumeConfigs := dev.GetVolumes()
			for i, oldUUID := range volumeConfigs {
				v, err := ctrl.GetVolume(oldUUID)
				if err != nil {
					return err
				}
				if explicitVolumes {
					skip := true
					for _, el := range volumesToPurge {
						if el == v.DisplayName {
							skip = false
							break
						}
					}
					if skip {
						continue
					}
				}
				newUUID, err := uuid.NewV4()
				if err != nil {
					return err
				}
				//update uuid to fire purge
				v.Uuid = newUUID.String()
				volumeConfigs[i] = newUUID.String()
				//fix volume ref to point onto new volume
				for _, el := range app.VolumeRefList {
					if el.Uuid == oldUUID {
						el.Uuid = newUUID.String()
					}
				}
			}
			dev.SetVolumeConfigs(volumeConfigs)
			if err = changer.setControllerAndDev(ctrl, dev); err != nil {
				return fmt.Errorf("setControllerAndDev: %s", err)
			}
			log.Infof("app %s purge done", appName)
			return nil
		}
	}
	log.Infof("not found app with name %s", appName)
	return nil
}

func PodRestart(appName string) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %s", err)
	}
	for _, el := range dev.GetApplicationInstances() {
		app, err := ctrl.GetApplicationInstanceConfig(el)
		if err != nil {
			return fmt.Errorf("no app in cloud %s: %s", el, err)
		}
		if app.Displayname == appName {
			if app.Restart == nil {
				app.Restart = &config.InstanceOpsCmd{Counter: 0}
			}
			app.Restart.Counter++
			if err = changer.setControllerAndDev(ctrl, dev); err != nil {
				return fmt.Errorf("setControllerAndDev: %s", err)
			}
			log.Infof("app %s restart done", appName)
			return nil
		}
	}
	log.Infof("not found app with name %s", appName)
	return nil
}

func PodStart(appName string) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %s", err)
	}
	for _, el := range dev.GetApplicationInstances() {
		app, err := ctrl.GetApplicationInstanceConfig(el)
		if err != nil {
			return fmt.Errorf("no app in cloud %s: %s", el, err)
		}
		if app.Displayname == appName {
			app.Activate = true
			if err = changer.setControllerAndDev(ctrl, dev); err != nil {
				return fmt.Errorf("setControllerAndDev: %s", err)
			}
			log.Infof("app %s start done", appName)
			return nil
		}
	}
	log.Infof("not found app with name %s", appName)
	return nil
}

func PodDelete(appName string, deleteVolumes bool) (bool, error) {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return false, fmt.Errorf("getControllerAndDev: %s", err)
	}
	for id, el := range dev.GetApplicationInstances() {
		app, err := ctrl.GetApplicationInstanceConfig(el)
		if err != nil {
			return false, fmt.Errorf("no app in cloud %s: %s", el, err)
		}
		if app.Displayname == appName {
			if deleteVolumes {
				volumeIDs := dev.GetVolumes()
				utils.DelEleInSliceByFunction(&volumeIDs, func(i interface{}) bool {
					vol, err := ctrl.GetVolume(i.(string))
					if err != nil {
						log.Errorf("no volume in cloud %s: %s", i.(string), err)
						return false
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
				return false, fmt.Errorf("setControllerAndDev: %s", err)
			}
			log.Infof("app %s delete done", appName)
			return false, nil
		}
	}
	log.Infof("not found app with name %s", appName)
	return false, nil
}

func PodLogs(appName string, outputTail uint, outputFields []string, outputFormat types.OutputFormat) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %s", err)
	}
	for _, el := range dev.GetApplicationInstances() {
		app, err := ctrl.GetApplicationInstanceConfig(el)
		if err != nil {
			return fmt.Errorf("no app in cloud %s: %s", el, err)
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
					if err = ctrl.LogChecker(dev.GetID(), logsQ, elog.HandleFactory(outputFormat, false), logType, 0); err != nil {
						return fmt.Errorf("LogChecker: %s", err)
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
					if err = ctrl.InfoChecker(dev.GetID(), infoQ, einfo.HandleFactory(outputFormat, false), infoType, 0); err != nil {
						return fmt.Errorf("InfoChecker: %s", err)
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
						return fmt.Errorf("MetricChecker: %s", err)
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
					if err = ctrl.FlowLogChecker(dev.GetID(), logsQ, eflowlog.HandleFactory(outputFormat, false), flowLogType, 0); err != nil {
						return fmt.Errorf("FlowLogChecker: %s", err)
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
						return err
					}
					if err = ctrl.LogAppsChecker(dev.GetID(), appID, nil, eapps.HandleFactory(outputFormat, false), appLogType, 0); err != nil {
						return fmt.Errorf("MetricChecker: %s", err)
					}
				}
			}
			return nil
		}
	}
	log.Infof("not found app with name %s", appName)
	return nil
}

func PodModify(appName string, cfg *EdenSetupArgs) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %s", err)
	}
	for _, appID := range dev.GetApplicationInstances() {
		app, err := ctrl.GetApplicationInstanceConfig(appID)
		if err != nil {
			return fmt.Errorf("no app in cloud %s: %s", appID, err)
		}
		if app.Displayname == appName {
			portPublishCombined := cfg.Runtime.PortPublish
			if cfg.Runtime.PortPublish == nil {
				portPublishCombined = []string{}
				for _, intf := range app.Interfaces {
					for _, acls := range intf.Acls {
						lport := ""
						var appPort uint32
						for _, match := range acls.Matches {
							if match.Type == "lport" {
								lport = match.Value
								break
							}
						}
						for _, action := range acls.Actions {
							if action.Portmap {
								appPort = action.AppPort
								break
							}
						}
						if lport != "" && appPort != 0 {
							portPublishCombined = append(portPublishCombined, fmt.Sprintf("%s:%d", lport, appPort))
						}
					}
				}
			}
			var opts []expect.ExpectationOption
			if len(cfg.Runtime.PodNetworks) > 0 {
				for i, el := range cfg.Runtime.PodNetworks {
					if i == 0 {
						//allocate ports on first network
						opts = append(opts, expect.AddNetInstanceNameAndPortPublish(el, portPublishCombined))
					} else {
						opts = append(opts, expect.AddNetInstanceNameAndPortPublish(el, nil))
					}
				}
			} else {
				opts = append(opts, expect.WithPortsPublish(portPublishCombined))
			}
			opts = append(opts, expect.WithACL(processAcls(cfg.Runtime.Acl)))
			vlansParsed, err := processVLANs(cfg.Runtime.Vlans)
			if err != nil {
				return err
			}
			opts = append(opts, expect.WithVLANs(vlansParsed))
			opts = append(opts, expect.WithOldApp(appName))
			opts = append(opts, expect.WithStartDelay(cfg.Runtime.StartDelay))
			expectation := expect.AppExpectationFromURL(ctrl, dev, defaults.DefaultDummyExpect, appName, opts...)
			appInstanceConfig := expectation.Application()
			needPurge := false
			if len(app.Interfaces) != len(appInstanceConfig.Interfaces) {
				needPurge = true
			} else {
				for ind, el := range app.Interfaces {
					equals, err := utils.CompareProtoMessages(el, appInstanceConfig.Interfaces[ind])
					if err != nil {
						return fmt.Errorf("CompareMessages: %v", err)
					}
					if !equals {
						needPurge = true
						break
					}
				}
			}
			if needPurge {
				if app.Purge == nil {
					app.Purge = &config.InstanceOpsCmd{Counter: 0}
				}
				app.Purge.Counter++
			}
			if cfg.Runtime.StartDelay != 0 {
				app.StartDelayInSeconds = appInstanceConfig.StartDelayInSeconds
			}
			//now we only change networks
			app.Interfaces = appInstanceConfig.Interfaces
			if err = changer.setControllerAndDev(ctrl, dev); err != nil {
				return fmt.Errorf("setControllerAndDev: %s", err)
			}
			if needPurge {
				processingFunction := func(im *info.ZInfoMsg) bool {
					if im.Ztype == info.ZInfoTypes_ZiApp {
						//waiting for purging state
						if im.GetAinfo().State == info.ZSwState_PURGING {
							return true
						}
					}
					return false
				}
				infoQ := make(map[string]string)
				infoQ["InfoContent.Ainfo.AppID"] = app.Uuidandversion.Uuid
				if err = ctrl.InfoChecker(dev.GetID(), infoQ, processingFunction, einfo.InfoNew, defaults.DefaultRepeatTimeout*defaults.DefaultRepeatCount); err != nil {
					return fmt.Errorf("InfoChecker: %s", err)
				}
			}
			log.Infof("app %s modify done", appName)
			return nil
		}
	}
	log.Infof("not found app with name %s", appName)
	return nil
}

// convert a "path:type" to a Disk struct
func diskToStruct(path string) (*edgeRegistry.Disk, error) {
	parts := strings.SplitN(path, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("expected structure <path>:<type>")
	}
	// get the disk type
	diskType, ok := edgeRegistry.NameToType[parts[1]]
	if !ok {
		return nil, fmt.Errorf("unknown disk type: %s", parts[1])
	}
	return &edgeRegistry.Disk{
		Source: &edgeRegistry.FileSource{Path: parts[0]},
		Type:   diskType,
	}, nil
}

func PodPublish(appName, kernelFile, initrdFile, rootFile, formatStr, arch string, local bool, disks []string, cfg *EdenSetupArgs) error {
	var (
		rootDisk     *edgeRegistry.Disk
		kernelSource *edgeRegistry.FileSource
		initrdSource *edgeRegistry.FileSource
		remoteTarget resolver.ResolverCloser
		err          error
	)
	ctx := context.TODO()
	if local {
		_, remoteTarget, err = utils.NewRegistryHTTP(ctx)
		if err != nil {
			return fmt.Errorf("unexpected error when created NewRegistry resolver: %v", err)
		}
		appName = fmt.Sprintf("%s:%d/%s", cfg.Registry.Ip, cfg.Registry.Port, appName)
	} else {
		_, remoteTarget, err = resolver.NewRegistry(ctx)
		if err != nil {
			return fmt.Errorf("unexpected error when created NewRegistry resolver: %v", err)
		}
	}
	if rootFile != "" {
		rootDisk, err = diskToStruct(rootFile)
		if err != nil {
			return fmt.Errorf("unable to read root disk %s: %v", rootFile, err)
		}
	}
	if kernelFile != "" {
		kernelSource = &edgeRegistry.FileSource{Path: kernelFile}
	}
	if initrdFile != "" {
		initrdSource = &edgeRegistry.FileSource{Path: initrdFile}
	}

	artifact := &edgeRegistry.Artifact{
		Kernel: kernelSource,
		Initrd: initrdSource,
		Root:   rootDisk,
	}
	for _, disk := range disks {
		additionalDisk, err := diskToStruct(disk)
		if err != nil {
			return fmt.Errorf("unable to read disk %s: %v", disk, err)
		}
		artifact.Disks = append(artifact.Disks, additionalDisk)
	}
	if kernelFile == "" {
		artifact.Kernel = nil
	}
	if initrdFile == "" {
		artifact.Initrd = nil
	}
	pusher := edgeRegistry.Pusher{
		Artifact: artifact,
		Image:    appName,
	}
	var format edgeRegistry.Format
	switch formatStr {
	case "artifacts":
		format = edgeRegistry.FormatArtifacts
	case "legacy":
		format = edgeRegistry.FormatLegacy
	default:
		return fmt.Errorf("unknown format: %v", formatStr)
	}
	hash, err := pusher.Push(format, true, os.Stdout, edgeRegistry.ConfigOpts{
		Author:       edgeRegistry.DefaultAuthor,
		OS:           edgeRegistry.DefaultOS,
		Architecture: arch,
	}, remoteTarget)
	if err != nil {
		return fmt.Errorf("error pushing to registry: %v", err)
	}
	fmt.Printf("Pushed image %s with digest %s\n", appName, hash)

	return nil
}
