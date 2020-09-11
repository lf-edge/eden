package integration

import (
	"flag"
	"fmt"
	"github.com/lf-edge/eden/pkg/eden"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/spf13/viper"
	"gotest.tools/assert"
)

var (
	dockerYML = flag.String("app-docker.yml", "", "docker yml file to build")
	vmYML     = flag.String("app-vm.yml", "", "vm yml file to build")
)

//TestApplication test base image loading into eve
func TestApplication(t *testing.T) {
	viperLoaded, err := utils.LoadConfigFile("")
	if err != nil {
		t.Fatalf("error reading config: %s", err.Error())
	}
	if dockerYML == nil || *dockerYML == "" {
		t.Fatal("app-docker.yml has no value")
	} else {
		t.Logf("dockerYML: %s", *dockerYML)
	}
	dockerYMLAbs := utils.ResolveAbsPath(*dockerYML)
	if vmYML == nil || *vmYML == "" {
		t.Fatal("app-vm.yml has no value")
	} else {
		t.Logf("vmYML: %s", *vmYML)
	}
	vmYMLAbs := utils.ResolveAbsPath(*vmYML)
	if viperLoaded {
		eserverImageDist = utils.ResolveAbsPath(viper.GetString("eden.images.dist"))
	}
	ctx, err := controller.CloudPrepare()
	if err != nil {
		t.Fatalf("CloudPrepare: %s", err)
	}

	deviceCtx, err := ctx.GetDeviceCurrent()
	if err != nil {
		t.Fatal("Fail in get first device: ", err)
	}

	deviceModel, err := ctx.GetDevModelByName(defaults.DefaultEVEModel)
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
		ymlToBuild      string
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
			vmYMLAbs,
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
			dockerYMLAbs,
		},
	}
	for _, tt := range applicationTests {
		t.Run(tt.appDefinition.appName, func(t *testing.T) {
			t.Run("Setup", func(t *testing.T) {
				SetupApplication(t, tt.appDefinition.imageFormat, tt.ymlToBuild)
			})
			metadata, metaDataToTest := metadataBuilder(tt.appDefinition.imageFormat)
			err = prepareApplicationLocal(ctx, tt.appDefinition, metadata, tt.vncDisplay, tt.networkAdapters)

			if err != nil {
				t.Fatal("Fail in prepare app from local file: ", err)
			}
			appInstances = append(appInstances, tt.appDefinition.appID)
			deviceCtx.SetApplicationInstanceConfig(appInstances)
			devUUID := deviceCtx.GetID()
			err = ctx.ConfigSync(deviceCtx)
			if err != nil {
				t.Fatal("Fail in sync config with controller: ", err)
			}
			t.Run("Started", func(t *testing.T) {
				err = ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "AppID": tt.appDefinition.appID}, einfo.HandleFirst, einfo.InfoAny, 1200)
				if err != nil {
					t.Fatal("Fail in waiting for app started status: ", err)
				}
			})
			t.Run("Downloaded", func(t *testing.T) {
				if !checkLogs {
					t.Skip("no LOGS flag set - skipped")
				}
				err = ctx.LogChecker(devUUID, map[string]string{"devId": devUUID.String(), "msg": fmt.Sprintf(".*AppID:\"%s\".*downloadProgress:100.*", tt.appDefinition.appID)}, elog.HandleFactory(elog.LogLines, true), elog.LogAny, 1200)
				if err != nil {
					t.Fatal("Fail in waiting for app downloaded status: ", err)
				}
			})
			t.Run("Installed", func(t *testing.T) {
				if !checkLogs {
					t.Skip("no LOGS flag set - skipped")
				}
				err = ctx.LogChecker(devUUID, map[string]string{"devId": devUUID.String(), "msg": fmt.Sprintf(".*AppID:\"%s\".*state:INSTALLED.*", tt.appDefinition.appID)}, elog.HandleFactory(elog.LogLines, true), elog.LogAny, 1200)
				if err != nil {
					t.Fatal("Fail in waiting for app installed status: ", err)
				}
			})
			timeout := time.Duration(1200)

			if !checkLogs {
				timeout = 2400
			}
			t.Run("Running", func(t *testing.T) {
				err = ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "InfoContent.ainfo.AppID": tt.appDefinition.appID, "InfoContent.ainfo.state": "RUNNING"}, einfo.HandleFirst, einfo.InfoAny, timeout)
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
				handler := func(im *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface) bool {
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
				err = ctx.InfoLastCallback(devUUID, map[string]string{"devId": devUUID.String(), "InfoContent.ainfo.AppID": tt.appDefinition.appID, "InfoContent.ainfo.state": "RUNNING", "InfoContent.ainfo.network": ".*"}, handler)
			})
			t.Run("RemoteConsole", func(t *testing.T) {
				desktopName, err := utils.GetDesktopName(fmt.Sprintf("127.0.0.1:591%d", tt.vncDisplay+1), "")
				if err != nil {
					t.Fatal("Fail in connect to VNC ", err)
				}
				t.Logf("VNC DesktopName: %s", desktopName)
			})
			t.Run("Clean", func(t *testing.T) {
				CleanApplication(t, tt.appDefinition.imageFormat, tt.ymlToBuild)
			})
		})
	}

}

