package integration

import (
	"fmt"
	"github.com/itmo-eve/eden/pkg/adam"
	"github.com/itmo-eve/eden/pkg/cloud"
	"github.com/itmo-eve/eden/pkg/device"
	"github.com/itmo-eve/eden/pkg/einfo"
	"github.com/itmo-eve/eden/pkg/elog"
	"github.com/itmo-eve/eden/pkg/utils"
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
	ctx := adam.Ctx{
		Dir: adamDir,
		URL: fmt.Sprintf("https://%s:%s", ip, port),
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

func TestAdamSetConfig(t *testing.T) {
	ctx, devUUID, err := adamPrepare()
	if err != nil {
		t.Fatal(err)
	}
	cloudCxt := &cloud.Ctx{}
	deviceCtx := device.CreateWithBaseConfig(*devUUID, cloudCxt)
	b, err := deviceCtx.GenerateJSONBytes()
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
	err = elog.LogChecker(ctx.GetLogsDir(devUUID), map[string]string{"devId": devUUID.String()}, 600)
	if err != nil {
		t.Fatal(err)
	}
}

func TestAdamInfo(t *testing.T) {
	ctx, devUUID, err := adamPrepare()
	if err != nil {
		t.Fatal(err)
	}
	err = einfo.InfoChecker(ctx.GetInfoDir(devUUID), map[string]string{"devId": devUUID.String()}, einfo.ZInfoDevSW, 300)
	if err != nil {
		t.Fatal(err)
	}
}
