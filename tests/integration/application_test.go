package integration

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/info"
	"gotest.tools/assert"
	"testing"
	"time"
)

//TestApplication test base image loading into eve
func TestApplication(t *testing.T) {
	ctx, err := controllerPrepare()
	if err != nil {
		t.Fatal("Fail in controller prepare: ", err)
	}

	deviceModel, err := ctx.GetDevModel(controller.DevModelTypeQemu)
	if err != nil {
		t.Fatal("Fail in get deviceModel: ", err)
	}

	networkInstanceForTest := networkInstanceLocal

	networkInstanceForTestSecond := networkInstanceLocalSecond

	err = prepareNetworkInstance(ctx, networkInstanceForTest, deviceModel)
	if err != nil {
		t.Fatal("Fail in prepare network instance: ", err)
	}

	err = prepareNetworkInstance(ctx, networkInstanceForTestSecond, deviceModel)
	if err != nil {
		t.Fatal("Fail in prepare network instance: ", err)
	}

	var appInstances []string

	var lastIP string

	//metadataBuilder is metadata generator
	//if lastIP already set (we already have another app), metadata will set url to http://lastIP/user-data.html
	//metaDataToTest - is metadata vision from test suite
	var metadataBuilder = func(format config.Format) (metaDataToEVE string, metaDataToTest string) {
		url := "127.0.0.1"
		if lastIP != "" {
			url = lastIP
		}
		toRet := fmt.Sprintf("http://%s/user-data.html", url)
		if format == config.Format_CONTAINER {
			return fmt.Sprintf("url=%s", toRet), toRet
		}
		return toRet, toRet
	}

	var applicationTests = []struct {
		appDefinition   *appInstLocal
		networkAdapters []*config.NetworkAdapter
		vncDisplay      uint32
	}{
		{
			appInstanceLocalVM,
			[]*config.NetworkAdapter{{
				Name:      "eth0",
				NetworkId: networkInstanceForTest.networkInstanceID,
				Acls: []*config.ACE{{
					Matches: []*config.ACEMatch{{Type: "host"}},
					Id:      1,
				}}}, {
				Name:      "eth1",
				NetworkId: networkInstanceForTestSecond.networkInstanceID,
				Acls: []*config.ACE{{
					Matches: []*config.ACEMatch{{
						Type:  "protocol",
						Value: "tcp",
					}, {
						Type:  "lport",
						Value: "8027",
					}},
					Actions: []*config.ACEAction{{
						Drop:       false,
						Limit:      false,
						Limitrate:  0,
						Limitunit:  "",
						Limitburst: 0,
						Portmap:    true,
						AppPort:    80,
					}},
					Name: "",
					Id:   1,
					Dir:  config.ACEDirection_BOTH,
				}, {
					Matches: []*config.ACEMatch{{Type: "host"}},
					Id:      2,
				}}},
			},
			0,
		},
		{
			appInstanceLocalContainer,
			[]*config.NetworkAdapter{{
				Name:      "eth0",
				NetworkId: networkInstanceForTest.networkInstanceID,
				Acls: []*config.ACE{{
					Matches: []*config.ACEMatch{{Type: "host"}},
					Id:      1,
				}}}, {
				Name:      "eth1",
				NetworkId: networkInstanceForTestSecond.networkInstanceID,
				Acls: []*config.ACE{{
					Matches: []*config.ACEMatch{{
						Type:  "protocol",
						Value: "tcp",
					}, {
						Type:  "lport",
						Value: "8028",
					}},
					Actions: []*config.ACEAction{{
						Drop:       false,
						Limit:      false,
						Limitrate:  0,
						Limitunit:  "",
						Limitburst: 0,
						Portmap:    true,
						AppPort:    80,
					}},
					Name: "",
					Id:   1,
					Dir:  config.ACEDirection_BOTH,
				}, {
					Matches: []*config.ACEMatch{{Type: "host"}},
					Id:      2,
				}}}},
			1,
		},
	}
	for _, tt := range applicationTests {
		t.Run(tt.appDefinition.appName, func(t *testing.T) {
			metadata, metaDataToTest := metadataBuilder(tt.appDefinition.imageFormat)
			err = prepareApplicationLocal(ctx, tt.appDefinition, metadata, tt.vncDisplay, tt.networkAdapters)

			if err != nil {
				t.Fatal("Fail in prepare app from local file: ", err)
			}
			deviceCtx, err := ctx.GetDeviceFirst()
			if err != nil {
				t.Fatal("Fail in get first device: ", err)
			}
			err = ctx.ApplyDevModel(deviceCtx, deviceModel)
			if err != nil {
				t.Fatal("Fail in ApplyDevModel: ", err)
			}
			appInstances = append(appInstances, tt.appDefinition.appID)
			deviceCtx.SetApplicationInstanceConfig(appInstances)
			devUUID := deviceCtx.GetID()
			err = ctx.ConfigSync(deviceCtx)
			if err != nil {
				t.Fatal("Fail in sync config with controller: ", err)
			}
			t.Run("Started", func(t *testing.T) {
				err = ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "AppID": tt.appDefinition.appID}, einfo.ZInfoAppInstance, 1200)
				if err != nil {
					t.Fatal("Fail in waiting for app started status: ", err)
				}
			})
			t.Run("Downloaded", func(t *testing.T) {
				if !checkLogs {
					t.Skip("no LOGS flag set - skipped")
				}
				err = ctx.LogChecker(devUUID, map[string]string{"devId": devUUID.String(), "msg": fmt.Sprintf(".*AppID:\"%s\".*downloadProgress:100.*", tt.appDefinition.appID)}, 1200)
				if err != nil {
					t.Fatal("Fail in waiting for app downloaded status: ", err)
				}
			})
			t.Run("Installed", func(t *testing.T) {
				if !checkLogs {
					t.Skip("no LOGS flag set - skipped")
				}
				err = ctx.LogChecker(devUUID, map[string]string{"devId": devUUID.String(), "msg": fmt.Sprintf(".*AppID:\"%s\".*state:INSTALLED.*", tt.appDefinition.appID)}, 1200)
				if err != nil {
					t.Fatal("Fail in waiting for app installed status: ", err)
				}
			})
			timeout := time.Duration(1200)

			if !checkLogs {
				timeout = 2400
			}
			t.Run("Running", func(t *testing.T) {
				err = ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "AppID": tt.appDefinition.appID, "state": "RUNNING"}, einfo.ZInfoAppInstance, timeout)
				if err != nil {
					t.Fatal("Fail in waiting for app running status: ", err)
				}
			})
			t.Run("Cloud-init", func(t *testing.T) {
				//we point urlToTest to external app port of app
				//where nginx serve on url http://eveIP:accessPortExternal/user-data.html data, obtained from cloud-init metadata
				urlToTest := fmt.Sprintf("http://%s:%s/user-data.html", tt.appDefinition.eveIP, tt.appDefinition.accessPortExternal)
				result, err := utils.RequestHTTPRepeatWithTimeout(urlToTest, false, 300)
				if err != nil {
					t.Fatalf("Fail in waiting for app http response to %s: %s", urlToTest, err)
				}
				assert.Equal(t, metaDataToTest, result)
			})
			t.Run("Connectivity", func(t *testing.T) {
				if lastIP != "" {
					t.Log("will be test connectivity to app and between apps")
				} else {
					t.Log("will be test connectivity to app")
				}
				//we point urlToTest to external app port of app
				//where nginx serve on url http://eveIP:accessPortExternal/received-data.html data, received by curl
				//from url which defined in metadata
				//So, for the first app it is just http://eveIP:accessPortExternal/received-data.html -> http://127.0.0.1/user-data.html
				//for the second app it is http://eveIP:accessPortExternal/received-data.html -> http://FIRSTAPPIP:LOCALPORT/user-data.html
				//so for both apps request to http://eveIP:accessPortExternal/received-data.html must return http://127.0.0.1/user-data.html
				urlToTest := fmt.Sprintf("http://%s:%s/received-data.html", tt.appDefinition.eveIP, tt.appDefinition.accessPortExternal)
				result, err := utils.RequestHTTPRepeatWithTimeout(urlToTest, false, 200)
				if err != nil {
					t.Fatalf("Fail in waiting for app http response to %s: %s", urlToTest, err)
				}
				assert.Equal(t, "http://127.0.0.1/user-data.html", result)
			})
			t.Run("ObtainIP", func(t *testing.T) {
				handler := func(im *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface, infoType einfo.ZInfoType) bool {
					appNetwork := im.GetAinfo().Network
					isOk := false
					for _, el := range appNetwork {
						if el.GetDevName() == "eth0" {
							if len(el.IPAddrs) > 0 {
								lastIP = appNetwork[0].IPAddrs[0]
								t.Logf("IPAddrs: %s", lastIP)
								isOk = true
								break
							}
						}
					}
					if !isOk {
						t.Fatal("Fail in get IP from info")
					}
					return true
				}
				err = ctx.InfoLastCallback(devUUID, map[string]string{"devId": devUUID.String(), "AppID": tt.appDefinition.appID, "state": "RUNNING", "network": ".*"}, einfo.ZInfoAppInstance, handler)
			})
		})
	}

}
