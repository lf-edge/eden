package configitems

import (
	"github.com/lf-edge/eden/sdn/pkg/maclookup"
	"github.com/lf-edge/eve/libs/reconciler"
)

// RegisterItems : register all configurators implemented by this package.
func RegisterItems(
	registry *reconciler.DefaultRegistry, macLookup *maclookup.MacLookup) error {
	type configurator struct {
		c reconciler.Configurator
		t string
	}
	configurators := []configurator{
		{c: &IPForwardingConfigurator{}, t: IPForwardingTypename},
		{c: &NetNamespaceConfigurator{}, t: NetNamespaceTypename},
		{c: &IfHandleConfigurator{MacLookup: macLookup}, t: IfHandleTypename},
		{c: &DhcpClientConfigurator{MacLookup: macLookup}, t: DhcpClientTypename},
		{c: &DhcpServerConfigurator{}, t: DhcpServerTypename},
		{c: &DnsServerConfigurator{}, t: DnsServerTypename},
		{c: &BondConfigurator{MacLookup: macLookup}, t: BondTypename},
		{c: &BridgeConfigurator{MacLookup: macLookup}, t: BridgeTypename},
		{c: &VethConfigurator{}, t: VethTypename},
		{c: &RouteConfigurator{MacLookup: macLookup}, t: RouteTypename},
		{c: &IPRuleConfigurator{}, t: IPRuleTypename},
		{c: &IptablesChainConfigurator{}, t: IPtablesChainTypename},
		{c: &IptablesChainConfigurator{}, t: IP6tablesChainTypename},
	}
	for _, configurator := range configurators {
		err := registry.Register(configurator.c, configurator.t)
		if err != nil {
			return err
		}
	}
	return nil
}
