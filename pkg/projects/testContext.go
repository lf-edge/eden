package projects

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"

	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/adam"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/edensdn"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/spf13/viper"
)

//GetControllerMode parse url with controller
func GetControllerMode(controllerMode string) (modeType, modeURL string, err error) {
	params := utils.GetParams(controllerMode, defaults.DefaultControllerModePattern)
	if len(params) == 0 {
		return "", "", fmt.Errorf("cannot parse mode (not [file|proto|adam|zedcloud]://<URL>): %s", controllerMode)
	}
	ok := false
	if modeType, ok = params["Type"]; !ok {
		return "", "", fmt.Errorf("cannot parse modeType (not [file|proto|adam|zedcloud]://<URL>): %s", controllerMode)
	}
	if modeURL, ok = params["URL"]; !ok {
		return "", "", fmt.Errorf("cannot parse modeURL (not [file|proto|adam|zedcloud]://<URL>): %s", controllerMode)
	}
	return
}

//TestContext is main structure for running tests
type TestContext struct {
	cloud     controller.Cloud
	project   *Project
	nodes     []*device.Ctx
	sdnClient *edensdn.SdnClient
	withSdn   bool
	procBus   *processingBus
	tests     map[*device.Ctx]*testing.T
	states    map[*device.Ctx]*State
	stopTime  time.Time
	addTime   time.Duration
}

//NewTestContext creates new TestContext
func NewTestContext() *TestContext {
	var (
		err       error
		sdnClient *edensdn.SdnClient
		withSdn   bool
	)
	viperLoaded := false
	if edenConfigEnv := os.Getenv(defaults.DefaultConfigEnv); edenConfigEnv != "" {
		viperLoaded, err = utils.LoadConfigFile(utils.GetConfig(edenConfigEnv))
	} else {
		viperLoaded, err = utils.LoadConfigFile("")
	}
	if err != nil {
		log.Fatalf("LoadConfigFile %s", err)
	}
	if viperLoaded {
		modeType, modeURL, err := GetControllerMode(viper.GetString("test.controller"))
		if err != nil {
			log.Debug(err)
		}
		if modeType != "" {
			if modeType != "adam" {
				log.Fatalf("Not implemented controller type %s", modeType)
			}
		}
		if modeURL != "" { //overwrite config only if url defined
			ipPort := strings.Split(modeURL, ":")
			ip := ipPort[0]
			if ip == "" {
				log.Fatalf("cannot get ip/hostname from %s", modeURL)
			}
			port := "80"
			if len(ipPort) > 1 {
				port = ipPort[1]
			}
			viper.Set("adam.ip", ip)
			viper.Set("adam.port", port)
		}
		devModel := viper.GetString("eve.devmodel")
		eveRemote := viper.GetBool("eve.remote")
		withSdn = !viper.GetBool("sdn.disable") &&
			devModel == defaults.DefaultQemuModel &&
			!eveRemote
		if withSdn {
			sdnSSHPort := viper.GetInt("sdn.ssh-port")
			sdnMgmtPort := viper.GetInt("sdn.mgmt-port")
			sdnSourceDir := utils.ResolveAbsPath(viper.GetString("sdn.source-dir"))
			sdnSSHKeyPath := filepath.Join(sdnSourceDir, "cert/ssh/id_rsa")
			sdnClient = &edensdn.SdnClient{
				SSHPort:    uint16(sdnSSHPort),
				SSHKeyPath: sdnSSHKeyPath,
				MgmtPort:   uint16(sdnMgmtPort),
			}
		}
	}
	vars, err := utils.InitVars()
	if err != nil {
		log.Fatalf("utils.InitVars: %s", err)
	}
	ctx := &controller.CloudCtx{Controller: &adam.Ctx{}}
	ctx.SetVars(vars)
	if err := ctx.InitWithVars(vars); err != nil {
		log.Fatalf("cloud.InitWithVars: %s", err)
	}
	ctx.GetAllNodes()
	tstCtx := &TestContext{
		cloud:     ctx,
		tests:     map[*device.Ctx]*testing.T{},
		sdnClient: sdnClient,
		withSdn:   withSdn,
	}
	tstCtx.procBus = initBus(tstCtx)
	return tstCtx
}

