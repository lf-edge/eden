package configitems

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/lf-edge/eve/libs/depgraph"
	"github.com/lf-edge/eve/libs/reconciler"
	log "github.com/sirupsen/logrus"
)

const (
	dnsmasqBinary       = "/usr/local/sbin/dnsmasq"
	dnsmasqStartTimeout = 3 * time.Second
	dnsmasqStopTimeout  = 30 * time.Second
	dnsmasqConfDir      = "/etc/dnsmasq"
	dnsmasqRunDir       = "/run/dnsmasq"

	dhcpSrvNamePrefix = "dhcpsrv-"
)

// DhcpServer : DHCP server.
type DhcpServer struct {
	// ServerName : logical name for the DHCP server.
	ServerName string
	// NetNamespace : network namespace where the server should be running.
	NetNamespace string
	// VethName : logical name of the veth pair on which the server operates.
	// (other types of interfaces are currently not supported)
	VethName string
	// VethPeerIfName : interface name of that side of the veth pair on which
	// the server should listen. It should be inside NetNamespace.
	VethPeerIfName string
	// Subnet : network address + netmask (IPv4 or IPv6).
	Subnet *net.IPNet
	// IPRange : a range of IP addresses to allocate from.
	// Not applicable for IPv6 (SLAAC is used instead).
	IPRange IPRange
	// GatewayIP : address of the default gateway to advertise (DHCP option 3).
	GatewayIP net.IP
	// DomainName : name of the domain assigned to the network.
	// It is propagated to clients using the DHCP option 15 (24 in DHCPv6).
	DomainName string
	// DNSServers : list of IP addresses of DNS servers to announce via DHCP option 6.
	DNSServers []net.IP
	// NTP server to announce via DHCP option 42 (56 in DHCPv6).
	// Optional argument, leave empty to disable.
	NTPServer string
	// WPAD : URL with a location of a PAC file, announced using the Web Proxy Auto-Discovery
	// Protocol (WPAD) and DHCP.
	// The client will learn the PAC file location using the DHCP option 252.
	// Optional argument, leave empty to disable.
	WPAD string
	// TODO: Netboot
	//  Example dnsmasq.conf:
	//    # use custom tftp-server instead machine running dnsmasq
	//    dhcp-boot=pxelinux,server.name,192.168.1.100
	//    # Boot for iPXE. The idea is to send two different
	//    # filenames, the first loads iPXE, and the second tells iPXE what to
	//    # load. The dhcp-match sets the ipxe tag for requests from iPXE.
	//    dhcp-boot=undionly.kpxe
	//    dhcp-match=set:ipxe,175 # iPXE sends a 175 option.
	//    dhcp-boot=tag:ipxe,http://boot.ipxe.org/demo/boot.php
}

// IPRange : a range of IP addresses.
type IPRange struct {
	// FromIP : start of the range (includes the address itself).
	FromIP net.IP
	// ToIP : end of the range (includes the address itself).
	ToIP net.IP
}

// Name
func (s DhcpServer) Name() string {
	return s.ServerName
}

// Label
func (s DhcpServer) Label() string {
	return s.ServerName + " (DHCP server)"
}

// Type
func (s DhcpServer) Type() string {
	return DhcpServerTypename
}

// Equal is a comparison method for two equally-named DhcpServer instances.
func (s DhcpServer) Equal(other depgraph.Item) bool {
	s2 := other.(DhcpServer)
	return s.NetNamespace == s2.NetNamespace &&
		s.VethName == s2.VethName &&
		s.VethPeerIfName == s2.VethPeerIfName &&
		equalIPNets(s.Subnet, s2.Subnet) &&
		s.IPRange.FromIP.Equal(s2.IPRange.FromIP) &&
		s.IPRange.ToIP.Equal(s2.IPRange.ToIP) &&
		s.GatewayIP.Equal(s2.GatewayIP) &&
		s.DomainName == s2.DomainName &&
		equalIPLists(s.DNSServers, s2.DNSServers) &&
		s.NTPServer == s2.NTPServer &&
		s.WPAD == s2.WPAD
}

// External returns false.
func (s DhcpServer) External() bool {
	return false
}

// String describes the DHCP server config.
func (s DhcpServer) String() string {
	return fmt.Sprintf("DHCP Server: %#+v", s)
}

