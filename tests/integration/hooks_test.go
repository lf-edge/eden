package integration

import (
	"context"
	"fmt"
	"testing"
	"time"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/lf-edge/eve/api/go/info"
	"github.com/lf-edge/eden/pkg/controller/einfo"
)

type EdenHookArgs map[string]interface{}
type EdenHookReturn interface{}
type EdenHookFunc func(t *testing.T, args EdenHookArgs) EdenHookReturn

type EdenHook struct {
	name string
	hook EdenHookFunc
	args EdenHookArgs
	result EdenHookReturn
}

var Hooks EdenHooks

type EdenHooks map[string][]EdenHook

var hooks EdenHooks = EdenHooks{
	"TestHooks":[]EdenHook{
		EdenHook{
			name: "CheckRebootInfo",
			hook: CheckRebootInfo,
			args: EdenHookArgs{},
			result: true,
		},
		EdenHook{
			name: "wait",
			hook: WaitHook,
			args: EdenHookArgs{"secs":10000},
			result: true,
		},/*
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

func CheckRebootInfo(t *testing.T, args EdenHookArgs) EdenHookReturn {
	fmt.Println("CheckRebootInfo")
	ctx, err := controllerPrepare()
	if err != nil {
		t.Fatal("Fail in controller prepare: ", err)
		return false
	}
	devUUID, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal("Fail in get first device: ", err)
		return false
	}
	err = ctx.InfoChecker(devUUID.GetID(), map[string]string{"devId": devUUID.GetID().String(), "lastRebootTime":".*"}, einfo.ZInfoDinfo, checkRebootTime, einfo.InfoAny, 300)
	if err != nil {
		t.Fatal("Fail in waiting for info: ", err)
		return false
	}
	t.Logf("1. Rebooted at %s with reason '%s'\n", lastRebootTime, lastRebootReason)
	for {
		err = ctx.InfoChecker(devUUID.GetID(), map[string]string{"devId": devUUID.GetID().String(), "lastRebootTime":".*"}, einfo.ZInfoDinfo, checkRebootTime, einfo.InfoNew, 3000)
		if err != nil {
			t.Fatal("Fail in waiting for info: ", err)
			return false
		}
		if rebooted {
			t.Logf("2. Rebooted at %s with reason '%s'\n", lastRebootTime, lastRebootReason)
			return true
		} else {
			t.Logf("Not rebooted: %s\n", lastRebootTime)
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
	fmt.Println("WaitHook", secs)
	time.Sleep(time.Duration(secs)*time.Second)
	fmt.Println("WaitHook finished")
	return secs
}

func ReadHook(t *testing.T, args EdenHookArgs) EdenHookReturn {
	var inp string
	prompt := args["prompt"].(string)
	fmt.Println(prompt)
	fmt.Scan(&inp)
	fmt.Println("Printed:", inp)
	return inp
}

func setupTestCase(t *testing.T) func(t *testing.T) {
	t.Log("Setup test case", t.Name())
	Hooks = hooks
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

func runSubTest(t *testing.T, name string, hook EdenHookFunc, args EdenHookArgs, result EdenHookReturn, cntx context.Context, cancel context.CancelFunc) {
	var res EdenHookReturn
	fmt.Println("L1", name)
	for {
		fmt.Println("L3", name)
		select {
		case <-cntx.Done():
			fmt.Println("L4", name)
			res = false
			fmt.Println("L5", name)
			return
		default:
			fmt.Println("L6", name)
			res = hook(t, args)
			fmt.Println("L7", name)
			if res != result {
					t.Errorf("%s got %v; want %v", name, res, result)
			}
			cancel()
			fmt.Println("L8", name)
			return
		}
	}
	fmt.Println("L7:", res)
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
