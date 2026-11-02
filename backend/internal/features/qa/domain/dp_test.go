package domain

import (
	"testing"

	"pmt/internal/kernel"
)

func TestDPCannotManageTestPlans(t *testing.T) {
	dp := kernel.Actor{Name: "dp", Profile: kernel.ProfileDP}
	if EnsureQAP(dp) == nil {
		t.Error("DP no debe gestionar planes QAP")
	}
	if EnsureTL(dp) == nil {
		t.Error("DP no debe decidir planes QAP")
	}
	if (Plan{Status: StatusSubmitted}).EnsureDecidable(dp) == nil {
		t.Error("DP no debe aprobar ni devolver planes QAP")
	}
}
