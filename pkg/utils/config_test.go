// Copyright (c) 2026 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"testing"

	"github.com/lf-edge/eden/pkg/utils"
)

// TestGenerateRandomSerialFormat asserts the returned serial is 10 numeric
// digits — the SMBIOS-friendly format eden writes into qemu and adam.
func TestGenerateRandomSerialFormat(t *testing.T) {
	s, err := utils.GenerateRandomSerial()
	if err != nil {
		t.Fatalf("GenerateRandomSerial: %v", err)
	}
	if len(s) != 10 {
		t.Fatalf("GenerateRandomSerial length=%d, want 10: %q", len(s), s)
	}
	for i, c := range s {
		if c < '0' || c > '9' {
			t.Fatalf("GenerateRandomSerial[%d]=%q non-digit in %q", i, c, s)
		}
	}
}

// TestGenerateRandomSerialUniqueness checks that 1 000 draws don't collide.
// The serial space is 10^10 so the expected collision probability across
// 1 000 draws (birthday bound) is ~5e-5; a single collision here would
// almost certainly indicate a broken RNG path, not bad luck.
func TestGenerateRandomSerialUniqueness(t *testing.T) {
	const draws = 1000
	seen := make(map[string]struct{}, draws)
	for i := 0; i < draws; i++ {
		s, err := utils.GenerateRandomSerial()
		if err != nil {
			t.Fatalf("GenerateRandomSerial[%d]: %v", i, err)
		}
		if _, dup := seen[s]; dup {
			t.Fatalf("collision after %d draws: %q", i, s)
		}
		seen[s] = struct{}{}
	}
}
