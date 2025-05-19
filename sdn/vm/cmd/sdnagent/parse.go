package main

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"

	"github.com/lf-edge/eden/sdn/vm/api"
	log "github.com/sirupsen/logrus"
)

const (
	// Minimum accepted MTU value.
	// Eventually, Eden-SDN will incorporate IPv6 support.
	// As per RFC 8200, the MTU must not be less than 1280 bytes to accommodate IPv6 packets.
	minMTU = 1280
	// Maximum MTU supported by the e1000 driver (used for interfaces connecting
	// Eden-SDN with the EVE VM).
	maxMTU = 16110
)

type parsedNetModel struct {
	api.NetworkModel
	items    labeledItems
	hostIPv4 net.IP
	hostIPv6 net.IP
}

type labeledItems map[itemID]*labeledItem

func (li labeledItems) getItem(typename, logicalLabel string) *labeledItem {
	return li[itemID{
		typename:     typename,
		logicalLabel: logicalLabel,
	}]
}

type itemID struct {
	typename     string
	logicalLabel string
}

type itemRef struct {
	itemID
	refKey string
}

type labeledItem struct {
	api.LabeledItem
	category     string            // empty if not categorized
	referencing  []itemRef         // other items referenced by this item
	referencedBy map[string]itemID // RefKey -> item
}

// Parse and validate network model.
func (a *agent) parseNetModel(netModel api.NetworkModel) (parsedModel parsedNetModel, err error) {
	parsedModel.NetworkModel = netModel

	// Parse and validate logical labels and their referencing.
	eps := netModel.Endpoints
	items := a.slicesToLabeledItems(netModel.Ports, netModel.Bonds, netModel.Bridges,
		netModel.Networks, eps.DNSServers, eps.NTPServers, eps.NetbootServers,
		eps.HTTPServers, eps.ExplicitProxies, eps.TransparentProxies, eps.Clients)
	parsedModel.items, err = a.parseLabeledItems(items)
	if err != nil {
		return
	}

	if err = a.validatePorts(&parsedModel); err != nil {
		return
	}
	if err = a.validateHostConfig(&parsedModel); err != nil {
		return
	}
	if err = a.validateNetworks(&parsedModel); err != nil {
		return
	}
	if err = a.validateEndpoints(&parsedModel); err != nil {
		return
	}
	if err = a.validateFirewall(&parsedModel); err != nil {
		return
	}
	return
}

func (a *agent) validatePorts(netModel *parsedNetModel) (err error) {
	// Every port should have a valid and unique MAC address.
	macs := make(map[string]struct{})
	for _, port := range netModel.Ports {
		if _, duplicate := macs[port.MAC]; duplicate {
			err = fmt.Errorf("port %s has duplicate MAC address %s",
				port.LogicalLabel, port.MAC)
			return
		}
		macs[port.MAC] = struct{}{}
		var mac net.HardwareAddr
		mac, err = net.ParseMAC(port.MAC)
		if err != nil {
			err = fmt.Errorf("port %s has invalid MAC address: %v", port.LogicalLabel, err)
			return
		}
		if bytes.HasPrefix(mac, hostPortMACPrefix) {
			err = fmt.Errorf("port %s has MAC address with prefix reserved for the host port",
				port.LogicalLabel)
			return
		}
		if _, err = net.ParseMAC(port.EVEConnect.MAC); err != nil {
			err = fmt.Errorf("EVE-side of port %s has invalid MAC address: %v",
				port.LogicalLabel, err)
			return
		}
	}

	// QueueLimit and BurstLimit are mandatory when RateLimit is set.
	for _, port := range netModel.Ports {
		if port.TC.RateLimit != 0 {
			if port.TC.QueueLimit == 0 {
				err = fmt.Errorf("RateLimit set for port %s without QueueLimit",
					port.LogicalLabel)
				return
			}
			if port.TC.BurstLimit == 0 {
				err = fmt.Errorf("RateLimit set for port %s without BurstLimit",
					port.LogicalLabel)
				return
			}
		}
	}
	return nil
}

