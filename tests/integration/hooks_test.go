package integration

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eve/api/go/info"
	"testing"
	"time"
)

type EdenHookArgs map[string]interface{}
type EdenHookReturn interface{}
type EdenHookFunc func(t *testing.T, args EdenHookArgs) EdenHookReturn

type EdenHook struct {
	name   string
	hook   EdenHookFunc
	args   EdenHookArgs
	result EdenHookReturn
}

var Hooks EdenHooks

type EdenHooks map[string][]EdenHook

var lastRebootTime *timestamp.Timestamp
var lastRebootReason string
var rebooted bool

func setupTestCase(t *testing.T) func(t *testing.T) {
	t.Log("Setup test case", t.Name())
	// TODO -- replace to reading from config file
	Hooks = EdenHooks{
		"TestHooks": []EdenHook{
			EdenHook{
				name:   "CheckRebootInfo",
				hook:   CheckRebootInfo,
				args:   EdenHookArgs{},
				result: true,
			},
			EdenHook{
				name:   "wait",
				hook:   WaitHook,
				args:   EdenHookArgs{"secs": 1000},
				result: 1000,
			}, /*
				EdenHook{
					name: "false",
					hook: BoolHook,
					args: EdenHookArgs{"val":false},
					//result: false,
					result: true,
				},
				EdenHook{
					name: "true",
					hook: BoolHook,
					args: EdenHookArgs{"val":true},
					result: true,
				},*/
		},
	}
	return func(t *testing.T) {
		t.Log("Teardown test case", t.Name())
	}
}

func setupSubTest(t *testing.T) func(t *testing.T) {
	t.Log("Setup sub test", t.Name())
	return func(t *testing.T) {
		t.Log("Teardown sub test", t.Name())
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
		fmt.Printf("lastRebootTime: %s\n", lastRebootTime)
		fmt.Printf("lrbt: %s\n", lrbt)
		if proto.Equal(lastRebootTime, lrbt) {
			rebooted = false
		} else {
			lastRebootTime = lrbt
			lastRebootReason = lrbr
			rebooted = true
		}
	}
	fmt.Printf("rebooted: %v\n", rebooted)
	return rebooted
}

func CheckRebootInfo(t *testing.T, args EdenHookArgs) EdenHookReturn {
	fmt.Println("CheckRebootInfo")
	ctx, err := controller.CloudPrepare()
	if err != nil {
		t.Fatalf("CloudPrepare: %s", err)
	}
	devUUID, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal("Fail in get first device: ", err)
		return false
	}
	err = ctx.InfoChecker(devUUID.GetID(), map[string]string{"devId": devUUID.GetID().String(), "lastRebootTime": ".*"}, einfo.ZInfoDinfo, checkRebootTime, einfo.InfoAny, 300)
	if err != nil {
		t.Fatal("Fail in waiting for info: ", err)
		return false
	}
	t.Logf("Previous reboot at %s with reason '%s'\n", lastRebootTime, lastRebootReason)
	for {
		err = ctx.InfoChecker(devUUID.GetID(), map[string]string{"devId": devUUID.GetID().String(), "lastRebootTime": ".*"}, einfo.ZInfoDinfo, checkRebootTime, einfo.InfoNew, 3000)
		if err != nil {
			t.Fatal("Fail in waiting for info: ", err)
			return false
		}
		if rebooted {
			t.Logf("Rebooted again at %s with reason '%s'\n", lastRebootTime, lastRebootReason)
			return true
		} else {
			t.Logf("Not rebooted, lastRebootTime: %s\n", lastRebootTime)
			return false
		}
	}

}

func BoolHook(t *testing.T, args EdenHookArgs) EdenHookReturn {
	fmt.Println("BoolHook", args["val"].(bool))
	return args["val"].(bool)
}

func WaitHook(t *testing.T, args EdenHookArgs) EdenHookReturn {
	secs := args["secs"].(int)
	fmt.Println("WaitHook", secs, "sec.")
	time.Sleep(time.Duration(secs) * time.Second)
	fmt.Println("WaitHook finished")
	return secs
}

func runSubTest(t *testing.T, name string, hook EdenHookFunc, args EdenHookArgs, result EdenHookReturn, cntx context.Context, cancel context.CancelFunc) {
	for {
		select {
		case <-cntx.Done():
			return
		default:
			res := hook(t, args)
			if res != result {
				t.Errorf("%s got %v; want %v", name, res, result)
			}
			cancel()
			return
		}
	}
}

func TestHooks(t *testing.T) {
	teardownTestCase := setupTestCase(t)
	defer teardownTestCase(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // cancel when we are finished tasks
	for _, tc := range Hooks[t.Name()] {
		go runSubTest(t, tc.name, tc.hook, tc.args, tc.result, ctx, cancel)
	}
	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}
