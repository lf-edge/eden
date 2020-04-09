package integration

import (
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"os"
	"path"
	"testing"
	"time"
)

//TestAdamOnBoard test onboarding into controller
//environment variable EVE_CERT - path to eve onboarding cert
//environment variable EVE_SERIAL - serial number of eve
func TestAdamOnBoard(t *testing.T) {
	ctx, err := controllerPrepare()
	if ctx == nil {
		t.Fatal("Fail in controller prepare: ", err)
	}
	devUUID, err := ctx.GetDeviceFirst()
	if devUUID == nil {
		eveCert := os.Getenv("EVE_CERT")
		if len(eveCert) == 0 {
			eveCert = path.Join(ctx.GetDir(), "run", "config", "onboard.cert.pem")
		}
		serial := os.Getenv("EVE_SERIAL")
		if len(serial) == 0 {
			serial = "31415926"
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
}

//TestAdamOnBoard test config set via controller
func TestControllerSetConfig(t *testing.T) {
	ctx, err := controllerPrepare()
	if err != nil {
		t.Fatal("Fail in controller prepare: ", err)
	}
	deviceCtx, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal("Fail in get first device: ", err)
	}
	err = ctx.ConfigSync(deviceCtx)
	if err != nil {
		t.Fatal("Fail in config sync with device: ", err)
	}
}

//TestAdamOnBoard test config get via controller
func TestControllerGetConfig(t *testing.T) {
	ctx, err := controllerPrepare()
	if err != nil {
		t.Fatal("Fail in controller prepare: ", err)
	}
	devUUID, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal("Fail in get first device: ", err)
	}
	config, err := ctx.ConfigGet(devUUID.GetID())
	if err != nil {
		t.Fatal("Fail in set config: ", err)
	}
	t.Log(config)
}

//TestAdamOnBoard test logs flow
func TestControllerLogs(t *testing.T) {
	ctx, err := controllerPrepare()
	if err != nil {
		t.Fatal("Fail in controller prepare: ", err)
	}
	devUUID, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal("Fail in get first device: ", err)
	}
	err = ctx.LogChecker(devUUID.GetID(), map[string]string{"devId": devUUID.GetID().String()}, 600)
	if err != nil {
		t.Fatal("Fail in waiting for logs: ", err)
	}
}

//TestAdamOnBoard test info flow
func TestControllerInfo(t *testing.T) {
	ctx, err := controllerPrepare()
	if err != nil {
		t.Fatal("Fail in controller prepare: ", err)
	}
	devUUID, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal("Fail in get first device: ", err)
	}
	err = ctx.InfoChecker(devUUID.GetID(), map[string]string{"devId": devUUID.GetID().String()}, einfo.ZInfoDevSW, 300)
	if err != nil {
		t.Fatal("Fail in waiting for info: ", err)
	}
}
