package integration

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	"os"
	"path"
	"path/filepath"
	"testing"
)

func TestBaseImage(t *testing.T) {
	ctx, err := controllerPrepare()
	if err != nil {
		t.Fatal(err)
	}
	devCtx, err := ctx.GetDeviceFirst()
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

	dsID := "eab8761b-5f89-4e0b-b757-4b87a9fa93ec"

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
	err = ctx.AddDatastore(&config.DatastoreConfig{
		Id:       dsID,
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
		DsId:      dsID,
		SizeBytes: size,
		Siginfo: &config.SignatureInfo{
			Intercertsurl: "",
			Signercerturl: "",
			Signature:     nil,
		},
	}
	err = ctx.AddImage(img)
	if err != nil {
		t.Fatal(err)
	}
	err = ctx.AddBaseOsConfig(&config.BaseOSConfig{
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
	devCtx.SetBaseOSConfig([]string{baseID})
	devUUID := devCtx.GetID()
	err = ctx.ConfigSync(devUUID)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("Started", func(t *testing.T) {
		err := ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "shortVersion": baseOSVersion}, einfo.ZInfoDevSW, 300)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Downloaded", func(t *testing.T) {
		err := ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "shortVersion": baseOSVersion, "downloadProgress": "100"}, einfo.ZInfoDevSW, 1500)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Logs", func(t *testing.T) {
		err = ctx.LogChecker(devUUID, map[string]string{"devId": devUUID.String(), "eveVersion": baseOSVersion}, 1200)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Active", func(t *testing.T) {
		err = ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "shortVersion": baseOSVersion, "status": "INSTALLED"}, einfo.ZInfoDevSW, 1200)
		if err != nil {
			t.Fatal(err)
		}
	})
}
