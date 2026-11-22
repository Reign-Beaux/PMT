package domain

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"pmt/internal/kernel"
)

func TestValidateUsername(t *testing.T) {
	for _, ok := range []string{"saul", "ana-1", "a", "x9", strings.Repeat("a", 40)} {
		if _, err := ValidateUsername(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "Saul", "ana_1", "-ana", "ana-", "ana maria", "añil", strings.Repeat("a", 41)} {
		if _, err := ValidateUsername(bad); !errors.Is(err, ErrUsernameInvalid) {
			t.Errorf("%q: %v", bad, err)
		}
	}
}

func TestValidateEmail(t *testing.T) {
	if got, err := ValidateEmail("  Ana@Example.COM "); err != nil || got != "ana@example.com" {
		t.Errorf("got %q, %v", got, err)
	}
	for _, bad := range []string{"", "ana", "@x", "ana@", "a na@x.com"} {
		if _, err := ValidateEmail(bad); !errors.Is(err, ErrEmailInvalid) {
			t.Errorf("%q: %v", bad, err)
		}
	}
}

func TestValidatePasswordAndName(t *testing.T) {
	if err := ValidatePassword("1234567"); !errors.Is(err, ErrPasswordTooShort) {
		t.Errorf("corta: %v", err)
	}
	if err := ValidatePassword("ñññññññ8"); err != nil {
		t.Errorf("8 caracteres: %v", err)
	}
	if _, err := ValidateName("   "); !errors.Is(err, ErrNameRequired) {
		t.Errorf("nombre vacío: %v", err)
	}
}

func TestPassword(t *testing.T) {
	hash, err := HashPassword("secreto-largo")
	if err != nil || !strings.HasPrefix(hash, "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Fatalf("hash = %q, %v", hash, err)
	}
	if ok, err := VerifyPassword("secreto-largo", hash); !ok || err != nil {
		t.Errorf("correcta: %t %v", ok, err)
	}
	if ok, err := VerifyPassword("otro", hash); ok || err != nil {
		t.Errorf("incorrecta: %t %v", ok, err)
	}
	if _, err := VerifyPassword("x", "$argon2id$v=19$m=999999,t=3,p=2$AAAA$AAAA"); err == nil {
		t.Error("parámetros fuera de rango")
	}
}

func TestTokens(t *testing.T) {
	a, _ := NewAPIToken()
	b, _ := NewAPIToken()
	if a == b || !IsAPIToken(a) || len(a) != len("pmt_")+64 {
		t.Errorf("tokens = %q %q", a, b)
	}
	agente, _ := NewToken()
	if IsAPIToken(agente) {
		t.Error("un token sin prefijo es de agente")
	}
	if string(HashToken(a)) == a || len(HashToken(a)) != 32 {
		t.Error("se guarda el SHA-256")
	}
}

func TestTokenName(t *testing.T) {
	if got, err := ValidateTokenName(" laptop "); err != nil || got != "laptop" {
		t.Errorf("got %q %v", got, err)
	}
	for _, bad := range []string{" ", strings.Repeat("x", 101)} {
		if _, err := ValidateTokenName(bad); !errors.Is(err, ErrTokenNameInvalid) {
			t.Errorf("%d: %v", len(bad), err)
		}
	}
}

func TestInvitation(t *testing.T) {
	if got := InvitationURL("https://pmt.example.com/", "abc"); got != "https://pmt.example.com/invitacion/abc" {
		t.Errorf("url = %q", got)
	}
	now := time.Now()
	if !InvitationValid(now.Add(time.Hour), false, false, now) {
		t.Error("vigente")
	}
	for _, tc := range []struct {
		expires          time.Time
		accepted, exists bool
	}{
		{now.Add(-time.Second), false, false},
		{now.Add(time.Hour), true, false},
		{now.Add(time.Hour), false, true},
	} {
		if InvitationValid(tc.expires, tc.accepted, tc.exists, now) {
			t.Errorf("%+v no vale", tc)
		}
	}
}

type finder map[string]User

func (f finder) UserByUsername(_ context.Context, username string) (User, error) {
	if u, ok := f[username]; ok {
		return u, nil
	}
	return User{}, ErrUserNotFound
}

type agentNames map[string]bool

func (a agentNames) Exists(_ context.Context, name string) (bool, error) { return a[name], nil }

func TestCurrentUserAndOwner(t *testing.T) {
	ctx := context.Background()
	users := finder{"saul": {Username: "saul", IsInstanceOwner: true}, "ana": {Username: "ana"}}

	if u, err := CurrentOwner(ctx, users, kernel.Actor{Name: "saul", Profile: kernel.ProfileUser}); err != nil || u.Username != "saul" {
		t.Errorf("dueño: %v", err)
	}
	if _, err := CurrentOwner(ctx, users, kernel.Actor{Name: "ana", Profile: kernel.ProfileUser}); !errors.Is(err, ErrOwnerOnly) {
		t.Errorf("ana no es dueña: %v", err)
	}
	for _, actor := range []kernel.Actor{
		{Name: "dbp-tl", Profile: kernel.ProfileTL},
		{Name: "agente-user", Profile: kernel.ProfileUser},
		{},
	} {
		if _, err := CurrentUser(ctx, users, actor); !errors.Is(err, ErrUsersOnly) {
			t.Errorf("%+v: %v", actor, err)
		}
	}
	// Un usuario con perfil declarado sigue siendo él, pero las operaciones de
	// cuenta son del perfil User.
	if _, err := CurrentUser(ctx, users, kernel.Actor{Name: "saul", Profile: kernel.ProfilePP}); !errors.Is(err, ErrProfileDeclared) {
		t.Errorf("usuario como PP: %v", err)
	}
}

func TestEnsureUsernameFree(t *testing.T) {
	agents := agentNames{"dbp-tl": true}
	if err := EnsureUsernameFree(context.Background(), agents, "dbp-tl"); !errors.Is(err, ErrUsernameTaken) {
		t.Errorf("nombre de agente: %v", err)
	}
	if err := EnsureUsernameFree(context.Background(), agents, "ana"); err != nil {
		t.Errorf("libre: %v", err)
	}
}

func TestUserActor(t *testing.T) {
	if got := (User{Username: "ana", Name: "Ana López"}).Actor(); got != (kernel.Actor{Name: "ana", Profile: kernel.ProfileUser}) {
		t.Errorf("actor = %+v", got)
	}
}
