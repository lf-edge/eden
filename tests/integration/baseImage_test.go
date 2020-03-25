package integration

import (
	"fmt"
	"github.com/itmo-eve/eden/pkg/cloud"
	"github.com/itmo-eve/eden/pkg/device"
	"github.com/itmo-eve/eden/pkg/einfo"
	"github.com/itmo-eve/eden/pkg/elog"
	"github.com/itmo-eve/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	"github.com/lf-edge/eve/api/go/info"
	"os"
	"path"
	"path/filepath"
	"testing"
)

func TestBaseImage(t *testing.T) {
	ctx, devUUID, err := adamPrepare()
	if err != nil {
		t.Fatal(err)
	}
	baseOSVersion := os.Getenv("EVE_BASE_VERSION")
	if len(baseOSVersion) == 0 {
		eveBaseRef := os.Getenv("EVE_BASE_REF")
		if len(eveBaseRef) == 0 {
			eveBaseRef = "4.10.0"
		}
		zArch := os.Getenv("ZARCH")
		if len(eveBaseRef) == 0 {
			zArch = "amd64"
		}
		baseOSVersion = fmt.Sprintf("%s-%s", eveBaseRef, zArch)
	}

	dsId := "eab8761b-5f89-4e0b-b757-4b87a9fa93ec"

	imageID := "1ab8761b-5f89-4e0b-b757-4b87a9fa93ec"

	baseID := "22b8761b-5f89-4e0b-b757-4b87a9fa93ec"

	imageName := path.Join(filepath.Dir(ctx.Dir), "images", "baseos.qcow2")

	fi, err := os.Stat(imageName)
	if err != nil {
		t.Fatal(err)
	}
	size := fi.Size()

	sha256sum, err := utils.SHA256SUM(imageName)
	if err != nil {
		t.Fatal(err)
	}

	cloudCxt := &cloud.CloudCtx{}
	err = cloudCxt.AddDatastore(&config.DatastoreConfig{
		Id:       dsId,
		DType:    config.DsType_DsHttp,
		Fqdn:     "http://mydomain.adam:8888",
		ApiKey:   "",
		Password: "",
		Dpath:    "",
		Region:   "",
	})
	if err != nil {
		t.Fatal(err)
	}
	img := &config.Image{
		Uuidandversion: &config.UUIDandVersion{
			Uuid:    imageID,
			Version: "4",
		},
		Name:      filepath.Base(imageName),
		Sha256:    sha256sum,
		Iformat:   config.Format_QCOW2,
		DsId:      dsId,
		SizeBytes: size,
		Siginfo: &config.SignatureInfo{
			Intercertsurl: "",
			Signercerturl: "",
			Signature:     nil,
		},
	}
	err = cloudCxt.AddImage(img)
	if err != nil {
		t.Fatal(err)
	}
	err = cloudCxt.AddBaseOsConfig(&config.BaseOSConfig{
		Uuidandversion: &config.UUIDandVersion{
			Uuid:    baseID,
			Version: "4",
		},
		Drives: []*config.Drive{{
			Image:        img,
			Readonly:     false,
			Preserve:     false,
			Drvtype:      config.DriveType_Unclassified,
			Target:       config.Target_TgtUnknown,
			Maxsizebytes: size,
		}},
		Activate:      true,
		BaseOSVersion: baseOSVersion,
		BaseOSDetails: nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	deviceCtx := device.CreateWithBaseConfig(*devUUID, cloudCxt)
	deviceCtx.SetBaseOSConfig([]string{baseID})
	b, err := deviceCtx.GenerateJsonBytes()
	if err != nil {
		t.Fatal(err)
	}
	configToSet := fmt.Sprintf("%s", string(b))
	t.Log(configToSet)
	res, err := ctx.ConfigSet(devUUID.String(), configToSet)
	if err != nil {
		t.Log(res)
		t.Fatal(err)
	}
	t.Run("Started", func(t *testing.T) {
		done := make(chan error)

		go func() {
			err = einfo.InfoWatchWithTimeout(ctx.GetInfoDir(devUUID), map[string]string{"devId": devUUID.String(), "shortVersion": baseOSVersion}, einfo.ZInfoDevSWFind, einfo.HandleFirst, 300)
			done <- err
		}()
		go func() {
			handler := func(im *info.ZInfoMsg, ds []*info.ZInfoDevSW) bool {
				einfo.ZInfoDevSWPrn(im, ds)
				done <- nil
				return true
			}
			err = einfo.InfoLast(ctx.GetInfoDir(devUUID), map[string]string{"devId": devUUID.String(), "shortVersion": baseOSVersion}, einfo.ZInfoDevSWFind, handler)
			if err != nil {
				done <- err
			}
		}()
		err = <-done
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Downloaded", func(t *testing.T) {
		done := make(chan error)

		go func() {
			err = einfo.InfoWatchWithTimeout(ctx.GetInfoDir(devUUID), map[string]string{"devId": devUUID.String(), "shortVersion": baseOSVersion, "downloadProgress": "100"}, einfo.ZInfoDevSWFind, einfo.HandleFirst, 1500)
			done <- err
		}()
		go func() {
			handler := func(im *info.ZInfoMsg, ds []*info.ZInfoDevSW) bool {
				einfo.ZInfoDevSWPrn(im, ds)
				done <- nil
				return true
			}
			err = einfo.InfoLast(ctx.GetInfoDir(devUUID), map[string]string{"devId": devUUID.String(), "shortVersion": baseOSVersion, "downloadProgress": "100"}, einfo.ZInfoDevSWFind, handler)
			if err != nil {
				done <- err
			}
		}()
		err = <-done
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Active", func(t *testing.T) {
		done := make(chan error)

		go func() {
			err = elog.LogWatchWithTimeout(ctx.GetLogsDir(devUUID), map[string]string{"devId": devUUID.String(), "eveVersion": baseOSVersion}, elog.HandleFirst, 1000)
			done <- err
		}()
		go func() {
			handler := func(item *elog.LogItem) bool {
				done <- nil
				return elog.HandleFirst(item)
			}
			err = elog.LogLast(ctx.GetLogsDir(devUUID), map[string]string{"devId": devUUID.String(), "eveVersion": baseOSVersion}, handler)
			if err != nil {
				done <- err
			}
		}()
		err = <-done
		if err != nil {
			t.Fatal(err)
		}
	})
}
