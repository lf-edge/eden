package eve

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/types"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/api/go/metrics"
)

// AppInstState stores state of app
type AppInstState struct {
	Name         string
	UUID         string
	Image        string
	AdamState    string
	EVEState     string
	InternalIP   []string
	ExternalIP   string
	InternalPort string
	ExternalPort string
	Metadata     string
	MemoryUsed   uint32
	MemoryAvail  uint32
	CPUUsage     int
	Macs         []string
	Volumes      map[string]uint32

	PrevCPUNS     uint64
	PrevCPUNSTime time.Time
	Deleted       bool
	InfoTime      time.Time
}

func appStateHeader() string {
	return "NAME\tIMAGE\tUUID\tINTERNAL\tEXTERNAL\tMEMORY\tSTATE(ADAM)\tLAST_STATE(EVE)"
}

func (appStateObj *AppInstState) toString() string {
	internal := "-"
	if len(appStateObj.InternalIP) == 1 {
		if appStateObj.InternalPort == "" {
			internal = appStateObj.InternalIP[0] //if one internal IP and not forward, display it
		} else {
			internal = fmt.Sprintf("%s:%s", appStateObj.InternalIP[0], appStateObj.InternalPort)
		}
	} else if len(appStateObj.InternalIP) > 1 {
		var els []string
		for i, el := range appStateObj.InternalIP {
			if i == 0 { //forward only on first network
				if appStateObj.InternalPort == "" {
					els = append(els, el) //if multiple internal IPs and not forward, display them
				} else {
					els = append(els, fmt.Sprintf("%s:%s", el, appStateObj.InternalPort))
				}
			} else {
				els = append(els, el)
			}
		}
		internal = strings.Join(els, "; ")
	}
	external := fmt.Sprintf("%s:%s", appStateObj.ExternalIP, appStateObj.ExternalPort)
	if appStateObj.ExternalPort == "" {
		external = "-"
	}
	memory := fmt.Sprintf("%s/%s",
		humanize.Bytes((uint64)(appStateObj.MemoryUsed*humanize.MByte)),
		humanize.Bytes((uint64)(appStateObj.MemoryAvail*humanize.MByte)))
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s",
		appStateObj.Name, appStateObj.Image, appStateObj.UUID,
		internal, external, memory,
		appStateObj.AdamState, appStateObj.EVEState)
}

func getPortMapping(appConfig *config.AppInstanceConfig, qemuPorts map[string]string) (intports, extports string) {
	iports := []string{}
	eports := []string{}
	for _, intf := range appConfig.Interfaces {
		fromPort := ""
		toPort := ""
		for _, acl := range intf.Acls {
			for _, match := range acl.Matches {
				if match.Type == "lport" {
					fromPort = match.Value
				}
			}
			for _, action := range acl.Actions {
				if action.Portmap {
					toPort = strconv.Itoa(int(action.AppPort))
				}
			}
			if fromPort != "" && toPort != "" {
				if len(qemuPorts) > 0 {
					for p1, p2 := range qemuPorts {
						if p2 == fromPort {
							fromPort = p1
							break
						}
					}
				}
				iports = append(iports, toPort)
				eports = append(eports, fromPort)
			}
		}
	}
	return strings.Join(iports, ","), strings.Join(eports, ",")
}

func (ctx *State) initApplications(ctrl controller.Cloud, dev *device.Ctx) error {
	ctx.Applications = make(map[string]*AppInstState)
	for _, el := range dev.GetApplicationInstances() {
		app, err := ctrl.GetApplicationInstanceConfig(el)
		if err != nil {
			return fmt.Errorf("no app in cloud %s: %s", el, err)
		}
		imageName := ""
		if len(app.Drives) > 0 {
			imageName = app.Drives[0].Image.Name
		}
		intPort, extPort := getPortMapping(app, ctrl.GetVars().EveQemuPorts)
		volumes := make(map[string]uint32)
		for _, el := range app.GetVolumeRefList() {
			volumes[el.Uuid] = 0
		}
		appStateObj := &AppInstState{
			Name:         app.Displayname,
			Image:        imageName,
			AdamState:    inControllerConfig,
			EVEState:     "UNKNOWN",
			InternalIP:   []string{"-"},
			ExternalIP:   "-",
			InternalPort: intPort,
			ExternalPort: extPort,
			Volumes:      volumes,
			UUID:         app.Uuidandversion.Uuid,
		}
		ctx.Applications[app.Uuidandversion.Uuid] = appStateObj
	}
	return nil
}

