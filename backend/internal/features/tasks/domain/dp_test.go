package domain

import (
	"slices"
	"testing"

	"pmt/internal/kernel"
)

func TestDPHasNoTransitions(t *testing.T) {
	for from, targets := range statusProfiles {
		for to, profiles := range targets {
			if slices.Contains(profiles, kernel.ProfileDP) {
				t.Errorf("DP no debe poder %s → %s", from, to)
			}
		}
	}
}
