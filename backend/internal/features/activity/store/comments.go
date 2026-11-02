// Package store implementa la persistencia de la feature activity.
package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"pmt/internal/features/activity/domain"
	"pmt/internal/features/activity/store/sqlcgen"
	"pmt/internal/kernel"
)

// Comments implementa con PostgreSQL las interfaces que piden las slices.
type Comments struct {
	queries *sqlcgen.Queries
}

// NewComments crea el repositorio de comentarios.
func NewComments(db sqlcgen.DBTX) *Comments {
	return &Comments{queries: sqlcgen.New(db)}
}

// Create guarda un comentario nuevo.
func (s *Comments) Create(ctx context.Context, c domain.Comment) (domain.Comment, error) {
	row, err := s.queries.CreateComment(ctx, sqlcgen.CreateCommentParams{
		ID:           c.ID,
		ItemKind:     string(c.ItemKind),
		ItemID:       c.ItemID,
		Body:         c.Body,
		ActorName:    c.ActorName,
		ActorProfile: string(c.ActorProfile),
	})
	if err != nil {
		return domain.Comment{}, fmt.Errorf("crear comentario: %w", err)
	}
	return toDomain(row), nil
}

// List devuelve los comentarios de un elemento por fecha.
func (s *Comments) List(ctx context.Context, kind domain.ItemKind, itemID uuid.UUID) ([]domain.Comment, error) {
	rows, err := s.queries.ListComments(ctx, sqlcgen.ListCommentsParams{ItemKind: string(kind), ItemID: itemID})
	if err != nil {
		return nil, fmt.Errorf("listar comentarios: %w", err)
	}
	comments := make([]domain.Comment, 0, len(rows))
	for _, row := range rows {
		comments = append(comments, toDomain(row))
	}
	return comments, nil
}

func toDomain(row sqlcgen.Comment) domain.Comment {
	return domain.Comment{
		ID:           row.ID,
		ItemKind:     domain.ItemKind(row.ItemKind),
		ItemID:       row.ItemID,
		Body:         row.Body,
		ActorName:    row.ActorName,
		ActorProfile: kernel.Profile(row.ActorProfile),
		CreatedAt:    row.CreatedAt,
	}
}
