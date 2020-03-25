package integration

import (
	"errors"
	"fmt"
	"github.com/itmo-eve/eden/pkg/adam"
	"github.com/itmo-eve/eden/pkg/cloud"
	"github.com/itmo-eve/eden/pkg/device"
	"github.com/itmo-eve/eden/pkg/einfo"
	"github.com/itmo-eve/eden/pkg/elog"
	"github.com/itmo-eve/eden/pkg/utils"
	"github.com/lf-edge/eve/api/go/config"
	uuid "github.com/satori/go.uuid"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"
)

const eveCert = "/adam/run/config/onboard.cert.pem"

func TestAdamOnBoard(t *testing.T) {
	currentPath, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	ip := os.Getenv("IP")
	if len(ip) == 0 {
		ip, err = utils.GetIPForDockerAccess()
		if err != nil {
			t.Fatal(err)
		}
	}
	port := os.Getenv("ADAM_PORT")
	if len(port) == 0 {
		port = "3333"
	}
	adamDir := os.Getenv("ADAM_DIST")
	if len(adamDir) == 0 {
		adamDir = path.Join(filepath.Dir(filepath.Dir(currentPath)), "dist", "adam")
		if stat, err := os.Stat(adamDir); err != nil || !stat.IsDir() {
			t.Fatal("Failed to get adam dir")
		}
	}
	serial := os.Getenv("EVE_SERIAL")
	if len(serial) == 0 {
		serial = "31415926"
	}
	ctx := adam.AdamCtx{
		Dir: adamDir,
		Url: fmt.Sprintf("https://%s:%s", ip, port),
	}
	t.Logf("Try to add onboarding")
	err = ctx.Register(eveCert, serial)
	if err != nil {
		t.Fatal(err)
	}
	res, err := ctx.OnBoardList()
	if err != nil {
		t.Fatal(err)
	}
	if len(res) == 0 {
		t.Fatal("No onboard in list")
	}
	t.Log(res)

	maxRepeat := 20
	delayTime := 20 * time.Second

	for i := 0; i < maxRepeat; i++ {
		cmdOut, err := ctx.DeviceList()
		if err != nil {
			t.Fatal(err)
		}
		if len(cmdOut) > 0 {
			t.Logf("Done onboarding in adam!")
			t.Logf("Device uuid: %s", cmdOut)
			return
		}
		t.Logf("Attempt to list devices (%d) of (%d)", i, maxRepeat)
		time.Sleep(delayTime)
	}
	t.Fatal("Onboarding timeout")
}

func adamPrepare() (adamCtx *adam.AdamCtx, id *uuid.UUID, err error) {
	currentPath, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	ip := os.Getenv("IP")
	if len(ip) == 0 {
		ip, err = utils.GetIPForDockerAccess()
		if err != nil {
			return nil, nil, err
		}
	}
	port := os.Getenv("ADAM_PORT")
	if len(port) == 0 {
		port = "3333"
	}
	adamDir := os.Getenv("ADAM_DIST")
	if len(adamDir) == 0 {
		adamDir = path.Join(filepath.Dir(filepath.Dir(currentPath)), "dist", "adam")
		if stat, err := os.Stat(adamDir); err != nil || !stat.IsDir() {
			return nil, nil, err
		}
	}
	ctx := adam.AdamCtx{
		Dir: adamDir,
		Url: fmt.Sprintf("https://%s:%s", ip, port),
	}
	cmdOut, err := ctx.DeviceList()
	if err != nil {
		return nil, nil, err
	}
	if len(cmdOut) > 0 {
		devUUID, err := uuid.FromString(cmdOut[0])
		if err != nil {
			return nil, nil, err
		}
		return &ctx, &devUUID, nil
	} else {
		return nil, nil, errors.New("no device found")
	}
}

func TestAdamSetConfig(t *testing.T) {
	ctx, devUUID, err := adamPrepare()
	if err != nil {
		t.Fatal(err)
	}
	cloudCxt := &cloud.CloudCtx{}
	deviceCtx := device.CreateWithBaseConfig(*devUUID, cloudCxt)
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
}

func TestAdamLogs(t *testing.T) {
	ctx, devUUID, err := adamPrepare()
	if err != nil {
		t.Fatal(err)
	}
	err = elog.LogWatchWithTimeout(ctx.GetLogsDir(devUUID), map[string]string{"devId": devUUID.String()}, elog.HandleFirst, 600)
	if err != nil {
		t.Fatal(err)
	}
}

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

	err = elog.LogWatchWithTimeout(ctx.GetLogsDir(devUUID), map[string]string{"devId": devUUID.String(), "eveVersion": baseOSVersion}, elog.HandleFirst, 600)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAdamInfo(t *testing.T) {
	ctx, devUUID, err := adamPrepare()
	if err != nil {
		t.Fatal(err)
	}
	err = einfo.InfoWatchWithTimeout(ctx.GetInfoDir(devUUID), map[string]string{"devId": devUUID.String()}, einfo.ZInfoDevSWFind, einfo.HandleFirst, 600)
	if err != nil {
		t.Fatal(err)
	}
}
