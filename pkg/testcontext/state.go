package testcontext

import (
	"reflect"

	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/emetric"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve-api/go/info"
	"github.com/lf-edge/eve-api/go/metrics"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type infoState struct {
	Dinfo  *info.ZInfoDevice
	Ainfo  []*info.ZInfoApp
	Niinfo []*info.ZInfoNetworkInstance
	Vinfo  []*info.ZInfoVolume
	Cinfo  []*info.ZInfoContentTree
	Binfo  []*info.ZInfoBlob

	AppMetrics             []*metrics.AppMetric
	NetworkInstanceMetrics []*metrics.ZMetricNetworkInstance
	VolumeMetrics          []*metrics.ZMetricVolume
	DeviceMetrics          *metrics.DeviceMetric

	LastInfoMessageTime *timestamppb.Timestamp
}

// State aggregates device state
type State struct {
	device     *device.Ctx
	deviceInfo *infoState
}

// InitState init State object for device
func InitState(device *device.Ctx) *State {
	return &State{device: device, deviceInfo: &infoState{}}
}

func (state *State) processInfo(infoMsg *info.ZInfoMsg) error {
	if infoMsg.DevId != state.device.GetID().String() {
		return nil
	}
	state.deviceInfo.LastInfoMessageTime = infoMsg.AtTimeStamp
	switch infoMsg.GetZtype() {
	case info.ZInfoTypes_ZiDevice:
		state.deviceInfo.Dinfo = infoMsg.GetDinfo()
	case info.ZInfoTypes_ZiApp:
		aInfo := infoMsg.GetAinfo()
		for ind, app := range state.deviceInfo.Ainfo {
			if app.AppID == aInfo.AppID {
				state.deviceInfo.Ainfo[ind] = aInfo
				return nil
			}
		}
		state.deviceInfo.Ainfo = append(state.deviceInfo.Ainfo, aInfo)
	case info.ZInfoTypes_ZiNetworkInstance:
		niInfo := infoMsg.GetNiinfo()
		for ind, ni := range state.deviceInfo.Niinfo {
			if ni.NetworkID == niInfo.NetworkID {
				state.deviceInfo.Niinfo[ind] = niInfo
				return nil
			}
		}
		state.deviceInfo.Niinfo = append(state.deviceInfo.Niinfo, niInfo)
	case info.ZInfoTypes_ZiVolume:
		vInfo := infoMsg.GetVinfo()
		for ind, volume := range state.deviceInfo.Vinfo {
			if volume.Uuid == vInfo.Uuid {
				state.deviceInfo.Vinfo[ind] = vInfo
				return nil
			}
		}
		state.deviceInfo.Vinfo = append(state.deviceInfo.Vinfo, vInfo)
	case info.ZInfoTypes_ZiContentTree:
		cInfo := infoMsg.GetCinfo()
		for ind, contentTree := range state.deviceInfo.Cinfo {
			if contentTree.Uuid == cInfo.Uuid {
				state.deviceInfo.Cinfo[ind] = cInfo
				return nil
			}
		}
		state.deviceInfo.Cinfo = append(state.deviceInfo.Cinfo, cInfo)
	case info.ZInfoTypes_ZiBlobList:
		bInfoList := infoMsg.GetBinfo()
	blobsLoop:
		for _, newBlob := range bInfoList.Blob {
			for ind, blob := range state.deviceInfo.Binfo {
				if blob.Sha256 == newBlob.Sha256 {
					state.deviceInfo.Binfo[ind] = newBlob
					continue blobsLoop
				}
			}
			state.deviceInfo.Binfo = append(state.deviceInfo.Binfo, newBlob)
		}
	}
	return nil
}

func (state *State) getProcessorInfo() einfo.HandlerFunc {
	return func(im *info.ZInfoMsg) bool {
		_ = state.processInfo(im)
		//process all events from controller
		return false
	}
}

// GetInfoProcessingFunction returns processing function for ZInfoMsg
func (state *State) GetInfoProcessingFunction() ProcInfoFunc {
	return func(infoMsg *info.ZInfoMsg) error {
		return state.processInfo(infoMsg)
	}
}

func (state *State) processMetric(metricMsg *metrics.ZMetricMsg) error {
	if metricMsg.DevID != state.device.GetID().String() {
		return nil
	}
	state.deviceInfo.AppMetrics = metricMsg.GetAm()
	state.deviceInfo.NetworkInstanceMetrics = metricMsg.GetNm()
	state.deviceInfo.VolumeMetrics = metricMsg.GetVm()
	state.deviceInfo.DeviceMetrics = metricMsg.GetDm()
	return nil
}

func (state *State) getProcessorMetric() emetric.HandlerFunc {
	return func(msg *metrics.ZMetricMsg) bool {
		_ = state.processMetric(msg)
		//process all events from controller
		return false
	}
}

// GetMetricProcessingFunction returns processing function for ZMetricMsg
func (state *State) GetMetricProcessingFunction() ProcMetricFunc {
	return func(metricMsg *metrics.ZMetricMsg) error {
		return state.processMetric(metricMsg)
	}
}

// GetDinfo get *info.ZInfoDevice from obtained info
func (state *State) GetDinfo() *info.ZInfoDevice {
	return state.deviceInfo.Dinfo
}

// GetAinfoSlice get []*info.ZInfoApp from obtained info
func (state *State) GetAinfoSlice() []*info.ZInfoApp {
	return state.deviceInfo.Ainfo
}

// GetNiinfoSlice get []*info.ZInfoNetworkInstance from obtained info
func (state *State) GetNiinfoSlice() []*info.ZInfoNetworkInstance {
	return state.deviceInfo.Niinfo
}

// GetVinfoSlice get []*info.ZInfoVolume from obtained info
func (state *State) GetVinfoSlice() []*info.ZInfoVolume {
	return state.deviceInfo.Vinfo
}

// GetCinfoSlice get []*info.ZInfoContentTree from obtained info
func (state *State) GetCinfoSlice() []*info.ZInfoContentTree {
	return state.deviceInfo.Cinfo
}

// GetBinfoSlice get []*info.ZInfoBlob from obtained info
func (state *State) GetBinfoSlice() []*info.ZInfoBlob {
	return state.deviceInfo.Binfo
}

// GetAppMetrics get []*metrics.AppMetric from obtained metrics
func (state *State) GetAppMetrics() []*metrics.AppMetric {
	return state.deviceInfo.AppMetrics
}

// GetNetworkInstanceMetrics get []*metrics.ZMetricNetworkInstance from obtained metrics
func (state *State) GetNetworkInstanceMetrics() []*metrics.ZMetricNetworkInstance {
	return state.deviceInfo.NetworkInstanceMetrics
}

// GetVolumeMetrics get []*metrics.ZMetricVolume from obtained metrics
func (state *State) GetVolumeMetrics() []*metrics.ZMetricVolume {
	return state.deviceInfo.VolumeMetrics
}

// GetDeviceMetrics get *metrics.DeviceMetric from obtained metrics
func (state *State) GetDeviceMetrics() *metrics.DeviceMetric {
	return state.deviceInfo.DeviceMetrics
}

// GetLastInfoTime get *timestamp.Timestamp for last received info
func (state *State) GetLastInfoTime() *timestamppb.Timestamp {
	return state.deviceInfo.LastInfoMessageTime
}

// LookUp access fields of State objects by path
// path contains address to lookup
// for example: LookUp("Dinfo.Network[0].IPAddrs[0]") will return first IP of first network of EVE
// All top fields to lookup in:
// Dinfo      *info.ZInfoDevice
// Ainfo      []*info.ZInfoApp
// Niinfo     []*info.ZInfoNetworkInstance
// Vinfo      []*info.ZInfoVolume
// Cinfo      []*info.ZInfoContentTree
// Binfo      []*info.ZInfoBlob
// Cipherinfo []*info.ZInfoCipher
// AppMetrics []*metrics.AppMetric
// NetworkInstanceMetrics []*metrics.ZMetricNetworkInstance
// VolumeMetrics []*metrics.ZMetricVolume
// DeviceMetrics *metrics.DeviceMetric
func (state *State) LookUp(path string) (value reflect.Value, err error) {
	value, err = utils.LookUp(state.deviceInfo, path)
	return
}

// CheckReady returns true in all needed information obtained from controller
func (state *State) CheckReady() bool {
	if state.deviceInfo.Dinfo == nil {
		return false
	}
	if state.deviceInfo.DeviceMetrics == nil {
		return false
	}
	return true
}