// Dependencies lists the veth and network namespace as dependencies.
func (s DhcpServer) Dependencies() (deps []depgraph.Dependency) {
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

// DhcpServerConfigurator implements Configurator interface for DhcpServer.
type DhcpServerConfigurator struct{}

// Create starts dnsmasq (in DHCP-only mode).
func (c *DhcpServerConfigurator) Create(ctx context.Context, item depgraph.Item) error {
	config := item.(DhcpServer)
	if err := c.createDnsmasqConfFile(config); err != nil {
		return err
	}
	done := reconciler.ContinueInBackground(ctx)
	go func() {
		err := startDnsmasq(dhcpSrvNamePrefix+config.ServerName, config.NetNamespace)
		done(err)
	}()
	return nil
}

func (c *DhcpServerConfigurator) createDnsmasqConfFile(server DhcpServer) error {
	isIPv6 := len(server.Subnet.IP) == net.IPv6len
	if err := ensureDir(dnsmasqConfDir); err != nil {
		return err
	}
	srvName := dhcpSrvNamePrefix + server.ServerName
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
	// To enable dnsmasq's DHCP server functionality.
	if isIPv6 {
		// Do Router Advertisements and stateless DHCPv6 for this subnet. Clients will
		// not get addresses from DHCP, but they will get other configuration information.
		// They will use SLAAC for addresses.
		file.WriteString(fmt.Sprintf("dhcp-range=%s,ra-stateless\n", server.Subnet))
	} else {
		netmask := net.IP(server.Subnet.Mask)
		file.WriteString(fmt.Sprintf("dhcp-range=%s,%s,%s,60m\n",
			server.IPRange.FromIP, server.IPRange.ToIP, netmask))
	}
	file.WriteString(fmt.Sprintf("dhcp-leasefile=%s\n",
		dnsmasqLeaseFile(srvName)))
	// To disable dnsmasq's DNS server functionality.
	file.WriteString("port=0\n")
	// Set the interface on which dnsmasq operates.
	file.WriteString(fmt.Sprintf("interface=%s\n", server.VethPeerIfName))
	// Logging.
	file.WriteString("log-dhcp\n")
	file.WriteString(fmt.Sprintf("log-facility=%s\n", dnsmasqLogFile(srvName)))
	// Domain name.
	if server.DomainName != "" {
		if isIPv6 {
			file.WriteString(fmt.Sprintf("dhcp-option=option:domain-search,%s\n",
				server.DomainName))
		} else {
			file.WriteString(fmt.Sprintf("dhcp-option=option:domain-name,%s\n",
				server.DomainName))
		}
	}
	// Default gateway.
	if len(server.GatewayIP) != 0 {
		// IPv6 needs to be handled with radvd.
		if !isIPv6 {
			gwIP := server.GatewayIP.String()
			file.WriteString(fmt.Sprintf("dhcp-option=option:router,%s\n", gwIP))
		}
	}
	// DNS servers.
	if len(server.DNSServers) > 0 {
		var addrList []string
		for _, srvIP := range server.DNSServers {
			addrList = append(addrList, srvIP.String())
		}
		file.WriteString(fmt.Sprintf("dhcp-option=option:dns-server,%s\n",
			strings.Join(addrList, ",")))
	}
	// NTP Server.
	if server.NTPServer != "" {
		file.WriteString(fmt.Sprintf("dhcp-option=option:ntp-server,%s\n", server.NTPServer))
	}
	// WPAD.
	if server.WPAD != "" {
		file.WriteString(fmt.Sprintf("dhcp-option=252,%s\n", server.WPAD))
	}
	if err = file.Sync(); err != nil {
		err = fmt.Errorf("failed to sync config file %s: %w", cfgPath, err)
		log.Error(err)
		return err
	}
	return nil
}

// Modify is not implemented.
func (c *DhcpServerConfigurator) Modify(ctx context.Context, oldItem, newItem depgraph.Item) (err error) {
	return errors.New("not implemented")
}

// Delete stops dnsmasq.
func (c *DhcpServerConfigurator) Delete(ctx context.Context, item depgraph.Item) error {
	config := item.(DhcpServer)
	done := reconciler.ContinueInBackground(ctx)
	go func() {
		srvName := dhcpSrvNamePrefix + config.ServerName
		err := stopDnsmasq(srvName)
		if err == nil {
			// ignore errors from here
			_ = removeDnsmasqConfFile(srvName)
			_ = removeDnsmasqLeaseFile(srvName)
			_ = removeDnsmasqLogFile(srvName)
			_ = removeDnsmasqPidFile(srvName)
		}
		done(err)
	}()
	return nil
}

// NeedsRecreate always returns true - Modify is not implemented.
func (c *DhcpServerConfigurator) NeedsRecreate(oldItem, newItem depgraph.Item) (recreate bool) {
	return true
}

func dnsmasqConfigPath(srvName string) string {
	return filepath.Join(dnsmasqConfDir, srvName+".conf")
}

func dnsmasqPidFile(srvName string) string {
	return filepath.Join(dnsmasqRunDir, srvName+".pid")
}

func dnsmasqLogFile(srvName string) string {
	return filepath.Join(dnsmasqRunDir, srvName+".log")
}

func dnsmasqLeaseFile(srvName string) string {
	return filepath.Join(dnsmasqRunDir, srvName+".leases")
}

func startDnsmasq(srvName, netNamespace string) error {
	if err := ensureDir(dnsmasqRunDir); err != nil {
		return err
	}
	cmd := "nohup"
	cfgPath := dnsmasqConfigPath(srvName)
	args := []string{
		dnsmasqBinary,
		"-C",
		cfgPath,
	}
	pidFile := dnsmasqPidFile(srvName)
	// Do not run in background - dnsmasq will detach itself.
	return startProcess(netNamespace, cmd, args, pidFile, dnsmasqStartTimeout, false)
}

func startProcess(netNamespace, cmd string, args []string, pidFile string,
	timeout time.Duration, background bool) error {
	startTime := time.Now()
	execCmd := namespacedCmd(netNamespace, cmd, args...)
	if background {
		err := execCmd.Start()
		if err != nil {
			err = fmt.Errorf("failed to start command %s (args: %v): %v", cmd, args, err)
			log.Error(err)
			return err
		}
	} else {
		out, err := execCmd.CombinedOutput()
		if err != nil {
			err = fmt.Errorf("failed to start command %s (args: %v): %s", cmd, args, out)
			log.Error(err)
			return err
		}
	}
	// Wait for the process to start.
	for !isProcessRunning(pidFile) {
		if time.Since(startTime) > timeout {
			err := fmt.Errorf("command %s (args: %v) failed to start in time", cmd, args)
			log.Error(err)
			return err
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

func removeDnsmasqConfFile(srvName string) error {
	cfgPath := dnsmasqConfigPath(srvName)
	if err := os.Remove(cfgPath); err != nil {
		err = fmt.Errorf("failed to remove dnsmasq config %s: %w",
			cfgPath, err)
		log.Error(err)
		return err
	}
	return nil
}

func removeDnsmasqPidFile(srvName string) error {
	pidPath := dnsmasqPidFile(srvName)
	if err := os.Remove(pidPath); err != nil {
		err = fmt.Errorf("failed to remove dnsmasq PID file %s: %w",
			pidPath, err)
		log.Error(err)
		return err
	}
	return nil
}

func removeDnsmasqLogFile(srvName string) error {
	logPath := dnsmasqLogFile(srvName)
	if err := os.Remove(logPath); err != nil {
		err = fmt.Errorf("failed to remove dnsmasq log file %s: %w",
			logPath, err)
		log.Error(err)
		return err
	}
	return nil
}

func removeDnsmasqLeaseFile(srvName string) error {
	leasePath := dnsmasqLeaseFile(srvName)
	if err := os.Remove(leasePath); err != nil {
		err = fmt.Errorf("failed to remove dnsmasq lease file %s: %w",
			leasePath, err)
		log.Error(err)
		return err
	}
	return nil
}

func stopDnsmasq(srvName string) error {
	pidFile := dnsmasqPidFile(srvName)
	return stopProcess(pidFile, dnsmasqStopTimeout)
}

func stopProcess(pidFile string, timeout time.Duration) error {
	process := getProcess(pidFile)
	if process == nil {
		err := fmt.Errorf("process pid-file=%s is not running", pidFile)
		log.Error(err)
		return err
	}
	stopTime := time.Now()
	err := process.Signal(syscall.SIGTERM)
	if err != nil {
		err := fmt.Errorf("SIGTERM signal sent to process pid-file=%s failed: %w",
			pidFile, err)
		log.Error(err)
		return err
	}
	// Wait for the process to stop.
	for isProcessRunning(pidFile) {
		if time.Since(stopTime) > timeout {
			err := fmt.Errorf("process pid-file=%s failed to stop in time", pidFile)
			log.Error(err)
			return err
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

func isProcessRunning(pidFile string) bool {
	process := getProcess(pidFile)
	if process == nil {
		return false
	}
	err := process.Signal(syscall.Signal(0))
	if err != nil {
		log.Errorf("isProcessRunning(%s): signal failed %s", pidFile, err)
		return false
	}
	return true
}

func getProcess(pidFile string) (process *os.Process) {
	pidBytes, err := ioutil.ReadFile(pidFile)
	if err != nil {
		// Not running, return nil.
		return nil
	}
	pidStr := strings.TrimSpace(string(pidBytes))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		log.Errorf("getProcess(%s): strconv.Atoi of %s failed: %v",
			pidFile, pidStr, err)
		return nil
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		log.Errorf("getProcess(%s): process PID=%d not found: %v",
			pidFile, pid, err)
		return nil
	}
	return p
}

func equalIPLists(ips1, ips2 []net.IP) bool {
	if len(ips1) != len(ips2) {
		return false
	}
	for i := range ips1 {
		if !ips1[i].Equal(ips2[i]) {
			return false
		}
	}
	return true
}
