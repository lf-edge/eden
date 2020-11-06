package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/info"
	log "github.com/sirupsen/logrus"
)

func podsList(logLevel log.Level) error {
	currentLogLevel := log.GetLevel()
	log.SetLevel(logLevel)
	defer log.SetLevel(currentLogLevel)
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %s", err)
	}
	appStates := make(map[string]*appState)
	for _, el := range dev.GetApplicationInstances() {
		app, err := ctrl.GetApplicationInstanceConfig(el)
		if err != nil {
			return fmt.Errorf("no app in cloud %s: %s", el, err)
		}
		imageName := ""
		if len(app.Drives) > 0 {
			imageName = app.Drives[0].Image.Name
		}
		intPort, extPort := getPortMapping(app, qemuPorts)
		volumes := make(map[string]uint32)
		for _, el := range app.GetVolumeRefList() {
			volumes[el.Uuid] = 0
		}
		appStateObj := &appState{
			name:      app.Displayname,
			image:     imageName,
			adamState: "IN_CONFIG",
			eveState:  "UNKNOWN",
			intIP:     []string{"-"},
			extIP:     "-",
			intPort:   intPort,
			extPort:   extPort,
			volumes:   volumes,
			uuid:      app.Uuidandversion.Uuid,
		}
		appStates[app.Uuidandversion.Uuid] = appStateObj
	}
	var handleInfo = func(im *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface) bool {
		switch im.GetZtype() {
		case info.ZInfoTypes_ZiVolume:
			for _, app := range appStates {
				if len(app.volumes) == 0 {
					continue
				}
				var percent uint32 = 0
				for vol := range app.volumes {
					percent += app.volumes[vol] //we sum all percents of all volumes and will divide them by count
					if im.GetVinfo().Uuid == vol {
						app.volumes[vol] = im.GetVinfo().ProgressPercentage
						break
					}
				}
				if strings.HasPrefix(app.eveState, info.ZSwState_DOWNLOAD_STARTED.String()) {
					app.eveState = fmt.Sprintf("%s (%d%%)", info.ZSwState_DOWNLOAD_STARTED.String(), int(percent)/len(app.volumes))
				}
			}
		case info.ZInfoTypes_ZiApp:
			appStateObj, ok := appStates[im.GetAinfo().AppID]
			if !ok {
				appStateObj = &appState{
					name:      im.GetAinfo().AppName,
					image:     "-",
					adamState: "NOT_IN_CONFIG",
					uuid:      im.GetAinfo().AppID,
				}
				appStates[im.GetAinfo().AppID] = appStateObj
			}
			appStateObj.eveState = im.GetAinfo().State.String()
			if len(im.GetAinfo().AppErr) > 0 {
				//if AppErr, show them
				appStateObj.eveState = fmt.Sprintf("%s: %s", im.GetAinfo().State.String(), im.GetAinfo().AppErr)
			}
			if len(im.GetAinfo().Network) != 0 && len(im.GetAinfo().Network[0].IPAddrs) != 0 {
				if len(im.GetAinfo().Network) > 1 {
					appStateObj.intIP = []string{}
					appStateObj.macs = []string{}
					for _, el := range im.GetAinfo().Network {
						if len(im.GetAinfo().Network[0].IPAddrs) != 0 {
							appStateObj.intIP = append(appStateObj.intIP, el.IPAddrs[0])
							appStateObj.macs = append(appStateObj.macs, el.MacAddr)
						}
					}
				} else {
					if len(im.GetAinfo().Network[0].IPAddrs) != 0 {
						appStateObj.intIP = []string{im.GetAinfo().Network[0].IPAddrs[0]}
						appStateObj.macs = []string{im.GetAinfo().Network[0].MacAddr}
					}
				}
			} else {
				appStateObj.intIP = []string{"-"}
				appStateObj.macs = []string{}
			}
		case info.ZInfoTypes_ZiNetworkInstance: //try to find ips from NetworkInstances
			for _, el := range im.GetNiinfo().IpAssignments {
				for _, appStateObj := range appStates {
					for ind, mac := range appStateObj.macs {
						if mac == el.MacAddress {
							appStateObj.intIP[ind] = el.IpAddress[0]
						}
					}
				}
			}
		case info.ZInfoTypes_ZiDevice:
			for _, appStateObj := range appStates {
				seen := false
				for _, el := range im.GetDinfo().AppInstances {
					if appStateObj.uuid == el.Uuid {
						seen = true
						break
					}
				}
				if !seen {
					appStateObj.eveState = "UNKNOWN" //UNKNOWN if not found in recent AppInstances
				}
				if devModel == defaults.DefaultRPIModel || devModel == defaults.DefaultGCPModel {
					for _, nw := range im.GetDinfo().Network {
						for _, addr := range nw.IPAddrs {
							if addr != "" {
								s := strings.Split(addr, ";")
								for _, oneip := range s {
									if strings.Contains(oneip, ".") {
										appStateObj.extIP = oneip
									}
								}
							}
						}
					}
				} else {
					appStateObj.extIP = "127.0.0.1"
				}
			}
		}
		return false
	}
	if err = ctrl.InfoLastCallback(dev.GetID(), map[string]string{"devId": dev.GetID().String()}, handleInfo); err != nil {
		return fmt.Errorf("fail in get InfoLastCallback: %s", err)
	}
	var handleInfoDevice = func(im *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface) bool {
		for _, appStateObj := range appStates {
			//check appStateObj not defined in adam
			if appStateObj.adamState == "NOT_IN_CONFIG" {
				switch im.GetZtype() {
				case info.ZInfoTypes_ZiApp:
					if im.GetAinfo().AppID == appStateObj.uuid {
						appStateObj.deleted = false //if in recent ZInfoTypes_ZiApp, than not deleted
					}
				case info.ZInfoTypes_ZiDevice:
					appStateObj.deleted = true
					for _, el := range im.GetDinfo().AppInstances {
						if el.Uuid == appStateObj.uuid {
							appStateObj.deleted = false //if in recent ZInfoTypes_ZiDevice, than not deleted
						}
					}
				}
			}
		}
		return false
	}
	if err = ctrl.InfoLastCallback(dev.GetID(), map[string]string{"devId": dev.GetID().String()}, handleInfoDevice); err != nil {
		return fmt.Errorf("fail in get InfoLastCallback: %s", err)
	}
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	if _, err = fmt.Fprintln(w, appStateHeader()); err != nil {
		return err
	}
	appStatesSlice := make([]*appState, 0, len(appStates))
	for _, k := range appStates {
		appStatesSlice = append(appStatesSlice, k)
	}
	sort.SliceStable(appStatesSlice, func(i, j int) bool {
		return appStatesSlice[i].name < appStatesSlice[j].name
	})
	for _, el := range appStatesSlice {
		if !el.deleted {
			if _, err = fmt.Fprintln(w, el.toString()); err != nil {
				return err
			}
		}
	}
	return w.Flush()
}