//GetNodeDescriptions returns list of nodes from config
func (tc *TestContext) GetNodeDescriptions() (nodes []*EdgeNodeDescription) {
	if eveList := viper.GetStringMap("test.eve"); len(eveList) > 0 {
		for name := range eveList {
			eveKey := viper.GetString(fmt.Sprintf("test.eve.%s.onboard-cert", name))
			eveSerial := viper.GetString(fmt.Sprintf("test.eve.%s.serial", name))
			eveModel := viper.GetString(fmt.Sprintf("test.eve.%s.model", name))
			nodes = append(nodes, &EdgeNodeDescription{Key: eveKey, Serial: eveSerial, Model: eveModel})
		}
	} else {
		log.Debug("NodeDescriptions not found. Will use default one.")
		nodes = append(nodes, &EdgeNodeDescription{
			Key:    utils.ResolveAbsPath(viper.GetString("eve.cert")),
			Serial: viper.GetString("eve.serial"),
			Model:  viper.GetString("eve.devModel"),
		})
	}
	return
}

//GetController returns current controller
func (tc *TestContext) GetController() controller.Cloud {
	if tc.cloud == nil {
		log.Fatal("Controller not initialized")
	}
	return tc.cloud
}

//InitProject init project object with defined name
func (tc *TestContext) InitProject(name string) {
	tc.project = &Project{name: name}
}

//AddEdgeNodesFromDescription adds EdgeNodes from description in test.eve param
func (tc *TestContext) AddEdgeNodesFromDescription() {
	for _, node := range tc.GetNodeDescriptions() {
		edgeNode := node.GetEdgeNode(tc)
		if edgeNode == nil {
			edgeNode = tc.NewEdgeNode(tc.WithNodeDescription(node), tc.WithCurrentProject())
		} else {
			edgeNode.SetProject(tc.project.name)
		}

		tc.ConfigSync(edgeNode)

		if edgeNode.GetState() == device.NotOnboarded {
			log.Fatal("Node is not onboarded now")
		}

		tc.AddNode(edgeNode)
	}
}

//GetEdgeNodeOpts pattern to pass device modifications
type GetEdgeNodeOpts func(*device.Ctx) bool

//WithTest assign *testing.T for device
func (tc *TestContext) WithTest(t *testing.T) GetEdgeNodeOpts {
	return func(d *device.Ctx) bool {
		tc.tests[d] = t
		return true
	}
}

//GetEdgeNode return node from context
func (tc *TestContext) GetEdgeNode(opts ...GetEdgeNodeOpts) *device.Ctx {
Node:
	for _, el := range tc.nodes {
		for _, opt := range opts {
			if !opt(el) {
				continue Node
			}
		}
		return el
	}
	return nil
}

//AddNode add node to test context
func (tc *TestContext) AddNode(node *device.Ctx) {
	tc.nodes = append(tc.nodes, node)
}

//UpdateEdgeNode update edge node
func (tc *TestContext) UpdateEdgeNode(edgeNode *device.Ctx, opts ...EdgeNodeOption) {
	for _, opt := range opts {
		opt(edgeNode)
	}
	tc.ConfigSync(edgeNode)
}

//NewEdgeNode creates edge node
func (tc *TestContext) NewEdgeNode(opts ...EdgeNodeOption) *device.Ctx {
	d := device.CreateEdgeNode()
	for _, opt := range opts {
		opt(d)
	}
	if tc.project == nil {
		log.Fatal("You must setup project before add node")
	}
	tc.ConfigSync(d)
	return d
}

//ConfigSync send config to controller
func (tc *TestContext) ConfigSync(edgeNode *device.Ctx) {
	if edgeNode.GetState() == device.NotOnboarded {
		if err := tc.GetController().OnBoardDev(edgeNode); err != nil {
			log.Fatalf("OnBoardDev %s", err)
		}
	} else {
		log.Debugf("Device %s onboarded", edgeNode.GetID().String())
	}
	if err := tc.GetController().ConfigSync(edgeNode); err != nil {
		log.Fatalf("Cannot send config of %s", edgeNode.GetID())
	}
}