func (ctx *State) applyOldStateApps(state *State) {
	for stateID, stateEL := range state.Applications {
		found := false
		for id := range ctx.Applications {
			if id != stateID {
				continue
			}
			ctx.Applications[id] = stateEL
			found = true
		}
		if !found {
			if stateEL.Deleted {
				continue
			}
			stateEL.AdamState = notInControllerConfig
			ctx.Applications[stateID] = stateEL
		}
	}
}

func (ctx *State) processApplicationsByMetric(msg *metrics.ZMetricMsg) {
	if appMetrics := msg.GetAm(); appMetrics != nil {
		for _, appMetric := range appMetrics {
			for _, el := range ctx.Applications {
				if appMetric.AppID == el.UUID {
					el.MemoryAvail = appMetric.Memory.GetAvailMem()
					el.MemoryUsed = appMetric.Memory.GetUsedMem()
					// if not restarted
					if el.PrevCPUNS < appMetric.Cpu.TotalNs {
						el.CPUUsage = int(float32(appMetric.Cpu.TotalNs-el.PrevCPUNS) / float32(msg.GetAtTimeStamp().AsTime().Sub(el.PrevCPUNSTime).Nanoseconds()) * 100.0)
					}
					el.PrevCPUNS = appMetric.Cpu.TotalNs
					el.PrevCPUNSTime = msg.GetAtTimeStamp().AsTime()
					break
				}
			}
		}
	}
}

