package configitems

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/lf-edge/eden/sdn/pkg/netmonitor"
	"github.com/lf-edge/eve/libs/depgraph"
	"github.com/lf-edge/eve/libs/reconciler"
	log "github.com/sirupsen/logrus"
)

const (
	dhcpcdBinary       = "/sbin/dhcpcd"
	dhcpcdStartTimeout = 3 * time.Second
	dhcpcdStopTimeout  = 30 * time.Second
)

// Dhcpcd : DHCP client (https://wiki.archlinux.org/title/dhcpcd).
// Can be only used with physical network interface (not with virtual interfaces like VETH).
type Dhcpcd struct {
	// PhysIfLL : logical label of the physical interface to associate dhcpcd with.
	PhysIfLL string
	// PhysIfMAC : MAC address of the physical interface to associate dhcpcd with.
	PhysIfMAC net.HardwareAddr
	// LogFile : where to put dhcpcd logs.
	LogFile string
}

// Name
func (c Dhcpcd) Name() string {
	return c.PhysIfMAC.String()
}

// Label
func (c Dhcpcd) Label() string {
	return "dhcpcd for " + c.PhysIfLL
}

// Type
func (c Dhcpcd) Type() string {
	return DhcpcdTypename
}

// Equal is a comparison method for two equally-named Dhcpcd instances.
func (c Dhcpcd) Equal(other depgraph.Item) bool {
	c2 := other.(Dhcpcd)
	return c.PhysIfLL == c2.PhysIfLL &&
		c.LogFile == c2.LogFile
}

// External returns false.
func (c Dhcpcd) External() bool {
	return false
}

// String describes the DHCP client config.
func (c Dhcpcd) String() string {
	return fmt.Sprintf("DHCP Client: %#+v", c)
}

// Dependencies lists the IfHandle as the only dependency of the DHCP client.
func (c Dhcpcd) Dependencies() (deps []depgraph.Dependency) {
	return []depgraph.Dependency{
		{
			RequiredItem: depgraph.ItemRef{
				ItemType: IfHandleTypename,
				ItemName: c.PhysIfMAC.String(),
			},
			MustSatisfy: func(item depgraph.Item) bool {
				ifHandle := item.(IfHandle)
				return ifHandle.Usage == IfUsageL3
			},
			Description: "Physical network interface must exist and be used in the L3 mode",
		},
	}
}

// IfHandleConfigurator implements Configurator interface for Dhcpcd.
type DhcpcdConfigurator struct {
	netMonitor *netmonitor.NetworkMonitor
}

// Create starts dhcpcd.
func (c *DhcpcdConfigurator) Create(ctx context.Context, item depgraph.Item) error {
	config := item.(Dhcpcd)
	mac := config.PhysIfMAC
	netIf, found := c.netMonitor.LookupInterfaceByMAC(mac)
	if !found {
		err := fmt.Errorf("failed to get physical interface with MAC %v", mac)
		log.Error(err)
		return err
	}
	ifName := netIf.IfName
	done := reconciler.ContinueInBackground(ctx)

	go func() {
		if c.dhcpcdExists(ifName) {
			err := fmt.Errorf("dhcpcd for interface %s is already running", ifName)
			log.Error(err)
			done(err)
			return
		}
		// Start DHCP client.
		var args []string
		if config.LogFile != "" {
			args = append(args, "-j", config.LogFile)
		}
		args = append(args, "-t", "0") // wait for release forever
		args = append(args, ifName)
		startTime := time.Now()
		cmd := exec.Command(dhcpcdBinary, args...)
		cmd.Stdout = nil
		cmd.Stderr = nil
		go func() {
			if err := cmd.Run(); err != nil {
				log.Errorf("dhcpcd %v: failed: %s", args, err)
			}
		}()
		// Wait for a bit then give up.
		for !c.dhcpcdExists(ifName) {
			if time.Since(startTime) > dhcpcdStartTimeout {
				err := fmt.Errorf("dhcpcd for interface %s failed to start in time",
					ifName)
				log.Error(err)
				done(err)
				return
			}
			time.Sleep(1 * time.Second)
		}
		log.Debugf("dhcpcd for interface %s is running", ifName)
		done(nil)
		return
	}()
	return nil
}

