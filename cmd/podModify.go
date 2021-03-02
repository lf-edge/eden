package cmd

import (
	"fmt"
	"github.com/dustin/go-humanize"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var podLink string

//podModifyCmd is a command to modify app
var podModifyCmd = &cobra.Command{
	Use:   "modify <app>",
	Short: "Modify pod",
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
				var opts []expect.ExpectationOption
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
				opts = append(opts, expect.WithACL(acl))
				opts = append(opts, expect.WithOldApp(appName))
				opts = append(opts, expect.WithHTTPDirectLoad(directLoad))
				diskSizeParsed, err := humanize.ParseBytes(diskSize)
				if err != nil {
					log.Fatal(err)
				}
				opts = append(opts, expect.WithDiskSize(int64(diskSizeParsed)))
				link := defaults.DefaultDummyExpect
				newLink := false
				needPurge := false
				if podLink != "" {
					needPurge = true
					newLink = true
					link = podLink
				}
				volumes:=dev.GetVolumes()
				expectation := expect.AppExpectationFromURL(ctrl, dev, link, appName, opts...)
				appInstanceConfig := expectation.Application()
				if len(app.Interfaces) != len(appInstanceConfig.Interfaces) {
					needPurge = true
				} else {
					for ind, el := range app.Interfaces {
						if el.NetworkId != appInstanceConfig.Interfaces[ind].NetworkId {
							needPurge = true
							break
						}
					}
				}
				app.Interfaces = appInstanceConfig.Interfaces
				if newLink {
					diffInDrives := true
					if len(app.Drives) == len(appInstanceConfig.Drives) {
						diffInDrives = false
						for i, d := range app.Drives {
							// check if we have difference in volumes and its images
							if appInstanceConfig.Drives[i].Image.Name != d.Image.Name || appInstanceConfig.Drives[i].Image.Sha256 != d.Image.Sha256 {
								diffInDrives = true
							}
						}
					}
					if diffInDrives {
						// user provides different link, so we need to purge volumes
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
						for _, volRef := range appInstanceConfig.VolumeRefList {
							volumeIDs = append(volumeIDs, volRef.Uuid)
						}
						dev.SetVolumeConfigs(volumeIDs)
						app.VolumeRefList = appInstanceConfig.VolumeRefList
						app.Drives = appInstanceConfig.Drives
					} else {
						// we will increase generation number if user provide the same link
						// to run image update
						dev.SetVolumeConfigs(volumes)
						for _, ctID := range dev.GetContentTrees() {
							ct, err := ctrl.GetContentTree(ctID)
							if err != nil {
								log.Fatalf("no ContentTree in cloud %s: %s", ctID, err)
							}
							for _, v := range app.VolumeRefList {
								volumeConfig, err := ctrl.GetVolume(v.Uuid)
								if err != nil {
									log.Fatalf("no volume in cloud %s: %s", v.Uuid, err)
								}
								volumeConfig.GenerationCount++
								if ct.GetUuid() == volumeConfig.Origin.DownloadContentTreeID {
									ct.GenerationCount++
								}
							}
						}
					}
				}
				if needPurge {
					if app.Purge == nil {
						app.Purge = &config.InstanceOpsCmd{Counter: 0}
					}
					app.Purge.Counter++
				}
				if err = changer.setControllerAndDev(ctrl, dev); err != nil {
					log.Fatalf("setControllerAndDev: %s", err)
				}
				log.Infof("app %s modify done", appName)
				return
			}
		}
		log.Infof("not found app with name %s", appName)
	},
}

func podModifyInit() {
	podCmd.AddCommand(podModifyCmd)
	podModifyCmd.Flags().StringSliceVarP(&portPublish, "publish", "p", nil, "Ports to publish in format EXTERNAL_PORT:INTERNAL_PORT")
	podModifyCmd.Flags().BoolVar(&aclOnlyHost, "only-host", false, "Allow access only to host and external networks")
	podModifyCmd.Flags().StringSliceVar(&podNetworks, "networks", nil, "Networks to connect to app (ports will be mapped to first network)")
	podModifyCmd.Flags().StringVar(&podLink, "link", "", "Set new app link for pod")
	podModifyCmd.Flags().StringVar(&diskSize, "disk-size", humanize.Bytes(0), "disk size (empty or 0 - same as in image)")
	podModifyCmd.Flags().BoolVar(&directLoad, "direct", true, "Use direct download for image instead of eserver")
	podModifyCmd.Flags().StringSliceVar(&acl, "acl", nil, "Allow access only to defined hosts/ips/subnets")
}
