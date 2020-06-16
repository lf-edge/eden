package reboot

import (
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/projects"
	"github.com/lf-edge/eve/api/go/info"
	"testing"
)

var rebootTime *timestamp.Timestamp
var rebootReason string
var rebooted bool

func checkReboot(im *info.ZInfoMsg, ds []*einfo.ZInfoMsgInterface, infoType einfo.ZInfoType) bool {
	lrbt := im.GetDinfo().LastRebootTime
	lrbr := im.GetDinfo().LastRebootReason

	if rebootTime == nil {
		rebootTime = lrbt
		rebootReason = lrbr
	} else {
		if !proto.Equal(rebootTime, lrbt) {
			rebootTime = lrbt
			rebootReason = lrbr
			rebooted = true
		}
	}
	return rebooted
}

func CheckRebootInfo(t *testing.T, tc *projects.TestContext, description string, args projects.AssertArgs) {
	t.Log("CheckRebootInfo")

	ctx, err := controller.CloudPrepare()
	if err != nil {
		t.Fatalf("Fail in CloudPrepare: %s", err)
	}
	devUUID, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal("Fail in get first device: ", err)
	}

	for {
		err = ctx.InfoChecker(devUUID.GetID(), map[string]string{"devId": devUUID.GetID().String(), "rebootConfigCounter": ".*"}, einfo.ZInfoDinfo, checkReboot, einfo.InfoNew, 3000)
		if err != nil {
			t.Fatal("Fail in waiting for info: ", err)
		}
		if rebooted {
			t.Logf("Rebooted at %s with reason '%s'\n", ptypes.TimestampString(rebootTime), rebootReason)
			return
		} else {
			t.Logf("Not rebooted, last reboot counter was at %s\n", ptypes.TimestampString(rebootTime))
		}
	}
}
