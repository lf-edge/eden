package integration

import (
	"fmt"
	"strconv"
	"testing"
	"time"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/einfo"
)

var lastRebootTime *timestamp.Timestamp
var lastRebootReason string
var rebooted bool

func setupTestCase(t *testing.T) func(t *testing.T) {
	t.Log("Setup test case", t.Name())
	ETests["BoolTest"] = []EdenTest{
		EdenTest{
			Name: "WaitHook",
			Test: WaitHook,
			Args: EdenTestArgs{"secs":"5"},
			Result: "5",
		},
		EdenTest{
			Name: "FalseHook",
			Test: BoolHook,
			Args: EdenTestArgs{"val":"false"},
			Result: "false",
			//Result: "true",
		},/*
		EdenTest{
			Name: "TrueHook",
			Test: BoolHook,
			Args: EdenTestArgs{"val":"true"},
			Result: "true",
		},*/
	}
	
	ETests["RebootTest"] = []EdenTest{
		EdenTest{
			Name: "CheckRebootInfo",
			Test: CheckRebootInfo,
			Args: EdenTestArgs{},
			Result: "reboot",
		},
		EdenTest{
			Name: "WaitHook",
			Test: WaitHook,
			Args: EdenTestArgs{"secs":"1000"},
			Result: "1000",
		},
	}
	
	return func(t *testing.T) {
		t.Log("Teardown test case", t.Name())
	}
}

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

func CheckRebootInfo(t *testing.T, name string, args EdenTestArgs) string {
	fmt.Println("CheckRebootInfo")
	ctx, err := controller.CloudPrepare()
	if err != nil {
		t.Fatalf("Fail in CloudPrepare: %s", err)
		return "fail"
	}
	devUUID, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal("Fail in get first device: ", err)
		return "fail"
	}
	err = ctx.InfoChecker(devUUID.GetID(), map[string]string{"devId": devUUID.GetID().String(), "lastRebootTime":".*"}, einfo.ZInfoDinfo, checkRebootTime, einfo.InfoAny, 300)
	if err != nil {
		t.Fatal("Fail in waiting for info: ", err)
		return "fail"
	}
	t.Logf("Previous reboot at %s with reason '%s'\n", lastRebootTime, lastRebootReason)
	for {
		err = ctx.InfoChecker(devUUID.GetID(), map[string]string{"devId": devUUID.GetID().String(), "lastRebootTime":".*"}, einfo.ZInfoDinfo, checkRebootTime, einfo.InfoNew, 3000)
		if err != nil {
			t.Fatal("Fail in waiting for info: ", err)
			return "fail"
		}
		if rebooted {
			t.Logf("Rebooted again at %s with reason '%s'\n", lastRebootTime, lastRebootReason)
			return "reboot"
		} else {
			t.Logf("Not rebooted, lastRebootTime: %s\n", lastRebootTime)
			return "fail"
		}
	}
}

func BoolHook(t *testing.T, name string, args EdenTestArgs) string {
	fmt.Println("BoolHook", t.Name(), args["val"])
	return args["val"]
}

func WaitHook(t *testing.T, name string, args EdenTestArgs) string {
	s := args["secs"]
	if secs, err:=strconv.Atoi(s); err != nil {
		t.Fatalf("Can't convert '%s' to seconds\n", s)
		return "fail"
	} else {
		fmt.Println("WaitHook", secs, "sec.")
		time.Sleep(time.Duration(secs)*time.Second)
		fmt.Println("WaitHook finished")
	}
	return s
}

func TestSets(t *testing.T) {
	teardownTestCase := setupTestCase(t)
	defer teardownTestCase(t)

	for name, _ := range ETests {
		t.Run(name, runTestSets)
	}
}
