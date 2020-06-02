package cmd

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var podName = ""

var podCmd = &cobra.Command{
	Use: "pod",
}

//podDeployCmd is command for deploy application on EVE
var podDeployCmd = &cobra.Command{
	Use:   "deploy <docker>://<TAG>[:<VERSION>]",
	Short: "Deploy app in pod",
	Long:  `Deploy app in pod.`,
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		appLink := args[0]
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev: %s", err)
		}
		expectation := expect.AppExpectationFromUrl(ctrl, appLink, podName)
		appInstanceConfig := expectation.Application()
		devModel, err := ctrl.GetDevModelByName(viper.GetString("eve.devmodel"))
		if err != nil {
			log.Fatalf("fail to get dev model %s: %s", viper.GetString("eve.devmodel"), err)
		}
		if err = ctrl.ApplyDevModel(dev, devModel); err != nil {
			log.Fatalf("ApplyDevModel: %s", err)
		}
		dev.SetApplicationInstanceConfig(append(dev.GetApplicationInstances(), appInstanceConfig.Uuidandversion.Uuid))
		if err = changer.setControllerAndDev(ctrl, dev); err != nil {
			log.Fatalf("setControllerAndDev: %s", err)
		}
		log.Infof("deploy pod %s with %s request sent", appInstanceConfig.Displayname, appLink)
	},
}

func podInit() {
	podCmd.AddCommand(podDeployCmd)
	podDeployCmd.Flags().StringVarP(&podName, "name", "n", "", "name for pod")
}
