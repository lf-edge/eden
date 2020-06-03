package cmd

import (
	"fmt"
	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/info"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"math/rand"
	"os"
	"text/tabwriter"
	"time"
)

var subnetForApp = "10.1.0.0/24"

var podName = ""

var podCmd = &cobra.Command{
	Use: "pod",
}

func checkDataStore(ds *config.DatastoreConfig, appType string, appUrl string) bool {
	if ds == nil {
		return false
	}
	if appType == "docker" && ds.DType == config.DsType_DsContainerRegistry && ds.Fqdn == "docker://docker.io" {
		return true
	}
	return false
}

func createDataStore(appType string, appUrl string) (*config.DatastoreConfig, error) {
	var ds *config.DatastoreConfig
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	switch appType {
	case "docker":
		ds = &config.DatastoreConfig{
			Id:         id.String(),
			DType:      config.DsType_DsContainerRegistry,
			Fqdn:       "docker://docker.io",
			ApiKey:     "",
			Password:   "",
			Dpath:      "",
			Region:     "",
			CipherData: nil,
		}
		return ds, nil
	default:
		return nil, fmt.Errorf("not supported appType")
	}
}

func checkImage(img *config.Image, dsId string, appType string, appUrl string, appVersion string) bool {
	if img == nil {
		return false
	}
	if appType == "docker" {
		if img.DsId == dsId && img.Name == fmt.Sprintf("%s:%s", appUrl, appVersion) && img.Iformat == config.Format_CONTAINER {
			return true
		}
	}
	return false
}

func createImage(dsId string, appType string, appUrl string, appVersion string) (*config.Image, error) {
	var img *config.Image
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	switch appType {
	case "docker":
		img = &config.Image{
			Uuidandversion: &config.UUIDandVersion{
				Uuid:    id.String(),
				Version: "1",
			},
			Name:    fmt.Sprintf("%s:%s", appUrl, appVersion),
			Iformat: config.Format_CONTAINER,
			DsId:    dsId,
			Siginfo: &config.SignatureInfo{},
		}
		return img, nil
	default:
		return nil, fmt.Errorf("not supported appType")
	}
}

func checkNetworkInstance(netInst *config.NetworkInstanceConfig) bool {
	if netInst == nil {
		return false
	}
	if netInst.Ip.Subnet == subnetForApp {
		return true
	}
	return false
}

func createNetworkInstance() (*config.NetworkInstanceConfig, error) {
	var netInst *config.NetworkInstanceConfig
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	subentIPs := utils.GetSubnetIPs(subnetForApp)
	netInst = &config.NetworkInstanceConfig{
		Uuidandversion: &config.UUIDandVersion{
			Uuid:    id.String(),
			Version: "1",
		},
		Displayname: "local",
		InstType:    config.ZNetworkInstType_ZnetInstLocal,
		Activate:    false,
		Port: &config.Adapter{
			Name: "uplink",
		},
		Cfg:    &config.NetworkInstanceOpaqueConfig{},
		IpType: config.AddressType_IPV4,
		Ip: &config.Ipspec{
			Subnet:  subnetForApp,
			Gateway: subentIPs[1].String(),
			Dns:     []string{subentIPs[1].String()},
			DhcpRange: &config.IpRange{
				Start: subentIPs[2].String(),
				End:   subentIPs[len(subentIPs)-2].String(),
			},
		},
		Dns: nil,
	}
	return netInst, nil
}

func checkAppInstanceConfig(app *config.AppInstanceConfig, appName string, appType string, appUrl string, appVersion string) bool {
	if app == nil {
		return false
	}
	if app.Displayname == appName {
		return true
	}
	return false
}

