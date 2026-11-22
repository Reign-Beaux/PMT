package kernel

import (
	"errors"
	"testing"
)

func TestParseProfile(t *testing.T) {
	for value, want := range map[string]Profile{
		"":            "",
		"  ":          "",
		"user":        "",
		"User":        "",
		"pp":          ProfilePP,
		"PC":          ProfilePC,
		" tl ":        ProfileTL,
		"dp":          ProfileDP,
		"DevFrontend": ProfileDevFrontend,
		"devbackend":  ProfileDevBackend,
		"cr":          ProfileCR,
		"QAP":         ProfileQAP,
		"qa":          ProfileQA,
	} {
		got, err := ParseProfile(value)
		if err != nil || got != want {
			t.Errorf("ParseProfile(%q) = %q, %v; quiero %q", value, got, err, want)
		}
	}
	for _, value := range []string{"xx", "dev", "owner", "t l", "pp,tl"} {
		if _, err := ParseProfile(value); !errors.Is(err, ErrProfileUnknown) {
			t.Errorf("ParseProfile(%q) = %v; quiero ErrProfileUnknown", value, err)
		}
	}
	var business *Error
	if !errors.As(ErrProfileUnknown, &business) || business.Kind != KindInvalid {
		t.Error("un perfil desconocido debe ser KindInvalid (400)")
	}
}

func TestAsUser(t *testing.T) {
	if got := AsUser("saul", ""); got != (Actor{Name: "saul", Profile: ProfileUser}) {
		t.Errorf("sin perfil = %+v", got)
	}
	for _, p := range []Profile{ProfilePP, ProfilePC, ProfileTL, ProfileDP, ProfileDevFrontend, ProfileDevBackend, ProfileCR, ProfileQAP, ProfileQA} {
		if got := AsUser("saul", p); got != (Actor{Name: "saul", Profile: p}) {
			t.Errorf("con %s = %+v", p, got)
		}
	}
}

func TestAsAgent(t *testing.T) {
	tl := Actor{Name: "dbp-tl", Profile: ProfileTL}
	for _, declared := range []Profile{"", ProfileTL} {
		got, err := AsAgent(tl, declared)
		if err != nil || got != tl {
			t.Errorf("declara %q = %+v, %v", declared, got, err)
		}
	}
	got, err := AsAgent(tl, ProfilePP)
	var business *Error
	if !errors.As(err, &business) || business.Kind != KindInvalid || got != (Actor{}) {
		t.Errorf("otro perfil = %+v, %v; quiero KindInvalid", got, err)
	}
}
