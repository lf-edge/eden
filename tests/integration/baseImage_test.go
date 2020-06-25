package integration

import (
	"flag"
	"fmt"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/defaults"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/spf13/viper"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

var (
	eveBaseTag          = flag.String("baseos.eve.tag", "", "eve tag for base os")
	eveBaseDistRelative = flag.String("baseos.eve.location", "evebaseos", "location of EVE base os directory")
	download            = flag.Bool("baseos.eve.download", true, "EVE downloading flag")
	//eveRepo = flag.String("e", "https://github.com/lf-edge/eve.git", "eve repo used in clone mode")
	eveRepo  string
	adamDist string
	eveArch  string
	eveHV    string
)

//TestBaseImage test base image loading into eve
func TestBaseImage(t *testing.T) {
	if eveBaseTag == nil || *eveBaseTag == "" {
		t.Fatal("baseos.eve.tag has no value")
	} else {
		t.Logf("baseos.eve.tag: %s", *eveBaseTag)
	}
	ctx, err := controller.CloudPrepare()
	if err != nil {
		t.Fatalf("CloudPrepare: %s", err)
	}
	var baseImageTests = []struct {
		dataStoreID string
		imageID     string
		baseID      string
		imageFormat config.Format
	}{
		{
			defaults.DefaultDataStoreID,

			defaults.DefaultImageID,

			defaults.DefaultBaseID,

			config.Format_QCOW2,
		},
	}
	for ind, tt := range baseImageTests {
		rootFsPath := ""
		t.Run(fmt.Sprintf("Setup %d/%d", ind+1, len(baseImageTests)), func(t *testing.T) {
			rootFsPath = SetupBaseImage(t)
		})
		t.Logf("rootFsPath: %s", rootFsPath)
		rootFSName := strings.TrimSuffix(filepath.Base(rootFsPath), filepath.Ext(rootFsPath))
		rootFSName = strings.TrimPrefix(rootFSName, "rootfs-")
		re := regexp.MustCompile(defaults.DefaultRootFSVersionPattern)
		if !re.MatchString(rootFSName) {
			t.Fatalf("Filename of rootfs %s does not match pattern %s", rootFSName, defaults.DefaultRootFSVersionPattern)
		}
		baseOSVersion := rootFSName
		t.Run(baseOSVersion, func(t *testing.T) {
			err = prepareBaseImageLocal(ctx, tt.dataStoreID, tt.imageID, tt.baseID, rootFsPath, tt.imageFormat, baseOSVersion)

			if err != nil {
				t.Fatal("Fail in prepare base image from local file: ", err)
			}
			deviceCtx, err := ctx.GetDeviceFirst()
			if err != nil {
				t.Fatal("Fail in get first device: ", err)
			}
			deviceCtx.SetBaseOSConfig([]string{tt.baseID})
			devUUID := deviceCtx.GetID()
			err = ctx.ConfigSync(deviceCtx)
			if err != nil {
				t.Fatal("Fail in sync config with controller: ", err)
			}
			t.Run("Started", func(t *testing.T) {
				err := ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "shortVersion": baseOSVersion}, einfo.ZInfoDevSW, einfo.HandleFirst, einfo.InfoAny, 300)
				if err != nil {
					t.Fatal("Fail in waiting for base image update init: ", err)
				}
			})
			t.Run("Downloaded", func(t *testing.T) {
				err := ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "shortVersion": baseOSVersion, "downloadProgress": "100"}, einfo.ZInfoDevSW, einfo.HandleFirst, einfo.InfoAny, 1500)
				if err != nil {
					t.Fatal("Fail in waiting for base image download progress: ", err)
				}
			})
			t.Run("Logs", func(t *testing.T) {
				if !checkLogs {
					t.Skip("no LOGS flag set - skipped")
				}
				err = ctx.LogChecker(devUUID, map[string]string{"devId": devUUID.String(), "eveVersion": baseOSVersion}, elog.HandleFirst, elog.LogAny, 1200)
				if err != nil {
					t.Fatal("Fail in waiting for base image logs: ", err)
				}
			})
			timeout := time.Duration(1200)

			if !checkLogs {
				timeout = 2400
			}
			t.Run("Active", func(t *testing.T) {
				err = ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "shortVersion": baseOSVersion, "status": "INSTALLED", "partitionState": "(inprogress|active)"}, einfo.ZInfoDevSW, einfo.HandleFirst, einfo.InfoAny, timeout)
				if err != nil {
					t.Fatal("Fail in waiting for base image installed status: ", err)
				}
			})
		})
		t.Run(fmt.Sprintf("Clean %d/%d", ind+1, len(baseImageTests)), func(t *testing.T) {
			CleanBaseImage(t)
		})
	}

}