func (a *agent) validateNetworks(netModel *parsedNetModel) (err error) {
	// Validate network IP config.
	for _, network := range netModel.Networks {
		if network.IsDualStack() {
			ipv4Conf := network.DualStack.IPv4
			ipv6Conf := network.DualStack.IPv6
			err = a.validateNetworkIPConfig(network.LogicalLabel, ipv4Conf, true, false)
			if err != nil {
				return
			}
			err = a.validateNetworkIPConfig(network.LogicalLabel, ipv6Conf, false, true)
			if err != nil {
				return
			}
			// It does not make sense to define different domain name for IPv4 and IPv6.
			if ipv4Conf.DHCP.DomainName != "" && ipv6Conf.DHCP.DomainName != "" {
				if ipv4Conf.DHCP.DomainName != ipv6Conf.DHCP.DomainName {
					err = fmt.Errorf(
						"dual-stack network %s is defined with two different domain names",
						network.LogicalLabel)
					return
				}
			}
		} else {
			err = a.validateNetworkIPConfig(network.LogicalLabel, network.NetworkIPConfig,
				false, false)
			if err != nil {
				return
			}
		}
	}

	// Validate routes towards EVE.
	for _, network := range netModel.Networks {
		if network.Router == nil {
			continue
		}
		// Subnets are already validated.
		var subnet1, subnet2 *net.IPNet
		if network.IsDualStack() {
			_, subnet1, _ = net.ParseCIDR(network.DualStack.IPv4.Subnet)
			_, subnet2, _ = net.ParseCIDR(network.DualStack.IPv6.Subnet)
		} else {
			_, subnet1, _ = net.ParseCIDR(network.Subnet)
		}
		for _, route := range network.Router.RoutesTowardsEVE {
			if _, _, err = net.ParseCIDR(route.DstNetwork); err != nil {
				err = fmt.Errorf("network %s route %+v has invalid destination: %w",
					network.LogicalLabel, route, err)
				return
			}
			gwIP := net.ParseIP(route.Gateway)
			if gwIP == nil {
				err = fmt.Errorf("network %s route %+v has invalid gateway IP (%s)",
					network.LogicalLabel, route, route.Gateway)
				return
			}
			routable := (subnet1 != nil && subnet1.Contains(gwIP)) ||
				(subnet2 != nil && subnet2.Contains(gwIP))
			if !routable {
				err = fmt.Errorf("network %s route %+v has gateway IP (%s) "+
					"which is not from within the network subnet(s)",
					network.LogicalLabel, route, route.Gateway)
				return
			}
		}
	}

	// Validate MTU settings.
	for _, network := range netModel.Networks {
		if network.MTU != 0 && network.MTU < minMTU {
			err = fmt.Errorf("MTU %d configured for network %s is too small",
				network.MTU, network.LogicalLabel)
			return
		}
		if network.MTU > maxMTU {
			err = fmt.Errorf("MTU %d configured for network %s is too large",
				network.MTU, network.LogicalLabel)
			return
		}
	}
	return nil
}

