package main

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"

	"github.com/lf-edge/eden/sdn/api"
	log "github.com/sirupsen/logrus"
)

const maxMTU = 9000

type parsedNetModel struct {
	api.NetworkModel
	items  labeledItems
	hostIP net.IP
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
		eps.HTTPServers, eps.ExplicitProxies, eps.Clients)
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
		if port.EVEConnect != nil {
			if _, err = net.ParseMAC(port.EVEConnect.MAC); err != nil {
				err = fmt.Errorf("EVE-side of port %s has invalid MAC address: %v",
					port.LogicalLabel, err)
				return
			}
		}
	}

	// MTU should be no more than 9000.
	for _, port := range netModel.Ports {
		if port.MTU > maxMTU {
			err = fmt.Errorf("MTU %d configured for port %s is too large",
				port.MTU, port.LogicalLabel)
			return
		}
	}
	return nil
}

func (a *agent) validateNetworks(netModel *parsedNetModel) (err error) {
	// Validate network Subnet, gateway IP and VLANs.
	for _, network := range netModel.Networks {
		if _, _, err = net.ParseCIDR(network.Subnet); err != nil {
			err = fmt.Errorf("network %s has invalid subnet: %w",
				network.LogicalLabel, err)
			return
		}
		if gwIP := net.ParseIP(network.GwIP); gwIP == nil {
			err = fmt.Errorf("network %s has invalid gateway IP (%s)",
				network.LogicalLabel, network.GwIP)
			return
		}
	}

	// Validate DHCP config.
	for _, network := range netModel.Networks {
		_, subnet, _ := net.ParseCIDR(network.Subnet)
		dhcp := network.DHCP
		if !dhcp.Enable {
			continue
		}
		if dhcp.IPRange.FromIP != "" {
			fromIP := net.ParseIP(dhcp.IPRange.FromIP)
			if fromIP == nil {
				err = fmt.Errorf("network %s has invalid DHCP range FromIP (%s)",
					network.LogicalLabel, dhcp.IPRange.FromIP)
				return
			}
			toIP := net.ParseIP(dhcp.IPRange.ToIP)
			if toIP == nil {
				err = fmt.Errorf("network %s has invalid DHCP range ToIP (%s)",
					network.LogicalLabel, dhcp.IPRange.ToIP)
				return
			}
			if !subnet.Contains(fromIP) || !subnet.Contains(toIP) {
				err = fmt.Errorf("network %s has DHCP IP range outside of the subnet",
					network.LogicalLabel)
				return
			}
			if bytes.Compare(fromIP, toIP) > 0 {
				err = fmt.Errorf("network %s has DHCP IP range where FromIP > ToIP",
					network.LogicalLabel)
				return
			}
		}
		for _, dns := range dhcp.PublicDNS {
			if dnsIP := net.ParseIP(dns); dnsIP == nil {
				err = fmt.Errorf("network %s has invalid public DNS server IP (%s)",
					network.LogicalLabel, dns)
				return
			}
		}
		if dhcp.PrivateNTP != "" && dhcp.PublicNTP != "" {
			err = fmt.Errorf("network %s has both public and private NTP configured",
				network.LogicalLabel)
			return
		}
	}

	// Do not mix VLAN and non-VLAN network with the same bridge
	for _, bridge := range a.netModel.Bridges {
		var netWithVlan, netWithoutVlan bool
		labeledItem := a.netModel.items.getItem(api.Bridge{}.ItemType(), bridge.LogicalLabel)
		for refKey, refBy := range labeledItem.referencedBy {
			if !strings.HasPrefix(refKey, api.NetworkBridgeRefPrefix) {
				continue
			}
			network := a.netModel.items[refBy].LabeledItem
			vlanID := network.(api.Network).VlanID
			if (vlanID == 0 && netWithVlan) || (vlanID != 0 && netWithoutVlan) {
				err = fmt.Errorf("bridge %s with both VLAN and non-VLAN networks",
					bridge.LogicalLabel)
				return
			}
			if vlanID == 0 {
				netWithoutVlan = true
			} else {
				netWithVlan = true
			}
		}
	}

	// Validate transparent proxy.
	for _, network := range netModel.Networks {
		if network.TransparentProxy != nil {
			proxy := network.TransparentProxy
			if err = a.validateCertPEM(proxy.CACertPEM, proxy.CAKeyPEM); err != nil {
				return
			}
			ruleHosts := make(map[string]struct{})
			for _, rule := range proxy.ProxyRules {
				if _, duplicate := ruleHosts[rule.ReqHost]; duplicate {
					err = fmt.Errorf("network %s with transparent proxy "+
						"which has duplicate rules", network.LogicalLabel)
					return
				}
				ruleHosts[rule.ReqHost] = struct{}{}
			}
		}
	}

	// TODO: check that within a network it is IPv4 or IPv6, not both (for now)
	return nil
}

func (a *agent) validateEndpoints(netModel *parsedNetModel) (err error) {
	// TODO
	//  - ExplicitProxy:
	//      - non-zero HTTP, HTTPS port should be different
	//      - parsable private DNS IPs
	//      - non-empty username, password
	//  - NetbootArtifacts:
	//      - exactly one is entrypoint, Filename and URL are non-empty
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
				strings.HasPrefix(entry.IP, api.AdamIPRef) {
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
			if err = a.validateCertPEM(httpSrv.CertPEM, httpSrv.KeyPEM); err != nil {
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
	// Validate Subnet.
	_, subnet, err := net.ParseCIDR(endpoint.Subnet)
	if err != nil {
		err = fmt.Errorf("endpoint %s with invalid subnet '%s': %w",
			endpoint.LogicalLabel, endpoint.Subnet, err)
		return
	}
	ones, bits := subnet.Mask.Size()
	if bits-ones < 2 {
		err = fmt.Errorf("endpoint %s uses subnet with less than 2 host IPs (%s)",
			endpoint.LogicalLabel, endpoint.Subnet)
		return
	}
	// Validate IP address.
	ip := net.ParseIP(endpoint.IP)
	if ip == nil {
		err = fmt.Errorf("endpoint %s with invalid IP address (%s)",
			endpoint.LogicalLabel, endpoint.IP)
		return
	}
	if !subnet.Contains(ip) {
		err = fmt.Errorf("endpoint %s has IP (%s) address outside of the configured "+
			"subnet (%s)", endpoint.LogicalLabel, endpoint.IP, endpoint.Subnet)
		return
	}
	// MTU should be no more than 9000.
	if endpoint.MTU > maxMTU {
		err = fmt.Errorf("MTU %d configured for endpoint %s is too large",
			endpoint.MTU, endpoint.LogicalLabel)
		return
	}
	return nil
}

func (a *agent) validateFirewall(netModel *parsedNetModel) (err error) {
	for _, rule := range netModel.Firewall.Rules {
		if _, _, err = net.ParseCIDR(rule.SrcSubnet); err != nil {
			err = fmt.Errorf("firewall rule with invalid subnet '%s': %w",
				rule.SrcSubnet, err)
			return
		}
		if _, _, err = net.ParseCIDR(rule.DstSubnet); err != nil {
			err = fmt.Errorf("firewall rule with invalid subnet '%s': %w",
				rule.SrcSubnet, err)
			return
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
			netModel.hostIP = ip
			break
		}
	}
	if !routableHostIP {
		err = errors.New("eden SDN requires at least one routable host IP address")
		return
	}
	return nil
}

func (a *agent) validateCertPEM(certPem, keyPem string) error {
	// TODO : should be parsable and correspond with each other
	return nil
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
