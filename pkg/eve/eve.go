package eve

import (
	"time"

	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/api/go/metrics"
)

// NodeState describes state of edge node
type NodeState struct {
	UsedMem           uint32
	AvailMem          uint32
	UsedPercentageMem float64

	LastRebootTime   time.Time
	LastRebootReason string

	// interface to ip mapping
	RemoteIPs map[string][]string

	LastSeen time.Time

	Version string
}

func (ctx *State) initNodeState(_ controller.Cloud, _ *device.Ctx) error {
	ctx.EveState = &NodeState{}
	return nil
}

func (ctx *State) applyOldStateNodeState(state *State) {
	ctx.EveState = state.EveState
}

func (ctx *State) processNodeStateByInfo(msg *info.ZInfoMsg) {
	infoTime := msg.AtTimeStamp.AsTime()
	if infoTime.After(ctx.EveState.LastSeen) {
		ctx.EveState.LastSeen = infoTime
	}
	if deviceInfo := msg.GetDinfo(); deviceInfo != nil {
		ctx.EveState.RemoteIPs = make(map[string][]string)
		for _, nw := range deviceInfo.Network {
			ctx.EveState.RemoteIPs[nw.LocalName] = nw.IPAddrs
		}
		ctx.EveState.LastRebootTime = deviceInfo.LastRebootTime.AsTime()
		ctx.EveState.LastRebootReason = deviceInfo.LastRebootReason
		if len(deviceInfo.SwList) > 0 {
			ctx.EveState.Version = deviceInfo.SwList[0].ShortVersion
		}
	}
}

func (ctx *State) processNodeStateByMetric(msg *metrics.ZMetricMsg) {
	metricTime := msg.AtTimeStamp.AsTime()
	if metricTime.After(ctx.EveState.LastSeen) {
		ctx.EveState.LastSeen = metricTime
	}
	if deviceMetric := msg.GetDm(); deviceMetric != nil {
		ctx.EveState.AvailMem = deviceMetric.Memory.GetAvailMem()
		ctx.EveState.UsedMem = deviceMetric.Memory.GetUsedMem()
		ctx.EveState.UsedPercentageMem = deviceMetric.Memory.GetUsedPercentage()
	}
}