//ExpandOnSuccess adds additional time to global timeout on every success check
func (tc *TestContext) ExpandOnSuccess(secs int) {
	tc.addTime = time.Duration(secs) * time.Second
}

//WaitForProcWithErrorCallback blocking execution until the time elapses or all Procs gone
//and fires callback in case of timeout
func (tc *TestContext) WaitForProcWithErrorCallback(secs int, callback Callback) {
	defer func() { tc.addTime = 0 }() //reset addTime on exit
	defer tc.procBus.clean()
	timeout := time.Duration(secs) * time.Second
	tc.stopTime = time.Now().Add(timeout)
	ticker := time.NewTicker(defaults.DefaultRepeatTimeout)
	defer ticker.Stop()
	waitChan := make(chan struct{}, 1)
	go func() {
		tc.procBus.wg.Wait()
		waitChan <- struct{}{}
	}()
	for {
		select {
		case <-waitChan:
			for node, el := range tc.tests {
				el.Logf("done for device %s", node.GetID())
			}
			return
		case <-ticker.C:
			for _, el := range tc.tests {
				if el.Failed() {
					// if one of tests failed, we are failed
					callback()
					return
				}
			}
			if time.Now().After(tc.stopTime) {
				callback()
				for _, el := range tc.tests {
					el.Errorf("WaitForProcWithErrorCallback terminated by timeout %s", timeout)
				}
				return
			}
		}
	}
}

//WaitForProc blocking execution until the time elapses or all Procs gone
//returns error on timeout
func (tc *TestContext) WaitForProc(secs int) {
	timeout := time.Duration(secs) * time.Second
	callback := func() {
		if len(tc.tests) == 0 {
			log.Fatalf("WaitForProc terminated by timeout %s", timeout)
		}
		for _, el := range tc.tests {
			el.Errorf("WaitForProc terminated by timeout %s", timeout)
		}
	}
	tc.WaitForProcWithErrorCallback(secs, callback)
}

//AddProcLog add processFunction, that will get all logs for edgeNode
func (tc *TestContext) AddProcLog(edgeNode *device.Ctx, processFunction ProcLogFunc) {
	tc.procBus.addProc(edgeNode, processFunction)
}

//AddProcAppLog add processFunction, that will get all app logs for edgeNode
func (tc *TestContext) AddProcAppLog(edgeNode *device.Ctx, appUUID uuid.UUID, processFunction ProcAppLogFunc) {
	tc.procBus.addAppProc(edgeNode, appUUID, processFunction)
}

//AddProcFlowLog add processFunction, that will get all FlowLogs for edgeNode
func (tc *TestContext) AddProcFlowLog(edgeNode *device.Ctx, processFunction ProcLogFlowFunc) {
	tc.procBus.addProc(edgeNode, processFunction)
}

//AddProcInfo add processFunction, that will get all info for edgeNode
func (tc *TestContext) AddProcInfo(edgeNode *device.Ctx, processFunction ProcInfoFunc) {
	tc.procBus.addProc(edgeNode, processFunction)
}

//AddProcMetric add processFunction, that will get all metrics for edgeNode
func (tc *TestContext) AddProcMetric(edgeNode *device.Ctx, processFunction ProcMetricFunc) {
	tc.procBus.addProc(edgeNode, processFunction)
}

//AddProcTimer add processFunction, that will fire with time intervals for edgeNode
func (tc *TestContext) AddProcTimer(edgeNode *device.Ctx, processFunction ProcTimerFunc) {
	tc.procBus.addProc(edgeNode, processFunction)
}

