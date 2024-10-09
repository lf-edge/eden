package evetestkit

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/eve"
	"github.com/lf-edge/eden/pkg/openevec"
	"github.com/lf-edge/eden/pkg/projects"
	"github.com/lf-edge/eden/pkg/tests"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/tmc/scp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/exp/rand"
)

const (
	// AppDefaultSSHPass is a default ssh password for the VM running on the EVE node
	AppDefaultSSHPass = "passw0rd"
	// AppDefaultSSHUser is a default ssh user for the VM running on the EVE node
	AppDefaultSSHUser = "ubuntu"
	// AppDefaultCloudConfig is a default cloud-init configuration for the VM which just
	// enables ssh password authentication and sets the password to "passw0rd".
	AppDefaultCloudConfig = "#cloud-config\npassword: " + AppDefaultSSHPass + "\nchpasswd: { expire: False }\nssh_pwauth: True\n"
)

var (
	controllerVerbosiry = "warn"
	edenConfEnv         = defaults.DefaultConfigEnv
)

// appInstanceConfig is a struct that holds the information about the app
// running on the EVE node
type appInstanceConfig struct {
	name    string
	sshPort string
	sshUser string
	sshPass string
}

// EveNode is a struct that holds the information about the remote node
type EveNode struct {
	controller *openevec.OpenEVEC
	edgenode   *device.Ctx
	cfg        *openevec.EdenSetupArgs
	tc         *projects.TestContext
	apps       []appInstanceConfig
	ip         string
	t          *testing.T
}

// AppOption is a function that sets the configuration for the app running on
// the EVE node
type AppOption func(n *EveNode, appName string)

// TestOption is a function that sets the configuration for the test
type TestOption func()

func getEdenConfig() (*openevec.EdenSetupArgs, error) {
	conf := os.Getenv(edenConfEnv)
	configName := utils.GetConfig(conf)
	cfg, err := openevec.FromViper(configName, controllerVerbosiry)
	if err != nil {
		return nil, fmt.Errorf("can't get the config: %w", err)
	}

	return cfg, nil
}

func getOpenEVEC() (*openevec.OpenEVEC, *openevec.EdenSetupArgs, error) {
	cfg, err := getEdenConfig()
	if err != nil {
		return nil, nil, err
	}

	return openevec.CreateOpenEVEC(cfg), cfg, nil
}

func createEveNode(node *device.Ctx, tc *projects.TestContext) (*EveNode, error) {
	evec, cfg, err := getOpenEVEC()
	if err != nil {
		return nil, fmt.Errorf("can't create OpenEVEC: %w", err)
	}

	return &EveNode{controller: evec, edgenode: node, tc: tc, apps: []appInstanceConfig{}, cfg: cfg}, nil
}

func (node *EveNode) getAppConfig(appName string) *appInstanceConfig {
	for i := range node.apps {
		if node.apps[i].name == appName {
			return &node.apps[i]
		}
	}
	return nil
}

// EveRunCommand runs a command on the EVE node
func (node *EveNode) EveRunCommand(command string) ([]byte, error) {
	realStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	os.Stdout = w
	// unfortunately, we can't capture command return value from SSHEve
	err = node.controller.SSHEve(command)
	os.Stdout = realStdout
	w.Close()

	if err != nil {
		return nil, err
	}

	out, _ := io.ReadAll(r)
	return out, nil
}

// EveFileExists checks if a file exists on EVE node
func (node *EveNode) EveFileExists(fileName string) (bool, error) {
	command := fmt.Sprintf("if stat \"%s\"; then echo \"1\"; else echo \"0\"; fi", fileName)
	out, err := node.EveRunCommand(command)
	if err != nil {
		return false, err
	}

	if strings.TrimSpace(string(out)) == "0" {
		return false, nil
	}

	return true, nil
}

// EveReadFile reads a file from EVE node
func (node *EveNode) EveReadFile(fileName string) ([]byte, error) {
	exist, err := node.EveFileExists(fileName)
	if err != nil {
		return nil, err
	}

	if !exist {
		return nil, fmt.Errorf("file %s does not exist", fileName)
	}

	command := fmt.Sprintf("cat %s", fileName)
	return node.EveRunCommand(command)
}

