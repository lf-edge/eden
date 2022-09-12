package configitems

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"

	sdnapi "github.com/lf-edge/eden/sdn/api"
	goproxycfg "github.com/lf-edge/eden/sdn/cmd/goproxy/config"
	"github.com/lf-edge/eve/libs/depgraph"
	"github.com/lf-edge/eve/libs/reconciler"
	log "github.com/sirupsen/logrus"
)

const (
	goproxyBinary  = "/bin/goproxy"
	goproxyConfDir = "/etc/goproxy"
	goproxyRunDir  = "/run/goproxy"

	goproxyStartTimeout = 3 * time.Second
	goproxyStopTimeout  = 10 * time.Second
)

// HttpProxy : HTTP(S) proxy
type HttpProxy struct {
	sdnapi.Proxy
	// ProxyName : logical name for the HTTP proxy.
	ProxyName string
	// NetNamespace : network namespace where the server should be running.
	NetNamespace string
	// VethName : logical name of the veth pair on which the proxy operates.
	// (other types of interfaces are currently not supported)
	// Can be empty (if the proxy is not associated with any particular interface).
	VethName string
	// ListenIP : IP address on which the proxy should listen.
	// Can be empty to listen on all available interfaces instead of just
	// the interface with the given host address
	ListenIP net.IP
	// HTTPPort : HTTP proxy port.
	// Zero value can be used to disable HTTP proxy.
	HTTPPort uint16
	// HTTPSPorts : HTTPS proxy port(s).
	// Empty list can be used to disable HTTPS proxy.
	HTTPSPorts []uint16
	// Transparent : enable for transparent proxy (not known to the client).
	Transparent bool
	// Users : define for username/password authentication, leave empty otherwise.
	Users []sdnapi.UserCredentials
}

// Name
func (p HttpProxy) Name() string {
	return p.ProxyName
}

// Label
func (p HttpProxy) Label() string {
	return p.ProxyName + " (HTTP proxy)"
}

// Type
func (p HttpProxy) Type() string {
	return HTTPProxyTypename
}

// Equal is a comparison method for two equally-named HttpProxy instances.
func (p HttpProxy) Equal(other depgraph.Item) bool {
	p2 := other.(HttpProxy)
	if len(p.ProxyRules) != len(p2.ProxyRules) {
		return false
	}
	for i := range p.ProxyRules {
		if p.ProxyRules[i] != p2.ProxyRules[i] {
			return false
		}
	}
	if len(p.Users) != len(p2.Users) {
		return false
	}
	for i := range p.Users {
		if p.Users[i] != p2.Users[i] {
			return false
		}
	}
	if len(p.HTTPSPorts) != len(p2.HTTPSPorts) {
		return false
	}
	for i := range p.HTTPSPorts {
		if p.HTTPSPorts[i] != p2.HTTPSPorts[i] {
			return false
		}
	}
	return p.NetNamespace == p2.NetNamespace &&
		p.VethName == p2.VethName &&
		p.ListenIP.Equal(p2.ListenIP) &&
		p.HTTPPort == p2.HTTPPort &&
		p.Transparent == p2.Transparent &&
		p.CACertPEM == p2.CACertPEM &&
		p.CAKeyPEM == p2.CAKeyPEM
}

// External returns false.
func (p HttpProxy) External() bool {
	return false
}

// String describes the HTTP proxy.
func (p HttpProxy) String() string {
	return fmt.Sprintf("HTTP proxy: %#+v", p)
}

// Dependencies lists the (optional) veth and network namespace as dependencies.
func (p HttpProxy) Dependencies() (deps []depgraph.Dependency) {
	deps = append(deps, depgraph.Dependency{
		RequiredItem: depgraph.ItemRef{
			ItemType: NetNamespaceTypename,
			ItemName: normNetNsName(p.NetNamespace),
		},
		Description: "Network namespace must exist",
	})
	if p.VethName != "" {
		deps = append(deps, depgraph.Dependency{
			RequiredItem: depgraph.ItemRef{
				ItemType: VethTypename,
				ItemName: p.VethName,
			},
			Description: "veth interface must exist",
		})
	}
	return deps
}

// HttpProxyConfigurator implements Configurator interface for HttpProxy.
type HttpProxyConfigurator struct{}

// Create starts goproxy (see sdn/cmd/goproxy).
func (c *HttpProxyConfigurator) Create(ctx context.Context, item depgraph.Item) error {
	config := item.(HttpProxy)
	if err := c.createGoproxyConfFile(config); err != nil {
		return err
	}
	done := reconciler.ContinueInBackground(ctx)
	go func() {
		err := startGoproxy(config.ProxyName, config.NetNamespace)
		done(err)
	}()
	return nil
}

