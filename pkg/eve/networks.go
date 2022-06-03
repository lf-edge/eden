package eve

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/api/go/metrics"
)

//NetInstState stores state of network instance
type NetInstState struct {
	Name        string
	UUID        string
	NetworkType config.ZNetworkInstType
	CIDR        string
	Stats       string
	AdamState   string
	EveState    string
	Activated   bool
	deleted     bool
}

func netInstStateHeader() string {
	return "NAME\tUUID\tTYPE\tCIDR\tSTATS\tSTATE(ADAM)\tLAST_STATE(EVE)"
}

func (netInstStateObj *NetInstState) toString() string {
	return fmt.Sprintf("%s\t%s\t%v\t%s\t%s\t%s\t%s",
		netInstStateObj.Name, netInstStateObj.UUID,
		netInstStateObj.NetworkType, netInstStateObj.CIDR, netInstStateObj.Stats,
		netInstStateObj.AdamState, netInstStateObj.EveState)
}

func (ctx *State) initNetworks(ctrl controller.Cloud, dev *device.Ctx) error {
	ctx.networks = make(map[string]*NetInstState)
	for _, el := range dev.GetNetworkInstances() {
		ni, err := ctrl.GetNetworkInstanceConfig(el)
		if err != nil {
			return fmt.Errorf("no netInst in cloud %s: %s", el, err)
		}
		netInstStateObj := &NetInstState{
			Name:        ni.GetDisplayname(),
			UUID:        ni.Uuidandversion.Uuid,
			Stats:       "-",
			AdamState:   inControllerConfig,
			EveState:    "UNKNOWN",
			CIDR:        ni.Ip.Subnet,
			NetworkType: ni.InstType,
		}
		ctx.networks[ni.Uuidandversion.Uuid] = netInstStateObj
	}
	return nil
}

func (ctx *State) processNetworksByInfo(im *info.ZInfoMsg) {
	switch im.GetZtype() {
	case info.ZInfoTypes_ZiNetworkInstance:
		netInstStateObj, ok := ctx.networks[im.GetNiinfo().GetNetworkID()]
		if !ok {
			netInstStateObj = &NetInstState{
				Name:        im.GetNiinfo().GetDisplayname(),
				UUID:        im.GetNiinfo().GetNetworkID(),
				Stats:       "-",
				AdamState:   notInControllerConfig,
				EveState:    "UNKNOWN",
				NetworkType: (config.ZNetworkInstType)(int32(im.GetNiinfo().InstType)),
			}
			ctx.networks[im.GetNiinfo().GetNetworkID()] = netInstStateObj
		}
		netInstStateObj.EveState = im.GetNiinfo().State.String()
		netInstStateObj.Activated = im.GetNiinfo().Activated

		if len(im.GetNiinfo().GetNetworkErr()) > 0 {
			netInstStateObj.EveState = fmt.Sprintf("%s ERRORS: %s", im.GetNiinfo().GetState().String(), im.GetNiinfo().GetNetworkErr())
		} else {
			if im.GetNiinfo().State == info.ZNetworkInstanceState_ZNETINST_STATE_ONLINE || netInstStateObj.Activated {
				netInstStateObj.EveState = "ACTIVATED"
			} else {
				netInstStateObj.EveState = "NOT_ACTIVATED"
			}
		}
		// XXX Guard against old EVE which doesn't send state
		// sends INIT state when deleting network instance
		if !netInstStateObj.Activated &&
			im.GetNiinfo().State != info.ZNetworkInstanceState_ZNETINST_STATE_INIT &&
			netInstStateObj.AdamState == notInControllerConfig {
			netInstStateObj.deleted = true
		}

		if im.GetNiinfo().State == info.ZNetworkInstanceState_ZNETINST_STATE_UNSPECIFIED &&
			netInstStateObj.AdamState == notInControllerConfig {
			netInstStateObj.deleted = true
		}
	}
}

func (ctx *State) processNetworksByMetric(msg *metrics.ZMetricMsg) {
	if networkMetrics := msg.GetNm(); networkMetrics != nil {
		for _, networkMetric := range networkMetrics {
			// XXX use [uuid] instead of loop
			for _, el := range ctx.networks {
				if networkMetric.NetworkID == el.UUID {
					el.Stats = networkMetric.GetNetworkStats().String()
					break
				}
			}
		}
	}
}

//NetList prints networks
func (ctx *State) NetList() error {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	if _, err := fmt.Fprintln(w, netInstStateHeader()); err != nil {
		return err
	}
	netInstStatesSlice := make([]*NetInstState, 0, len(ctx.Networks()))
	netInstStatesSlice = append(netInstStatesSlice, ctx.Networks()...)
	sort.SliceStable(netInstStatesSlice, func(i, j int) bool {
		return netInstStatesSlice[i].Name < netInstStatesSlice[j].Name
	})
	for _, el := range netInstStatesSlice {
		if _, err := fmt.Fprintln(w, el.toString()); err != nil {
			return err
		}
	}
	return w.Flush()
}
