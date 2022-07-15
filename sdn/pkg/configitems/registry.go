package configitems

import (
	"github.com/lf-edge/eden/sdn/pkg/netmonitor"
	"github.com/lf-edge/eve/libs/reconciler"
)

// RegisterItems : register all configurators implemented by this package.
func RegisterItems(
	registry *reconciler.DefaultRegistry, netMonitor *netmonitor.NetworkMonitor) error {
	type configurator struct {
		c reconciler.Configurator
		t string
	}
	configurators := []configurator{
		{c: &IPForwardingConfigurator{}, t: IPForwardingTypename},
		{c: &NetNamespaceConfigurator{}, t: NetNamespaceTypename},
		{c: &IfHandleConfigurator{netMonitor: netMonitor}, t: IfHandleTypename},
		{c: &DhcpcdConfigurator{netMonitor: netMonitor}, t: DhcpcdTypename},
	}
	for _, configurator := range configurators {
		err := registry.Register(configurator.c, configurator.t)
		if err != nil {
			return err
		}
	}
	return nil
}
