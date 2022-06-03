package eve

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/device"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eve/api/go/metrics"
)

//AppInstState stores state of app
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
	Memory       string
	macs         []string
	volumes      map[string]uint32
	deleted      bool
	infoTime     time.Time
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
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s",
		appStateObj.Name, appStateObj.Image, appStateObj.UUID,
		internal, external, appStateObj.Memory,
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
	ctx.applications = make(map[string]*AppInstState)
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
			volumes:      volumes,
			UUID:         app.Uuidandversion.Uuid,
		}
		ctx.applications[app.Uuidandversion.Uuid] = appStateObj
	}
	return nil
}

func (ctx *State) processApplicationsByMetric(msg *metrics.ZMetricMsg) {
	if appMetrics := msg.GetAm(); appMetrics != nil {
		for _, appMetric := range appMetrics {
			for _, el := range ctx.applications {
				if appMetric.AppID == el.UUID {
					el.Memory = fmt.Sprintf("%s/%s",
						humanize.Bytes((uint64)(appMetric.Memory.GetUsedMem()*humanize.MByte)),
						humanize.Bytes((uint64)(appMetric.Memory.GetAvailMem()*humanize.MByte)))
					break
				}
			}
		}
	}
}

func (ctx *State) processApplicationsByInfo(im *info.ZInfoMsg) {
	switch im.GetZtype() {
	case info.ZInfoTypes_ZiVolume:
		for _, app := range ctx.applications {
			if len(app.volumes) == 0 {
				continue
			}
			var percent uint32
			for vol := range app.volumes {
				percent += app.volumes[vol] //we sum all percents of all volumes and will divide them by count
				if im.GetVinfo().Uuid == vol {
					app.volumes[vol] = im.GetVinfo().ProgressPercentage
					break
				}
			}
			if strings.HasPrefix(app.EVEState, info.ZSwState_DOWNLOAD_STARTED.String()) {
				app.EVEState = fmt.Sprintf("%s (%d%%)", info.ZSwState_DOWNLOAD_STARTED.String(), int(percent)/len(app.volumes))
			}
		}
	case info.ZInfoTypes_ZiApp:
		appStateObj, ok := ctx.applications[im.GetAinfo().AppID]
		if !ok {
			appStateObj = &AppInstState{
				Name:      im.GetAinfo().AppName,
				Image:     "-",
				AdamState: notInControllerConfig,
				UUID:      im.GetAinfo().AppID,
			}
			ctx.applications[im.GetAinfo().AppID] = appStateObj
		}
		appStateObj.EVEState = im.GetAinfo().State.String()
		if len(im.GetAinfo().AppErr) > 0 {
			//if AppErr, show them
			appStateObj.EVEState = fmt.Sprintf("%s: %s", im.GetAinfo().State.String(), im.GetAinfo().AppErr)
		}
		if len(im.GetAinfo().Network) != 0 && len(im.GetAinfo().Network[0].IPAddrs) != 0 {
			if len(im.GetAinfo().Network) > 1 {
				appStateObj.InternalIP = []string{}
				appStateObj.macs = []string{}
				for _, el := range im.GetAinfo().Network {
					if len(im.GetAinfo().Network[0].IPAddrs) != 0 {
						appStateObj.InternalIP = append(appStateObj.InternalIP, el.IPAddrs[0])
						appStateObj.macs = append(appStateObj.macs, el.MacAddr)
					}
				}
			} else {
				if len(im.GetAinfo().Network[0].IPAddrs) != 0 {
					appStateObj.InternalIP = []string{im.GetAinfo().Network[0].IPAddrs[0]}
					appStateObj.macs = []string{im.GetAinfo().Network[0].MacAddr}
				}
			}
		} else {
			appStateObj.InternalIP = []string{"-"}
			appStateObj.macs = []string{}
		}
		//check appStateObj not defined in adam
		if appStateObj.AdamState != inControllerConfig {
			if im.GetAinfo().AppID == appStateObj.UUID {
				appStateObj.deleted = false //if in recent ZInfoTypes_ZiApp, then not deleted
			}
		}
		if im.GetAinfo().State == info.ZSwState_INVALID {
			appStateObj.deleted = true
		}
		appStateObj.infoTime = im.AtTimeStamp.AsTime()
	case info.ZInfoTypes_ZiNetworkInstance: //try to find ips from NetworkInstances
		for _, el := range im.GetNiinfo().IpAssignments {
			for _, appStateObj := range ctx.applications {
				for ind, mac := range appStateObj.macs {
					if mac == el.MacAddress {
						appStateObj.InternalIP[ind] = el.IpAddress[0]
					}
				}
			}
		}
	case info.ZInfoTypes_ZiDevice:
		for _, el := range im.GetDinfo().AppInstances {
			if _, ok := ctx.applications[el.Uuid]; !ok {
				appStateObj := &AppInstState{
					Name:      el.Name,
					Image:     "-",
					AdamState: notInControllerConfig,
					EVEState:  "UNKNOWN",
					UUID:      el.Uuid,
				}
				ctx.applications[el.Uuid] = appStateObj
			}
		}
		for _, appStateObj := range ctx.applications {
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
			if appStateObj.AdamState != inControllerConfig && appStateObj.infoTime.Before(im.AtTimeStamp.AsTime()) {
				appStateObj.deleted = true
				for _, el := range im.GetDinfo().AppInstances {
					//if in recent ZInfoTypes_ZiDevice with timestamp after ZInfoTypes_ZiApp, than not deleted
					if el.Uuid == appStateObj.UUID {
						appStateObj.deleted = false
					}
				}
			}
		}
	}
}

//PodsList prints applications
func (ctx *State) PodsList() error {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	if _, err := fmt.Fprintln(w, appStateHeader()); err != nil {
		return err
	}
	appStatesSlice := make([]*AppInstState, 0, len(ctx.Applications()))
	appStatesSlice = append(appStatesSlice, ctx.Applications()...)
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