func SetupBaseImage(t *testing.T) (fileToUse string) {
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
	viperLoaded, err := utils.LoadConfigFile("")
	if err != nil {
		t.Fatalf("error reading config: %s", err.Error())
	}
	if viperLoaded {
		eveRepo = viper.GetString("eve.repo")
		adamDist = utils.ResolveAbsPath(viper.GetString("adam.dist"))
		eveHV = viper.GetString("eve.hv")
		eveArch = viper.GetString("eve.arch")
		eserverImageDist = utils.ResolveAbsPath(viper.GetString("eden.images.dist"))
	}
	eveBaseDist := utils.ResolveAbsPath(*eveBaseDistRelative)
	if !*download {
		if _, err := os.Stat(eveBaseDist); os.IsNotExist(err) {
			if err := utils.CloneFromGit(eveBaseDist, eveRepo, *eveBaseTag); err != nil {
				t.Fatalf("cannot clone BASE EVE: %s", err)
			} else {
				t.Log("clone BASE EVE done")
			}
			if _, _, err = utils.MakeEveInRepo(eveBaseDist, adamDist, eveArch, eveHV, "raw", true); err != nil {
				t.Fatalf("cannot MakeEveInRepo base: %s", err)
			} else {
				t.Log("MakeEveInRepo base done")
			}
		} else {
			t.Logf("BASE EVE already exists in dir: %s", eveBaseDist)
		}
	} else {
		if _, err := os.Stat(eveBaseDist); os.IsNotExist(err) {
			if err := utils.DownloadEveFormDocker(command, eveBaseDist, eveArch, eveHV, *eveBaseTag, false); err != nil {
				t.Fatalf("cannot download Base EVE: %s", err)
			} else {
				t.Log("download Base EVE done")
			}
			if err := utils.ChangeConfigPartAndRootFs(command, eveBaseDist, adamDist, eveArch, eveHV); err != nil {
				t.Fatalf("cannot ChangeConfigPartAndRootFs Base EVE: %s", err)
			} else {
				t.Log("ChangeConfigPartAndRootFs Base EVE done")
			}
		} else {
			t.Logf("Base EVE already exists in dir: %s", eveBaseDist)
		}
	}
	rootFsPath, err := utils.GetFileFollowLinks(filepath.Join(eveBaseDist, "dist", eveArch, "installer", fmt.Sprintf("rootfs-%s.img", eveHV)))
	if err != nil {
		t.Fatalf("GetFileFollowLinks: %s", err)
	}
	if err = utils.CopyFileNotExists(rootFsPath, filepath.Join(eserverImageDist, "baseos", filepath.Base(rootFsPath))); err != nil {
		t.Fatalf("Copy EVE base image failed: %s", err)
	} else {
		t.Log("Copy EVE base image done")
	}
	return filepath.Base(rootFsPath)
}

func CleanBaseImage(t *testing.T) {
	eveBaseDist := utils.ResolveAbsPath(*eveBaseDistRelative)
	if _, err := os.Stat(eveBaseDist); !os.IsNotExist(err) {
		if err = os.RemoveAll(eveBaseDist); err != nil {
			t.Fatalf("error in %s delete: %s", eveBaseDist, err)
		}
	}
}
