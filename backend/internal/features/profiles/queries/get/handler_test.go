package get

import (
	"errors"
	"testing"

	"pmt/internal/features/profiles/catalog"
	"pmt/internal/kernel"
)

func TestHandleActor(t *testing.T) {
	h := New()
	for _, actor := range []kernel.Actor{
		{Name: "ana", Profile: kernel.ProfileTL},
		{Name: "ana", Profile: kernel.ProfilePC},
		{Name: "dbp-tl", Profile: kernel.ProfileTL, Agent: true},
	} {
		want, _ := catalog.Served(actor.Profile)
		got, err := h.HandleActor(actor)
		if err != nil || got != want || got == "" {
			t.Errorf("%+v: %.60q, %v", actor, got, err)
		}
	}
	for _, profile := range []kernel.Profile{kernel.ProfileUser, ""} {
		if _, err := h.HandleActor(kernel.Actor{Name: "ana", Profile: profile}); !errors.Is(err, ErrNoProfile) {
			t.Errorf("%q: err = %v", profile, err)
		}
	}
}

func TestHandleIgualQueHandleActor(t *testing.T) {
	h := New()
	byName, err := h.Handle("qa")
	if err != nil {
		t.Fatal(err)
	}
	if byActor, _ := h.HandleActor(kernel.Actor{Profile: kernel.ProfileQA}); byActor != byName {
		t.Error("REST y get_profile deben servir el mismo texto")
	}
}
