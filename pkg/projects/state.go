package projects

import (
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/emetric"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/api/go/metrics"
	"reflect"
)

type infoState struct {
	Dinfo  *info.ZInfoDevice
	Ainfo  []*info.ZInfoApp
	Niinfo []*info.ZInfoNetworkInstance
	Vinfo  []*info.ZInfoVolume
	Cinfo  []*info.ZInfoContentTree
	Binfo  []*info.ZInfoBlob

	Am []*metrics.AppMetric
	Nm []*metrics.ZMetricNetworkInstance
	Vm []*metrics.ZMetricVolume
	Dm *metrics.DeviceMetric
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
		for ind, blob := range state.deviceInfo.Binfo {
			for _, newBlob := range bInfoList.Blob {
				if blob.Sha256 == newBlob.Sha256 {
					state.deviceInfo.Binfo[ind] = newBlob
					continue blobsLoop
				}
			}
			state.deviceInfo.Binfo = append(state.deviceInfo.Binfo, blob)
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
	state.deviceInfo.Am = metricMsg.GetAm()
	state.deviceInfo.Nm = metricMsg.GetNm()
	state.deviceInfo.Vm = metricMsg.GetVm()
	state.deviceInfo.Dm = metricMsg.GetDm()
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
	return state.deviceInfo.Dinfo
}

//GetAinfoSlice get []*info.ZInfoApp from obtained info
func (state *state) GetAinfoSlice() []*info.ZInfoApp {
	return state.deviceInfo.Ainfo
}

//GetNiinfoSlice get []*info.ZInfoNetworkInstance from obtained info
func (state *state) GetNiinfoSlice() []*info.ZInfoNetworkInstance {
	return state.deviceInfo.Niinfo
}

//GetVinfoSlice get []*info.ZInfoVolume from obtained info
func (state *state) GetVinfoSlice() []*info.ZInfoVolume {
	return state.deviceInfo.Vinfo
}

//GetCinfoSlice get []*info.ZInfoContentTree from obtained info
func (state *state) GetCinfoSlice() []*info.ZInfoContentTree {
	return state.deviceInfo.Cinfo
}

//GetBinfoSlice get []*info.ZInfoBlob from obtained info
func (state *state) GetBinfoSlice() []*info.ZInfoBlob {
	return state.deviceInfo.Binfo
}

//GetAm get []*metrics.AppMetric from obtained metrics
func (state *state) GetAm() []*metrics.AppMetric {
	return state.deviceInfo.Am
}

//GetNm get []*metrics.ZMetricNetworkInstance from obtained metrics
func (state *state) GetNm() []*metrics.ZMetricNetworkInstance {
	return state.deviceInfo.Nm
}

//GetVm get []*metrics.ZMetricVolume from obtained metrics
func (state *state) GetVm() []*metrics.ZMetricVolume {
	return state.deviceInfo.Vm
}

//GetDm get *metrics.DeviceMetric from obtained metrics
func (state *state) GetDm() *metrics.DeviceMetric {
	return state.deviceInfo.Dm
}

//LookUp access fields of state objects by path
//path contains address to lookup
//for example: LookUp("Dinfo.Network[0].IPAddrs[0]") will return first IP of first network of EVE
//All top fields to lookup in:
//Dinfo      *info.ZInfoDevice
//Ainfo      []*info.ZInfoApp
//Niinfo     []*info.ZInfoNetworkInstance
//Vinfo      []*info.ZInfoVolume
//Cinfo      []*info.ZInfoContentTree
//Binfo      []*info.ZInfoBlob
//Cipherinfo []*info.ZInfoCipher
//Am []*metrics.AppMetric
//Nm []*metrics.ZMetricNetworkInstance
//Vm []*metrics.ZMetricVolume
//Dm *metrics.DeviceMetric
func (state *state) LookUp(path string) (value reflect.Value, err error) {
	value, err = utils.LookUp(state.deviceInfo, path)
	return
}

//CheckReady returns true in all needed information obtained from controller
func (state *state) CheckReady() bool {
	if state.deviceInfo.Dinfo == nil {
		return false
	}
	if state.deviceInfo.Dm == nil {
		return false
	}
	return true
}
