package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"pmt/internal/kernel"
)

func TestNewComment(t *testing.T) {
	actor := kernel.Actor{Name: "tl", Profile: kernel.ProfileTL}
	id := uuid.New()

	c, err := NewComment(ItemStage, id, "  Va a QA  ", actor)

	if err != nil {
		t.Fatal(err)
	}
	if c.ID == uuid.Nil || c.ItemKind != ItemStage || c.ItemID != id || c.Body != "Va a QA" || c.ActorName != "tl" || c.ActorProfile != kernel.ProfileTL {
		t.Errorf("comentario = %+v", c)
	}
}

func TestNewCommentRejects(t *testing.T) {
	tests := []struct {
		name string
		kind ItemKind
		body string
		want error
	}{
		{"vacío", ItemTask, "", ErrBodyRequired},
		{"solo espacios", ItemTask, " \n\t ", ErrBodyRequired},
		{"tipo desconocido", "finding", "Texto", ErrItemKindInvalid},
		{"tipo vacío", "", "Texto", ErrItemKindInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewComment(tt.kind, uuid.New(), tt.body, kernel.Owner); !errors.Is(err, tt.want) {
				t.Errorf("error = %v", err)
			}
		})
	}
}
