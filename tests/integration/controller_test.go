package integration

import (
	"testing"

	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/device"
)

//TestAdamOnBoard test onboarding into controller
func TestAdamOnBoard(t *testing.T) {
	ctx, err := controller.CloudPrepare()
	if err != nil {
		t.Fatalf("CloudPrepare: %s", err)
	}
	vars := ctx.GetVars()
	dev := device.CreateEdgeNode()
	dev.SetSerial(vars.EveSerial)
	dev.SetOnboardKey(vars.EveCert)
	dev.SetDevModel(vars.DevModel)
	t.Logf("Try to add onboarding")
	err = ctx.OnBoardDev(dev)
	if err != nil {
		t.Fatal(err)
	}
}

//TestControllerSetConfig test config set via controller
func TestControllerSetConfig(t *testing.T) {
	ctx, err := controller.CloudPrepare()
	if err != nil {
		t.Fatalf("CloudPrepare: %s", err)
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

//TestControllerGetConfig test config get via controller
func TestControllerGetConfig(t *testing.T) {
	ctx, err := controller.CloudPrepare()
	if err != nil {
		t.Fatalf("CloudPrepare: %s", err)
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
	ctx, err := controller.CloudPrepare()
	if err != nil {
		t.Fatalf("CloudPrepare: %s", err)
	}
	devUUID, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal("Fail in get first device: ", err)
	}
	t.Log(devUUID.GetID())
	err = ctx.LogChecker(devUUID.GetID(), map[string]string{"devId": devUUID.GetID().String()}, elog.HandleFactory(elog.LogLines, true), elog.LogAny, 600)
	if err != nil {
		t.Fatal("Fail in waiting for logs: ", err)
	}
}

//TestControllerInfo test info flow
func TestControllerInfo(t *testing.T) {
	ctx, err := controller.CloudPrepare()
	if err != nil {
		t.Fatalf("CloudPrepare: %s", err)
	}
	devUUID, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal("Fail in get first device: ", err)
	}
	t.Log(devUUID.GetID())
	err = ctx.InfoChecker(devUUID.GetID(), map[string]string{"devId": devUUID.GetID().String()}, einfo.HandleFirst, einfo.InfoAny, 300)
	if err != nil {
		t.Fatal("Fail in waiting for info: ", err)
	}
}