func (a *agent) validateNetworkIPConfig(netLabel string, netIPConf api.NetworkIPConfig,
	shouldBeIPv4, shouldBeIPv6 bool) error {
	// Validate network Subnet and gateway IP.
	_, subnet, err := net.ParseCIDR(netIPConf.Subnet)
	if err != nil {
		return fmt.Errorf("network %s has invalid subnet: %w", netLabel, err)
	}
	if shouldBeIPv4 && subnet.IP.To4() == nil {
		return fmt.Errorf("expected IPv4 subnet for network %s, got: %v",
			netLabel, subnet)
	}
	if shouldBeIPv6 && subnet.IP.To4() != nil {
		return fmt.Errorf("expected IPv6 subnet for network %s, got: %v",
			netLabel, subnet)
	}
	// Make sure that remaining fields have the same IP version as Subnet.
	shouldBeIPv4 = subnet.IP.To4() != nil
	shouldBeIPv6 = subnet.IP.To4() == nil
	gwIP := net.ParseIP(netIPConf.GwIP)
	if gwIP == nil {
		return fmt.Errorf("network %s has invalid gateway IP (%s)",
			netLabel, netIPConf.GwIP)
	}
	// This also checks that gwIP has the correct IP version.
	if !subnet.Contains(gwIP) {
		return fmt.Errorf(
			"network %s has gateway IP (%s) which is not inside the subnet (%s)",
			netLabel, netIPConf.GwIP, netIPConf.Subnet)
	}

	// Validate DHCP config.
	dhcp := netIPConf.DHCP
	if !dhcp.Enable {
		return nil
	}
	if dhcp.IPRange.FromIP != "" {
		fromIP := net.ParseIP(dhcp.IPRange.FromIP)
		if fromIP == nil {
			return fmt.Errorf("network %s has invalid DHCP range FromIP (%s)",
				netLabel, dhcp.IPRange.FromIP)
		}
		toIP := net.ParseIP(dhcp.IPRange.ToIP)
		if toIP == nil {
			return fmt.Errorf("network %s has invalid DHCP range ToIP (%s)",
				netLabel, dhcp.IPRange.ToIP)
		}
		// This also checks that fromIP and toIP have the correct IP version.
		if !subnet.Contains(fromIP) || !subnet.Contains(toIP) {
			return fmt.Errorf("network %s has DHCP IP range outside of the subnet",
				netLabel)
		}
		if bytes.Compare(fromIP, toIP) > 0 {
			return fmt.Errorf("network %s has DHCP IP range where FromIP > ToIP",
				netLabel)
		}
	}
	for _, dns := range dhcp.PublicDNS {
		dnsIP := net.ParseIP(dns)
		if dnsIP == nil {
			return fmt.Errorf("network %s has invalid public DNS server IP (%s)",
				netLabel, dns)
		}
		if shouldBeIPv4 && dnsIP.To4() == nil {
			return fmt.Errorf("expected IPv4 DNS server address for network %s, got: %v",
				netLabel, dnsIP)
		}
		if shouldBeIPv6 && dnsIP.To4() != nil {
			return fmt.Errorf("expected IPv6 DNS server address for network %s, got: %v",
				netLabel, dnsIP)
		}
	}
	if dhcp.PrivateNTP != "" && dhcp.PublicNTP != "" {
		return fmt.Errorf("network %s has both public and private NTP configured",
			netLabel)
	}
	for _, entry := range dhcp.StaticEntries {
		if _, err = net.ParseMAC(entry.MAC); err != nil {
			return fmt.Errorf("network %s has static DHCP entry with invalid MAC (%s)",
				netLabel, entry.MAC)
		}
		ip := net.ParseIP(entry.IP)
		if ip == nil {
			return fmt.Errorf("network %s has static DHCP entry with invalid IP (%s)",
				netLabel, entry.IP)
		}
		if shouldBeIPv4 && ip.To4() == nil {
			return fmt.Errorf("expected IPv4 static DHCP entry for network %s, got: %v",
				netLabel, ip)
		}
		if shouldBeIPv6 && ip.To4() != nil {
			return fmt.Errorf("expected IPv6 static DHCP entry for network %s, got: %v",
				netLabel, ip)
		}
	}
	if shouldBeIPv6 && dhcp.WPAD != "" {
		return fmt.Errorf(
			"network %s configured with WPAD URL (%s) which is not supported for IPv6",
			netLabel, dhcp.WPAD)
	}
	return nil
}

