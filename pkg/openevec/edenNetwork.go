package openevec

import (
	"fmt"

	"github.com/lf-edge/eden/pkg/controller/eflowlog"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/eve"
	"github.com/lf-edge/eden/pkg/expect"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
)

func NetworkLs() error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %w", err)
	}
	state := eve.Init(ctrl, dev)
	if err := ctrl.InfoLastCallback(dev.GetID(), nil, state.InfoCallback()); err != nil {
		return fmt.Errorf("fail in get InfoLastCallback: %w", err)
	}
	if err := ctrl.MetricLastCallback(dev.GetID(), nil, state.MetricCallback()); err != nil {
		return fmt.Errorf("fail in get MetricLastCallback: %w", err)
	}
	if err := state.NetList(); err != nil {
		return err
	}
	return nil
}

func NetworkDelete(niName string) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %w", err)
	}
	for id, el := range dev.GetNetworkInstances() {
		ni, err := ctrl.GetNetworkInstanceConfig(el)
		if err != nil {
			return fmt.Errorf("no network in cloud %s: %w", el, err)
		}
		if ni.Displayname == niName {
			configs := dev.GetNetworkInstances()
			utils.DelEleInSlice(&configs, id)
			dev.SetNetworkInstanceConfig(configs)
			if err = changer.setControllerAndDev(ctrl, dev); err != nil {
				return fmt.Errorf("setControllerAndDev: %w", err)
			}
			log.Infof("network %s delete done", niName)
			return nil
		}
	}
	log.Infof("not found network with name %s", niName)
	return nil
}

func NetworkNetstat(niName string, outputFormat types.OutputFormat, outputTail uint) error {
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %w", err)
	}
	for _, el := range dev.GetNetworkInstances() {
		ni, err := ctrl.GetNetworkInstanceConfig(el)
		if err != nil {
			return fmt.Errorf("no network in cloud %s: %s", el, err)
		}
		if ni.Displayname == niName {
			// block for process FlowLog
			fmt.Printf("netstat list for network %s:\n", ni.Uuidandversion.Uuid)
			// process only existing elements
			flowLogType := eflowlog.FlowLogExist

			if outputTail > 0 {
				// process only outputTail elements from end
				flowLogType = eflowlog.FlowLogTail(outputTail)
			}

			// logsQ for filtering logs by app
			logsQ := make(map[string]string)
			logsQ["scope.netInstUUID"] = ni.Uuidandversion.Uuid
			if err = ctrl.FlowLogChecker(dev.GetID(), logsQ, eflowlog.HandleFactory(outputFormat, false), flowLogType, 0); err != nil {
				return fmt.Errorf("FlowLogChecker: %w", err)
			}
			return nil
		}
	}
	log.Infof("not found network with name %s", niName)

	return nil
}

func NetworkCreate(subnet, networkType, networkName, uplinkAdapter string, staticDNSEntries []string) error {
	if networkType != "local" && networkType != "switch" {
		return fmt.Errorf("network type %s not supported now", networkType)
	}
	if networkType == "local" && subnet == "" {
		return fmt.Errorf("you must define subnet as first arg for local network")
	}
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %w", err)
	}
	var opts []expect.ExpectationOption
	opts = append(opts, expect.AddNetInstanceAndPortPublish(subnet, networkType, networkName, nil, uplinkAdapter))
	opts = append(opts, expect.WithStaticDNSEntries(networkName, staticDNSEntries))
	expectation := expect.AppExpectationFromURL(ctrl, dev, defaults.DefaultDummyExpect, "", opts...)
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
		return fmt.Errorf("setControllerAndDev: %w", err)
	}

	return nil
}
