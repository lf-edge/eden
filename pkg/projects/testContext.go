package projects

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/adam"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"strings"
	"testing"
	"time"
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
	cloud    controller.Cloud
	project  *Project
	nodes    []*device.Ctx
	procBus  *processingBus
	tests    map[*device.Ctx]*testing.T
	states   map[*device.Ctx]*State
	stopTime time.Time
	addTime  time.Duration
}

//NewTestContext creates new TestContext
func NewTestContext() *TestContext {
	var err error
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
			log.Fatal(err)
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
		cloud: ctx,
		tests: map[*device.Ctx]*testing.T{},
	}
	tstCtx.procBus = initBus(tstCtx)
	return tstCtx
}

//GetNodeDescriptions returns list of nodes from config
func (tc *TestContext) GetNodeDescriptions() (nodes []*EdgeNodeDescription) {
	eveList := viper.GetStringMap("test.eve")
	for name := range eveList {
		eveKey := viper.GetString(fmt.Sprintf("test.eve.%s.onboard-cert", name))
		eveSerial := viper.GetString(fmt.Sprintf("test.eve.%s.serial", name))
		eveModel := viper.GetString(fmt.Sprintf("test.eve.%s.model", name))
		nodes = append(nodes, &EdgeNodeDescription{Name: name, Key: eveKey, Serial: eveSerial, Model: eveModel})
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
			tc.UpdateEdgeNode(edgeNode, tc.WithCurrentProject(), tc.WithDeviceModel(node.Model))
		}

		if edgeNode.GetState() == device.NotOnboarded {
			log.Fatal("Node is not onboarded now")
		}

		tc.AddNode(edgeNode)
	}
}

//GetEdgeNodeOpts pattern to pass device modifications
type GetEdgeNodeOpts func(*device.Ctx) bool

//FilterByName check EdgeNode name
func (tc *TestContext) FilterByName(name string) GetEdgeNodeOpts {
	return func(d *device.Ctx) bool {
		return d.GetName() == name
	}
}

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
		log.Fatalf("Cannot send config of %s", edgeNode.GetName())
	}
}

//ExpandOnSuccess adds additional time to global timeout on every success check
func (tc *TestContext) ExpandOnSuccess(secs int) {
	tc.addTime = time.Duration(secs) * time.Second
}

//WaitForProcWithErrorCallback blocking execution until the time elapses or all Procs gone
//and fires callback in case of timeout
func (tc *TestContext) WaitForProcWithErrorCallback(secs int, callback TimeoutCallback) {
	defer func() { tc.addTime = 0 }() //reset addTime on exit
	timeout := time.Duration(secs) * time.Second
	tc.stopTime = time.Now().Add(timeout)
	ticker := time.NewTicker(defaults.DefaultRepeatTimeout)
	defer ticker.Stop()
	waitChan := make(chan struct{})
	go func() {
		tc.procBus.wg.Wait()
		close(waitChan)
	}()
	for {
		select {
		case <-waitChan:
			for node, el := range tc.tests {
				el.Logf("done for device %s", node.GetName())
			}
			return
		case <-ticker.C:
			if time.Now().After(tc.stopTime) {
				callback()
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
		tc.procBus.clean()
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
		log.Fatalf("edgeNode not found with name %s", edgeNode.GetName())
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
