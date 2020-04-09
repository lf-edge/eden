package integration

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eve/api/go/config"
	"os"
	"testing"
	"time"
)

//TestBaseImage test base image loading into eve
//environment variable EVE_BASE_REF - version of eve image
//environment variable ZARCH - architecture of eve image
func TestBaseImage(t *testing.T) {
	ctx, err := controllerPrepare()
	if err != nil {
		t.Fatal("Fail in controller prepare: ", err)
	}
	eveBaseRef := os.Getenv("EVE_BASE_REF")
	if len(eveBaseRef) == 0 {
		eveBaseRef = "4.10.0"
	}
	zArch := os.Getenv("ZARCH")
	if len(eveBaseRef) == 0 {
		zArch = "amd64"
	}
	HV := os.Getenv("HV")
	if HV == "xen" {
		HV = ""
	}
	var baseImageTests = []struct {
		dataStoreID       string
		imageID           string
		baseID            string
		imageRelativePath string
		imageFormat       config.Format
		eveBaseRef        string
		zArch             string
		HV                string
	}{
		{eServerDataStoreID,

			"1ab8761b-5f89-4e0b-b757-4b87a9fa93ec",

			"22b8761b-5f89-4e0b-b757-4b87a9fa93ec",

			"baseos.qcow2",
			config.Format_QCOW2,
			eveBaseRef,
			zArch,
			HV,
		},
	}
	for _, tt := range baseImageTests {
		baseOSVersion := fmt.Sprintf("%s-%s", tt.eveBaseRef, tt.zArch)
		if tt.HV != "" {
			baseOSVersion = fmt.Sprintf("%s-%s-%s", tt.eveBaseRef, tt.zArch, tt.HV)
		}
		t.Run(baseOSVersion, func(t *testing.T) {

			err = prepareBaseImageLocal(ctx, tt.dataStoreID, tt.imageID, tt.baseID, tt.imageRelativePath, tt.imageFormat, baseOSVersion)

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
				err := ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "shortVersion": baseOSVersion}, einfo.ZInfoDevSW, 300)
				if err != nil {
					t.Fatal("Fail in waiting for base image update init: ", err)
				}
			})
			t.Run("Downloaded", func(t *testing.T) {
				err := ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "shortVersion": baseOSVersion, "downloadProgress": "100"}, einfo.ZInfoDevSW, 1500)
				if err != nil {
					t.Fatal("Fail in waiting for base image download progress: ", err)
				}
			})
			t.Run("Logs", func(t *testing.T) {
				if !checkLogs {
					t.Skip("no LOGS flag set - skipped")
				}
				err = ctx.LogChecker(devUUID, map[string]string{"devId": devUUID.String(), "eveVersion": baseOSVersion}, 1200)
				if err != nil {
					t.Fatal("Fail in waiting for base image logs: ", err)
				}
			})
			timeout := time.Duration(1200)

			if !checkLogs {
				timeout = 2400
			}
			t.Run("Active", func(t *testing.T) {
				err = ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "shortVersion": baseOSVersion, "status": "INSTALLED", "partitionState": "(inprogress|active)"}, einfo.ZInfoDevSW, timeout)
				if err != nil {
					t.Fatal("Fail in waiting for base image installed status: ", err)
				}
			})
		})
	}

}
