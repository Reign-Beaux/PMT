// Package get consulta un proyecto.
package get

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
	"pmt/internal/kernel"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	Get(ctx context.Context, id uuid.UUID) (domain.Project, error)
	Member(ctx context.Context, projectID, userID uuid.UUID) (domain.Member, error)
}

// Result es el proyecto consultado, con el rol de quien consulta.
type Result struct {
	ID             uuid.UUID     `json:"id"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	Status         domain.Status `json:"status"`
	ProductVersion string        `json:"product_version"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	Role           domain.Role   `json:"role,omitempty"`
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve el proyecto y el rol del actor en él. Sin usuario en el actor
// (las comprobaciones internas de otras features) o sin membresía, Role queda
// vacío; la guardia de membresía ya rechazó antes a quien no es miembro.
func (h *Handler) Handle(ctx context.Context, actor kernel.Actor, id uuid.UUID) (Result, error) {
	p, err := h.reader.Get(ctx, id)
	if err != nil {
		return Result{}, err
	}
	result := Result{
		ID:             p.ID,
		Name:           p.Name,
		Description:    p.Description,
		Status:         p.Status,
		ProductVersion: p.ProductVersion,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
	if actor.UserID == uuid.Nil {
		return result, nil
	}
	member, err := h.reader.Member(ctx, id, actor.UserID)
	if errors.Is(err, domain.ErrMemberNotFound) {
		return result, nil
	}
	if err != nil {
		return Result{}, err
	}
	result.Role = member.Role
	return result, nil
}
