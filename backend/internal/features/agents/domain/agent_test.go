package domain

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"pmt/internal/kernel"
)

func TestNewAgent(t *testing.T) {
	tests := []struct {
		name    string
		agent   string
		profile kernel.Profile
		want    error
	}{
		{"válido", " dev ", kernel.ProfileDevBackend, nil},
		{"usuario", "persona", kernel.ProfileUser, nil},
		{"planeación de desarrollo", "dev", kernel.ProfileDP, nil},
		{"sin nombre", "  ", kernel.ProfileTL, ErrAgentNameRequired},
		{"sin perfil", "dev", "", ErrAgentProfileInvalid},
		{"perfil desconocido", "dev", "Dev", ErrAgentProfileInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := NewAgent(tt.agent, tt.profile)

			if !errors.Is(err, tt.want) {
				t.Fatalf("error = %v", err)
			}
			if err == nil && agent.Name != "dev" && agent.Name != "persona" {
				t.Errorf("nombre = %q", agent.Name)
			}
		})
	}
}

func TestAgentActorAndRevoked(t *testing.T) {
	agent, err := NewAgent("tl", kernel.ProfileTL)
	if err != nil {
		t.Fatal(err)
	}

	if got := agent.Actor(); got != (kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}) {
		t.Errorf("actor = %+v", got)
	}
	if agent.Revoked() {
		t.Error("un agente nuevo no está revocado")
	}
	now := time.Now()
	agent.RevokedAt = &now
	if !agent.Revoked() {
		t.Error("debería estar revocado")
	}
}

func TestToken(t *testing.T) {
	first, second := NewToken(), NewToken()

	if len(first) != 2*tokenBytes || first == second {
		t.Errorf("tokens = %d, repetidos = %v", len(first), first == second)
	}
	if !bytes.Equal(HashToken(first), HashToken(first)) || bytes.Equal(HashToken(first), HashToken(second)) {
		t.Error("el hash debe ser determinista y distinguir tokens")
	}
	if bytes.Contains(HashToken(first), []byte(first)) {
		t.Error("el hash no debe contener el token")
	}
}
