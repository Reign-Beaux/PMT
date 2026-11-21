package seedowner

import (
	"context"
	"errors"
	"strings"
	"testing"

	"pmt/internal/features/auth/authtest"
	"pmt/internal/features/auth/domain"
)

func TestCreaAlDuenoUnaSolaVez(t *testing.T) {
	ctx := context.Background()
	repo := authtest.New()
	h := New(repo, authtest.Agents{})

	if err := h.Handle(ctx, Command{Email: " Saul@Example.com", Password: "contrasena", Username: "saul"}); err != nil {
		t.Fatal(err)
	}
	u, err := repo.UserByUsername(ctx, "saul")
	if err != nil || !u.IsInstanceOwner || u.Email != "saul@example.com" || u.Name != "saul" {
		t.Fatalf("dueño = %+v, %v", u, err)
	}
	// Con usuarios ya no valida ni crea, aunque la configuración cambie.
	if err := h.Handle(ctx, Command{Email: "otro", Password: "x", Username: "Otro"}); err != nil {
		t.Fatalf("con usuarios: %v", err)
	}
	if users, _ := repo.ListUsers(ctx); len(users) != 1 {
		t.Fatalf("usuarios = %d", len(users))
	}
}

func TestSinConfiguracionNoHaceNada(t *testing.T) {
	repo := authtest.New()
	for _, cmd := range []Command{{}, {Email: "a@b.c"}, {Password: "contrasena"}} {
		if err := New(repo, authtest.Agents{}).Handle(context.Background(), cmd); err != nil {
			t.Errorf("%+v: %v", cmd, err)
		}
	}
	if has, _ := repo.HasUsers(context.Background()); has {
		t.Error("no debe crear usuarios")
	}
}

func TestConfiguracionInvalida(t *testing.T) {
	for _, tc := range []struct {
		cmd  Command
		want error
		key  string
	}{
		{Command{Email: "sin-arroba", Password: "contrasena", Username: "saul"}, domain.ErrEmailInvalid, "owner.email"},
		{Command{Email: "s@x.com", Password: "contrasena", Username: ""}, domain.ErrUsernameInvalid, "owner.username"},
		{Command{Email: "s@x.com", Password: "corta", Username: "saul"}, domain.ErrPasswordTooShort, "owner.password"},
		{Command{Email: "s@x.com", Password: "contrasena", Username: "dbp-tl"}, domain.ErrUsernameTaken, "owner.username"},
	} {
		err := New(authtest.New(), authtest.Agents{"dbp-tl": true}).Handle(context.Background(), tc.cmd)
		if !errors.Is(err, tc.want) || !strings.Contains(err.Error(), tc.key) {
			t.Errorf("%+v: %v", tc.cmd, err)
		}
	}
}
