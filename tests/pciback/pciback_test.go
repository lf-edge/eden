// Package pciback verifies that domainmgr keeps a network port in the host
// (does not reserve it to pciback/vfio-pci) when the controller's device model
// disagrees with what the kernel presents for that port's PCI device. It covers
// a port whose model interface name differs from the kernel-assigned name, and
// (as a behavior-documenting baseline) a non-network device declared at the same
// PCI address as an in-use network port. The port must stay recognized by its
// PCI address and kept in the host so the device stays reachable.
package pciback

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	tk "github.com/lf-edge/eden/pkg/evetestkit"
)

var eveNode *tk.EveNode

const (
	projectName = "pciback"
	// rebootWait is how long to wait for EVE to come back after a reboot.
	rebootWait = uint(10 * 60)
	// settleWait bounds how long we wait for domainmgr to reach a steady state
	// after a device-model change.
	settleWait   = 3 * time.Minute
	pollInterval = 10 * time.Second
	// victimPort is the NIC we make the model disagree about; the other NIC
	// (eth0) is left untouched so EVE stays reachable throughout the test.
	victimPort = "eth1"
)

func TestMain(m *testing.M) {
	node, err := tk.InitializeTest(projectName, tk.WithControllerVerbosity("debug"))
	if err != nil {
		fmt.Printf("Failed to initialize test: %v\n", err)
		os.Exit(1)
	}
	eveNode = node
	// Shorten the config poll so the reboots this suite triggers are picked up
	// quickly instead of on the default ~1 minute cadence.
	_ = eveNode.UpdateNodeGlobalConfig(nil, map[string]string{"timer.config.interval": "10"})
	os.Exit(m.Run())
}

// ioBundle is the subset of types.IoBundle this test asserts on. IsPCIBack is a
// pointer because the status serializes it as null (not false) when unset.
type ioBundle struct {
	Phylabel   string `json:"Phylabel"`
	PciLong    string `json:"PciLong"`
	KeepInHost bool   `json:"KeepInHost"`
	IsPCIBack  *bool  `json:"IsPCIBack"`
}

// readAssignableAdapters reads domainmgr's AssignableAdapters status off EVE.
// It returns an error (rather than failing the test) because the status file may
// be briefly absent right after a reboot and ssh can hiccup - callers retry.
// The "|| true" keeps the remote exit code 0 when the file is not yet present.
func readAssignableAdapters() ([]ioBundle, error) {
	out, err := eveNode.EveRunCommand("eve exec pillar sh -c " +
		"'cat /run/domainmgr/AssignableAdapters/global.json 2>/dev/null || true'")
	if err != nil {
		return nil, err
	}
	// Skip any leading noise before the JSON object.
	for i := 0; i < len(out); i++ {
		if out[i] == '{' {
			out = out[i:]
			break
		}
	}
	var aa struct {
		IoBundleList []ioBundle `json:"IoBundleList"`
	}
	if err := json.Unmarshal(out, &aa); err != nil {
		return nil, fmt.Errorf("AssignableAdapters not readable yet: %w", err)
	}
	return aa.IoBundleList, nil
}

func lookupBundle(list []ioBundle, phylabel string) *ioBundle {
	for i := range list {
		if list[i].Phylabel == phylabel {
			return &list[i]
		}
	}
	return nil
}

