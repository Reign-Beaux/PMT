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

func TestDPIsNotALead(t *testing.T) {
	dp := kernel.Actor{Name: "dp", Profile: kernel.ProfileDP}
	if (Stage{Lead: "dp"}).EnsureLead(dp) == nil {
		t.Error("DP no actúa como TL aunque coincida el nombre")
	}
	if EnsureCanAssignLead(dp) == nil {
		t.Error("DP no asigna TL")
	}
}
