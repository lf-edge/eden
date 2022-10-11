package configitems

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/lf-edge/eve/libs/depgraph"
	"github.com/lf-edge/eve/libs/reconciler"
	log "github.com/sirupsen/logrus"
)

const dnsSrvNamePrefix = "dnssrv-"

// DnsServer : DNS server.
type DnsServer struct {
	// ServerName : logical name for the DNS server.
	ServerName string
	// NetNamespace : network namespace where the server should be running.
	NetNamespace string
	// VethName : logical name of the veth pair on which the server operates.
	// (other types of interfaces are currently not supported)
	VethName string
	// VethPeerIfName : interface name of that side of the veth pair on which
	// the server should listen. It should be inside NetNamespace.
	VethPeerIfName string
	// StaticEntries : list of FQDN->IP entries statically configured for the server.
	StaticEntries []DnsEntry
	// UpstreamServers : list of IP addresses of public DNS servers to forward
	// requests to (unless there is a static entry).
	UpstreamServers []net.IP
}

// DnsEntry : Mapping between FQDN and an IP address.
type DnsEntry struct {
	FQDN string
	IP   net.IP
}

// Name
func (s DnsServer) Name() string {
	return s.ServerName
}

// Label
func (s DnsServer) Label() string {
	return s.ServerName + " (DNS server)"
}

// Type
func (s DnsServer) Type() string {
	return DnsServerTypename
}

// Equal is a comparison method for two equally-named DnsServer instances.
func (s DnsServer) Equal(other depgraph.Item) bool {
	s2 := other.(DnsServer)
	if len(s.UpstreamServers) != len(s2.UpstreamServers) {
		return false
	}
	for i := range s.UpstreamServers {
		if !s.UpstreamServers[i].Equal(s2.UpstreamServers[i]) {
			return false
		}
	}
	if len(s.StaticEntries) != len(s2.StaticEntries) {
		return false
	}
	for i := range s.StaticEntries {
		if !s.StaticEntries[i].IP.Equal(s2.StaticEntries[i].IP) ||
			s.StaticEntries[i].FQDN != s2.StaticEntries[i].FQDN {
			return false
		}
	}
	return s.NetNamespace == s2.NetNamespace &&
		s.VethName == s2.VethName &&
		s.VethPeerIfName == s2.VethPeerIfName
}

// External returns false.
func (s DnsServer) External() bool {
	return false
}

// String describes the DNS server.
func (s DnsServer) String() string {
	return fmt.Sprintf("DNS Server: %#+v", s)
}

// Dependencies lists the veth and network namespace as dependencies.
func (s DnsServer) Dependencies() (deps []depgraph.Dependency) {
	return []depgraph.Dependency{
		{
			RequiredItem: depgraph.ItemRef{
				ItemType: NetNamespaceTypename,
				ItemName: normNetNsName(s.NetNamespace),
			},
			Description: "Network namespace must exist",
		},
		{
			RequiredItem: depgraph.ItemRef{
				ItemType: VethTypename,
				ItemName: s.VethName,
			},
			Description: "veth interface must exist",
		},
	}
}

// DnsServerConfigurator implements Configurator interface for DnsServer.
type DnsServerConfigurator struct{}

// Create starts dnsmasq (in DNS-only mode).
func (c *DnsServerConfigurator) Create(ctx context.Context, item depgraph.Item) error {
	config := item.(DnsServer)
	if err := c.createDnsmasqConfFile(config); err != nil {
		return err
	}
	done := reconciler.ContinueInBackground(ctx)
	go func() {
		err := startDnsmasq(dnsSrvNamePrefix+config.ServerName, config.NetNamespace)
		done(err)
	}()
	return nil
}

func (c *DnsServerConfigurator) createDnsmasqConfFile(server DnsServer) error {
	if err := ensureDir(dnsmasqConfDir); err != nil {
		return err
	}
	srvName := dnsSrvNamePrefix + server.ServerName
	cfgPath := dnsmasqConfigPath(srvName)
	file, err := os.Create(cfgPath)
	if err != nil {
		err = fmt.Errorf("failed to create config file %s: %w", cfgPath, err)
		log.Error(err)
		return err
	}
	defer file.Close()
	// PID file is also used by Delete method.
	file.WriteString(fmt.Sprintf("pid-file=%s\n", dnsmasqPidFile(srvName)))
	// Set the interface on which dnsmasq operates.
	file.WriteString(fmt.Sprintf("interface=%s\n", server.VethPeerIfName))
	// Disable DHCP.
	file.WriteString(fmt.Sprintf("no-dhcp-interface=%s\n", server.VethPeerIfName))
	// Logging.
	file.WriteString("log-queries\n")
	file.WriteString(fmt.Sprintf("log-facility=%s\n", dnsmasqLogFile(srvName)))
	// Upstream DNS servers.
	for _, upstreamSrv := range server.UpstreamServers {
		file.WriteString(fmt.Sprintf("server=%s\n", upstreamSrv))
	}
	file.WriteString("no-resolv\n")
	// Static DNS entries.
	for _, entry := range server.StaticEntries {
		file.WriteString(fmt.Sprintf("address=/%s/%s\n", entry.FQDN, entry.IP.String()))
	}
	file.WriteString("no-hosts\n")
	if err = file.Sync(); err != nil {
		err = fmt.Errorf("failed to sync config file %s: %w", cfgPath, err)
		log.Error(err)
		return err
	}
	return nil
}

// Modify is not implemented.
func (c *DnsServerConfigurator) Modify(ctx context.Context, oldItem, newItem depgraph.Item) (err error) {
	return errors.New("not implemented")
}

// Delete stops dnsmasq.
func (c *DnsServerConfigurator) Delete(ctx context.Context, item depgraph.Item) error {
	config := item.(DnsServer)
	done := reconciler.ContinueInBackground(ctx)
	go func() {
		srvName := dnsSrvNamePrefix + config.ServerName
		err := stopDnsmasq(srvName)
		if err == nil {
			// ignore errors from here
			_ = removeDnsmasqConfFile(srvName)
			_ = removeDnsmasqLogFile(srvName)
			_ = removeDnsmasqPidFile(srvName)
		}
		done(err)
	}()
	return nil
}

// NeedsRecreate always returns true - Modify is not implemented.
func (c *DnsServerConfigurator) NeedsRecreate(oldItem, newItem depgraph.Item) (recreate bool) {
	return true
}