// portPci returns the PCI address EVE resolved for the given port, retrying
// until the status is available.
func portPci(t *testing.T, phylabel string) string {
	t.Helper()
	deadline := time.Now().Add(settleWait)
	for time.Now().Before(deadline) {
		if list, err := readAssignableAdapters(); err == nil {
			if b := lookupBundle(list, phylabel); b != nil && b.PciLong != "" {
				return b.PciLong
			}
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("no IoBundle with a PCI address for %s within %s", phylabel, settleWait)
	return ""
}

// applyDeviceModel fetches the controller config, lets fn edit the parsed JSON,
// pushes it back and reboots (deviceIoList changes take effect on reboot). It
// registers a cleanup that restores the original config.
func applyDeviceModel(t *testing.T, fn func(cfg map[string]any)) {
	t.Helper()
	const cfgFile = "/tmp/pciback-device-config.json"
	if err := eveNode.GetConfig(cfgFile); err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	orig, err := os.ReadFile(cfgFile)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	t.Cleanup(func() {
		if err := os.WriteFile(cfgFile, orig, 0o600); err != nil {
			return
		}
		_ = eveNode.SetConfig(cfgFile)
		_ = eveNode.EveRebootAndWait(rebootWait)
	})
	var cfg map[string]any
	if err := json.Unmarshal(orig, &cfg); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if _, ok := cfg["deviceIoList"].([]any); !ok {
		t.Fatalf("deviceIoList missing from controller config")
	}
	fn(cfg)
	edited, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(cfgFile, edited, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := eveNode.SetConfig(cfgFile); err != nil {
		t.Fatalf("SetConfig: %v", err)
	}
	if err := eveNode.EveRebootAndWait(rebootWait); err != nil {
		t.Fatalf("reboot after config change: %v", err)
	}
}

// assertKeptInHost fails unless every named adapter is kept in the host and not
// reserved to pciback, held steadily (the bug's reserve/release churn never
// settles into a kept state). Reading the status also requires EVE to be
// reachable, which is the ultimate thing the fix protects.
func assertKeptInHost(t *testing.T, phylabels ...string) {
	t.Helper()
	deadline := time.Now().Add(settleWait)
	stable := 0
	var last []ioBundle
	var lastErr error
	for time.Now().Before(deadline) {
		list, err := readAssignableAdapters()
		if err != nil {
			// Status not published yet or a transient ssh failure; keep polling.
			lastErr, stable = err, 0
			time.Sleep(pollInterval)
			continue
		}
		last, lastErr = list, nil
		ok := true
		for _, pl := range phylabels {
			b := lookupBundle(list, pl)
			if b == nil || !b.KeepInHost || (b.IsPCIBack != nil && *b.IsPCIBack) {
				ok = false
			}
		}
		if ok {
			if stable++; stable >= 3 {
				return
			}
		} else {
			stable = 0
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("adapters %v not kept in host / reserved to pciback within %s (lastErr=%v): %+v",
		phylabels, settleWait, lastErr, last)
}

// TestPortKeptOnIfnameMismatch: the model's interface name for a port differs
// from the kernel-assigned name (e.g. model enpNsN vs kernel ethN). domainmgr
// must match the port to its device by PCI address and keep it in the host.
func TestPortKeptOnIfnameMismatch(t *testing.T) {
	pci := portPci(t, victimPort)
	t.Logf("%s is backed by %s; giving it a mismatched model interface name", victimPort, pci)

	applyDeviceModel(t, func(cfg map[string]any) {
		found := false
		for _, e := range cfg["deviceIoList"].([]any) {
			m, ok := e.(map[string]any)
			if ok && m["logicallabel"] == victimPort {
				m["phyaddrs"] = map[string]any{"Ifname": "enpMock" + victimPort, "PciLong": pci}
				found = true
			}
		}
		if !found {
			t.Fatalf("port %s not found in deviceIoList", victimPort)
		}
	})

	assertKeptInHost(t, victimPort)
}

// TestPortKeptOnPhantomAdapterSamePci documents behavior for a device model that
// declares a non-network device (audio) at the same PCI address as a NIC that is
// in use as a network port. domainmgr must not reserve that PCI to pciback for
// the phantom device, or it would unbind the live port.
//
// This holds today regardless of the PCI-identity fix: EVE groups devices that
// share a PCI controller and keeps the whole group in the host. The test
// therefore documents existing behavior rather than gating the fix; it is a
// starting point to extend once EVE reports such device-model inconsistencies
// back to the controller (the phantom adapter's IoBundle error).
func TestPortKeptOnPhantomAdapterSamePci(t *testing.T) {
	pci := portPci(t, victimPort)
	t.Logf("declaring a phantom audio adapter at %s (same PCI as %s)", pci, victimPort)

	applyDeviceModel(t, func(cfg map[string]any) {
		phantom := map[string]any{
			"ptype":        4, // PhyIoAudio (non-network)
			"phylabel":     "MockAudio",
			"logicallabel": "mockaudio",
			"assigngrp":    "mockaudio",
			"usage":        0, // PhyIoUsageNone
			"phyaddrs":     map[string]any{"PciLong": pci},
		}
		cfg["deviceIoList"] = append(cfg["deviceIoList"].([]any), phantom)
	})

	// Neither the real port nor the phantom sharing its PCI may be reserved.
	assertKeptInHost(t, victimPort, "MockAudio")
}
