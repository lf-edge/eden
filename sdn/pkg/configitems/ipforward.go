package configitems

import (
	"context"
	"fmt"
	"github.com/lf-edge/eve/libs/depgraph"
	log "github.com/sirupsen/logrus"
)

const (
	ipv4ForwardingKey = "net.ipv4.ip_forward"
	ipv6ForwardingKey = "net.ipv6.conf.all.forwarding"
)

// IPForwarding : item representing IP forwarding settings inside a given net namespace.
type IPForwarding struct {
	NetNamespace  string // network namespace name
	EnableForIPv4 bool
	EnableForIPv6 bool
}

// Name
func (f IPForwarding) Name() string {
	return normNetNsName(f.NetNamespace)
}

// Label
func (f IPForwarding) Label() string {
	return fmt.Sprintf("IP Forwarding in %s ns", normNetNsName(f.NetNamespace))
}

// Type
func (f IPForwarding) Type() string {
	return IPForwardingTypename
}

// Equal compares IP forwarding settings.
func (f IPForwarding) Equal(other depgraph.Item) bool {
	f2 := other.(IPForwarding)
	return f.EnableForIPv4 == f2.EnableForIPv4 &&
		f.EnableForIPv6 == f2.EnableForIPv6
}

// External returns false.
func (f IPForwarding) External() bool {
	return false
}

// String prints IP forwarding settings.
func (f IPForwarding) String() string {
	return fmt.Sprintf("Namespace: %s\nIPv4 Forwarding: %v\nIPv6 Forwarding: %v",
		normNetNsName(f.NetNamespace), f.EnableForIPv4, f.EnableForIPv6)
}

// Dependencies returns dependency on the network namespace.
func (f IPForwarding) Dependencies() (deps []depgraph.Dependency) {
	if isMainNetNs(f.NetNamespace) {
		return nil
	}
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

// IPForwardingConfigurator implements Configurator for IP forwarding settings.
type IPForwardingConfigurator struct{}

// Create applies IP forwarding settings.
func (c *IPForwardingConfigurator) Create(ctx context.Context, item depgraph.Item) error {
	f := item.(IPForwarding)
	return c.setIPForwarding(f.NetNamespace, f.EnableForIPv4, f.EnableForIPv6)
}

// Modify updates IP forwarding settings.
func (c *IPForwardingConfigurator) Modify(ctx context.Context, oldItem, newItem depgraph.Item) (err error) {
	f := newItem.(IPForwarding)
	return c.setIPForwarding(f.NetNamespace, f.EnableForIPv4, f.EnableForIPv6)
}

// Delete disables IP forwarding (default settings).
func (c *IPForwardingConfigurator) Delete(ctx context.Context, item depgraph.Item) error {
	f := item.(IPForwarding)
	return c.setIPForwarding(f.NetNamespace,false, false)
}

func (c *IPForwardingConfigurator) setIPForwarding(netNs string, v4, v6 bool) error {
	strValue := func(enable bool) string {
		if enable {
			return "1"
		}
		return "0"
	}
	sysctlKV := fmt.Sprintf("%s=%s", ipv4ForwardingKey, strValue(v4))
	out, err := namespacedCmd(netNs, "sysctl", "-w", sysctlKV).CombinedOutput()
	if err != nil {
		errMsg := fmt.Errorf("failed to set IPv4 forwarding: %s", out)
		log.Error(errMsg)
		return err
	}
	sysctlKV = fmt.Sprintf("%s=%s", ipv6ForwardingKey, strValue(v6))
	out, err = namespacedCmd(netNs, "sysctl", "-w", sysctlKV).CombinedOutput()
	if err != nil {
		errMsg := fmt.Errorf("failed to set IPv6 forwarding: %s", out)
		log.Error(errMsg)
		return err
	}
	return nil
}

// NeedsRecreate returns false - Modify is able to apply any change.
func (c *IPForwardingConfigurator) NeedsRecreate(oldItem, newItem depgraph.Item) (recreate bool) {
	return false
}