func CleanApplication(t *testing.T, format config.Format, ymlPath string) {
	imageFile := strings.TrimSuffix(filepath.Base(ymlPath), filepath.Ext(ymlPath))
	switch format {
	case config.Format_CONTAINER:
		containerImageFile := filepath.Join(eserverImageDist, "docker", fmt.Sprintf("%s.tar", imageFile))
		if _, err := os.Stat(containerImageFile); !os.IsNotExist(err) {
			if err = os.Remove(containerImageFile); err != nil {
				t.Fatal(err)
			}
		}
		t.Log("Container image remove done")
	case config.Format_QCOW2:
		vmImageFile := filepath.Join(eserverImageDist, "vm", fmt.Sprintf("%s.qcow2", imageFile))
		if _, err := os.Stat(vmImageFile); !os.IsNotExist(err) {
			if err = os.Remove(vmImageFile); err != nil {
				t.Fatal(err)
			}
		}
		t.Log("VM image remove done")
	default:
		t.Fatalf("Unsupported format: %d", format)
	}
}

func SetupApplication(t *testing.T, format config.Format, ymlPath string) {
	vars, err := utils.InitVars()
	if err != nil {
		t.Fatalf("error reading config: %s\n", err)
	}
	command := vars.EdenProg
	_, err = exec.LookPath(command)
	if err != nil {
		command = utils.ResolveAbsPath(vars.EdenBinDir + "/" + command)
		_, err = exec.LookPath(command)
		if err != nil {
			t.Fatalf("cannot obtain executable path: %s", err)
		}
	}
	imageFile := strings.TrimSuffix(filepath.Base(ymlPath), filepath.Ext(ymlPath))
	switch format {
	case config.Format_CONTAINER:
		containerImageFile := filepath.Join(eserverImageDist, "docker", fmt.Sprintf("%s.tar", imageFile))
		if _, err := os.Stat(containerImageFile); os.IsNotExist(err) {
			if err = utils.BuildContainer(ymlPath, defaults.DefaultImageTag); err != nil {
				t.Fatalf("Cannot build container image: %s", err)
			} else {
				t.Log("Container image build done")
			}
			if err = utils.DockerImageRepack(command, containerImageFile, defaults.DefaultImageTag); err != nil {
				t.Fatalf("Cannot repack container image: %s", err)
			} else {
				t.Log("Container image repack done")
			}
		} else {
			t.Log("Container image build done")
		}
	case config.Format_QCOW2:
		binDir := filepath.Dir(command)
		if _, err := os.Lstat(binDir); os.IsNotExist(err) {
			if err := os.MkdirAll(binDir, 0755); err != nil {
				t.Fatalf("Cannot create binDir: %s", err)
			}
		}
		linuxKitPath := filepath.Join(binDir, fmt.Sprintf("linuxkit-%s-%s", runtime.GOOS, runtime.GOARCH))
		linuxKitSymlinkPath := filepath.Join(binDir, "linuxkit")
		if _, err := os.Stat(linuxKitPath); os.IsNotExist(err) {
			linuxKitUrl := fmt.Sprintf("https://github.com/linuxkit/linuxkit/releases/download/%s/linuxkit-%s-%s", defaults.DefaultLinuxKitVersion, runtime.GOOS, runtime.GOARCH)
			if err = utils.DownloadFile(linuxKitPath, linuxKitUrl); err != nil {
				t.Fatalf("Download LinuxKit from %s failed: %s", linuxKitUrl, err)
			} else {
				if err := os.Chmod(linuxKitPath, 755); err != nil {
					t.Fatalf("Cannot Chmod LinuxKit: %s", err)
				}
				if err := os.Symlink(linuxKitPath, linuxKitSymlinkPath); err != nil {
					t.Fatalf("Cannot make LinuxKit symlink: %s", err)
				}
			}
			t.Log("LinuxKit download done")
		} else {
			t.Log("LinuxKit already exists")
		}
		vmImageFile := filepath.Join(eserverImageDist, "vm", fmt.Sprintf("%s.qcow2", imageFile))
		if _, err := os.Stat(vmImageFile); os.IsNotExist(err) {
			if err = eden.BuildVM(linuxKitSymlinkPath, ymlPath, vmImageFile); err != nil {
				t.Fatalf("Cannot build VM image: %s", err)
			} else {
				t.Log("VM image build done")
			}
		} else {
			t.Log("VM image build done")
		}
	default:
		t.Fatalf("Unsupported format: %d", format)
	}
}
