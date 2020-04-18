package integration

import (
	"fmt"
	"testing"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eden/pkg/controller/einfo"
)

var lastRebootTime *timestamp.Timestamp
var lastRebootReason string
var rebooted bool

func checkRebootTime(im *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface, infoType einfo.ZInfoType) bool {
	lrbt := im.GetDinfo().LastRebootTime
	lrbr := im.GetDinfo().LastRebootReason

	if lastRebootTime == nil {
		lastRebootTime = lrbt
		lastRebootReason = lrbr
		rebooted = true
	} else {
		fmt.Printf("lastRebootTime: %s\n",lastRebootTime)
		fmt.Printf("lrbt: %s\n",lrbt)
		if proto.Equal(lastRebootTime, lrbt) {
			rebooted = false
		} else {
			lastRebootTime = lrbt
			lastRebootReason = lrbr
			rebooted = true
		}
	}
	fmt.Printf("rebooted: %v\n",rebooted)
	return rebooted
}

//TestAdamOnBoard test info flow
func TestRebootInfo(t *testing.T) {
	ctx, err := controllerPrepare()
	if err != nil {
		t.Fatal("Fail in controller prepare: ", err)
	}
	devUUID, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal("Fail in get first device: ", err)
	}
	err = ctx.InfoChecker(devUUID.GetID(), map[string]string{"devId": devUUID.GetID().String(), "lastRebootTime":".*"}, einfo.ZInfoDinfo, checkRebootTime, einfo.InfoAny, 300)
	if err != nil {
		t.Fatal("Fail in waiting for info: ", err)
	}
	t.Logf("1. Rebooted at %s with reason '%s'\n", lastRebootTime, lastRebootReason)
	for {
		err = ctx.InfoChecker(devUUID.GetID(), map[string]string{"devId": devUUID.GetID().String(), "lastRebootTime":".*"}, einfo.ZInfoDinfo, checkRebootTime, einfo.InfoNew, 3000)
		if err != nil {
			t.Fatal("Fail in waiting for info: ", err)
		}
		if rebooted {
			t.Logf("2. Rebooted at %s with reason '%s'\n", lastRebootTime, lastRebootReason)
			break
		} else {
			t.Logf("Not rebooted: %s\n", lastRebootTime)
		}
	}
}
