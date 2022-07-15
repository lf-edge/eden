package configitems

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/lf-edge/eve/libs/depgraph"
	log "github.com/sirupsen/logrus"
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
	err := os.MkdirAll(namedNsDir, 0755)
	if err != nil {
		err = fmt.Errorf("failed to create directory %s: %w", namedNsDir, err)
		log.Error(err)
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
