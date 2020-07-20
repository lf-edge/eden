package integration

import (
	"fmt"
	"github.com/lf-edge/eden/pkg/controller"
	"github.com/lf-edge/eden/pkg/controller/einfo"
	"github.com/lf-edge/eden/pkg/controller/elog"
	"github.com/lf-edge/eden/pkg/defaults"
	"testing"
	"time"
)

//TestNetworkInstance test network instances creation in EVE
func TestNetworkInstance(t *testing.T) {
	ctx, err := controller.CloudPrepare()
	if err != nil {
		t.Fatalf("CloudPrepare: %s", err)
	}

	deviceCtx, err := ctx.GetDeviceFirst()
	if err != nil {
		t.Fatal("Fail in get first device: ", err)
	}
	devModel, err := ctx.GetDevModelByName(defaults.DefaultEVEModel)
	if err != nil {
		t.Fatal("Fail in get dev model: ", err)
	}

	var networkInstances []string
	var networkInstanceTests = []struct {
		networkInstance *netInst
	}{
		{networkInstanceLocal},
		{networkInstanceSwitch},
		{networkInstanceCloud},
	}
	for _, tt := range networkInstanceTests {
		t.Run(tt.networkInstance.networkInstanceName, func(t *testing.T) {
			err = prepareNetworkInstance(ctx, tt.networkInstance, devModel)
			if err != nil {
				t.Fatal("Fail in prepare network instance: ", err)
			}

			devUUID := deviceCtx.GetID()
			//append networkInstance for run all of them together
			networkInstances = append(networkInstances, tt.networkInstance.networkInstanceID)
			deviceCtx.SetNetworkInstanceConfig(networkInstances)
			err = ctx.ConfigSync(deviceCtx)
			if err != nil {
				t.Fatal("Fail in sync config with controller: ", err)
			}
			t.Run("Process", func(t *testing.T) {
				err = ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "networkID": tt.networkInstance.networkInstanceID}, einfo.HandleFirst, einfo.InfoAny, 300)
				if err != nil {
					t.Fatal("Fail in waiting for process start from info: ", err)
				}
			})
			t.Run("Handled", func(t *testing.T) {
				if !checkLogs {
					t.Skip("no LOGS flag set - skipped")
				}
				err = ctx.LogChecker(devUUID, map[string]string{"devId": devUUID.String(), "msg": fmt.Sprintf(".*handleNetworkInstanceModify\\(%s\\) done.*", tt.networkInstance.networkInstanceID), "level": "info"}, elog.HandleFirst, elog.LogAny, 600)
				if err != nil {
					t.Fatal("Fail in waiting for handleNetworkInstanceModify done from zedagent: ", err)
				}
			})
			timeout := time.Duration(200)

			if !checkLogs {
				timeout = 800
			}
			t.Run("Active", func(t *testing.T) {
				err = ctx.InfoChecker(devUUID, map[string]string{"devId": devUUID.String(), "networkID": tt.networkInstance.networkInstanceID, "activated": "true"}, einfo.HandleFirst, einfo.InfoAny, timeout)
				if err != nil {
					t.Fatal("Fail in waiting for activated state from info: ", err)
				}
			})
		})
	}
}
