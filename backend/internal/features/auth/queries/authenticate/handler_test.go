package authenticate

import (
	"context"
	"errors"
	"testing"
	"time"

	"pmt/internal/features/auth/authtest"
	"pmt/internal/features/auth/domain"
	"pmt/internal/kernel"
)

func TestTokenSinPrefijoNoEsDeUsuario(t *testing.T) {
	_, err := New(authtest.New()).Token(context.Background(), "abc123")
	if !errors.Is(err, kernel.ErrUnauthenticated) {
		t.Errorf("error = %v", err)
	}
}

func TestSesionVaciaOInexistente(t *testing.T) {
	h := New(authtest.New())
	for _, s := range []string{"", "no-existe"} {
		if _, err := h.Session(context.Background(), s); !errors.Is(err, kernel.ErrSessionInvalid) {
			t.Errorf("%q: %v", s, err)
		}
	}
}

func TestSesionYTokenActuanComoUsuario(t *testing.T) {
	ctx := context.Background()
	repo := authtest.New()
	u := repo.AddUser("ana@example.com", "ana", false, "contrasena")
	repo.CreateSession(ctx, domain.HashToken("sesion"), u.ID, time.Now().Add(domain.SessionTTL))
	token, _ := domain.NewAPIToken()
	repo.CreateAPIToken(ctx, u.ID, u.ID, "x", domain.HashToken(token))
	want := kernel.Actor{Name: "ana", Profile: kernel.ProfileUser, UserID: u.ID}

	if got, err := New(repo).Session(ctx, "sesion"); err != nil || got != want {
		t.Errorf("sesión: %+v %v", got, err)
	}
	if got, err := New(repo).Token(ctx, token); err != nil || got != want {
		t.Errorf("token: %+v %v", got, err)
	}
	repo.DisableUser(ctx, u.ID)
	if _, err := New(repo).Token(ctx, token); !errors.Is(err, kernel.ErrUnauthenticated) {
		t.Errorf("token de desactivado: %v", err)
	}
}
