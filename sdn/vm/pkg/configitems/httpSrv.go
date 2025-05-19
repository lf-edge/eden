package configitems

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	sdnapi "github.com/lf-edge/eden/sdn/vm/api"
	httpsrvcfg "github.com/lf-edge/eden/sdn/vm/cmd/httpsrv/config"
	"github.com/lf-edge/eve/libs/depgraph"
	"github.com/lf-edge/eve/libs/reconciler"
	log "github.com/sirupsen/logrus"
)

const (
	httpSrvBinary  = "/bin/httpsrv"
	httpSrvConfDir = "/etc/httpsrv"
	httpSrvRunDir  = "/run/httpsrv"

	httpSrvStartTimeout = 3 * time.Second
	httpSrvStopTimeout  = 10 * time.Second
)

// HttpServer : HTTP server
type HttpServer struct {
	// ServerName : logical name for the HTTP server.
	ServerName string
	// NetNamespace : network namespace where the server should be running.
	NetNamespace string
	// VethName : logical name of the veth pair on which the server operates.
	// (other types of interfaces are currently not supported)
	// Can be empty (if the server is not associated with any particular interface).
	VethName string
	// ListenIPs : IP addresses on which the server should listen.
	// Can be empty to listen on all available interfaces instead of just
	// the interfaces with the given host addresses.
	ListenIPs []net.IP
	// HTTPPort : port to listen for HTTP requests.
	// Zero value can be used to disable HTTP.
	HTTPPort uint16
	// HTTPSPort : port to listen for HTTPS requests.
	// Zero value can be used to disable HTTPS.
	HTTPSPort uint16
	// CertPEM : Server certificate in the PEM format. Required for HTTPS.
	CertPEM string
	// KeyPEM : Server key in the PEM format. Required for HTTPS.
	KeyPEM string
	// Maps URL Path to a content to be returned inside the HTTP(s) response body.
	Paths map[string]sdnapi.HTTPContent
}

// Name
func (s HttpServer) Name() string {
	return s.ServerName
}

// Label
func (s HttpServer) Label() string {
	return s.ServerName + " (HTTP server)"
}

// Type
func (s HttpServer) Type() string {
	return HTTPServerTypename
}

// Equal is a comparison method for two equally-named HttpServer instances.
func (s HttpServer) Equal(other depgraph.Item) bool {
	s2 := other.(HttpServer)
	if len(s.Paths) != len(s2.Paths) {
		return false
	}
	for path, content := range s.Paths {
		if content2, ok := s2.Paths[path]; !ok || content != content2 {
			return false
		}
	}
	return s.NetNamespace == s2.NetNamespace &&
		s.VethName == s2.VethName &&
		equalIPLists(s.ListenIPs, s2.ListenIPs) &&
		s.HTTPPort == s2.HTTPPort &&
		s.HTTPSPort == s2.HTTPSPort &&
		s.CertPEM == s2.CertPEM &&
		s.KeyPEM == s2.KeyPEM
}

// External returns false.
func (s HttpServer) External() bool {
	return false
}

// String describes the HTTP server.
func (s HttpServer) String() string {
	return fmt.Sprintf("HTTP server: %#+v", s)
}

// Dependencies lists the (optional) veth and network namespace as dependencies.
func (s HttpServer) Dependencies() (deps []depgraph.Dependency) {
	deps = append(deps, depgraph.Dependency{
		RequiredItem: depgraph.ItemRef{
			ItemType: NetNamespaceTypename,
			ItemName: normNetNsName(s.NetNamespace),
		},
		Description: "Network namespace must exist",
	})
	if s.VethName != "" {
		deps = append(deps, depgraph.Dependency{
			RequiredItem: depgraph.ItemRef{
				ItemType: VethTypename,
				ItemName: s.VethName,
			},
			Description: "veth interface must exist",
		})
	}
	return deps
}

// HttpServerConfigurator implements Configurator interface for HttpServer.
type HttpServerConfigurator struct{}

// Create starts httpsrv (see sdn/cmd/httpsrv).
func (c *HttpServerConfigurator) Create(ctx context.Context, item depgraph.Item) error {
	config := item.(HttpServer)
	if err := c.createHttpSrvConfFile(config); err != nil {
		return err
	}
	done := reconciler.ContinueInBackground(ctx)
	go func() {
		err := startHttpSrv(config.ServerName, config.NetNamespace)
		done(err)
	}()
	return nil
}

