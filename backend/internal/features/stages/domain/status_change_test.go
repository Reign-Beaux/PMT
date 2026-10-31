package domain

import (
	"testing"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

func TestNewStatusChange(t *testing.T) {
	id := uuid.New()
	actor := kernel.Actor{Name: "agente", Profile: "PP"}

	change := NewStatusChange(id, "desde", "hacia", actor, "  motivo  ")

	if change.ID == uuid.Nil || change.StageID != id {
		t.Errorf("identificadores = %v, %v", change.ID, change.StageID)
	}
	if change.From != "desde" || change.To != "hacia" {
		t.Errorf("status = %q, %q", change.From, change.To)
	}
	if change.ActorName != "agente" || change.ActorProfile != "PP" {
		t.Errorf("actor = %q, %q", change.ActorName, change.ActorProfile)
	}
	if change.Reason != "motivo" {
		t.Errorf("motivo = %q", change.Reason)
	}
}