func createAppInstanceConfig(img *config.Image, appName string, netInstId string, appType string, appUrl string, appVersion string) (*config.AppInstanceConfig, error) {
	var app *config.AppInstanceConfig
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	switch appType {
	case "docker":
		app = &config.AppInstanceConfig{
			Uuidandversion: &config.UUIDandVersion{
				Uuid:    id.String(),
				Version: "1",
			},
			Fixedresources: &config.VmConfig{
				Memory:     1024000,
				Maxmem:     1024000,
				Vcpus:      1,
				Rootdev:    "/dev/xvda1",
				Bootloader: "/usr/bin/pygrub",
			},
			Drives: []*config.Drive{{
				Image: img,
			}},
			Activate:    true,
			Displayname: appName,
			Interfaces: []*config.NetworkAdapter{{
				Name:      "default",
				NetworkId: netInstId,
				Acls: []*config.ACE{{
					Matches: []*config.ACEMatch{{
						Type: "host",
					}},
					Id: 1,
				}},
			}},
		}
		return app, nil
	default:
		return nil, fmt.Errorf("not supported appType")
	}
}

var podDeployCmd = &cobra.Command{
	Use:   "deploy <docker>://<TAG>[:<VERSION>]",
	Short: "Deploy app in pod",
	Long:  `Deploy app in pod.`,
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		appLink := args[0]
		rand.Seed(time.Now().UnixNano())
		appName := namesgenerator.GetRandomName(0)
		if podName != "" {
			appName = podName
		}
		params := getParams(appLink, defaults.DefaultPodLinkPattern)
		if len(params) == 0 {
			log.Fatalf("fail to parse <docker>://<TAG>[:<VERSION>] from argument (%s)", appLink)
		}
		appType := ""
		appUrl := ""
		appVersion := ""
		ok := false
		if appType, ok = params["TYPE"]; !ok || appType == "" {
			log.Fatalf("cannot parse appType (not [docker]): %s", appLink)
		}
		if appUrl, ok = params["TAG"]; !ok || appUrl == "" {
			log.Fatalf("cannot parse appTag: %s", appLink)
		}
		if appVersion, ok = params["VERSION"]; !ok || appVersion == "" {
			log.Debugf("cannot parse appVersion from %s will use latest", appLink)
			appVersion = "latest"
		}

		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev: %s", err)
		}
		var datastore *config.DatastoreConfig
		for _, ds := range ctrl.ListDataStore() {
			if checkDataStore(ds, appType, appUrl) {
				datastore = ds
				break
			}
		}
		if datastore == nil {
			if datastore, err = createDataStore(appType, appUrl); err != nil {
				log.Fatalf("cannot create datastore: %s", err)
			}
			if err = ctrl.AddDataStore(datastore); err != nil {
				log.Fatalf("AddDataStore: %s", err)
			}
			log.Infof("new datastore created %s", datastore.Id)
		}
		var image *config.Image
		for _, img := range ctrl.ListImage() {
			if checkImage(img, datastore.Id, appType, appUrl, appVersion) {
				image = img
				break
			}
		}
		if image == nil {
			if image, err = createImage(datastore.Id, appType, appUrl, appVersion); err != nil {
				log.Fatalf("cannot create image: %s", err)
			}
			if err = ctrl.AddImage(image); err != nil {
				log.Fatalf("AddImage: %s", err)
			}
			log.Infof("new image created %s", image.Uuidandversion.Uuid)
		}

		var networkInstance *config.NetworkInstanceConfig
		for _, netInst := range ctrl.ListNetworkInstanceConfig() {
			if checkNetworkInstance(netInst) {
				networkInstance = netInst
				break
			}
		}
		if networkInstance == nil {
			if networkInstance, err = createNetworkInstance(); err != nil {
				log.Fatalf("cannot create NetworkInstance: %s", err)
			}
			if err = ctrl.AddNetworkInstanceConfig(networkInstance); err != nil {
				log.Fatalf("AddNetworkInstanceConfig: %s", err)
			}
		}
		var appInstanceConfig *config.AppInstanceConfig
		for _, app := range ctrl.ListApplicationInstanceConfig() {
			if checkAppInstanceConfig(app, appName, appType, appUrl, appVersion) {
				appInstanceConfig = app
				break
			}
		}
		if appInstanceConfig == nil {
			if appInstanceConfig, err = createAppInstanceConfig(image, appName, networkInstance.Uuidandversion.Uuid, appType, appUrl, appVersion); err != nil {
				log.Fatalf("cannot create app: %s", err)
			}
			if err = ctrl.AddApplicationInstanceConfig(appInstanceConfig); err != nil {
				log.Fatalf("AddApplicationInstanceConfig: %s", err)
			}
			log.Infof("new app created %s", appInstanceConfig.Uuidandversion.Uuid)
		}
		for _, el := range dev.GetApplicationInstances() {
			if el == appInstanceConfig.Uuidandversion.Uuid {
				log.Info("Already deployed")
				return
			}
		}
		devModel, err := ctrl.GetDevModelByName(viper.GetString("eve.devmodel"))
		if err != nil {
			log.Fatalf("fail to get dev model %s: %s", viper.GetString("eve.devmodel"), err)
		}
		if err = ctrl.ApplyDevModel(dev, devModel); err != nil {
			log.Fatalf("ApplyDevModel: %s", err)
		}
		dev.SetApplicationInstanceConfig(append(dev.GetApplicationInstances(), appInstanceConfig.Uuidandversion.Uuid))
		if err = changer.setControllerAndDev(ctrl, dev); err != nil {
			log.Fatalf("setControllerAndDev: %s", err)
		}
		log.Infof("deploy pod %s with %s://%s:%s request sent", appName, appType, appUrl, appVersion)
	},
}

