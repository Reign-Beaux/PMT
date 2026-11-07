package kernel

import "testing"

func TestIsPlanner(t *testing.T) {
	for _, p := range []Profile{ProfilePP, ProfileUser} {
		if !p.IsPlanner() {
			t.Errorf("%s debe planear", p)
		}
	}
	for _, p := range []Profile{ProfilePC, ProfileTL, ProfileDevFrontend, ProfileDevBackend, ProfileCR, ProfileQAP, ProfileQA, ProfileDP, ""} {
		if p.IsPlanner() {
			t.Errorf("%q no debe planear", p)
		}
	}
}

func TestHasAnyTransition(t *testing.T) {
	table := map[string]map[string][]Profile{
		"a": {"b": {ProfileTL}},
		"b": {"c": {ProfileQA}, "a": {ProfilePP}},
		"c": {},
	}
	for _, p := range []Profile{ProfileTL, ProfileQA, ProfilePP} {
		if !HasAnyTransition(table, p) {
			t.Errorf("%s debe tener transición", p)
		}
	}
	for _, p := range []Profile{ProfilePC, ProfileDP, ""} {
		if HasAnyTransition(table, p) {
			t.Errorf("%q no debe tener transición", p)
		}
	}
}
