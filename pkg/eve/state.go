package eve

import (
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/emetric"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/projects"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/api/go/metrics"
	log "github.com/sirupsen/logrus"
)

const (
	inControllerConfig    = "IN_CONFIG"
	notInControllerConfig = "NOT_IN_CONFIG"
)

//State stores representation of EVE state
//we should assign InfoCallback and MetricCallback to update state
type State struct {
	applications   map[string]*AppInstState
	networks       map[string]*NetInstState
	volumes        map[string]*VolInstState
	infoAndMetrics *projects.State
	device         *device.Ctx
}

//Init State object with controller and device
func Init(ctrl controller.Cloud, dev *device.Ctx) (ctx *State) {
	ctx = &State{device: dev, infoAndMetrics: projects.InitState(dev)}
	ctx.applications = make(map[string]*AppInstState)
	ctx.networks = make(map[string]*NetInstState)
	if err := ctx.initApplications(ctrl, dev); err != nil {
		log.Fatalf("EVE State initApplications error: %s", err)
	}
	if err := ctx.initVolumes(ctrl, dev); err != nil {
		log.Fatalf("EVE State initVolumes error: %s", err)
	}
	if err := ctx.initNetworks(ctrl, dev); err != nil {
		log.Fatalf("EVE State initNetworks error: %s", err)
	}
	return
}

//InfoAndMetrics returns last info and metric objects
func (ctx *State) InfoAndMetrics() *projects.State {
	return ctx.infoAndMetrics
}

//Applications extracts applications states
func (ctx *State) Applications() []*AppInstState {
	v := make([]*AppInstState, 0, len(ctx.applications))
	for _, value := range ctx.applications {
		if !value.deleted {
			v = append(v, value)
		}
	}
	return v
}

//Networks extracts networks states
func (ctx *State) Networks() []*NetInstState {
	v := make([]*NetInstState, 0, len(ctx.networks))
	for _, value := range ctx.networks {
		if !value.deleted {
			v = append(v, value)
		}
	}
	return v
}

//Volumes extracts volumes states
func (ctx *State) Volumes() []*VolInstState {
	v := make([]*VolInstState, 0, len(ctx.volumes))
	for _, value := range ctx.volumes {
		if !value.deleted {
			v = append(v, value)
		}
	}
	return v
}

//InfoCallback should be assigned to feed new values from info messages into state
func (ctx *State) InfoCallback() einfo.HandlerFunc {
	return func(msg *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface) bool {
		ctx.processVolumesByInfo(msg)
		ctx.processApplicationsByInfo(msg)
		ctx.processNetworksByInfo(msg)
		if err := ctx.infoAndMetrics.GetInfoProcessingFunction()(msg); err != nil {
			log.Fatalf("EVE State GetInfoProcessingFunction error: %s", err)
		}
		return false
	}
}

//MetricCallback should be assigned to feed new values from metric messages into state
func (ctx *State) MetricCallback() emetric.HandlerFunc {
	return func(msg *metrics.ZMetricMsg) bool {
		ctx.processVolumesByMetric(msg)
		ctx.processApplicationsByMetric(msg)
		ctx.processNetworksByMetric(msg)
		if err := ctx.infoAndMetrics.GetMetricProcessingFunction()(msg); err != nil {
			log.Fatalf("EVE State GetMetricProcessingFunction error: %s", err)
		}
		return false
	}
}