func (a *agent) validateEndpoints(netModel *parsedNetModel) (err error) {
	//nolint:godox
	// TODO: NetbootArtifacts
	for _, client := range netModel.Endpoints.Clients {
		if err = a.validateEndpoint(client.Endpoint); err != nil {
			return
		}
	}
	for _, dnsSrv := range netModel.Endpoints.DNSServers {
		if err = a.validateEndpoint(dnsSrv.Endpoint); err != nil {
			return
		}
		for _, upstreamSrv := range dnsSrv.UpstreamServers {
			if ip := net.ParseIP(upstreamSrv); ip == nil {
				err = fmt.Errorf("DNS server %s has invalid upstream server IP (%s)",
					dnsSrv.LogicalLabel, upstreamSrv)
				return
			}
		}
		for _, entry := range dnsSrv.StaticEntries {
			if entry.FQDN == "" {
				err = fmt.Errorf("DNS server %s has static entry with empty FQDN",
					dnsSrv.LogicalLabel)
				return
			}
			if strings.HasPrefix(entry.IP, api.EndpointIPRefPrefix) ||
				strings.HasPrefix(entry.IP, api.EndpointIPv4RefPrefix) ||
				strings.HasPrefix(entry.IP, api.EndpointIPv6RefPrefix) ||
				entry.IP == api.AdamIPRef ||
				entry.IP == api.AdamIPv4Ref ||
				entry.IP == api.AdamIPv6Ref {
				// Do not try to parse IP, it is a symbolic reference.
				continue
			}
			if ip := net.ParseIP(entry.IP); ip == nil {
				err = fmt.Errorf("DNS server %s has invalid static entry IP (%s)",
					dnsSrv.LogicalLabel, entry.IP)
				return
			}

		}
	}
	for _, proxy := range netModel.Endpoints.ExplicitProxies {
		if err = a.validateEndpoint(proxy.Endpoint); err != nil {
			return
		}
		for _, dns := range proxy.PublicDNS {
			if dnsIP := net.ParseIP(dns); dnsIP == nil {
				err = fmt.Errorf("proxy %s has invalid public DNS server IP (%s)",
					proxy.LogicalLabel, dns)
				return
			}
		}
		if proxy.HTTPProxy.Port == 0 && proxy.HTTPSProxy.Port == 0 {
			err = fmt.Errorf("Proxy %s without port numbers",
				proxy.LogicalLabel)
			return
		}
		if proxy.HTTPProxy.Port != 0 && proxy.HTTPSProxy.Port != 0 {
			if proxy.HTTPProxy.Port == proxy.HTTPSProxy.Port {
				err = fmt.Errorf("proxy %s with colliding ports",
					proxy.LogicalLabel)
				return
			}
		}
		for _, user := range proxy.Users {
			if user.Username == "" {
				err = fmt.Errorf("Proxy %s with empty username",
					proxy.LogicalLabel)
				return
			}
		}
		if proxy.CACertPEM != "" {
			if err = a.validateCertPEM(proxy.CACertPEM, proxy.CAKeyPEM, true); err != nil {
				return
			}
		}
		ruleHosts := make(map[string]struct{})
		for _, rule := range proxy.ProxyRules {
			if _, duplicate := ruleHosts[rule.ReqHost]; duplicate {
				err = fmt.Errorf("proxy %s has duplicate rules", proxy.LogicalLabel)
				return
			}
			ruleHosts[rule.ReqHost] = struct{}{}
		}
	}
	for _, proxy := range netModel.Endpoints.TransparentProxies {
		if err = a.validateEndpoint(proxy.Endpoint); err != nil {
			return
		}
		for _, dns := range proxy.PublicDNS {
			if dnsIP := net.ParseIP(dns); dnsIP == nil {
				err = fmt.Errorf("proxy %s has invalid public DNS server IP (%s)",
					proxy.LogicalLabel, dns)
				return
			}
		}
		if proxy.CACertPEM != "" {
			if err = a.validateCertPEM(proxy.CACertPEM, proxy.CAKeyPEM, true); err != nil {
				return
			}
		}
		ruleHosts := make(map[string]struct{})
		for _, rule := range proxy.ProxyRules {
			if _, duplicate := ruleHosts[rule.ReqHost]; duplicate {
				err = fmt.Errorf("proxy %s has duplicate rules", proxy.LogicalLabel)
				return
			}
			ruleHosts[rule.ReqHost] = struct{}{}
		}
	}
	for _, httpSrv := range netModel.Endpoints.HTTPServers {
		if err = a.validateEndpoint(httpSrv.Endpoint); err != nil {
			return
		}
		if httpSrv.HTTPPort == 0 && httpSrv.HTTPSPort == 0 {
			err = fmt.Errorf("HTTP server %s without port numbers",
				httpSrv.LogicalLabel)
			return
		}
		if httpSrv.HTTPPort != 0 && httpSrv.HTTPSPort != 0 {
			if httpSrv.HTTPPort == httpSrv.HTTPSPort {
				err = fmt.Errorf("HTTP server %s with colliding ports",
					httpSrv.LogicalLabel)
				return
			}
		}
		if httpSrv.CertPEM != "" {
			if err = a.validateCertPEM(httpSrv.CertPEM, httpSrv.KeyPEM, false); err != nil {
				return
			}
		} else if httpSrv.HTTPSPort != 0 {
			err = fmt.Errorf("HTTPS server %s without certificate",
				httpSrv.LogicalLabel)
			return
		}
	}
	for _, netbootSrv := range netModel.Endpoints.NetbootServers {
		if err = a.validateEndpoint(netbootSrv.Endpoint); err != nil {
			return
		}
	}
	for _, ntpSrv := range netModel.Endpoints.NTPServers {
		if err = a.validateEndpoint(ntpSrv.Endpoint); err != nil {
			return
		}
	}
	return nil
}