// Modify is not implemented.
func (c *DhcpcdConfigurator) Modify(ctx context.Context, oldItem, newItem depgraph.Item) (err error) {
	return errors.New("not implemented")
}

// Delete stops dhcpcd.
func (c *DhcpcdConfigurator) Delete(ctx context.Context, item depgraph.Item) error {
	config := item.(Dhcpcd)
	mac := config.PhysIfMAC
	netIf, found := c.netMonitor.LookupInterfaceByMAC(mac)
	if !found {
		err := fmt.Errorf("failed to get physical interface with MAC %v", mac)
		log.Error(err)
		return err
	}
	ifName := netIf.IfName
	done := reconciler.ContinueInBackground(ctx)

	go func() {
		startTime := time.Now()
		// Run release, wait for a bit, then exit and give up.
		failed := false
		for {
			// Release DHCP lease and un-configure the interface.
			// It waits up to 10 seconds.
			// https://github.com/NetworkConfiguration/dhcpcd/blob/dhcpcd-8.1.6/src/dhcpcd.c#L1950-L1957
			_, err := exec.Command(dhcpcdBinary, "--release", ifName).CombinedOutput()
			if err != nil {
				log.Errorf("dhcpcd release failed for interface %s: %v, elapsed time %v",
					ifName, err, time.Since(startTime))
			}
			if !c.dhcpcdExists(ifName) {
				break
			}
			if time.Since(startTime) > dhcpcdStopTimeout {
				log.Errorf("dhcpcd for interface %s is still running, will exit it, elapsed time %v",
					ifName, time.Since(startTime))
				failed = true
				break
			}
			log.Warnf("dhcpcd for interface %s is still running, elapsed time %v",
				ifName, time.Since(startTime))
			time.Sleep(1 * time.Second)
		}
		if !failed {
			log.Debugf("dhcpcd for interface %s is gone, elapsed time %v",
				ifName, time.Since(startTime))
			done(nil)
			return
		}
		// Exit dhcpcd running on the interface.
		// It waits up to 10 seconds.
		// https://github.com/NetworkConfiguration/dhcpcd/blob/dhcpcd-8.1.6/src/dhcpcd.c#L1950-L1957
		_, err := exec.Command(dhcpcdBinary, "--exit", ifName).CombinedOutput()
		if err != nil {
			err = fmt.Errorf("dhcpcd exit failed for interface %s: %v, elapsed time %v",
				ifName, err, time.Since(startTime))
			log.Error(err)
			done(err)
			return
		}
		if !c.dhcpcdExists(ifName) {
			log.Infof("dhcpcd for interface %s is gone after exit, elapsed time %v",
				ifName, time.Since(startTime))
			done(nil)
			return
		}
		err = fmt.Errorf("exiting dhcpcd for interface %s is still running, elapsed time %v",
			ifName, time.Since(startTime))
		log.Error(err)
		done(err)
		return
	}()
	return nil
}

func (c *DhcpcdConfigurator) dhcpcdExists(ifName string) bool {
	pidFile := fmt.Sprintf("/run/dhcpcd-%s.pid", ifName)
	pidBytes, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return false
	}
	pidStr := strings.TrimSpace(string(pidBytes))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		log.Errorf("dhcpcdExists(%s): strconv.Atoi of %s failed %s; ignored\n",
			ifName, pidStr, err)
		return true // guess since we don't know
	}
	// Does the pid exist?
	p, err := os.FindProcess(pid)
	if err != nil {
		log.Errorf("dhcpcdExists(%s): process not found %s", ifName, err)
		return false
	}
	err = p.Signal(syscall.Signal(0))
	if err != nil {
		log.Errorf("dhcpcdExists(%s): signal failed %s", ifName, err)
		return false
	}
	return true
}

// NeedsRecreate always returns true - Modify is not implemented.
func (c *DhcpcdConfigurator) NeedsRecreate(oldItem, newItem depgraph.Item) (recreate bool) {
	return true
}
