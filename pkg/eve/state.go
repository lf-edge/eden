package eve

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/emetric"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/api/go/metrics"
	log "github.com/sirupsen/logrus"
)

const (
	inControllerConfig    = "IN_CONFIG"
	notInControllerConfig = "NOT_IN_CONFIG"
	stateFileTemplate     = "state_store_%s.json"
)

// State stores representation of EVE state
// we should assign InfoCallback and MetricCallback to update state
type State struct {
	Applications map[string]*AppInstState
	Networks     map[string]*NetInstState
	Volumes      map[string]*VolInstState
	EveState     *NodeState
	device       *device.Ctx
}

// Init State object with controller and device
func Init(ctrl controller.Cloud, dev *device.Ctx) (ctx *State) {
	ctx = &State{device: dev}
	if err := ctx.initApplications(ctrl, dev); err != nil {
		log.Fatalf("EVE State initApplications error: %s", err)
	}
	if err := ctx.initVolumes(ctrl, dev); err != nil {
		log.Fatalf("EVE State initVolumes error: %s", err)
	}
	if err := ctx.initNetworks(ctrl, dev); err != nil {
		log.Fatalf("EVE State initNetworks error: %s", err)
	}
	if err := ctx.initNodeState(ctrl, dev); err != nil {
		log.Fatalf("EVE State initNodeState error: %s", err)
	}
	if err := ctx.Load(); err != nil {
		log.Fatalf("EVE State Load error: %s", err)
	}
	return
}

func (ctx *State) getStateFile() (string, error) {
	edenDir, err := utils.DefaultEdenDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(edenDir, fmt.Sprintf(stateFileTemplate, ctx.device.GetID().String())), nil
}

// Store state into file
func (ctx *State) Store() error {
	data, err := json.Marshal(ctx)
	if err != nil {
		return err
	}
	stateFile, err := ctx.getStateFile()
	if err != nil {
		return err
	}
	return os.WriteFile(stateFile, data, 0600)
}

// Load state from file
func (ctx *State) Load() error {
	stateFile, err := ctx.getStateFile()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(stateFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var obj *State
	err = json.Unmarshal(data, &obj)
	if err != nil {
		return err
	}
	ctx.applyOldStateApps(obj)
	ctx.applyOldStateNetworks(obj)
	ctx.applyOldStateVolumes(obj)
	ctx.applyOldStateNodeState(obj)
	return nil
}

// Prepared returns true if we have enough info to work
func (ctx *State) Prepared() bool {
	return ctx.EveState.LastSeen.Unix() != 0
}

// NotDeletedApplications extracts AppInstState  which are not marked as deleted
func (ctx *State) NotDeletedApplications() []*AppInstState {
	v := make([]*AppInstState, 0, len(ctx.Applications))
	for _, value := range ctx.Applications {
		if !value.Deleted {
			v = append(v, value)
		}
	}
	return v
}

// NotDeletedNetworks extracts NetInstState  which are not marked as deleted
func (ctx *State) NotDeletedNetworks() []*NetInstState {
	v := make([]*NetInstState, 0, len(ctx.Networks))
	for _, value := range ctx.Networks {
		if !value.Deleted {
			v = append(v, value)
		}
	}
	return v
}

// NotDeletedVolumes extracts VolInstState which are not marked as deleted
func (ctx *State) NotDeletedVolumes() []*VolInstState {
	v := make([]*VolInstState, 0, len(ctx.Volumes))
	for _, value := range ctx.Volumes {
		if !value.Deleted {
			v = append(v, value)
		}
	}
	return v
}

// NodeState returns NodeState
func (ctx *State) NodeState() *NodeState {
	return ctx.EveState
}

// InfoCallback should be assigned to feed new values from info messages into state
func (ctx *State) InfoCallback() einfo.HandlerFunc {
	return func(msg *info.ZInfoMsg) bool {
		ctx.processVolumesByInfo(msg)
		ctx.processApplicationsByInfo(msg)
		ctx.processNetworksByInfo(msg)
		ctx.processNodeStateByInfo(msg)
		return false
	}
}

// MetricCallback should be assigned to feed new values from metric messages into state
func (ctx *State) MetricCallback() emetric.HandlerFunc {
	return func(msg *metrics.ZMetricMsg) bool {
		ctx.processVolumesByMetric(msg)
		ctx.processApplicationsByMetric(msg)
		ctx.processNetworksByMetric(msg)
		ctx.processNodeStateByMetric(msg)
		return false
	}
}
