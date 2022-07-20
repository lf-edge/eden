package main

import (
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
	items labeledItems
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
		if strings.HasPrefix(mac.String(), hostPortMACPrefix) {
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
		}
	}
	if !routableHostIP {
		err = errors.New("eden SDN requires at least one routable host IP address")
		return
	}

	// TODO
	//  - Network Subnet, GwIP parsable
	//  - Do not mix VLAN and non-VLAN network with the same bridge
	//  - valid IPRange (parsable, left <= right)
	//  - PublicDNS - valid IP addresses
	//  - Parsable PEM (CACertPEM, CAKeyPEM)
	//  - Endpoint - parsable Subnet, IP, IP is inside Subnet
	//  - DNSServer.UpstreamServers - parsable IPs
	//  - DNSEntry.IP - parsable, FQDN not empty
	//  - DHCP : do not configure private and public NTP at the same time
	//  - HTTPServer - non-zero HTTP, HTTPS port should be different, valid PEMs
	//  - ExplicitProxy - non-zero HTTP, HTTPS port should be different
	//  - FwRule - Src and Dst Subnets parsable
	//  - NetbootArtifacts - exactly one is entrypoint, Filename and URL are non-empty
	//  - check that within a network it is IPv4 or IPv6, not both (for now)
	return
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
