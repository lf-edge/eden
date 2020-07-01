package projects

import (
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/emetric"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/api/go/metrics"
	"net"
)

type infoState struct {
	device          *info.ZInfoDevice
	app             []*info.ZInfoApp
	networkInstance []*info.ZInfoNetworkInstance
	volume          []*info.ZInfoVolume
	contentTree     []*info.ZInfoContentTree
	blobs           []*info.ZInfoBlob

	appMetrics             []*metrics.AppMetric
	networkInstanceMetrics []*metrics.ZMetricNetworkInstance
	vmMetrics              []*metrics.ZMetricVolume
	deviceMetric           *metrics.DeviceMetric
}

type state struct {
	device     *device.Ctx
	deviceInfo *infoState
}

//InitState init state object for device
func InitState(device *device.Ctx) *state {
	return &state{device: device, deviceInfo: &infoState{}}
}

func (state *state) processInfo(infoMsg *info.ZInfoMsg) error {
	if infoMsg.DevId != state.device.GetID().String() {
		return nil
	}
	switch infoMsg.GetZtype() {
	case info.ZInfoTypes_ZiDevice:
		state.deviceInfo.device = infoMsg.GetDinfo()
	case info.ZInfoTypes_ZiApp:
		aInfo := infoMsg.GetAinfo()
		for ind, app := range state.deviceInfo.app {
			if app.AppID == aInfo.AppID {
				state.deviceInfo.app[ind] = aInfo
				return nil
			}
		}
		state.deviceInfo.app = append(state.deviceInfo.app, aInfo)
	case info.ZInfoTypes_ZiNetworkInstance:
		niInfo := infoMsg.GetNiinfo()
		for ind, ni := range state.deviceInfo.networkInstance {
			if ni.NetworkID == niInfo.NetworkID {
				state.deviceInfo.networkInstance[ind] = niInfo
				return nil
			}
		}
		state.deviceInfo.networkInstance = append(state.deviceInfo.networkInstance, niInfo)
	case info.ZInfoTypes_ZiVolume:
		vInfo := infoMsg.GetVinfo()
		for ind, volume := range state.deviceInfo.volume {
			if volume.Uuid == vInfo.Uuid {
				state.deviceInfo.volume[ind] = vInfo
				return nil
			}
		}
		state.deviceInfo.volume = append(state.deviceInfo.volume, vInfo)
	case info.ZInfoTypes_ZiContentTree:
		cInfo := infoMsg.GetCinfo()
		for ind, contentTree := range state.deviceInfo.contentTree {
			if contentTree.Uuid == cInfo.Uuid {
				state.deviceInfo.contentTree[ind] = cInfo
				return nil
			}
		}
		state.deviceInfo.contentTree = append(state.deviceInfo.contentTree, cInfo)
	case info.ZInfoTypes_ZiBlobList:
		bInfoList := infoMsg.GetBinfo()
	blobsLoop:
		for ind, blob := range state.deviceInfo.blobs {
			for _, newBlob := range bInfoList.Blob {
				if blob.Sha256 == newBlob.Sha256 {
					state.deviceInfo.blobs[ind] = newBlob
					continue blobsLoop
				}
			}
			state.deviceInfo.blobs = append(state.deviceInfo.blobs, blob)
		}
	}
	return nil
}

func (state *state) getProcessorInfo() einfo.HandlerFunc {
	return func(im *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface, infoType einfo.ZInfoType) bool {
		_ = state.processInfo(im)
		//process all events from controller
		return false
	}
}

//GetInfoProcessingFunction returns processing function for ZInfoMsg
func (state *state) GetInfoProcessingFunction() ProcInfoFunc {
	return func(infoMsg *info.ZInfoMsg) error {
		return state.processInfo(infoMsg)
	}
}

