package domain

import (
	"testing"

	"pmt/internal/kernel"
)

func TestDPCannotManageFindings(t *testing.T) {
	dp := kernel.Actor{Name: "dp", Profile: kernel.ProfileDP}
	if EnsureCanCreate(dp) == nil {
		t.Error("DP no debe crear hallazgos")
	}
	open := Finding{Status: StatusOpen}
	for _, resolution := range []Status{StatusFixed, StatusDismissed} {
		if open.EnsureClosable(resolution, dp) == nil {
			t.Errorf("DP no debe cerrar hallazgos como %s", resolution)
		}
	}
}
