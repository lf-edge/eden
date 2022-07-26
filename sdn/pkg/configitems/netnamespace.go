package configitems

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/lf-edge/eve/libs/depgraph"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const (
	// Symbolic name for the main network namespace (where SDN agent operates).
	MainNsName = "main"

	// Directory with references to named network namespaces.
	namedNsDir = "/var/run/netns"
)

func normNetNsName(name string) string {
	if name == "" {
		name = MainNsName
	}
	return name
}

func namespacedCmd(netNs string, cmd string, args ...string) *exec.Cmd {
	netNs = normNetNsName(netNs)
	if netNs == MainNsName {
		return exec.Command(cmd, args...)
	}
	var newArgs []string
	newArgs = append(newArgs, "netns", "exec", normNetNsName(netNs))
	newArgs = append(newArgs, cmd)
	newArgs = append(newArgs, args...)
	return exec.Command("ip", newArgs...)
}

func moveLinkToNamespace(link netlink.Link, netNs string) (err error) {
	nsHandle, err := netns.GetFromName(netNs)
	if err != nil {
		return err
	}
	if err := netlink.LinkSetNsFd(link, int(nsHandle)); err != nil {
		return err
	}
	return nil
}

func switchToNamespace(netNs string) (revert func(), err error) {
	// Save the current network namespace.
	origNs, err := netns.Get()
	if err != nil {
		return func() {}, err
	}
	closeNs := func(ns netns.NsHandle) {
		if err := ns.Close(); err != nil {
			log.Warnf("closing NsHandle (%v) failed: %v", ns, err)
		}
	}
	// Get network namespace file descriptor.
	nsHandle, err := netns.GetFromName(netNs)
	if err != nil {
		closeNs(origNs)
		return func() {}, err
	}
	defer closeNs(nsHandle)

	// Lock the OS Thread so we don't accidentally switch namespaces later.
	runtime.LockOSThread()

	// Switch the namespace.
	if err := netns.Set(nsHandle); err != nil {
		runtime.UnlockOSThread()
		closeNs(origNs)
		return func() {}, err
	}

	return func() {
		if err := netns.Set(origNs); err != nil {
			log.Errorf("Failed to switch to original Linux network namespace: %v", err)
		}
		closeNs(origNs)
		runtime.UnlockOSThread()
	}, nil
}

// NetNamespace : an item representing named network namespace.
type NetNamespace struct {
	// NsName : name of the network namespace.
	NsName string
}

// Name
func (n NetNamespace) Name() string {
	return normNetNsName(n.NsName)
}

// Label
func (n NetNamespace) Label() string {
	return normNetNsName(n.NsName) + " (net namespace)"
}

// Type
func (n NetNamespace) Type() string {
	return NetNamespaceTypename
}

// Equal is a comparison method for two equally-named NetNamespace instances.
// There are no attributes beyond the name - nothing to compare.
func (n NetNamespace) Equal(depgraph.Item) bool {
	return true
}

// External returns false.
func (n NetNamespace) External() bool {
	return false
}

// String describes the namespace.
func (n NetNamespace) String() string {
	return fmt.Sprintf("Network Namespace \"%s\"", n.NsName)
}

// Dependencies returns nothing.
func (n NetNamespace) Dependencies() (deps []depgraph.Dependency) {
	return nil
}

// mkdir -p /var/run/netns
// ip netns exec <ns-name> ip link set dev lo up

// NetNamespaceConfigurator implements Configurator interface for NetNamespace.
type NetNamespaceConfigurator struct{}

// Create adds network namespace.
func (c *NetNamespaceConfigurator) Create(ctx context.Context, item depgraph.Item) error {
	ns := item.(NetNamespace)
	nsName := normNetNsName(ns.NsName)
	if nsName == MainNsName {
		// Nothing to do, already exists.
		return nil
	}
	if err := ensureDir(namedNsDir); err != nil {
		return err
	}
	out, err := exec.Command("ip", "netns", "add", nsName).CombinedOutput()
	if err != nil {
		errMsg := fmt.Errorf("failed to add net namespace %s: %s", nsName, out)
		log.Error(errMsg)
		return err
	}
	// By default, the loopback interface is down.
	loUpArgs := []string{"link", "set", "dev", "lo", "up"}
	out, err = namespacedCmd(nsName, "ip", loUpArgs...).CombinedOutput()
	if err != nil {
		errMsg := fmt.Errorf("failed to set IPv4 forwarding: %s", out)
		log.Error(errMsg)
		return err
	}
	return nil
}

// Modify is not needed.
func (c *NetNamespaceConfigurator) Modify(ctx context.Context, oldItem, newItem depgraph.Item) (err error) {
	return errors.New("not implemented")
}

// Delete removes network namespace.
func (c *NetNamespaceConfigurator) Delete(ctx context.Context, item depgraph.Item) error {
	ns := item.(NetNamespace)
	nsName := normNetNsName(ns.NsName)
	if nsName == MainNsName {
		// Main network namespace cannot be deleted.
		return errors.New("not supported")
	}
	out, err := exec.Command("ip", "netns", "del", nsName).CombinedOutput()
	if err != nil {
		errMsg := fmt.Errorf("failed to del net namespace %s: %s", nsName, out)
		log.Error(errMsg)
		return err
	}
	return nil
}

// NeedsRecreate is not actually used (no attributes to Modify).
func (c *NetNamespaceConfigurator) NeedsRecreate(oldItem, newItem depgraph.Item) (recreate bool) {
	return false
}

func ensureDir(dirname string) error {
	err := os.MkdirAll(dirname, 0755)
	if err != nil {
		err = fmt.Errorf("failed to create directory %s: %w", dirname, err)
		log.Error(err)
		return err
	}
	return nil
}