func (a *agent) validateEndpoint(endpoint api.Endpoint) (err error) {
	if endpoint.IsDualStack() {
		err = a.validateEndpointIPConfig(endpoint.LogicalLabel, endpoint.DualStack.IPv4,
			true, false)
		if err != nil {
			return
		}
		err = a.validateEndpointIPConfig(endpoint.LogicalLabel, endpoint.DualStack.IPv6,
			false, true)
		if err != nil {
			return
		}
	} else {
		err = a.validateEndpointIPConfig(endpoint.LogicalLabel, endpoint.EndpointIPConfig,
			false, false)
		if err != nil {
			return
		}
	}

	// Validate MTU settings.
	if endpoint.MTU != 0 && endpoint.MTU < minMTU {
		return fmt.Errorf("MTU %d configured for endpoint %s is too small",
			endpoint.MTU, endpoint.LogicalLabel)
	}
	if endpoint.MTU > maxMTU {
		return fmt.Errorf("MTU %d configured for endpoint %s is too large",
			endpoint.MTU, endpoint.LogicalLabel)
	}
	return nil
}

func (a *agent) validateEndpointIPConfig(epLabel string, epIPConf api.EndpointIPConfig,
	shouldBeIPv4, shouldBeIPv6 bool) error {
	if epIPConf.Subnet == "" {
		// L2-only endpoint.
		return nil
	}
	_, subnet, err := net.ParseCIDR(epIPConf.Subnet)
	if err != nil {
		return fmt.Errorf("endpoint %s with invalid subnet '%s': %w",
			epLabel, epIPConf.Subnet, err)
	}
	if shouldBeIPv4 && subnet.IP.To4() == nil {
		return fmt.Errorf("expected IPv4 subnet for endpoint %s, got: %v",
			epLabel, subnet)
	}
	if shouldBeIPv6 && subnet.IP.To4() != nil {
		return fmt.Errorf("expected IPv6 subnet for endpoint %s, got: %v",
			epLabel, subnet)
	}
	ones, bits := subnet.Mask.Size()
	if bits-ones < 2 {
		return fmt.Errorf("endpoint %s uses subnet with less than 2 host IPs (%s)",
			epLabel, epIPConf.Subnet)
	}
	// Validate IP address.
	ip := net.ParseIP(epIPConf.IP)
	if ip == nil {
		return fmt.Errorf("endpoint %s with invalid IP address (%s)",
			epLabel, epIPConf.IP)
	}
	// This also checks that endpoint IP has the correct IP version.
	if !subnet.Contains(ip) {
		return fmt.Errorf("endpoint %s has IP (%s) address outside of the configured "+
			"subnet (%s)", epLabel, epIPConf.IP, epIPConf.Subnet)
	}
	return nil
}