// EveDeleteFile deletes a file from EVE node
func (node *EveNode) EveDeleteFile(fileName string) error {
	exist, err := node.EveFileExists(fileName)
	if err != nil {
		return err
	}

	if !exist {
		return nil
	}

	command := fmt.Sprintf("rm %s", fileName)
	_, err = node.EveRunCommand(command)
	return err
}

// AppWaitForRunningState waits for an app to start and become running on the EVE node
func (node *EveNode) AppWaitForRunningState(appName string, timeoutSeconds uint) error {
	start := time.Now()
	lastState := ""

	for {
		state, err := node.AppGetState(appName)
		if err != nil {
			return err
		}

		if lastState != state {
			fmt.Println(utils.AddTimestampf("App %s state changed to: %s", appName, state))
			lastState = state
		}

		state = strings.ToLower(state)
		if strings.Contains(state, "halting") {
			return fmt.Errorf("app %s is in halting state", appName)
		}

		if state == "running" {
			return nil
		}

		if time.Since(start) > time.Duration(timeoutSeconds)*time.Second {
			return fmt.Errorf("timeout waiting for app %s to start", appName)
		}

		time.Sleep(1 * time.Second)
	}
}

// AppWaitForSSH waits for the SSH connection to be established to the app VM that
// is running on the EVE node
func (node *EveNode) AppWaitForSSH(appName string, timeoutSeconds uint) error {
	start := time.Now()
	for {
		_, err := node.AppSSHExec(appName, "echo")
		if err == nil {
			return nil
		}

		if time.Since(start) > time.Duration(timeoutSeconds)*time.Second {
			return fmt.Errorf("timeout waiting for SSH connection")
		}

		fmt.Println(utils.AddTimestampf("Still waiting for SSH connection (%d/%d seconds)",
			int(time.Since(start).Seconds()), timeoutSeconds))

		time.Sleep(3 * time.Second)
	}
}

// AppStopAndRemove stops and removes an app from the EVE node
func (node *EveNode) AppStopAndRemove(appName string) error {
	if err := node.controller.PodStop(appName); err != nil {
		return err
	}

	if _, err := node.controller.PodDelete(appName, true); err != nil {
		return err
	}

	return nil
}

// AppGetState gets the state of an app running on the EVE node
func (node *EveNode) AppGetState(appName string) (string, error) {
	ctrl, err := controller.CloudPrepare()
	if err != nil {
		return "", fmt.Errorf("fail in CloudPrepare: %w", err)
	}

	state := eve.Init(ctrl, node.edgenode)
	if err := ctrl.InfoLastCallback(node.edgenode.GetID(), nil, state.InfoCallback()); err != nil {
		return "", fmt.Errorf("fail in get InfoLastCallback: %w", err)
	}
	if err := ctrl.MetricLastCallback(node.edgenode.GetID(), nil, state.MetricCallback()); err != nil {
		return "", fmt.Errorf("fail in get MetricLastCallback: %w", err)
	}
	appStatesSlice := make([]*eve.AppInstState, 0, len(state.Applications()))
	appStatesSlice = append(appStatesSlice, state.Applications()...)
	for _, app := range appStatesSlice {
		if app.Name == appName {
			return app.EVEState, nil
		}
	}

	return "", fmt.Errorf("app %s not found", appName)
}

