package cmd

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/eve"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var volumeCmd = &cobra.Command{
	Use: "volume",
}

//networkLsCmd is a command to list deployed volumes
var volumeLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List volumes",
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
		state := eve.Init(ctrl, dev)
		if err := ctrl.MetricLastCallback(dev.GetID(), nil, state.MetricCallback()); err != nil {
			log.Fatalf("fail in get InfoLastCallback: %s", err)
		}
		if err := ctrl.InfoLastCallback(dev.GetID(), nil, state.InfoCallback()); err != nil {
			log.Fatalf("fail in get InfoLastCallback: %s", err)
		}
		if err := state.VolumeList(); err != nil {
			log.Fatal(err)
		}
	},
}

func volumeInit() {
	volumeCmd.AddCommand(volumeLsCmd)
}
