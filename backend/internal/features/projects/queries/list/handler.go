// Package list consulta los proyectos de quien actúa.
package list

import (
	"context"
	"time"

	"github.com/google/uuid"

	"pmt/internal/features/projects/domain"
	"pmt/internal/kernel"
)

// Reader es la lectura que necesita la operación.
type Reader interface {
	List(ctx context.Context, userID uuid.UUID, roles []domain.Role) ([]domain.MemberProject, error)
}

// Item es un proyecto de la lista, con el rol de quien consulta.
type Item struct {
	ID             uuid.UUID     `json:"id"`
	Name           string        `json:"name"`
	Description    string        `json:"description"`
	Status         domain.Status `json:"status"`
	ProductVersion string        `json:"product_version"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	Role           domain.Role   `json:"role"`
}

// Handler ejecuta la operación.
type Handler struct {
	reader Reader
}

// New crea el handler.
func New(reader Reader) *Handler {
	return &Handler{reader: reader}
}

// Handle devuelve los proyectos de los que el actor es miembro: los suyos y los
// que le compartieron. Un token de agente ve los del dueño de la instancia. Con
// token no aparecen aquellos de los que es viewer: solo los ve el sitio web.
func (h *Handler) Handle(ctx context.Context, actor kernel.Actor) ([]Item, error) {
	if actor.UserID == uuid.Nil {
		return []Item{}, nil
	}
	projects, err := h.reader.List(ctx, actor.UserID, domain.ReachableRoles(actor))
	if err != nil {
		return nil, err
	}

	items := make([]Item, 0, len(projects))
	for _, p := range projects {
		items = append(items, Item{
			ID:             p.ID,
			Name:           p.Name,
			Description:    p.Description,
			Status:         p.Status,
			ProductVersion: p.ProductVersion,
			CreatedAt:      p.CreatedAt,
			UpdatedAt:      p.UpdatedAt,
			Role:           p.Role,
		})
	}
	return items, nil
}