// AppSSHExec executes a command on the app VM running on the EVE node.
func (node *EveNode) AppSSHExec(appName, command string) (string, error) {
	appConfig := node.getAppConfig(appName)
	if appConfig == nil {
		return "", fmt.Errorf("app %s not found, make sure to deploy app/vm with WithSSH option", appName)
	}

	host := fmt.Sprintf("%s:%s", node.ip, appConfig.sshPort)

	config := &ssh.ClientConfig{
		User: appConfig.sshUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(appConfig.sshPass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}
	client, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return "", fmt.Errorf("failed to dial: %s", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %s", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return "", fmt.Errorf("failed to run command: %s", err)
	}

	return string(output), nil
}

// AppSCPCopy copies a file from the local machine to the app VM running on the EVE node.
func (node *EveNode) AppSCPCopy(appName, localFile, remoteFile string) error {
	info, err := os.Stat(localFile)
	if os.IsNotExist(err) {
		return fmt.Errorf("file %s does not exist", localFile)
	}
	if info.IsDir() {
		return fmt.Errorf("file %s is a directory", localFile)
	}

	appConfig := node.getAppConfig(appName)
	if appConfig == nil {
		return fmt.Errorf("app %s not found, make sure to deploy app/vm with WithSSH option", appName)
	}

	host := fmt.Sprintf("%s:%s", node.ip, appConfig.sshPort)

	config := &ssh.ClientConfig{
		User: appConfig.sshUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(appConfig.sshPass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}
	client, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return fmt.Errorf("failed to dial: %s", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %s", err)
	}
	defer session.Close()

	err = scp.CopyPath(localFile, remoteFile, session)
	if err != nil {
		return nil
	}

	return nil
}

// WithSSH is an option that sets the SSH configuration for the app running on
// the EVE node, this should be use with DeployVM function.
func WithSSH(user, pass, port string) AppOption {
	return func(n *EveNode, appName string) {
		a := n.getAppConfig(appName)
		a.sshUser = user
		a.sshPass = pass
		a.sshPort = port
	}
}

// EveRebootNode reboots the EVE node.
func (node *EveNode) EveRebootNode() error {
	return node.controller.EdgeNodeReboot("")
}

// EveRebootAndWait reboots the EVE node and waits for it to come back.
func (node *EveNode) EveRebootAndWait(timeoutSeconds uint) error {
	out, err := node.EveRunCommand("uptime -s")
	if err != nil {
		return err
	}
	uptimeOne := strings.TrimSpace(string(out))

	if err := node.EveRebootNode(); err != nil {
		return err
	}

	start := time.Now()
	for {
		if time.Since(start) > time.Duration(timeoutSeconds)*time.Second {
			return fmt.Errorf("timeout waiting for the node to reboot and come back")
		}

		out, err := node.EveRunCommand("uptime -s")
		if err != nil {
			continue
		}

		if uptimeOne != strings.TrimSpace(string(out)) {
			break
		}

		fmt.Println(utils.AddTimestampf("Still waiting for node to boot up (%d/%d seconds)",
			int(time.Since(start).Seconds()), timeoutSeconds))
		time.Sleep(3 * time.Second)
	}

	return nil
}

// EveDeployApp deploys a VM/App on the EVE node
func (node *EveNode) EveDeployApp(appLink string, pc openevec.PodConfig, options ...AppOption) error {
	node.apps = append(node.apps, appInstanceConfig{name: pc.Name})
	for _, option := range options {
		option(node, pc.Name)
	}

	return node.controller.PodDeploy(appLink, pc, node.cfg)
}

// EveIsTpmEnabled checks if EVE node is running with (SW)TPM enabled
func (node *EveNode) EveIsTpmEnabled() bool {
	return node.cfg.Eve.TPM
}

func (node *EveNode) LogTimeFatalf(format string, args ...interface{}) {
	out := utils.AddTimestampf(format+"\n", args...)
	if node.t != nil {
		node.t.Fatal(out)
	} else {
		fmt.Print(out)
		os.Exit(1)
	}
}

func (node *EveNode) LogTimeInfof(format string, args ...interface{}) {
	out := utils.AddTimestampf(format+"\n", args...)
	if node.t != nil {
		node.t.Logf(out)
	} else {
		fmt.Print(out)
	}
}

func (node *EveNode) discoverEveIP() error {
	if node.edgenode.GetRemoteAddr() == "" {
		eveIPCIDR, err := node.tc.GetState(node.edgenode).LookUp("Dinfo.Network[0].IPAddrs[0]")
		if err != nil {
			return err
		}

		ip := net.ParseIP(eveIPCIDR.String())
		if ip == nil || ip.To4() == nil {
			return fmt.Errorf("failed to parse IP address: %s", eveIPCIDR.String())
		}

		node.ip = ip.To4().String()
		return nil
	}

	node.ip = node.edgenode.GetRemoteAddr()
	return nil
}

// GetDefaultVMConfig returns a default configuration for a VM
func GetDefaultVMConfig(appName, cloudConfig string, portPub []string) openevec.PodConfig {
	var pc openevec.PodConfig

	pc.Name = appName
	pc.AppMemory = humanize.Bytes(defaults.DefaultAppMem * 1024)
	pc.DiskSize = "4GB"
	pc.VolumeType = "QCOW2"
	pc.Metadata = cloudConfig
	pc.VncPassword = ""
	pc.ImageFormat = "QCOW2"
	pc.Registry = "remote"
	pc.VolumeSize = humanize.IBytes(defaults.DefaultVolumeSize)
	pc.PortPublish = portPub
	pc.VncDisplay = 0
	pc.AppCpus = defaults.DefaultAppCPU
	pc.AppAdapters = nil
	pc.Networks = nil
	pc.ACLOnlyHost = false
	pc.NoHyper = false
	pc.DirectLoad = true
	pc.SftpLoad = false
	pc.Disks = nil
	pc.Mount = nil
	pc.Profiles = nil
	pc.ACL = nil
	pc.Vlans = nil
	pc.OpenStackMetadata = false
	pc.DatastoreOverride = ""
	pc.StartDelay = 0
	pc.PinCpus = false

	return pc
}

// WithControllerVerbosity sets the verbosity level of the controller,
// possible values are: panic, fatal, error, debug, info, trace, warn
// This is an option for InitilizeTest.
func WithControllerVerbosity(verbosity string) TestOption {
	return func() {
		controllerVerbosiry = verbosity
	}
}

// WithEdenConfigEnv sets the environment variable that holds the path to the
// eden configuration file. This is an option for InitilizeTest.
func WithEdenConfigEnv(env string) TestOption {
	return func() {
		edenConfEnv = env
	}
}

// GetRandomAppName generates a random app name
func GetRandomAppName(prefix string) string {
	rnd := rand.New(rand.NewSource(uint64(time.Now().UnixNano())))
	return prefix + namesgenerator.GetRandomName(rnd.Intn(1))
}

// InitilizeTest is used to provide setup and teardown for the rest of the
// tests. As part of setup we make sure that context has a slice of
// EVE instances that we can operate on. It grabs the first one in the slice
// for running tests.
func InitilizeTest(projectName string, options ...TestOption) (*EveNode, error) {
	var edgenode *device.Ctx
	tests.TestArgsParse()
	tc := projects.NewTestContext()

	// Registering our own project namespace with controller for easy cleanup
	tc.InitProject(fmt.Sprintf("%s_%s", projectName, time.Now()))

	// Create representation of EVE instances (based on the names
	// or UUIDs that were passed in) in the context. This is the first place
	// where we're using zcli-like API:
	for _, node := range tc.GetNodeDescriptions() {
		edgeNode := node.GetEdgeNode(tc)
		if edgeNode == nil {
			// Couldn't find existing edgeNode record in the controller.
			// Need to create it from scratch now:
			// this is modeled after: zcli edge-node create <name>
			// --project=<project> --model=<model> [--title=<title>]
			// ([--edge-node-certificate=<certificate>] |
			// [--onboarding-certificate=<certificate>] |
			// [(--onboarding-key=<key> --serial=<serial-number>)])
			// [--network=<network>...]
			//
			// XXX: not sure if struct (giving us optional fields) would be better
			edgeNode = tc.NewEdgeNode(tc.WithNodeDescription(node), tc.WithCurrentProject())
		} else {
			// make sure to move EdgeNode to the project we created, again
			// this is modeled after zcli edge-node update <name> [--title=<title>]
			// [--lisp-mode=experimental|default] [--project=<project>]
			// [--clear-onboarding-certs] [--config=<key:value>...] [--network=<network>...]
			edgeNode.SetProject(projectName)
		}

		edgenode = edgeNode
		tc.ConfigSync(edgeNode)

		// finally we need to make sure that the edgeNode is in a state that we need
		// it to be, before the test can run -- this could be multiple checks on its
		// status, but for example:
		if edgeNode.GetState() == device.NotOnboarded {
			return nil, fmt.Errorf("node is not onboarded now")
		}

		// this is a good node -- lets add it to the test context
		tc.AddNode(edgeNode)
	}

	tc.StartTrackingState(false)

	// apply options
	for _, option := range options {
		option()
	}

	// create a remote node
	rnode, err := createEveNode(edgenode, tc)
	if err != nil {
		return nil, fmt.Errorf("can't create RemoteNode: %w", err)
	}

	// get the IP address of the EVE node
	err = rnode.discoverEveIP()
	if err != nil {
		return nil, fmt.Errorf("can't get the IP address of the EVE node: %w", err)
	}

	return rnode, nil
}
