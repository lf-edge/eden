package integration

import (
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"os"
	"path"
	"testing"
	"time"
)

func TestAdamOnBoard(t *testing.T) {
	ctx, err := controllerPrepare()
	if ctx == nil {
		t.Fatal(err)
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

func TestControllerSetConfig(t *testing.T) {
	ctx, err := controllerPrepare()
	if err != nil {
		t.Fatal(err)
	}
	devUUID, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal(err)
	}
	err = ctx.ConfigSync(devUUID.GetID())
	if err != nil {
		t.Fatal(err)
	}
}

func TestControllerGetConfig(t *testing.T) {
	ctx, err := controllerPrepare()
	if err != nil {
		t.Fatal(err)
	}
	devUUID, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal(err)
	}
	config, err := ctx.ConfigGet(devUUID.GetID())
	if err != nil {
		t.Fatal(err)
	}
	t.Log(config)
}

func TestControllerLogs(t *testing.T) {
	ctx, err := controllerPrepare()
	if err != nil {
		t.Fatal(err)
	}
	devUUID, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal(err)
	}
	err = ctx.LogChecker(devUUID.GetID(), map[string]string{"devId": devUUID.GetID().String()}, 600)
	if err != nil {
		t.Fatal(err)
	}
}

func TestControllerInfo(t *testing.T) {
	ctx, err := controllerPrepare()
	if err != nil {
		t.Fatal(err)
	}
	devUUID, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal(err)
	}
	err = ctx.InfoChecker(devUUID.GetID(), map[string]string{"devId": devUUID.GetID().String()}, einfo.ZInfoDevSW, 300)
	if err != nil {
		t.Fatal(err)
	}
}
