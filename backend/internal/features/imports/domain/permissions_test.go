package domain

import (
	"errors"
	"testing"

	"pmt/internal/kernel"
)

func TestEnsureCanImportOnlyPlanners(t *testing.T) {
	for _, p := range []kernel.Profile{kernel.ProfilePP, kernel.ProfileUser} {
		if err := EnsureCanImport(kernel.Actor{Name: "a", Profile: p}); err != nil {
			t.Errorf("%s: err = %v", p, err)
		}
	}
	for _, p := range []kernel.Profile{kernel.ProfilePC, kernel.ProfileTL, kernel.ProfileDP, kernel.ProfileQA} {
		if err := EnsureCanImport(kernel.Actor{Name: "a", Profile: p}); !errors.Is(err, ErrImportForbidden) {
			t.Errorf("%s: err = %v", p, err)
		}
	}
}
