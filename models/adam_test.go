package models

import (
	"../utils"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"
)

func TestAdamCtx_OnBoard(t *testing.T) {
	currentPath, err := os.Getwd()
	if err != nil {
		t.Errorf(err.Error())
	}
	ip := os.Getenv("IP")
	if len(ip) == 0 {
		ip, err = utils.GetIPForDockerAccess()
		if err != nil {
			t.Errorf(err.Error())
		}
	}
	adamDir := path.Join(filepath.Dir(currentPath), "dist", "adam")
	if stat, err := os.Stat(adamDir); err != nil || !stat.IsDir() {
		t.Errorf("Failed to get adam dir")
	}
	ctx := AdamCtx{
		Dir: adamDir,
		Url: fmt.Sprintf("https://%s:3333", ip),
	}
	t.Logf("Try to add onboarding")
	err = ctx.OnBoardAdd("31415926")
	if err != nil {
		t.Errorf(err.Error())
	}
	res, err := ctx.OnBoardList()
	if err != nil {
		t.Error(err)
	}
	if len(res) == 0 {
		t.Errorf("No onboard in list")
	}
	t.Logf(res)

	maxRepeat := 20
	delayTime := 20 * time.Second

	for i := 0; i < maxRepeat; i++ {
		cmdOut, err := ctx.DeviceList()
		if err != nil {
			t.Error(err)
		}
		if len(cmdOut) > 0 {
			t.Logf("Done onboarding in adam!")
			t.Logf("Device uuid: %s", cmdOut)
			return
		}
		t.Logf("Attempt to list devices (%d) of (%d)", i, maxRepeat)
		time.Sleep(delayTime)
	}
	t.Errorf("Onboarding timeout")
}
