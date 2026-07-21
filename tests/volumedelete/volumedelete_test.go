// Copyright (c) 2026 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package volumedelete_test exercises volumemgr's failed-volume-delete retry
// loop (lf-edge/eve#6176) end to end on EVE-k, driving the file-existence fault
// injection compiled in under the eve `faultinjection` build tag
// (FAULT_INJECTION=y).
//
// Mechanism (learned by observing a live EVE-k sandbox):
//   - A VM app's boot disk is a CSI/Longhorn PVC volume. volumemgr only invokes
//     volumeHandlerCSI.DestroyVolume (the delete path the fix guards) once the
//     VolumeStatus reaches SubState=Created; before that a delete just
//     unpublishes. So the tests wait for SubState=Created before deleting.
//   - On Longhorn the app volume is often REPLICATED, and DestroyVolume SKIPS
//     kubeapi.DeletePVC for replicated volumes - so the DeletePVC fault marker
//     does not fire for them. The VolumeDestroy marker gates DestroyVolume
//     BEFORE the replicated skip, so it fails the destroy for any volume. These
//     tests therefore use the VolumeDestroy marker.
//   - The fix's observable signature (no controller/adam round-trip needed) is
//     the VolumeStatus staying published in SubState=Deleting with an Error set
//     and being re-driven off the gc tick, instead of being unpublished
//     immediately (the pre-fix leak). We read VolumeStatus straight off the
//     device (/run/volumemgr/VolumeStatus) via `eve exec`, which is lag-free.
//
// The app guest never has to boot (it can't at depth-2 nested KVM in the
// Multipass sandbox); only the volume lifecycle is exercised.
package volumedelete_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lf-edge/eden/pkg/defaults"
	tk "github.com/lf-edge/eden/pkg/evetestkit"
	"github.com/lf-edge/eden/pkg/openevec"
	"github.com/lf-edge/eden/pkg/utils"
	log "github.com/sirupsen/logrus"
)

const (
	projectName = "voldel-test"
	// hvTypeKube is the eve.hv value for an EVE-k image (see the package doc for
	// why a `make HV=k` build tags the image -k-).
	hvTypeKube = "k"

	// Fault marker paths (see eve pkg/pillar/kubeapi/faultinjection_volumedelete.go).
	// VolumeDestroy fires before the replicated-volume skip, so it works for
	// replicated Longhorn volumes too - unlike DeletePVC.
	volumeDestroyFaultPath = "/tmp/VolumeDestroy_FaultInjection_Fail"

	// VolumeStatus.SubState values (eve types.volumeSubState): Initial=0,
	// Preparing=1, PrepareDone=2, Created=3, Deleting=4.
	subStateCreated  = 3
	subStateDeleting = 4

	// With timer.gc.vdisk at its 60s minimum the gc tick fires every 6s
	// (vdiskGCTime/10); maxVolumeDeleteRetries is 12, so a give-up takes ~72s.
	fastVdiskGCTime = "60"

	createTimeout  = 40 * time.Minute // volume create is slow nested
	retrySeen      = 3 * time.Minute  // window to observe the retry signature
	giveUpTimeout  = 5 * time.Minute
	recoverTimeout = 4 * time.Minute
)

var (
	eveNode *tk.EveNode
	evec    *openevec.OpenEVEC
)

