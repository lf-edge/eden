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
	states   map[*device.Ctx]*state
	stopTime time.Time
	addTime  time.Duration
}

//NewTestContext creates new TestContext
func NewTestContext() *TestContext {
	viperLoaded, err := utils.LoadConfigFile("")
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
func (ctx *TestContext) GetNodeDescriptions() (nodes []*EdgeNodeDescription) {
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
func (ctx *TestContext) GetController() controller.Cloud {
	if ctx.cloud == nil {
		log.Fatal("Controller not initialized")
	}
	return ctx.cloud
}

//InitProject init project object with defined name
func (ctx *TestContext) InitProject(name string) {
	ctx.project = &Project{name: name}
}

//AddEdgeNodesFromDescription adds EdgeNodes from description in test.eve param
func (ctx *TestContext) AddEdgeNodesFromDescription() {
	for _, node := range ctx.GetNodeDescriptions() {
		edgeNode := ctx.GetController().GetEdgeNode(node.Name)
		if edgeNode == nil {
			edgeNode = ctx.NewEdgeNode(ctx.WithNodeDescription(node), ctx.WithCurrentProject())
		} else {
			ctx.UpdateEdgeNode(edgeNode, ctx.WithCurrentProject(), ctx.WithDeviceModel(node.Model))
		}

		if edgeNode.GetState() == device.NotOnboarded {
			log.Fatal("Node is not onboarded now")
		}

		ctx.AddNode(edgeNode)
	}
}

type GetEdgeNodeOpts func(*device.Ctx) bool

//FilterByName check EdgeNode name
func (ctx *TestContext) FilterByName(name string) GetEdgeNodeOpts {
	return func(d *device.Ctx) bool {
		return d.GetName() == name
	}
}

//WithTest assign *testing.T for device
func (ctx *TestContext) WithTest(t *testing.T) GetEdgeNodeOpts {
	return func(d *device.Ctx) bool {
		ctx.tests[d] = t
		return true
	}
}

//GetEdgeNode return node from context
func (ctx *TestContext) GetEdgeNode(opts ...GetEdgeNodeOpts) *device.Ctx {
Node:
	for _, el := range ctx.nodes {
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
func (ctx *TestContext) AddNode(node *device.Ctx) {
	ctx.nodes = append(ctx.nodes, node)
}

//UpdateEdgeNode update edge node
func (ctx *TestContext) UpdateEdgeNode(edgeNode *device.Ctx, opts ...EdgeNodeOption) {
	for _, opt := range opts {
		opt(edgeNode)
	}
	ctx.ConfigSync(edgeNode)
}

//NewEdgeNode creates edge node
func (ctx *TestContext) NewEdgeNode(opts ...EdgeNodeOption) *device.Ctx {
	d := device.CreateEdgeNode()
	for _, opt := range opts {
		opt(d)
	}
	if ctx.project == nil {
		log.Fatal("You must setup project before add node")
	}
	ctx.ConfigSync(d)
	return d
}

//ConfigSync send config to controller
func (ctx *TestContext) ConfigSync(edgeNode *device.Ctx) {
	if edgeNode.GetState() == device.NotOnboarded {
		if err := ctx.GetController().OnBoardDev(edgeNode); err != nil {
			log.Fatalf("OnBoardDev %s", err)
		}
	} else {
		log.Debug("Device %s onboarded", edgeNode.GetID().String())
	}
	if err := ctx.GetController().ConfigSync(edgeNode); err != nil {
		log.Fatalf("Cannot send config of %s", edgeNode.GetName())
	}
}

//ExpandOnSuccess adds additional time to global timeout on every success check
func (tc *TestContext) ExpandOnSuccess(secs int) {
	tc.addTime = time.Duration(secs) * time.Second
}

//WaitForProc blocking execution until the time elapses or all Procs gone
//returns error on timeout
func (tc *TestContext) WaitForProc(secs int) {
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
				tc.procBus.clean()
				if len(tc.tests) == 0 {
					log.Fatalf("WaitForProc terminated by timeout %s", timeout)
				}
				for _, el := range tc.tests {
					el.Errorf("WaitForProc terminated by timeout %s", timeout)
				}
				return
			}
		}
	}
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

//StartTrackingState init function for state monitoring
//if onlyNewElements set no use old information from controller
func (tc *TestContext) StartTrackingState(onlyNewElements bool) {
	tc.states = map[*device.Ctx]*state{}
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

//WaitForState wait for state initialization from controller
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
			} else {
				time.Sleep(defaults.DefaultRepeatTimeout)
			}
		}
	}()
	select {
	case <-waitChan:
		if el, isOk := tc.tests[edgeNode]; !isOk {
			log.Println("done waiting for state")
		} else {
			el.Logf("done waiting for state")
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

//GetState returns state object for edgeNode
func (tc *TestContext) GetState(edgeNode *device.Ctx) *state {
	return tc.states[edgeNode]
}
