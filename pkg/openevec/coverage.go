// Copyright (c) 2026 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

package openevec

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/lf-edge/eden/pkg/defaults"
	log "github.com/sirupsen/logrus"
)

// waitForEveSSH polls until EVE accepts an SSH connection or the timeout
// expires.  Tests that reboot EVE leave SSH temporarily unreachable; retrying
// here avoids a spurious coverage-collection failure.
func (openEVEC *OpenEVEC) waitForEveSSH(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	interval := 10 * time.Second
	for {
		if err := openEVEC.sshEveProbe(); err == nil {
			return nil
		}
		if time.Now().Add(interval).After(deadline) {
			return fmt.Errorf("EVE SSH not available after %v", timeout)
		}
		log.Infof("EVE coverage: waiting for SSH to become available...")
		time.Sleep(interval)
	}
}

// waitForCoverageFile polls GOCOVERDIR on EVE until the covcounters file
// count exceeds the value recorded in countRef, or timeout expires.
// A higher count means the SIGUSR2-triggered write completed.
//
// We compare counts rather than timestamps because busybox find -newer uses
// 1-second granularity; the new covcounters file typically lands within
// milliseconds of the reference point and would be missed by a mtime check.
func (openEVEC *OpenEVEC) waitForCoverageFile(coverDir, countRef string, timeout time.Duration) error {
	const pollInterval = 2 * time.Second
	deadline := time.Now().Add(timeout)
	pollCmd := fmt.Sprintf(
		`before=$(cat %s 2>/dev/null || echo 0); `+
			`now=$(ls %s/covcounters.* 2>/dev/null | wc -l); `+
			`[ "$now" -gt "$before" ]`,
		countRef, coverDir)
	for {
		if err := openEVEC.SdnForwardSSHToEve(pollCmd); err == nil {
			return nil
		}
		if time.Until(deadline) <= pollInterval {
			return fmt.Errorf("no new coverage file in %s after %v", coverDir, timeout)
		}
		log.Infof("EVE coverage: waiting for coverage files to appear...")
		time.Sleep(pollInterval)
	}
}

// CollectEveCoverage dumps coverage counters from coverage-instrumented EVE
// binaries (built with COVER=y / go build -cover -covermode=atomic) and
// converts the result to a Go coverage text profile at
// <outputDir>/eden_e2e_coverage.txt.
//
// The function:
//  1. Waits for EVE SSH to be ready (EVE may be rebooting after tests).
//  2. Sends SIGUSR2 to all running zedbox processes so they write their
//     in-memory coverage counters to GOCOVERDIR without terminating.
//  3. Copies the binary coverage files to a local temporary directory.
//  4. Converts them to text profile format with "go tool covdata textfmt".
//
// Requirements:
//   - EVE must have been built with COVER=y so that zedbox is instrumented
//     and its init() sets GOCOVERDIR=/persist/coverage and registers a
//     SIGUSR2 handler that calls runtime/coverage.WriteCountersDir.
//   - SSH access to EVE must be configured (eden.ssh-key must exist and
//     debug.enable.ssh must be set in the EVE config item).
func (openEVEC *OpenEVEC) CollectEveCoverage(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("cannot create coverage output dir %s: %w", outputDir, err)
	}

	// Wait for EVE SSH to be ready.  A test may have rebooted EVE and SSH
	// might not be accepting connections immediately after tests complete.
	log.Infof("EVE coverage: waiting for EVE SSH (up to 5 min)...")
	if err := openEVEC.waitForEveSSH(5 * time.Minute); err != nil {
		return fmt.Errorf("cannot reach EVE via SSH: %w", err)
	}

	// Record the count of existing covcounters files before signalling.
	// We compare counts (not mtimes) because busybox find -newer has 1-second
	// timestamp granularity and would miss files written in the same second.
	const coverCountRef = "/tmp/eve_cov_count"
	countCmd := fmt.Sprintf(
		"ls %s/covcounters.* 2>/dev/null | wc -l > %s",
		defaults.DefaultEveCoverageDir, coverCountRef)
	tsErr := openEVEC.SdnForwardSSHToEve(countCmd)

	// Send SIGUSR2 to all zedbox processes to dump live coverage counters.
	// The SIGUSR2 handler (registered in zedbox when built with -cover) calls
	// runtime/coverage.WriteCountersDir(GOCOVERDIR) without exiting.
	log.Infof("EVE coverage: sending SIGUSR2 to zedbox processes")
	sigCmd := "kill -USR2 $(pgrep -x zedbox) 2>/dev/null; true"
	if err := openEVEC.SdnForwardSSHToEve(sigCmd); err != nil {
		log.Warnf("EVE coverage: SIGUSR2 delivery failed (%v); "+
			"coverage may be incomplete", err)
	}

	// Wait for a new coverage file to appear rather than sleeping a fixed duration.
	log.Infof("EVE coverage: waiting for coverage data to be written")
	if tsErr != nil {
		// Count recording failed — fall back to the fixed sleep.
		log.Warnf("EVE coverage: could not record coverage file count (%v); "+
			"using fixed sleep", tsErr)
		time.Sleep(3 * time.Second)
	} else if err := openEVEC.waitForCoverageFile(
		defaults.DefaultEveCoverageDir, coverCountRef, 30*time.Second); err != nil {
		log.Warnf("EVE coverage: %v; proceeding with whatever was written", err)
	}

	// Copy binary coverage files from EVE to a local temp directory.
	tmpDir, err := os.MkdirTemp("", "eve-coverage-*")
	if err != nil {
		return fmt.Errorf("cannot create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	log.Infof("EVE coverage: copying %s from EVE to %s",
		defaults.DefaultEveCoverageDir, tmpDir)
	if err := openEVEC.SdnForwardSCPDirFromEve(
		defaults.DefaultEveCoverageDir, tmpDir); err != nil {
		return fmt.Errorf("cannot copy coverage data from EVE: %w", err)
	}

	// The SCP copies the directory itself, so the files land under
	// tmpDir/coverage/.  Point covdata at that sub-directory.
	coverSubDir := filepath.Join(tmpDir, filepath.Base(defaults.DefaultEveCoverageDir))
	if _, err := os.Stat(coverSubDir); os.IsNotExist(err) {
		// Fallback: files landed directly in tmpDir.
		coverSubDir = tmpDir
	}

	// Convert binary coverage format → Go text profile format.
	outputFile := filepath.Join(outputDir, "eden_e2e_coverage.txt")
	log.Infof("EVE coverage: converting to text profile %s", outputFile)
	cmd := exec.Command("go", "tool", "covdata", "textfmt",
		"-i="+coverSubDir,
		"-o="+outputFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go tool covdata textfmt failed: %w\nOutput: %s",
			err, string(out))
	}

	log.Infof("EVE coverage: written to %s", outputFile)
	return nil
}