//StartTrackingState init function for State monitoring
//if onlyNewElements set no use old information from controller
func (tc *TestContext) StartTrackingState(onlyNewElements bool) {
	tc.states = map[*device.Ctx]*State{}
	for _, dev := range tc.nodes {
		curState := InitState(dev)
		tc.states[dev] = curState
		if !onlyNewElements {
			//process all events from controller
			_ = tc.GetController().InfoLastCallback(dev.GetID(), map[string]string{}, curState.getProcessorInfo())
			_ = tc.GetController().MetricLastCallback(dev.GetID(), map[string]string{}, curState.getProcessorMetric())
		}
		if _, exists := tc.procBus.proc[dev]; !exists {
			tc.procBus.initCheckers(dev)
		}
		tc.procBus.proc[dev] = append(tc.procBus.proc[dev], &absFunc{proc: curState.GetInfoProcessingFunction(), disabled: false, states: true})
		tc.procBus.proc[dev] = append(tc.procBus.proc[dev], &absFunc{proc: curState.GetMetricProcessingFunction(), disabled: false, states: true})
	}
}

//WaitForState wait for State initialization from controller
func (tc *TestContext) WaitForState(edgeNode *device.Ctx, secs int) {
	state, isOk := tc.states[edgeNode]
	if !isOk {
		log.Fatalf("edgeNode not found with name %s", edgeNode.GetID())
	}
	timeout := time.Duration(secs) * time.Second
	waitChan := make(chan struct{})
	go func() {
		for {
			if state.CheckReady() {
				close(waitChan)
				return
			}
			time.Sleep(defaults.DefaultRepeatTimeout)
		}
	}()
	select {
	case <-waitChan:
		if el, isOk := tc.tests[edgeNode]; !isOk {
			log.Println("done waiting for State")
		} else {
			el.Logf("done waiting for State")
		}
		return
	case <-time.After(timeout):
		if len(tc.tests) == 0 {
			log.Fatalf("WaitForState terminated by timeout %s", timeout)
		}
		for _, el := range tc.tests {
			el.Fatalf("WaitForState terminated by timeout %s", timeout)
		}
		return
	}
}

//GetState returns State object for edgeNode
func (tc *TestContext) GetState(edgeNode *device.Ctx) *State {
	return tc.states[edgeNode]
}

// PortForwardCommand forwards network traffic from a command run inside the host
// and targeted towards EVE VM.
// The command should be run from inside of `cmd` function, with localhost (or 127.0.0.1)
// as the destination IP and fwdPort as the destination port.
func (tc *TestContext) PortForwardCommand(cmd func(fwdPort uint16) error,
	eveIfName string, targetPort uint16) error {
	if !tc.withSdn {
		// Find out what the targetPort is (statically) mapped to in the host.
		targetHostPort := -1
		hostFwd := viper.GetStringMapString("eve.hostfwd")
		for k, v := range hostFwd {
			hostPort, err := strconv.Atoi(k)
			if err != nil {
				log.Errorf("failed to parse host port from eve.hostfwd: %v", err)
				continue
			}
			guestPort, err := strconv.Atoi(v)
			if err != nil {
				log.Errorf("failed to parse guest port from eve.hostfwd: %v", err)
				continue
			}
			if eveIfName == "eth1" {
				// For eth1 numbers of forwarded ports are shifted by 10.
				hostPort += 10
				guestPort += 10
			}
			if guestPort == int(targetPort) {
				targetHostPort = hostPort
				break
			}
		}
		if targetHostPort == -1 {
			log.Fatalf("Target EVE interface and port (%s, %d) are not port-forwarded "+
				"by config (see eve.hostfwd)", eveIfName, targetPort)
		}
		// Redirect command to localhost and the forwarded port.
		return cmd(uint16(targetHostPort))
	}
	// Temporarily establish port forwarding using SSH.
	targetIP, err := tc.sdnClient.GetEveIfIP(eveIfName)
	if err != nil {
		log.Errorf("failed to get EVE IP address: %v", err)
		return nil
	}
	localPort, err := utils.FindUnusedPort()
	if err != nil {
		log.Errorf("failed to find unused port number: %v", err)
		return nil
	}
	closeTunnel, err := tc.sdnClient.SSHPortForwarding(localPort, targetPort, targetIP)
	if err != nil {
		log.Errorf("failed to establish SSH port forwarding: %v", err)
		return nil
	}
	defer closeTunnel()
	return cmd(localPort)
}
