package configitems

import (
	"context"
	"fmt"
	"github.com/lf-edge/eve/libs/depgraph"
	log "github.com/sirupsen/logrus"
)

const (
	ipv4ForwardingKey  = "net.ipv4.ip_forward"
	ipv6ForwardingKey  = "net.ipv6.conf.all.forwarding"
	bridgeIptablesKey  = "net.bridge.bridge-nf-call-iptables"
	bridgeIp6tablesKey = "net.bridge.bridge-nf-call-ip6tables"
)

// Sysctl : item representing kernel parameters set using sysctl.
type Sysctl struct {
	// NetNamespace : network namespace name
	NetNamespace          string
	EnableIPv4Forwarding  bool
	EnableIPv6Forwarding  bool
	BridgeNfCallIptables  bool
	BridgeNfCallIp6tables bool
}

// Name
func (f Sysctl) Name() string {
	return normNetNsName(f.NetNamespace)
}

// Label
func (f Sysctl) Label() string {
	return fmt.Sprintf("sysctl for %s ns", normNetNsName(f.NetNamespace))
}

// Type
func (f Sysctl) Type() string {
	return SysctlTypename
}

// Equal compares sysctl settings.
func (f Sysctl) Equal(other depgraph.Item) bool {
	f2 := other.(Sysctl)
	return f == f2
}

// External returns false.
func (f Sysctl) External() bool {
	return false
}

// String prints sysctl settings.
func (f Sysctl) String() string {
	return fmt.Sprintf("Namespace: %s\nIPv4 Forwarding: %v\nIPv6 Forwarding: %v\n"+
		"Bridge uses Iptables: %v\nBridge uses Ip6tables: %v",
		normNetNsName(f.NetNamespace), f.EnableIPv4Forwarding, f.EnableIPv6Forwarding,
		f.BridgeNfCallIptables, f.BridgeNfCallIp6tables)
}

// Dependencies returns dependency on the network namespace.
func (f Sysctl) Dependencies() (deps []depgraph.Dependency) {
	return []depgraph.Dependency{
		{
			RequiredItem: depgraph.ItemRef{
				ItemType: NetNamespaceTypename,
				ItemName: normNetNsName(f.NetNamespace),
			},
			Description: "Network namespace must exist",
		},
	}
}

// SysctlConfigurator implements Configurator for sysctl settings.
type SysctlConfigurator struct{}

// Create applies sysctl settings.
func (c *SysctlConfigurator) Create(ctx context.Context, item depgraph.Item) error {
	f := item.(Sysctl)
	err := c.setIPForwarding(f.NetNamespace, f.EnableIPv4Forwarding, f.EnableIPv6Forwarding)
	if err != nil {
		return err
	}
	return c.setBridgeIptables(f.NetNamespace, f.BridgeNfCallIptables, f.BridgeNfCallIp6tables)
}

// Modify updates sysctl settings.
func (c *SysctlConfigurator) Modify(ctx context.Context, oldItem, newItem depgraph.Item) error {
	f := newItem.(Sysctl)
	err := c.setIPForwarding(f.NetNamespace, f.EnableIPv4Forwarding, f.EnableIPv6Forwarding)
	if err != nil {
		return err
	}
	return c.setBridgeIptables(f.NetNamespace, f.BridgeNfCallIptables, f.BridgeNfCallIp6tables)
}

// Delete sets default sysctl settings.
func (c *SysctlConfigurator) Delete(ctx context.Context, item depgraph.Item) error {
	f := item.(Sysctl)
	err := c.setIPForwarding(f.NetNamespace, false, false)
	if err != nil {
		return err
	}
	return c.setBridgeIptables(f.NetNamespace, true, true)
}

func (c *SysctlConfigurator) setIPForwarding(netNs string, v4, v6 bool) error {
	sysctlKV := fmt.Sprintf("%s=%s", ipv4ForwardingKey, c.boolValueToStr(v4))
	out, err := namespacedCmd(netNs, "sysctl", "-w", sysctlKV).CombinedOutput()
	if err != nil {
		errMsg := fmt.Errorf("failed to set IPv4 forwarding: %s", out)
		log.Error(errMsg)
		return err
	}
	sysctlKV = fmt.Sprintf("%s=%s", ipv6ForwardingKey, c.boolValueToStr(v6))
	out, err = namespacedCmd(netNs, "sysctl", "-w", sysctlKV).CombinedOutput()
	if err != nil {
		errMsg := fmt.Errorf("failed to set IPv6 forwarding: %s", out)
		log.Error(errMsg)
		return err
	}
	return nil
}

func (c *SysctlConfigurator) setBridgeIptables(netNs string, v4, v6 bool) error {
	sysctlKV := fmt.Sprintf("%s=%s", bridgeIptablesKey, c.boolValueToStr(v4))
	out, err := namespacedCmd(netNs, "sysctl", "-w", sysctlKV).CombinedOutput()
	if err != nil {
		errMsg := fmt.Errorf("failed to set BridgeNfCallIptables: %s", out)
		log.Error(errMsg)
		return err
	}
	sysctlKV = fmt.Sprintf("%s=%s", bridgeIp6tablesKey, c.boolValueToStr(v6))
	out, err = namespacedCmd(netNs, "sysctl", "-w", sysctlKV).CombinedOutput()
	if err != nil {
		errMsg := fmt.Errorf("failed to set BridgeNfCallIp6tables: %s", out)
		log.Error(errMsg)
		return err
	}
	return nil
}

// NeedsRecreate returns false - Modify is able to apply any change.
func (c *SysctlConfigurator) NeedsRecreate(oldItem, newItem depgraph.Item) (recreate bool) {
	return false
}

func (c *SysctlConfigurator) boolValueToStr(enable bool) string {
	if enable {
		return "1"
	}
	return "0"
}
