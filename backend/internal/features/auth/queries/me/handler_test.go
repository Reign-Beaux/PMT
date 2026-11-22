package me

import (
	"context"
	"errors"
	"testing"

	"pmt/internal/features/auth/domain"
	"pmt/internal/kernel"
)

type finder map[string]domain.User

func (f finder) UserByUsername(_ context.Context, username string) (domain.User, error) {
	if u, ok := f[username]; ok {
		return u, nil
	}
	return domain.User{}, domain.ErrUserNotFound
}

func TestHandleDevuelveElPerfilEfectivo(t *testing.T) {
	h := New(finder{"saul": {Username: "saul"}})
	for _, profile := range []kernel.Profile{kernel.ProfileUser, kernel.ProfileTL, kernel.ProfilePC} {
		got, err := h.Handle(context.Background(), kernel.Actor{Name: "saul", Profile: profile})
		if err != nil || got.Username != "saul" || got.Profile != profile {
			t.Errorf("%s: %+v, %v", profile, got, err)
		}
	}
}

func TestHandleRechazaAgentes(t *testing.T) {
	h := New(finder{"saul": {Username: "saul"}})
	for _, actor := range []kernel.Actor{{Name: "dbp-tl", Profile: kernel.ProfileTL}, {Name: "agente", Profile: kernel.ProfileUser}, {}} {
		if _, err := h.Handle(context.Background(), actor); !errors.Is(err, domain.ErrUsersOnly) {
			t.Errorf("%+v: %v", actor, err)
		}
	}
}
