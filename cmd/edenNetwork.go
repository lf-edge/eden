package cmd

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eve"
	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	networkType string
	networkName string
)

var networkCmd = &cobra.Command{
	Use: "network",
}

//networkLsCmd is a command to list deployed network instances
var networkLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List networks",
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
		if err := ctrl.InfoLastCallback(dev.GetID(), nil, state.InfoCallback()); err != nil {
			log.Fatalf("fail in get InfoLastCallback: %s", err)
		}
		if err := ctrl.MetricLastCallback(dev.GetID(), nil, state.MetricCallback()); err != nil {
			log.Fatalf("fail in get MetricLastCallback: %s", err)
		}
		if err := state.NetList(); err != nil {
			log.Fatal(err)
		}
	},
}

//networkDeleteCmd is a command to delete network instance from EVE
var networkDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete network",
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
		niName := args[0]
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev: %s", err)
		}
		for id, el := range dev.GetNetworkInstances() {
			ni, err := ctrl.GetNetworkInstanceConfig(el)
			if err != nil {
				log.Fatalf("no network in cloud %s: %s", el, err)
			}
			if ni.Displayname == niName {
				configs := dev.GetNetworkInstances()
				utils.DelEleInSlice(&configs, id)
				dev.SetNetworkInstanceConfig(configs)
				if err = changer.setControllerAndDev(ctrl, dev); err != nil {
					log.Fatalf("setControllerAndDev: %s", err)
				}
				log.Infof("network %s delete done", niName)
				return
			}
		}
		log.Infof("not found network with name %s", niName)
	},
}

//networkCreateCmd is command for create network instance in EVE
var networkCreateCmd = &cobra.Command{
	Use:   "create [subnet]",
	Short: "Create network instance in EVE",
	Args:  cobra.RangeArgs(0, 1),
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
		if networkType != "local" && networkType != "switch" {
			log.Fatalf("Network type %s not supported now", networkType)
		}
		subnet := ""
		if networkType == "local" {
			if len(args) != 1 {
				log.Fatal("You must define subnet as first arg for local network")
			}
			subnet = args[0]
		}
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev: %s", err)
		}
		var opts []expect.ExpectationOption
		opts = append(opts, expect.AddNetInstanceAndPortPublish(subnet, networkType, networkName, nil))
		expectation := expect.AppExpectationFromURL(ctrl, dev, defaults.DefaultDummyExpect, podName, opts...)
		netInstancesConfigs := expectation.NetworkInstances()
	mainloop:
		for _, el := range netInstancesConfigs {
			for _, element := range dev.GetNetworkInstances() {
				if element == el.Uuidandversion.Uuid {
					log.Infof("network with defined parameters already exists")
					continue mainloop
				}
			}
			dev.SetNetworkInstanceConfig(append(dev.GetNetworkInstances(), el.Uuidandversion.Uuid))
			log.Infof("deploy network %s with name %s request sent", el.Uuidandversion.Uuid, el.Displayname)
		}
		if err = changer.setControllerAndDev(ctrl, dev); err != nil {
			log.Fatalf("setControllerAndDev: %s", err)
		}
	},
}

func networkInit() {
	networkCmd.AddCommand(networkLsCmd)
	networkCmd.AddCommand(networkDeleteCmd)
	networkCmd.AddCommand(networkCreateCmd)
	networkCreateCmd.Flags().StringVar(&networkType, "type", "local", "Type of network: local or switch")
	networkCreateCmd.Flags().StringVarP(&networkName, "name", "n", "", "Name of network (empty for auto generation)")
}