func TestMain(m *testing.M) {
	log.Println("Volume-delete retry test suite started")
	defer log.Println("Volume-delete retry test suite finished")

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	twoLevelsUp := filepath.Dir(filepath.Dir(currentPath))

	cfg, err := openevec.LoadConfig(utils.GetConfig("default"))
	if err != nil {
		log.Fatalf("Failed to get config %v\n", err)
	}
	if cfg.Eve.HV != hvTypeKube {
		log.Fatalf("Incorrect eve.hv value %q, this suite requires EVE-k (eve.hv=%q)",
			cfg.Eve.HV, hvTypeKube)
	}

	evec = openevec.CreateOpenEVEC(cfg)
	configDir := filepath.Join(twoLevelsUp, "eve-config-dir")
	if err := evec.SetupEden("config", configDir, "", "", "", []string{}, false, false); err != nil {
		log.Fatalf("Failed to setup Eden: %v", err)
	}
	if err := evec.StartEden(defaults.DefaultVBoxVMName, "", ""); err != nil {
		log.Fatalf("Start eden failed: %s", err)
	}
	if err := evec.OnboardEve(cfg.Eve.CertsUUID); err != nil {
		log.Fatalf("Eve onboard failed: %s", err)
	}
	node, err := tk.InitializeTestFromConfig(projectName, cfg, tk.WithControllerVerbosity("debug"))
	if err != nil {
		log.Fatalf("Failed to initialize test: %v", err)
	}
	eveNode = node

	// Speed up the delete-retry loop for the whole suite: retries are driven off
	// volumemgr's gc tick (= timer.gc.vdisk/10 s), and give-up takes
	// maxVolumeDeleteRetries(12) ticks. At the default 3600 that is ~72 min; at
	// the 60 s minimum it is ~72 s. Set it ONCE here (not per-test): volumemgr
	// only re-creates its gc ticker on a ZedAgentStatus config-get, so toggling
	// it per-test raced the ticker and left give-up ticking at the old (slow)
	// rate. Setting it once and never restoring lets the ticker settle to 6 s and
	// stay there for both tests.
	if err := node.UpdateNodeGlobalConfig(nil, map[string]string{"timer.gc.vdisk": fastVdiskGCTime}); err != nil {
		log.Fatalf("failed to set timer.gc.vdisk: %v", err)
	}
	os.Exit(m.Run())
}

// vdVol is the subset of VolumeStatus fields the tests inspect. EVE embeds
// ErrorDescription, so its Error field is promoted to the top level in JSON.
type vdVol struct {
	DisplayName  string `json:"DisplayName"`
	State        int    `json:"State"`
	SubState     int    `json:"SubState"`
	RefCount     int    `json:"RefCount"`
	FileLocation string `json:"FileLocation"`
	Error        string `json:"Error"`
}

// vdVolumeStatuses reads all VolumeStatus objects off the device (lag-free,
// unlike the controller-reported state). The bool is false if the read failed
// (transient), which callers distinguish from "no volumes".
func vdVolumeStatuses() ([]vdVol, bool) {
	out, err := eveNode.EveRunCommand(
		`eve exec pillar sh -c 'for f in /run/volumemgr/VolumeStatus/*.json; do [ -e "$f" ] && { cat "$f"; echo; }; done'`)
	if err != nil {
		return nil, false
	}
	var vols []vdVol
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var v vdVol
		if json.Unmarshal([]byte(line), &v) == nil {
			vols = append(vols, v)
		}
	}
	return vols, true
}

// vdAppVol returns the VolumeStatus for the given app (DisplayName prefix) and
// whether it is present.
func vdAppVol(app string) (vdVol, bool) {
	vols, ok := vdVolumeStatuses()
	if !ok {
		return vdVol{}, false
	}
	for _, v := range vols {
		if strings.HasPrefix(v.DisplayName, app) {
			return v, true
		}
	}
	return vdVol{}, false
}

func vdTouchMarker(t *testing.T, path string) {
	t.Helper()
	if _, err := eveNode.EveRunCommand("eve exec pillar touch " + path); err != nil {
		t.Fatalf("failed to create fault marker %s: %v", path, err)
	}
	t.Logf("fault marker created: %s", path)
}

func vdRemoveMarker(t *testing.T, path string) {
	t.Helper()
	if _, err := eveNode.EveRunCommand("eve exec pillar rm -f " + path); err != nil {
		t.Logf("failed to remove fault marker %s: %v", path, err)
	}
}

// vdWaitVolumeCreated deploys is done by the caller; this waits until the app's
// volume reaches SubState=Created (the state from which a delete drives
// DestroyVolume).
func vdWaitVolumeCreated(t *testing.T, app string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if v, ok := vdAppVol(app); ok && v.SubState >= subStateCreated {
			t.Logf("volume for %s reached SubState=%d (created)", app, v.SubState)
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("volume for %s did not reach SubState=Created within %s", app, timeout)
		}
		time.Sleep(15 * time.Second)
	}
}

// smallVMImage is a tiny (~20 MB) VM disk so the CDI import / volume create
// finishes quickly. A full Ubuntu cloud image (~700 MB) takes ~40+ min to reach
// SubState=Created nested, which is impractical. The guest need not boot - the
// test only exercises the volume lifecycle.
const smallVMImage = "https://download.cirros-cloud.net/0.6.2/cirros-0.6.2-x86_64-disk.img"

