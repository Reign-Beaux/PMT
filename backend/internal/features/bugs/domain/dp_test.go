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

func TestDPCannotReportBugs(t *testing.T) {
	dp := kernel.Actor{Name: "dp", Profile: kernel.ProfileDP}
	if EnsureCanCreate(dp) == nil {
		t.Error("DP no debe crear bugs")
	}
	if EnsureCanUpdate(dp) == nil {
		t.Error("DP no debe corregir bugs")
	}
}