func (c *HttpProxyConfigurator) createGoproxyConfFile(proxy HttpProxy) error {
	if err := ensureDir(goproxyConfDir); err != nil {
		return err
	}
	proxyName := proxy.ProxyName
	// Prepare configuration.
	var listenIP string
	if proxy.ListenIP != nil {
		listenIP = proxy.ListenIP.String()
	}
	config := goproxycfg.ProxyConfig{
		ListenIP:    listenIP,
		HTTPPort:    proxy.HTTPPort,
		HTTPSPorts:  proxy.HTTPSPorts,
		Transparent: proxy.Transparent,
		LogFile:     goproxyLogFile(proxyName),
		PidFile:     goproxyPidFile(proxyName),
		Verbose:     true,
		CACertPEM:   proxy.CACertPEM,
		CAKeyPEM:    proxy.CAKeyPEM,
		ProxyRules:  proxy.ProxyRules,
		Users:       proxy.Users,
	}
	configBytes, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		err = fmt.Errorf("failed to marshal config to JSON: %w", err)
		log.Error(err)
		return err
	}
	// Write configuration to file.
	cfgPath := goproxyConfigPath(proxyName)
	err = ioutil.WriteFile(cfgPath, configBytes, 0644)
	if err != nil {
		err = fmt.Errorf("failed to create config file %s: %w", cfgPath, err)
		log.Error(err)
		return err
	}
	return nil
}

// Modify is not implemented.
func (c *HttpProxyConfigurator) Modify(ctx context.Context, oldItem, newItem depgraph.Item) (err error) {
	return errors.New("not implemented")
}

// Delete stops goproxy.
func (c *HttpProxyConfigurator) Delete(ctx context.Context, item depgraph.Item) error {
	config := item.(HttpProxy)
	done := reconciler.ContinueInBackground(ctx)
	go func() {
		err := stopGoproxy(config.ProxyName)
		if err == nil {
			// ignore errors from here
			_ = removeGoproxyConfFile(config.ProxyName)
			_ = removeGoproxyLogFile(config.ProxyName)
			_ = removeGoproxyPidFile(config.ProxyName)
		}
		done(err)
	}()
	return nil
}

// NeedsRecreate always returns true - Modify is not implemented.
func (c *HttpProxyConfigurator) NeedsRecreate(oldItem, newItem depgraph.Item) (recreate bool) {
	return true
}

func goproxyConfigPath(proxyName string) string {
	return filepath.Join(goproxyConfDir, proxyName+".conf")
}

func goproxyPidFile(proxyName string) string {
	return filepath.Join(goproxyRunDir, proxyName+".pid")
}

func goproxyLogFile(proxyName string) string {
	return filepath.Join(goproxyRunDir, proxyName+".log")
}

func removeGoproxyConfFile(proxyName string) error {
	cfgPath := goproxyConfigPath(proxyName)
	if err := os.Remove(cfgPath); err != nil {
		err = fmt.Errorf("failed to remove goproxy config %s: %w",
			cfgPath, err)
		log.Error(err)
		return err
	}
	return nil
}

func removeGoproxyPidFile(proxyName string) error {
	pidPath := goproxyPidFile(proxyName)
	if err := os.Remove(pidPath); err != nil {
		err = fmt.Errorf("failed to remove goproxy PID file %s: %w",
			pidPath, err)
		log.Error(err)
		return err
	}
	return nil
}

func removeGoproxyLogFile(proxyName string) error {
	logPath := goproxyLogFile(proxyName)
	if err := os.Remove(logPath); err != nil {
		err = fmt.Errorf("failed to remove proxy log file %s: %w",
			logPath, err)
		log.Error(err)
		return err
	}
	return nil
}

func startGoproxy(proxyName, netNamespace string) error {
	if err := ensureDir(goproxyRunDir); err != nil {
		return err
	}
	cfgPath := goproxyConfigPath(proxyName)
	cmd := goproxyBinary
	args := []string{
		"-c",
		cfgPath,
	}
	pidFile := goproxyPidFile(proxyName)
	return startProcess(netNamespace, cmd, args, pidFile, goproxyStartTimeout, true)
}

func stopGoproxy(proxyName string) error {
	pidFile := goproxyPidFile(proxyName)
	return stopProcess(pidFile, goproxyStopTimeout)
}
