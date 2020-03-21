package integration

import (
	"errors"
	"fmt"
	"github.com/itmo-eve/eden/pkg/adam"
	"github.com/itmo-eve/eden/pkg/cloud"
	"github.com/itmo-eve/eden/pkg/device"
	"github.com/itmo-eve/eden/pkg/elog"
	"github.com/itmo-eve/eden/pkg/utils"
	uuid "github.com/satori/go.uuid"
	"log"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"
)

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
		Url: fmt.Sprintf("https://%s:3333", ip),
	}
	t.Logf("Try to add onboarding")
	err = ctx.OnBoardAdd(serial)
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
	adamDir := os.Getenv("ADAM_DIST")
	if len(adamDir) == 0 {
		adamDir = path.Join(filepath.Dir(filepath.Dir(currentPath)), "dist", "adam")
		if stat, err := os.Stat(adamDir); err != nil || !stat.IsDir() {
			return nil, nil, err
		}
	}
	ctx := adam.AdamCtx{
		Dir: adamDir,
		Url: fmt.Sprintf("https://%s:3333", ip),
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
		log.Fatal(err)
	}
	cloudCxt := &cloud.CloudCtx{}
	deviceCtx := device.CreateWithBaseConfig(*devUUID, cloudCxt)
	b, err := deviceCtx.GenerateJsonBytes()
	if err != nil {
		log.Fatal(err)
	}
	configToSet := fmt.Sprintf("%s", string(b))
	log.Print(configToSet)
	res, err := ctx.ConfigSet(devUUID.String(), configToSet)
	if err != nil {
		t.Log(res)
		t.Fatal(err)
	}
}

func TestAdamLogs(t *testing.T) {
	ctx, devUUID, err := adamPrepare()
	if err != nil {
		log.Fatal(err)
	}
	err = elog.LogWatchWithTimeout(path.Join(ctx.Dir, "run", "adam", "device", devUUID.String(), "logs"), map[string]string{}, elog.HandleFirst, 180)
	if err != nil {
		t.Fatal(err)
	}
}