func netList(logLevel log.Level) error {
	currentLogLevel := log.GetLevel()
	log.SetLevel(logLevel)
	defer log.SetLevel(currentLogLevel)
	changer := &adamChanger{}
	ctrl, dev, err := changer.getControllerAndDev()
	if err != nil {
		return fmt.Errorf("getControllerAndDev: %s", err)
	}
	netInstStates := make(map[string]*netInstState)
	for _, el := range dev.GetNetworkInstances() {
		ni, err := ctrl.GetNetworkInstanceConfig(el)
		if err != nil {
			return fmt.Errorf("no netInst in cloud %s: %s", el, err)
		}
		netInstStateObj := &netInstState{
			name:      ni.GetDisplayname(),
			uuid:      ni.Uuidandversion.Uuid,
			adamState: "IN_CONFIG",
			eveState:  "UNKNOWN",
			cidr:      ni.Ip.Subnet,
			netType:   ni.InstType,
		}
		netInstStates[ni.Displayname] = netInstStateObj
	}
	var handleInfo = func(im *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface) bool {
		switch im.GetZtype() {
		case info.ZInfoTypes_ZiNetworkInstance:
			netInstStateObj, ok := netInstStates[im.GetNiinfo().GetDisplayname()]
			if !ok {
				netInstStateObj = &netInstState{
					name:      im.GetNiinfo().GetDisplayname(),
					uuid:      im.GetNiinfo().GetNetworkID(),
					adamState: "NOT_IN_CONFIG",
					eveState:  "IN_CONFIG",
					netType:   (config.ZNetworkInstType)(int32(im.GetNiinfo().InstType)),
				}
				netInstStates[im.GetNiinfo().GetDisplayname()] = netInstStateObj
			}
			if !im.GetNiinfo().Activated {
				if netInstStateObj.activated {
					//if previously Activated==true and now Activated==false then deleted
					netInstStateObj.deleted = true
				} else {
					netInstStateObj.deleted = false
				}
				netInstStateObj.eveState = "NOT_ACTIVATED"
			} else {
				netInstStateObj.eveState = "ACTIVATED"
			}
			netInstStateObj.activated = im.GetNiinfo().Activated
			//if errors, show them if in adam`s config
			if len(im.GetNiinfo().GetNetworkErr()) > 0 {
				netInstStateObj.eveState = fmt.Sprintf("ERRORS: %s", im.GetNiinfo().GetNetworkErr())
				if netInstStateObj.adamState == "NOT_IN_CONFIG" {
					netInstStateObj.deleted = true
				}
			}
		}
		return false
	}
	if err = ctrl.InfoLastCallback(dev.GetID(), map[string]string{"devId": dev.GetID().String()}, handleInfo); err != nil {
		return fmt.Errorf("fail in get InfoLastCallback: %s", err)
	}
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 1, '\t', 0)
	if _, err = fmt.Fprintln(w, netInstStateHeader()); err != nil {
		return err
	}
	netInstStatesSlice := make([]*netInstState, 0, len(netInstStates))
	for _, k := range netInstStates {
		netInstStatesSlice = append(netInstStatesSlice, k)
	}
	sort.SliceStable(netInstStatesSlice, func(i, j int) bool {
		return netInstStatesSlice[i].name < netInstStatesSlice[j].name
	})
	for _, el := range netInstStatesSlice {
		if !el.deleted {
			if _, err = fmt.Fprintln(w, el.toString()); err != nil {
				return err
			}
		}
	}
	return w.Flush()
}