func (a *agent) validateFirewall(netModel *parsedNetModel) (err error) {
	for _, rule := range netModel.Firewall.Rules {
		if rule.SrcSubnet != "" {
			if _, _, err = net.ParseCIDR(rule.SrcSubnet); err != nil {
				err = fmt.Errorf("firewall rule with invalid subnet '%s': %w",
					rule.SrcSubnet, err)
				return
			}
		}
		if rule.DstSubnet != "" {
			if _, _, err = net.ParseCIDR(rule.DstSubnet); err != nil {
				err = fmt.Errorf("firewall rule with invalid subnet '%s': %w",
					rule.DstSubnet, err)
				return
			}
		}
		if len(rule.Ports) > 0 {
			if rule.Protocol != api.TCP && rule.Protocol != api.UDP {
				err = fmt.Errorf("firewall rule with non-empty set of ports (%v) "+
					" but protocol is neither TCP nor UDP (%v)", rule.Ports, rule.Protocol)
				return
			}
		}
	}
	return nil
}

func (a *agent) validateHostConfig(netModel *parsedNetModel) (err error) {
	// Eden SDN requires at least one routable host IP address.
	if netModel.Host == nil {
		err = errors.New("missing host configuration")
		return
	}
	var routableHostIP bool
	for _, hostIP := range netModel.Host.HostIPs {
		ip := net.ParseIP(hostIP)
		if ip == nil {
			err = fmt.Errorf("failed to parse host IP address %s", hostIP)
			return
		}
		if ip.IsGlobalUnicast() {
			routableHostIP = true
			if ip.To4() != nil {
				netModel.hostIPv4 = ip.To4()
			} else {
				netModel.hostIPv6 = ip.To16()
			}
		}
	}
	if !routableHostIP {
		err = errors.New("eden SDN requires at least one routable host IP address")
		return
	}
	if netModel.Host.ControllerPort == 0 {
		err = errors.New("missing controller port")
		return
	}
	return nil
}

func (a *agent) validateCertPEM(certPem, keyPem string, isCA bool) error {
	// Check that certificate can be parsed.
	block, _ := pem.Decode([]byte(certPem))
	if block == nil {
		return errors.New("failed to decode PEM certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse PEM certificate: %v", err)
	}
	if isCA != cert.IsCA {
		return fmt.Errorf("invalid certificate purpose (IsCA=%t)", cert.IsCA)
	}
	// Check that private key can be parsed.
	block, _ = pem.Decode([]byte(keyPem))
	if block == nil {
		return errors.New("failed to decode PEM private key")
	}
	privateKey, err := a.parsePrivateKey(block.Bytes)
	if err != nil {
		return err
	}
	// Check that the public key and the private key correspond with each other.
	switch pub := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		rsaKey, ok := privateKey.(*rsa.PrivateKey)
		if !ok {
			return errors.New("private key type does not match public key type")
		}
		if pub.N.Cmp(rsaKey.N) != 0 {
			return errors.New("private key does not match public key")
		}
	case *ecdsa.PublicKey:
		ecdsaKey, ok := privateKey.(*ecdsa.PrivateKey)
		if !ok {
			return errors.New("private key type does not match public key type")
		}
		if pub.X.Cmp(ecdsaKey.X) != 0 || pub.Y.Cmp(ecdsaKey.Y) != 0 {
			return errors.New("private key does not match public key")
		}
	case ed25519.PublicKey:
		ed25519Key, ok := privateKey.(ed25519.PrivateKey)
		if !ok {
			return errors.New("private key type does not match public key type")
		}
		if !bytes.Equal(ed25519Key.Public().(ed25519.PublicKey), pub) {
			return errors.New("private key does not match public key")
		}
	default:
		return errors.New("unknown public key algorithm")
	}
	return nil
}

