package projects

import (
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eve/api/go/info"
)

type infoState struct {
	device          *info.ZInfoDevice
	app             []*info.ZInfoApp
	networkInstance []*info.ZInfoNetworkInstance
	volume          []*info.ZInfoVolume
	contentTree     []*info.ZInfoContentTree
	blobs           []*info.ZInfoBlob
}

type state struct {
	device     *device.Ctx
	deviceInfo *infoState
}

//InitState init state object for device
func InitState(device *device.Ctx) *state {
	return &state{device: device, deviceInfo: &infoState{}}
}

func (state *state) process(infoMsg *info.ZInfoMsg) error {
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
		return state.process(im) == nil
	}
}

//GetInfoProcessingFunction returns processing function for ZInfoMsg
func (state *state) GetInfoProcessingFunction() ProcInfoFunc {
	return func(infoMsg *info.ZInfoMsg) error {
		return state.process(infoMsg)
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

//CheckReady returns true in all needed information obtained from controller
func (state *state) CheckReady() bool {
	if state.deviceInfo.device == nil {
		return false
	}
	return true
}
