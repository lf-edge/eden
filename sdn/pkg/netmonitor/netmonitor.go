package netmonitor

import (
	"bytes"
	"context"
	"net"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

const (
	netlinkSubBufSize = 128 * 1024 // bytes
	eventChanBufSize = 64 // number of events
)

// NetworkMonitor currently allows to lookup network interfaces and obtain their attributes,
// watch for interface events and on top of that provides a layer of caching to minimize
// the number or netlink calls, while at the same time preventing from reading stale data.
// For now the monitor is limited to the network namespace of the caller, i.e. cannot
// monitor other network namespaces.
// NetworkMonitor is thread-safe.
type NetworkMonitor struct {
	sync.Mutex
	eventSubs   []subscriber
	staleCache  bool
	netIfsCache map[int]NetIf // key: interface index
}

// Event received from the network stack.
type Event interface {
	isNetworkEvent()
}

// NetIfChange : network interface (dis)appeared or attributes have changed.
type NetIfChange struct {
	// Attrs : interface attributes.
	Attrs NetIfAttrs
	// True if this is a newly added interface.
	Added bool
	// True if interface was removed.
	Deleted bool
}

func (e NetIfChange) isNetworkEvent() {}

// Equal allows to compare two IfChange events for equality.
func (e NetIfChange) Equal(e2 NetIfChange) bool {
	return e.Added == e2.Added &&
		e.Deleted == e2.Deleted &&
		e.Attrs.Equal(e2.Attrs)
}

// AddrChange : IP address was (un)assigned from/to interface.
type AddrChange struct {
	IfIndex   int
	IfAddress *net.IPNet
	Deleted   bool
}

func (e AddrChange) isNetworkEvent() {}

// NetIf : network interface.
type NetIf struct {
	NetIfAttrs
	// IPs : IP addresses assigned to the interface.
	IPs []*net.IPNet
}

// NetIfAttrs : network interface attributes.
type NetIfAttrs struct {
	// Index of the interface
	IfIndex int
	// Name of the interface.
	IfName string
	// IfType should be one of the link types as defined in ip-link(8).
	IfType string
	// MAC : MAC address of the interface.
	MAC net.HardwareAddr
	// True if interface is administratively enabled.
	AdminUp bool
	// True if interface is ready to transmit data at the L1 layer.
	LowerUp bool
	// True if interface is a slave of another interface (e.g. a sub-interface).
	Enslaved bool
	// If interface is enslaved, this should contain index of the master interface.
	MasterIfIndex int
}

// Equal allows to compare two sets of interface attributes for equality.
func (a NetIfAttrs) Equal(a2 NetIfAttrs) bool {
	return a.IfIndex == a2.IfIndex &&
		a.IfName == a2.IfName &&
		a.IfType == a2.IfType &&
		bytes.Equal(a.MAC, a2.MAC) &&
		a.AdminUp == a2.AdminUp &&
		a.LowerUp == a2.LowerUp &&
		a.Enslaved == a2.Enslaved &&
		a.MasterIfIndex == a2.MasterIfIndex
}

type subscriber struct {
	name   string
	events chan Event
	done   <-chan struct{}
}

// Init should be called first to prepare the monitor.
func (m *NetworkMonitor) Init() {
	go m.watcher()
	m.Lock()
	m.syncCache()
	m.Unlock()
}

// This method is run with the monitor in the locked state.
func (m *NetworkMonitor) syncCache() {
	newCache := make(map[int]NetIf)
	netIfs, err := net.Interfaces()
	if err != nil {
		log.Errorf("Failed to list network interfaces: %v", err)
		return
	}
	for _, netIf := range netIfs {
		ifName := netIf.Name
		link, err := netlink.LinkByName(ifName)
		if err != nil {
			if _, notFound := err.(netlink.LinkNotFoundError); notFound {
				continue
			}
			log.Errorf("Failed to get link for interface %s: %v", ifName, err)
			continue
		}
		attrs := m.ifAttrsFromLink(link)
		var ips []*net.IPNet
		ips4, err := netlink.AddrList(link, netlink.FAMILY_V4)
		if err != nil {
			log.Errorf("Failed to get IPv4 addresses for interface %s: %v", ifName, err)
			continue
		}
		ips6, err := netlink.AddrList(link, netlink.FAMILY_V6)
		if err != nil {
			log.Errorf("Failed to get IPv6 addresses for interface %s: %v", ifName, err)
			continue
		}
		for _, ip := range ips4 {
			ips = append(ips, ip.IPNet)
		}
		for _, ip := range ips6 {
			ips = append(ips, ip.IPNet)
		}
		newCache[link.Attrs().Index] = NetIf{
			NetIfAttrs: attrs,
			IPs:        ips,
		}
	}
	m.netIfsCache = newCache
	m.staleCache = false
}

// ListInterfaces returns all available network interfaces.
func (m *NetworkMonitor) ListInterfaces() (netIfs []NetIf) {
	m.Lock()
	defer m.Unlock()
	if m.staleCache {
		m.syncCache()
	}
	for _, netIf := range m.netIfsCache {
		netIfs = append(netIfs, netIf)
	}
	return netIfs
}

// LookupInterfaceByName : lookup interface by the index (ID used by the kernel).
func (m *NetworkMonitor) LookupInterfaceByIndex(ifIndex int) (netIf NetIf, found bool) {
	m.Lock()
	defer m.Unlock()
	if m.staleCache {
		m.syncCache()
	}
	netIf, found = m.netIfsCache[ifIndex]
	return
}

// LookupInterfaceByName : lookup interface by its name in the kernel.
func (m *NetworkMonitor) LookupInterfaceByName(ifName string) (
	netIf NetIf, found bool) {
	m.Lock()
	defer m.Unlock()
	if m.staleCache {
		m.syncCache()
	}
	for _, netIf = range m.netIfsCache {
		if netIf.IfName == ifName {
			return netIf, true
		}
	}
	return NetIf{}, false
}

// LookupInterfaceByMAC : lookup interface by its MAC address.
func (m *NetworkMonitor) LookupInterfaceByMAC(mac net.HardwareAddr) (
	netIf NetIf, found bool) {
	m.Lock()
	defer m.Unlock()
	if m.staleCache {
		m.syncCache()
	}
	for _, netIf = range m.netIfsCache {
		if bytes.Equal(netIf.MAC, mac) {
			return netIf, true
		}
	}
	return NetIf{}, false
}

// WatchEvents allows to subscribe to watch for events from the Linux network stack.
func (m *NetworkMonitor) WatchEvents(ctx context.Context, subName string) <-chan Event {
	m.Lock()
	defer m.Unlock()
	sub := subscriber{
		name:   subName,
		events: make(chan Event, eventChanBufSize),
		done:   ctx.Done(),
	}
	m.eventSubs = append(m.eventSubs, sub)
	return sub.events
}

func (m *NetworkMonitor) watcher() {
	doneChan := make(chan struct{})
	linkChan := m.linkSubscribe(doneChan)
	addrChan := m.addrSubscribe(doneChan)
	// Remember previously published NetIfChange notifications to avoid
	// spurious events.
	lastIfChange := make(map[int]NetIfChange)

	for {
		select {
		case linkUpdate, ok := <-linkChan:
			if !ok {
				log.Warn("Link subscription was closed")
				linkChan = m.linkSubscribe(doneChan)
				m.Lock()
				// We may have lost some notifications, mark the cache as stale.
				m.staleCache = true
				m.Unlock()
				continue
			}
			ifIndex := linkUpdate.Attrs().Index
			attrs := m.ifAttrsFromLink(linkUpdate)
			added := linkUpdate.Header.Type == syscall.RTM_NEWLINK
			deleted := linkUpdate.Header.Type == syscall.RTM_DELLINK
			event := NetIfChange{
				Attrs:   attrs,
				Added:   added,
				Deleted: deleted,
			}
			prevIfChange := lastIfChange[ifIndex]
			if prevIfChange.Equal(event) {
				continue
			}
			lastIfChange[ifIndex] = event
			m.Lock()
			m.staleCache = true
			m.publishEvent(event)
			m.Unlock()
		case addrUpdate, ok := <-addrChan:
			if !ok {
				log.Warn("Address subscription was closed")
				addrChan = m.addrSubscribe(doneChan)
				// We may have lost some notifications, sync the cached data.
				continue
			}
			event := AddrChange{
				IfIndex:   addrUpdate.LinkIndex,
				IfAddress: &addrUpdate.LinkAddress,
				Deleted:   !addrUpdate.NewAddr,
			}
			m.Lock()
			m.syncCache()
			m.publishEvent(event)
			m.Unlock()
		}
	}
}

// This method is run with the monitor in the locked state.
func (m *NetworkMonitor) publishEvent(ev Event) {
	var activeSubs []subscriber
	for _, sub := range m.eventSubs {
		select {
		case <-sub.done:
			// unsubscribe
			continue
		default:
			// continue subscription
		}
		select {
		case sub.events <- ev:
		default:
			log.Warnf("failed to deliver event %+v to subscriber %s",
				ev, sub.name)
		}
		activeSubs = append(activeSubs, sub)
	}
	m.eventSubs = activeSubs
}

func (m *NetworkMonitor) linkSubscribe(doneChan chan struct{}) chan netlink.LinkUpdate {
	linkChan := make(chan netlink.LinkUpdate, eventChanBufSize)
	linkErrFunc := func(err error) {
		log.Errorf("LinkSubscribe failed %s\n", err)
	}
	linkOpts := netlink.LinkSubscribeOptions{
		ErrorCallback: linkErrFunc,
	}
	if err := netlink.LinkSubscribeWithOptions(
		linkChan, doneChan, linkOpts); err != nil {
		log.Fatal(err)
	}
	return linkChan
}

func (m *NetworkMonitor) addrSubscribe(doneChan chan struct{}) chan netlink.AddrUpdate {
	addrChan := make(chan netlink.AddrUpdate, eventChanBufSize)
	addrErrFunc := func(err error) {
		log.Errorf("AddrSubscribe failed %s\n", err)
	}
	addrOpts := netlink.AddrSubscribeOptions{
		ErrorCallback:     addrErrFunc,
		ReceiveBufferSize: netlinkSubBufSize,
	}
	if err := netlink.AddrSubscribeWithOptions(
		addrChan, doneChan, addrOpts); err != nil {
		log.Fatal(err)
	}
	return addrChan
}

func (m *NetworkMonitor) ifAttrsFromLink(link netlink.Link) NetIfAttrs {
	return NetIfAttrs{
		IfIndex:       link.Attrs().Index,
		IfName:        link.Attrs().Name,
		IfType:        link.Type(),
		MAC:           link.Attrs().HardwareAddr,
		AdminUp:       (link.Attrs().Flags & net.FlagUp) != 0,
		LowerUp:       link.Attrs().OperState == netlink.OperUp,
		Enslaved:      link.Attrs().MasterIndex != 0,
		MasterIfIndex: link.Attrs().MasterIndex,
	}
}
