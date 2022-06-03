package eve

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/api/go/metrics"
)

//VolInstState stores state of volumes
type VolInstState struct {
	Name          string
	UUID          string
	Image         string
	VolumeType    config.Format
	Size          string
	MaxSize       string
	AdamState     string
	EveState      string
	Ref           string
	contentTreeID string
	MountPoint    string
	originType    config.VolumeContentOriginType
	deleted       bool
}

func volInstStateHeader() string {
	return "NAME\tUUID\tREF\tIMAGE\tTYPE\tSIZE\tMAX_SIZE\tMOUNT\tSTATE(ADAM)\tLAST_STATE(EVE)"
}

func (volInstStateObj *VolInstState) toString() string {
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%v\t%s\t%s\t%s\t%s\t%s",
		volInstStateObj.Name, volInstStateObj.UUID, volInstStateObj.Ref, volInstStateObj.Image,
		volInstStateObj.VolumeType, volInstStateObj.Size, volInstStateObj.MaxSize, volInstStateObj.MountPoint,
		volInstStateObj.AdamState, volInstStateObj.EveState)
}

func (ctx *State) initVolumes(ctrl controller.Cloud, dev *device.Ctx) error {
	ctx.volumes = make(map[string]*VolInstState)
	for _, el := range dev.GetVolumes() {
		vi, err := ctrl.GetVolume(el)
		if err != nil {
			return fmt.Errorf("no Volume in cloud %s: %s", el, err)
		}
		contentTreeID := vi.GetOrigin().GetDownloadContentTreeID()
		image := "-"
		iFormat := config.Format_RAW
		if vi.GetOrigin().GetType() == config.VolumeContentOriginType_VCOT_DOWNLOAD {
			ct, err := ctrl.GetContentTree(contentTreeID)
			if err != nil {
				return fmt.Errorf("no ContentTree in cloud %s: %s", contentTreeID, err)
			}
			image = ct.GetURL()
			iFormat = ct.Iformat
		}
		var ref []string
		var mountPoint []string
	appInstanceLoop:
		for _, id := range dev.GetApplicationInstances() {
			appInstanceConfig, err := ctrl.GetApplicationInstanceConfig(id)
			if err != nil {
				return fmt.Errorf("no Application instance in cloud %s: %s", contentTreeID, err)
			}
			for _, volumeRef := range appInstanceConfig.VolumeRefList {
				if volumeRef.Uuid == vi.GetUuid() {
					ref = append(ref, fmt.Sprintf("app: %s", appInstanceConfig.Displayname))
					mountPoint = append(mountPoint, volumeRef.MountDir)
					break appInstanceLoop
				}
			}
		}
		volInstStateObj := &VolInstState{
			Name:          vi.GetDisplayName(),
			UUID:          vi.GetUuid(),
			Image:         image,
			VolumeType:    iFormat,
			AdamState:     inControllerConfig,
			EveState:      "UNKNOWN",
			Size:          "-",
			MaxSize:       "-",
			MountPoint:    strings.Join(mountPoint, ";"),
			Ref:           strings.Join(ref, ";"),
			contentTreeID: contentTreeID,
			originType:    vi.GetOrigin().GetType(),
		}
		ctx.volumes[vi.GetUuid()] = volInstStateObj
	}
	return nil
}

func (ctx *State) processVolumesByInfo(im *info.ZInfoMsg) {
	switch im.GetZtype() {
	case info.ZInfoTypes_ZiVolume:
		infoObject := im.GetVinfo()
		volInstStateObj, ok := ctx.volumes[infoObject.GetUuid()]
		if !ok {
			volInstStateObj = &VolInstState{
				Name:       infoObject.GetDisplayName(),
				UUID:       infoObject.GetUuid(),
				AdamState:  notInControllerConfig,
				EveState:   infoObject.State.String(),
				Size:       "-",
				MaxSize:    "-",
				MountPoint: "-",
				Ref:        "-",
			}
			ctx.volumes[infoObject.GetUuid()] = volInstStateObj
		}
		volInstStateObj.deleted =
			infoObject.DisplayName == "" || infoObject.State == info.ZSwState_INVALID
		if volInstStateObj.VolumeType != config.Format_FmtUnknown &&
			volInstStateObj.VolumeType != config.Format_CONTAINER {
			//we cannot use limits for container or unknown types
			if infoObject.GetResources() != nil {
				//MaxSizeBytes to show in MAX_SIZE column
				if maxSize := infoObject.GetResources().GetMaxSizeBytes(); maxSize > 0 {
					volInstStateObj.MaxSize = humanize.Bytes(maxSize)
				}
			}
		}
		if infoObject.GetVolumeErr() != nil {
			volInstStateObj.EveState = fmt.Sprintf("%s: %s", infoObject.State.String(), infoObject.GetVolumeErr().String())
		} else {
			if volInstStateObj.originType == config.VolumeContentOriginType_VCOT_BLANK {
				volInstStateObj.EveState = infoObject.GetState().String()
			}
		}
	case info.ZInfoTypes_ZiContentTree:
		infoObject := im.GetCinfo()
		for _, el := range ctx.volumes {
			if infoObject.Uuid == el.contentTreeID {
				if infoObject.GetErr() != nil {
					el.EveState = fmt.Sprintf("%s: %s", infoObject.State.String(), infoObject.GetErr().String())
					continue
				}
				el.EveState = infoObject.State.String()
				if infoObject.State == info.ZSwState_DOWNLOAD_STARTED {
					el.EveState = fmt.Sprintf("%s (%d%%)", el.EveState, infoObject.ProgressPercentage)
				}
			}
		}
	}
}

func (ctx *State) processVolumesByMetric(msg *metrics.ZMetricMsg) {
	if volumeMetrics := msg.GetVm(); volumeMetrics != nil {
		for _, volumeMetric := range volumeMetrics {
			volInstStateObj, ok := ctx.volumes[volumeMetric.GetUuid()]
			if ok {
				volInstStateObj.Size = humanize.Bytes(volumeMetric.GetUsedBytes())
			}
		}
	}
}

//VolumeList prints volumes
func (ctx *State) VolumeList() error {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	if _, err := fmt.Fprintln(w, volInstStateHeader()); err != nil {
		return err
	}
	volInstStatesSlice := make([]*VolInstState, 0, len(ctx.Volumes()))
	volInstStatesSlice = append(volInstStatesSlice, ctx.Volumes()...)
	sort.SliceStable(volInstStatesSlice, func(i, j int) bool {
		return volInstStatesSlice[i].Name < volInstStatesSlice[j].Name
	})
	for _, el := range volInstStatesSlice {
		if _, err := fmt.Fprintln(w, el.toString()); err != nil {
			return err
		}
	}
	return w.Flush()
}
