package main

import (
	"fmt"
	"net"
	"reflect"
	"strings"

	"github.com/lf-edge/eden/sdn/api"
)

func (a *agent) validateNetModel(netModel api.NetworkModel) error {
	// TODO
	//  - check references to logical labels (sometimes cannot overlap, etc.)
	//  - unique VLAN IDs
	//  - Network Subnet, GwIP parsable
	//  - valid IPRange (parsable, left <= right)
	//  - PublicDNS - valid IP addresses
	//  - Parsable PEM (CACertPEM, CAKeyPEM)
	//  - Endpoint - parsable Subnet, IP, IP is inside Subnet
	//  - MTU is no more than 9000
	//  - DNSServer.UpstreamServers - parsable IPs
	//  - DNSEntry.IP - parsable, FQDN not empty
	//  - HTTPServer - non-zero HTTP, HTTPS port should be different, valid PEMs
	//  - ExplicitProxy - non-zero HTTP, HTTPS port should be different
	//  - FwRule - Src and Dst Subnets parsable
	//  - NetbootArtifacts - exactly one is entrypoint, Filename and URL are non-empty
	//  - check that within a network it is IPv4 or IPv6, not both

	// Logical label is mandatory and within each type should be unique
	if _, err := a.collectLogicalLabels("port", netModel.Ports); err != nil {
		return err
	}
	if _, err := a.collectLogicalLabels("bond", netModel.Bonds); err != nil {
		return err
	}
	if _, err := a.collectLogicalLabels("bridge", netModel.Bridges); err != nil {
		return err
	}
	if _, err := a.collectLogicalLabels("network", netModel.Networks); err != nil {
		return err
	}
	eps := netModel.Endpoints
	if _, err := a.collectLogicalLabels("DNS server", eps.DNSServers); err != nil {
		return err
	}
	if _, err := a.collectLogicalLabels("client endpoint", eps.Clients); err != nil {
		return err
	}
	if _, err := a.collectLogicalLabels("explicit proxy", eps.ExplicitProxies); err != nil {
		return err
	}
	if _, err := a.collectLogicalLabels("HTTP server", eps.HTTPServers); err != nil {
		return err
	}
	if _, err := a.collectLogicalLabels("netboot server", eps.NetbootServers); err != nil {
		return err
	}
	if _, err := a.collectLogicalLabels("NTP server", eps.NTPServers); err != nil {
		return err
	}

	// Every port should have valid MAC address
	for _, port := range netModel.Ports {
		mac, err := net.ParseMAC(port.MAC)
		if err != nil {
			return fmt.Errorf("port %s has invalid MAC address: %v", port.LogicalLabel, err)
		}
		if strings.HasPrefix(mac.String(), hostPortMACPrefix) {
			return fmt.Errorf("port %s has MAC address with prefix reserved for the host port",
				port.LogicalLabel)
		}
		if port.EVEConnect != nil {
			if _, err := net.ParseMAC(port.EVEConnect.MAC); err != nil {
				return fmt.Errorf("EVE-side of port %s has invalid MAC address: %v",
					port.LogicalLabel, err)
			}
		}
	}
	return nil
}

func (a *agent) collectLogicalLabels(typeName string, slice interface{}) ([]string, error) {
	var labels []string
	rv := reflect.ValueOf(slice)
	for i := 0; i < rv.Len(); i++ {
		item := rv.Index(i)
		label := item.FieldByName("LogicalLabel").String()
		if label == "" {
			return nil, fmt.Errorf("%s at index %d has empty logical label", typeName, i)
		}
		for _, prevLabel := range labels {
			if label == prevLabel {
				return nil, fmt.Errorf("%s at index %d with duplicate logical label: %s",
					typeName, i, label)
			}
		}
		labels = append(labels, label)
	}
	return labels, nil
}