// Attempt to parse the given private key DER block. OpenSSL 0.9.8 generates
// PKCS #1 private keys by default, while OpenSSL 1.0.0 generates PKCS #8 keys.
// OpenSSL ecparam generates SEC1 EC private keys for ECDSA. We try all three.
func (a *agent) parsePrivateKey(der []byte) (crypto.PrivateKey, error) {
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey, *ecdsa.PrivateKey, ed25519.PrivateKey:
			return key, nil
		default:
			return nil, errors.New("found unknown private key type in PKCS#8 wrapping")
		}
	}
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	}

	return nil, errors.New("failed to parse private key")
}

func (a *agent) parseLabeledItems(items []api.LabeledItem) (labeledItems, error) {
	parsedItems := make(labeledItems)
	for _, item := range items {
		id := itemID{
			typename:     item.ItemType(),
			logicalLabel: item.ItemLogicalLabel(),
		}
		if _, duplicate := parsedItems[id]; duplicate {
			return nil, fmt.Errorf("duplicate logical label: %s/%s",
				id.typename, id.logicalLabel)
		}
		var category string
		if categItem, withCategory := item.(api.LabeledItemWithCategory); withCategory {
			category = categItem.ItemCategory()
		}
		parsedItems[id] = &labeledItem{
			LabeledItem:  item,
			category:     category,
			referencedBy: make(map[string]itemID),
		}
	}
	for _, item := range items {
		id := itemID{
			typename:     item.ItemType(),
			logicalLabel: item.ItemLogicalLabel(),
		}
		for _, ref := range item.ReferencesFromItem() {
			refID := itemID{
				typename:     ref.ItemType,
				logicalLabel: ref.ItemLogicalLabel,
			}
			refItem, exists := parsedItems[refID]
			if !exists {
				return nil, fmt.Errorf("referenced item %s/%s does not exist "+
					"(ref-key: %s)", refID.typename, refID.logicalLabel, ref.RefKey)
			}
			if ref.ItemCategory != "" {
				if refItem.category != ref.ItemCategory {
					return nil, fmt.Errorf("category mismatch for referenced item %s/%s "+
						"(expected %s, has %s)", refID.typename, refID.logicalLabel,
						ref.ItemCategory, refItem.category)
				}
			}
			_, collision := refItem.referencedBy[ref.RefKey]
			if collision {
				return nil, fmt.Errorf("colliding referencing to logical label: %s/%s "+
					"(ref-key: %s)", refID.typename, refID.logicalLabel, ref.RefKey)
			}
			refItem.referencedBy[ref.RefKey] = id
			parsedItems[id].referencing = append(parsedItems[id].referencing, itemRef{
				itemID: refID,
				refKey: ref.RefKey,
			})
		}
	}
	return parsedItems, nil
}

func (a *agent) slicesToLabeledItems(slices ...interface{}) (items []api.LabeledItem) {
	for _, slice := range slices {
		rv := reflect.ValueOf(slice)
		for i := 0; i < rv.Len(); i++ {
			item := rv.Index(i)
			if labeledItem, ok := item.Interface().(api.LabeledItem); ok {
				items = append(items, labeledItem)
			} else {
				log.Warnf("Not an instance of labeled item: %+v", item)
			}
		}
	}
	return items
}