// vdDeployVM deploys a small VM app. Returns the app name.
func vdDeployVM(t *testing.T, suffix string) string {
	t.Helper()
	app := tk.GetRandomAppName(projectName + suffix)
	pc := tk.GetDefaultVMConfig(app, tk.AppDefaultCloudConfig, nil)
	if err := eveNode.EveDeployApp(smallVMImage, true, pc); err != nil {
		t.Fatalf("failed to deploy app: %v", err)
	}
	return app
}

// TestVolumeDeleteRetryRecovers: with the destroy failing, deleting the app must
// keep the VolumeStatus published in SubState=Deleting with an error (the #6176
// fix - the pre-fix code unpublishes immediately and leaks). Clearing the fault
// then lets a retry finish the delete and the VolumeStatus disappears.
func TestVolumeDeleteRetryRecovers(t *testing.T) {
	app := vdDeployVM(t, "-rec-")
	removed := false
	defer func() {
		if !removed {
			_ = eveNode.AppStopAndRemove(app)
		}
	}()
	vdWaitVolumeCreated(t, app, createTimeout)

	vdTouchMarker(t, volumeDestroyFaultPath)
	faultCleared := false
	defer func() {
		if !faultCleared {
			vdRemoveMarker(t, volumeDestroyFaultPath)
		}
	}()

	if err := eveNode.AppStopAndRemove(app); err != nil {
		t.Fatalf("failed to remove app %s: %v", app, err)
	}
	removed = true

	// Fix signature: the volume is kept in SubState=Deleting WITH an error and
	// re-driven, rather than vanishing. Observe it persist.
	deadline := time.Now().Add(retrySeen)
	sawRetry := false
	for time.Now().Before(deadline) {
		if v, ok := vdAppVol(app); ok && v.SubState == subStateDeleting && v.Error != "" {
			sawRetry = true
			t.Logf("observed retained failed-delete: SubState=Deleting, error=%q", v.Error)
			break
		}
		time.Sleep(5 * time.Second)
	}
	if !sawRetry {
		t.Fatalf("volume for %s was not kept in SubState=Deleting with an error after a "+
			"failed delete (pre-fix leak behavior?)", app)
	}

	// Clear the fault; a retry must now finish the delete and unpublish the volume.
	vdRemoveMarker(t, volumeDestroyFaultPath)
	faultCleared = true

	deadline = time.Now().Add(recoverTimeout)
	for {
		if _, ok := vdAppVol(app); !ok {
			t.Logf("volume for %s unpublished after fault cleared; retry recovered the delete", app)
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("volume for %s still published %s after clearing the fault", app, recoverTimeout)
		}
		time.Sleep(5 * time.Second)
	}
}

// TestVolumeDeleteGivesUp: with the destroy failing permanently, volumemgr must
// retry a bounded number of times and then give up (unpublish), rather than
// resubmitting forever. Observed as the VolumeStatus persisting in
// SubState=Deleting with an error and then disappearing while the fault is still
// armed.
func TestVolumeDeleteGivesUp(t *testing.T) {
	app := vdDeployVM(t, "-giveup-")
	removed := false
	defer func() {
		if !removed {
			_ = eveNode.AppStopAndRemove(app)
		}
	}()
	vdWaitVolumeCreated(t, app, createTimeout)

	vdTouchMarker(t, volumeDestroyFaultPath)
	defer vdRemoveMarker(t, volumeDestroyFaultPath)

	if err := eveNode.AppStopAndRemove(app); err != nil {
		t.Fatalf("failed to remove app %s: %v", app, err)
	}
	removed = true

	// Must first be retained for retry (SubState=Deleting + error)...
	deadline := time.Now().Add(retrySeen)
	sawRetry := false
	for time.Now().Before(deadline) {
		if v, ok := vdAppVol(app); ok && v.SubState == subStateDeleting && v.Error != "" {
			sawRetry = true
			break
		}
		time.Sleep(5 * time.Second)
	}
	if !sawRetry {
		t.Fatalf("volume for %s was not retained for retry after a failed delete", app)
	}

	// ...then, with the fault still armed, give up after the retry budget and
	// unpublish (leaving the underlying volume orphaned - the terminal fallback).
	deadline = time.Now().Add(giveUpTimeout)
	for {
		if _, ok := vdAppVol(app); !ok {
			t.Logf("volume for %s unpublished after the retry budget: give-up path", app)
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("volume for %s never gave up within %s (retrying forever?)", app, giveUpTimeout)
		}
		time.Sleep(5 * time.Second)
	}
}