//nolint:cyclop
func (ctx *State) processApplicationsByInfo(im *info.ZInfoMsg) {
	switch im.GetZtype() {
	case info.ZInfoTypes_ZiVolume:
		for _, app := range ctx.Applications {
			if len(app.Volumes) == 0 {
				continue
			}
			var percent uint32
			for vol := range app.Volumes {
				percent += app.Volumes[vol] //we sum all percents of all volumes and will divide them by count
				if im.GetVinfo().Uuid == vol {
					app.Volumes[vol] = im.GetVinfo().ProgressPercentage
					break
				}
			}
			if strings.HasPrefix(app.EVEState, info.ZSwState_DOWNLOAD_STARTED.String()) {
				app.EVEState = fmt.Sprintf("%s (%d%%)", info.ZSwState_DOWNLOAD_STARTED.String(), int(percent)/len(app.Volumes))
			}
		}
	case info.ZInfoTypes_ZiAppInstMetaData:
		for _, app := range ctx.Applications {
			if im.GetAmdinfo().Uuid == app.UUID {
				app.Metadata = string(im.GetAmdinfo().Data)
				break
			}
		}
	case info.ZInfoTypes_ZiApp:
		appStateObj, ok := ctx.Applications[im.GetAinfo().AppID]
		if !ok {
			appStateObj = &AppInstState{
				Name:      im.GetAinfo().AppName,
				Image:     "-",
				AdamState: notInControllerConfig,
				UUID:      im.GetAinfo().AppID,
			}
			ctx.Applications[im.GetAinfo().AppID] = appStateObj
		}
		appStateObj.EVEState = im.GetAinfo().State.String()
		if len(im.GetAinfo().AppErr) > 0 {
			//if AppErr, show them
			appStateObj.EVEState = fmt.Sprintf("%s: %s", im.GetAinfo().State.String(), im.GetAinfo().AppErr)
		}
		if len(im.GetAinfo().Network) != 0 && len(im.GetAinfo().Network[0].IPAddrs) != 0 {
			if len(im.GetAinfo().Network) > 1 {
				appStateObj.InternalIP = []string{}
				appStateObj.Macs = []string{}
				for _, el := range im.GetAinfo().Network {
					if len(el.IPAddrs) != 0 {
						appStateObj.InternalIP = append(appStateObj.InternalIP, el.IPAddrs[0])
						appStateObj.Macs = append(appStateObj.Macs, el.MacAddr)
					}
				}
			} else {
				if len(im.GetAinfo().Network[0].IPAddrs) != 0 {
					appStateObj.InternalIP = []string{im.GetAinfo().Network[0].IPAddrs[0]}
					appStateObj.Macs = []string{im.GetAinfo().Network[0].MacAddr}
				}
			}
		} else {
			appStateObj.InternalIP = []string{"-"}
			appStateObj.Macs = []string{}
		}
		//check appStateObj not defined in adam
		if appStateObj.AdamState != inControllerConfig {
			if im.GetAinfo().AppID == appStateObj.UUID {
				appStateObj.Deleted = false //if in recent ZInfoTypes_ZiApp, then not deleted
			}
		}
		if im.GetAinfo().State == info.ZSwState_INVALID {
			appStateObj.Deleted = true
		}
		appStateObj.InfoTime = im.AtTimeStamp.AsTime()
	case info.ZInfoTypes_ZiNetworkInstance: //try to find ips from NetworkInstances
		for _, el := range im.GetNiinfo().IpAssignments {
			// nothing to show if no IpAddress received
			if len(el.IpAddress) == 0 {
				continue
			}
			for _, appStateObj := range ctx.Applications {
				for ind, mac := range appStateObj.Macs {
					if mac == el.MacAddress {
						appStateObj.InternalIP[ind] = el.IpAddress[0]
					}
				}
			}
		}
	case info.ZInfoTypes_ZiDevice:
		for _, el := range im.GetDinfo().AppInstances {
			if _, ok := ctx.Applications[el.Uuid]; !ok {
				appStateObj := &AppInstState{
					Name:      el.Name,
					Image:     "-",
					AdamState: notInControllerConfig,
					EVEState:  "UNKNOWN",
					UUID:      el.Uuid,
				}
				ctx.Applications[el.Uuid] = appStateObj
			}
		}
		for _, appStateObj := range ctx.Applications {
			seen := false
			for _, el := range im.GetDinfo().AppInstances {
				if appStateObj.UUID == el.Uuid {
					seen = true
					break
				}
			}
			if !seen {
				appStateObj.EVEState = "UNKNOWN" //UNKNOWN if not found in recent AppInstances
			}
			if ctx.device.GetRemote() {
				for _, nw := range im.GetDinfo().Network {
					for _, addr := range nw.IPAddrs {
						if addr != "" {
							s := strings.Split(addr, ";")
							for _, oneip := range s {
								if strings.Contains(oneip, ".") {
									appStateObj.ExternalIP = oneip
								}
							}
						}
					}
				}
			} else {
				appStateObj.ExternalIP = "127.0.0.1"
			}
			//check appStateObj not defined in adam
			if appStateObj.AdamState != inControllerConfig && appStateObj.InfoTime.Before(im.AtTimeStamp.AsTime()) {
				appStateObj.Deleted = true
				for _, el := range im.GetDinfo().AppInstances {
					//if in recent ZInfoTypes_ZiDevice with timestamp after ZInfoTypes_ZiApp, then not deleted
					if el.Uuid == appStateObj.UUID {
						appStateObj.Deleted = false
					}
				}
			}
		}
	}
}

func (ctx *State) printPodListLines() error {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	if _, err := fmt.Fprintln(w, appStateHeader()); err != nil {
		return err
	}
	appStatesSlice := make([]*AppInstState, 0, len(ctx.NotDeletedApplications()))
	appStatesSlice = append(appStatesSlice, ctx.NotDeletedApplications()...)
	sort.SliceStable(appStatesSlice, func(i, j int) bool {
		return appStatesSlice[i].Name < appStatesSlice[j].Name
	})
	for _, el := range appStatesSlice {
		if _, err := fmt.Fprintln(w, el.toString()); err != nil {
			return err
		}
	}
	return w.Flush()
}

func (ctx *State) printPodListJSON() error {
	result, err := json.MarshalIndent(ctx.NotDeletedApplications(), "", "    ")
	if err != nil {
		return err
	}
	//nolint:forbidigo
	fmt.Println(string(result))
	return nil
}

// PodsList prints applications
func (ctx *State) PodsList(outputFormat types.OutputFormat) error {
	switch outputFormat {
	case types.OutputFormatLines:
		return ctx.printPodListLines()
	case types.OutputFormatJSON:
		return ctx.printPodListJSON()
	}
	return fmt.Errorf("unimplemented output format")
}