type appState struct {
	name      string
	image     string
	adamState string
	eveState  string
	ip        string
	deleted   bool
}

func appStateHeader() string {
	return "NAME\tIMAGE\tIP\tSTATE(ADAM)\tLAST_STATE(EVE)"
}

func (appStateObj *appState) String() string {
	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s", appStateObj.name, appStateObj.image, appStateObj.ip, appStateObj.adamState, appStateObj.eveState)
}

var podPsCmd = &cobra.Command{
	Use:   "ps",
	Short: "List pods",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev: %s", err)
		}
		appStates := make(map[string]*appState)
		for _, el := range dev.GetApplicationInstances() {
			app, err := ctrl.GetApplicationInstanceConfig(el)
			if err != nil {
				log.Fatalf("no app in cloud %s: %s", el, err)
			}
			imageName := ""
			if len(app.Drives) > 0 {
				imageName = app.Drives[0].Image.Name
			}
			appStateObj := &appState{name: app.Displayname, image: imageName, adamState: "IN_CONFIG", eveState: "UNKNOWN", ip: "-"}
			appStates[app.Uuidandversion.Uuid] = appStateObj
		}
		var handleInfo = func(im *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface, infoType einfo.ZInfoType) bool {
			appStateObj, ok := appStates[im.GetAinfo().AppID]
			if !ok {
				imageName := ""
				if len(im.GetAinfo().GetSoftwareList()) > 0 {
					imageName = im.GetAinfo().GetSoftwareList()[0].ImageName
				}
				appStateObj = &appState{name: im.GetAinfo().AppName, image: imageName, adamState: "NOT_IN_CONFIG"}
				appStates[im.GetAinfo().AppID] = appStateObj
			}
			appStateObj.eveState = im.GetAinfo().State.String()
			if len(im.GetAinfo().Network) > 0 && len(im.GetAinfo().Network[0].IPAddrs) > 0 {
				appStateObj.ip = im.GetAinfo().Network[0].IPAddrs[0]
			} else {
				appStateObj.ip = "-"
			}
			return false
		}
		if err = ctrl.InfoLastCallback(dev.GetID(), map[string]string{"devId": dev.GetID().String()}, einfo.ZInfoAppInstance, handleInfo); err != nil {
			log.Fatalf("Fail in get InfoLastCallback: %s", err)
		}
		var handleInfoDevice = func(im *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface, infoType einfo.ZInfoType) bool {
			for _, appStateObj := range appStates {
				if appStateObj.adamState == "NOT_IN_CONFIG" {
					appStateObj.deleted = true
					for _, el := range im.GetDinfo().AppInstances {
						if appStateObj.name == el.Name {
							appStateObj.deleted = false
						}
					}
				}
			}
			return false
		}
		if err = ctrl.InfoLastCallback(dev.GetID(), map[string]string{"devId": dev.GetID().String()}, einfo.ZInfoDinfo, handleInfoDevice); err != nil {
			log.Fatalf("Fail in get InfoLastCallback: %s", err)
		}
		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 0, 8, 1, '\t', 0)
		if _, err = fmt.Fprintln(w, appStateHeader()); err != nil {
			log.Fatal(err)
		}
		for _, el := range appStates {
			if el.deleted == false {
				if _, err = fmt.Fprintln(w, el.String()); err != nil {
					log.Fatal(err)
				}
			}
		}
		if err = w.Flush(); err != nil {
			log.Fatal(err)
		}
	},
}

var podStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop pod",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		appName := args[0]
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev: %s", err)
		}
		for _, el := range dev.GetApplicationInstances() {
			app, err := ctrl.GetApplicationInstanceConfig(el)
			if err != nil {
				log.Fatalf("no app in cloud %s: %s", el, err)
			}
			if app.Displayname == appName {
				app.Activate = false
				if err = changer.setControllerAndDev(ctrl, dev); err != nil {
					log.Fatalf("setControllerAndDev: %s", err)
				}
				log.Infof("app %s stop done", appName)
				return
			}
		}
		log.Infof("not found app with name %s", appName)
	},
}

var podStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start pod",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		appName := args[0]
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev: %s", err)
		}
		for _, el := range dev.GetApplicationInstances() {
			app, err := ctrl.GetApplicationInstanceConfig(el)
			if err != nil {
				log.Fatalf("no app in cloud %s: %s", el, err)
			}
			if app.Displayname == appName {
				app.Activate = true
				if err = changer.setControllerAndDev(ctrl, dev); err != nil {
					log.Fatalf("setControllerAndDev: %s", err)
				}
				log.Infof("app %s start done", appName)
				return
			}
		}
		log.Infof("not found app with name %s", appName)
	},
}

var podDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete pod",
	Args:  cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		assingCobraToViper(cmd)
		_, err := utils.LoadConfigFile(configFile)
		if err != nil {
			return fmt.Errorf("error reading config: %s", err.Error())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		appName := args[0]
		changer := &adamChanger{}
		ctrl, dev, err := changer.getControllerAndDev()
		if err != nil {
			log.Fatalf("getControllerAndDev: %s", err)
		}
		for id, el := range dev.GetApplicationInstances() {
			app, err := ctrl.GetApplicationInstanceConfig(el)
			if err != nil {
				log.Fatalf("no app in cloud %s: %s", el, err)
			}
			if app.Displayname == appName {
				configs := dev.GetApplicationInstances()
				utils.DelEleInSlice(&configs, id)
				dev.SetApplicationInstanceConfig(configs)
				if err = changer.setControllerAndDev(ctrl, dev); err != nil {
					log.Fatalf("setControllerAndDev: %s", err)
				}
				log.Infof("app %s delete done", appName)
				return
			}
		}
		log.Infof("not found app with name %s", appName)
	},
}

func podInit() {
	podCmd.AddCommand(podDeployCmd)
	podDeployCmd.Flags().StringVarP(&podName, "name", "n", "", "name for pod")
	podCmd.AddCommand(podPsCmd)
	podCmd.AddCommand(podStopCmd)
	podCmd.AddCommand(podStartCmd)
	podCmd.AddCommand(podDeleteCmd)
}