func (c *HttpServerConfigurator) createHttpSrvConfFile(httpSrv HttpServer) error {
	if err := ensureDir(httpSrvConfDir); err != nil {
		return err
	}
	serverName := httpSrv.ServerName
	// Prepare configuration.
	listenIPs := make([]string, 0, len(httpSrv.ListenIPs))
	for _, ip := range httpSrv.ListenIPs {
		listenIPs = append(listenIPs, ip.String())
	}
	config := httpsrvcfg.HttpSrvConfig{
		ListenIPs: listenIPs,
		LogFile:   httpSrvLogFile(serverName),
		PidFile:   httpSrvPidFile(serverName),
		Verbose:   true,
		HTTPPort:  httpSrv.HTTPPort,
		HTTPSPort: httpSrv.HTTPSPort,
		CertPEM:   httpSrv.CertPEM,
		KeyPEM:    httpSrv.KeyPEM,
		Paths:     httpSrv.Paths,
	}
	configBytes, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		err = fmt.Errorf("failed to marshal config to JSON: %w", err)
		log.Error(err)
		return err
	}
	// Write configuration to file.
	cfgPath := httpSrvConfigPath(serverName)
	err = os.WriteFile(cfgPath, configBytes, 0644)
	if err != nil {
		err = fmt.Errorf("failed to create config file %s: %w", cfgPath, err)
		log.Error(err)
		return err
	}
	return nil
}

// Modify is not implemented.
func (c *HttpServerConfigurator) Modify(ctx context.Context, oldItem, newItem depgraph.Item) (err error) {
	return errors.New("not implemented")
}

// Delete stops httpsrv.
func (c *HttpServerConfigurator) Delete(ctx context.Context, item depgraph.Item) error {
	config := item.(HttpServer)
	done := reconciler.ContinueInBackground(ctx)
	go func() {
		err := stopHttpSrv(config.ServerName)
		if err == nil {
			// ignore errors from here
			_ = removeHttpSrvConfFile(config.ServerName)
			_ = removeHttpSrvLogFile(config.ServerName)
			_ = removeHttpSrvPidFile(config.ServerName)
		}
		done(err)
	}()
	return nil
}

// NeedsRecreate always returns true - Modify is not implemented.
func (c *HttpServerConfigurator) NeedsRecreate(oldItem, newItem depgraph.Item) (recreate bool) {
	return true
}

func httpSrvConfigPath(srvName string) string {
	return filepath.Join(httpSrvConfDir, srvName+".conf")
}

func httpSrvPidFile(srvName string) string {
	return filepath.Join(httpSrvRunDir, srvName+".pid")
}

func httpSrvLogFile(srvName string) string {
	return filepath.Join(httpSrvRunDir, srvName+".log")
}

func removeHttpSrvConfFile(srvName string) error {
	cfgPath := httpSrvConfigPath(srvName)
	if err := os.Remove(cfgPath); err != nil {
		err = fmt.Errorf("failed to remove HTTP server config %s: %w",
			cfgPath, err)
		log.Error(err)
		return err
	}
	return nil
}

func removeHttpSrvPidFile(srvName string) error {
	pidPath := httpSrvPidFile(srvName)
	if err := os.Remove(pidPath); err != nil {
		err = fmt.Errorf("failed to remove HTTP server PID file %s: %w",
			pidPath, err)
		log.Error(err)
		return err
	}
	return nil
}

func removeHttpSrvLogFile(srvName string) error {
	logPath := httpSrvLogFile(srvName)
	if err := os.Remove(logPath); err != nil {
		err = fmt.Errorf("failed to remove HTTP server log file %s: %w",
			logPath, err)
		log.Error(err)
		return err
	}
	return nil
}

func startHttpSrv(srvName, netNamespace string) error {
	if err := ensureDir(httpSrvRunDir); err != nil {
		return err
	}
	cfgPath := httpSrvConfigPath(srvName)
	cmd := httpSrvBinary
	args := []string{
		"-c",
		cfgPath,
	}
	pidFile := httpSrvPidFile(srvName)
	return startProcess(netNamespace, cmd, args, pidFile, httpSrvStartTimeout, true)
}

func stopHttpSrv(srvName string) error {
	pidFile := httpSrvPidFile(srvName)
	return stopProcess(pidFile, httpSrvStopTimeout)
}
