package cmd

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

//podModifyCmd is a command to modify app
var podModifyCmd = &cobra.Command{
	Use:   "modify",
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
				opts = append(opts, expect.WithOldApp(appName))
				expectation := expect.AppExpectationFromURL(ctrl, dev, defaults.DefaultDummyExpect, appName, opts...)
				appInstanceConfig := expectation.Application()
				needPurge := false
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
}
