package cmd

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/info"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

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
		for _, appID := range dev.GetApplicationInstances() {
			app, err := ctrl.GetApplicationInstanceConfig(appID)
			if err != nil {
				log.Fatalf("no app in cloud %s: %s", appID, err)
			}
			if app.Displayname == appName {
				portPublishCombined := portPublish
				if !cmd.Flags().Changed("publish") {
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
				if len(podNetworks) > 0 {
					for i, el := range podNetworks {
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
				opts = append(opts, expect.WithACL(processAcls(acl)))
				vlansParsed, err := processVLANs(vlans)
				if err != nil {
					log.Fatal(err)
				}
				opts = append(opts, expect.WithVLANs(vlansParsed))
				opts = append(opts, expect.WithOldApp(appName))
				expectation := expect.AppExpectationFromURL(ctrl, dev, defaults.DefaultDummyExpect, appName, opts...)
				appInstanceConfig := expectation.Application()
				needPurge := false
				if len(app.Interfaces) != len(appInstanceConfig.Interfaces) {
					needPurge = true
				} else {
					for ind, el := range app.Interfaces {
						equals, err := utils.CompareProtoMessages(el, appInstanceConfig.Interfaces[ind])
						if err != nil {
							log.Fatalf("CompareMessages: %v", err)
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
				//now we only change networks
				app.Interfaces = appInstanceConfig.Interfaces
				if err = changer.setControllerAndDev(ctrl, dev); err != nil {
					log.Fatalf("setControllerAndDev: %s", err)
				}
				if needPurge {
					processingFunction := func(im *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface) bool {
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
						log.Fatalf("InfoChecker: %s", err)
					}
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
	podModifyCmd.Flags().StringSliceVar(&podNetworks, "networks", nil, "Networks to connect to app (ports will be mapped to first network). May have <name:[MAC address]> notation.")
	podModifyCmd.Flags().StringSliceVar(&acl, "acl", nil, `Allow access only to defined hosts/ips/subnets.
Without explicitly configured ACLs, all traffic is allowed.
You can set ACL for a particular network in format '<network_name[:endpoint[:action]]>', where 'action' is either 'allow' (default) or 'drop'.
With ACLs configured, endpoints not matched by any rule are blocked.
To block all traffic define ACL with no endpoints: '<network_name>:'`)
	podModifyCmd.Flags().StringSliceVar(&vlans, "vlan", nil, `Connect application to the (switch) network over an access port assigned to the given VLAN.
You can set access VLAN ID (VID) for a particular network in the format '<network_name:VID>'`)
}