func (state *state) processMetric(metricMsg *metrics.ZMetricMsg) error {
	if metricMsg.DevID != state.device.GetID().String() {
		return nil
	}
	state.deviceInfo.appMetrics = metricMsg.GetAm()
	state.deviceInfo.networkInstanceMetrics = metricMsg.GetNm()
	state.deviceInfo.vmMetrics = metricMsg.GetVm()
	state.deviceInfo.deviceMetric = metricMsg.GetDm()
	return nil
}

func (state *state) getProcessorMetric() emetric.HandlerFunc {
	return func(msg *metrics.ZMetricMsg) bool {
		_ = state.processMetric(msg)
		//process all events from controller
		return false
	}
}

//GetMetricProcessingFunction returns processing function for ZMetricMsg
func (state *state) GetMetricProcessingFunction() ProcMetricFunc {
	return func(metricMsg *metrics.ZMetricMsg) error {
		return state.processMetric(metricMsg)
	}
}

//GetDinfo get *info.ZInfoDevice from obtained info
func (state *state) GetDinfo() *info.ZInfoDevice {
	return state.deviceInfo.device
}

//GetAinfoSlice get []*info.ZInfoApp from obtained info
func (state *state) GetAinfoSlice() []*info.ZInfoApp {
	return state.deviceInfo.app
}

//GetNiinfoSlice get []*info.ZInfoNetworkInstance from obtained info
func (state *state) GetNiinfoSlice() []*info.ZInfoNetworkInstance {
	return state.deviceInfo.networkInstance
}

//GetVinfoSlice get []*info.ZInfoVolume from obtained info
func (state *state) GetVinfoSlice() []*info.ZInfoVolume {
	return state.deviceInfo.volume
}

//GetCinfoSlice get []*info.ZInfoContentTree from obtained info
func (state *state) GetCinfoSlice() []*info.ZInfoContentTree {
	return state.deviceInfo.contentTree
}

//GetBinfoSlice get []*info.ZInfoBlob from obtained info
func (state *state) GetBinfoSlice() []*info.ZInfoBlob {
	return state.deviceInfo.blobs
}

//GetAm get []*metrics.AppMetric from obtained metrics
func (state *state) GetAm() []*metrics.AppMetric {
	return state.deviceInfo.appMetrics
}

//GetNm get []*metrics.ZMetricNetworkInstance from obtained metrics
func (state *state) GetNm() []*metrics.ZMetricNetworkInstance {
	return state.deviceInfo.networkInstanceMetrics
}

//GetVm get []*metrics.ZMetricVolume from obtained metrics
func (state *state) GetVm() []*metrics.ZMetricVolume {
	return state.deviceInfo.vmMetrics
}

//GetDm get *metrics.DeviceMetric from obtained metrics
func (state *state) GetDm() *metrics.DeviceMetric {
	return state.deviceInfo.deviceMetric
}

//CheckEVERemote returns true if we use remote EVE
func (state *state) CheckEVERemote() bool {
	if state.device.GetDevModel() == defaults.DefaultRPIModel {
		return true
	}
	//we also should check connection to remote EVE in cloud
	return false
}

//GetEVEIPs returns EVE IPs from info
func (state *state) GetEVEIPs() (ips []string) {
	if state.CheckEVERemote() {
		if dInfo := state.GetDinfo(); dInfo != nil {
			if len(dInfo.Network) > 0 {
				if dInfo.Network[0] != nil {
					if len(dInfo.Network[0].IPAddrs) > 0 {
						ip, _, err := net.ParseCIDR(dInfo.Network[0].IPAddrs[0])
						if err != nil {
							return nil
						}
						ips = append(ips, ip.To4().String())
					}
				}
			}
		}
	} else {
		return []string{"127.0.0.1"}
	}
	return
}

//CheckReady returns true in all needed information obtained from controller
func (state *state) CheckReady() bool {
	if state.deviceInfo.device == nil {
		return false
	}
	if state.deviceInfo.deviceMetric == nil {
		return false
	}
	return true
}
