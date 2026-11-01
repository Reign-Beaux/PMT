// Package store implementa la persistencia de la feature reviews.
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"pmt/internal/features/reviews/domain"
	"pmt/internal/features/reviews/store/sqlcgen"
	"pmt/internal/kernel"
)

// Findings implementa con PostgreSQL las interfaces que piden las slices.
type Findings struct {
	queries *sqlcgen.Queries
}

// NewFindings crea el repositorio de hallazgos.
func NewFindings(db sqlcgen.DBTX) *Findings {
	return &Findings{queries: sqlcgen.New(db)}
}

// Create guarda un hallazgo nuevo.
func (s *Findings) Create(ctx context.Context, f domain.Finding) (domain.Finding, error) {
	row, err := s.queries.CreateFinding(ctx, sqlcgen.CreateFindingParams{
		ID:           f.ID,
		ItemKind:     string(f.ItemKind),
		ItemID:       f.ItemID,
		Description:  f.Description,
		Blocking:     f.Blocking,
		ActorName:    f.ActorName,
		ActorProfile: string(f.ActorProfile),
	})
	if err != nil {
		return domain.Finding{}, fmt.Errorf("crear hallazgo: %w", err)
	}
	return toDomain(row), nil
}

// Get busca un hallazgo por su ID.
func (s *Findings) Get(ctx context.Context, id uuid.UUID) (domain.Finding, error) {
	row, err := s.queries.GetFinding(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Finding{}, domain.ErrFindingNotFound
	}
	if err != nil {
		return domain.Finding{}, fmt.Errorf("obtener hallazgo: %w", err)
	}
	return toDomain(row), nil
}

// Close guarda el cierre solo si sigue abierto: de dos cierres simultáneos, el
// segundo responde que ya está cerrado.
func (s *Findings) Close(ctx context.Context, f domain.Finding) (domain.Finding, error) {
	row, err := s.queries.CloseFinding(ctx, sqlcgen.CloseFindingParams{
		ID:          f.ID,
		Status:      string(f.Status),
		ClosedBy:    pgtype.Text{String: f.ClosedBy, Valid: true},
		CloseReason: f.CloseReason,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Finding{}, domain.ErrFindingClosed
	}
	if err != nil {
		return domain.Finding{}, fmt.Errorf("cerrar hallazgo: %w", err)
	}
	return toDomain(row), nil
}

// List devuelve los hallazgos de un elemento por fecha. Con pending, solo los
// abiertos, bloqueantes primero.
func (s *Findings) List(ctx context.Context, kind domain.ItemKind, itemID uuid.UUID, pending bool) ([]domain.Finding, error) {
	var (
		rows []sqlcgen.Finding
		err  error
	)
	if pending {
		rows, err = s.queries.ListPendingFindings(ctx, sqlcgen.ListPendingFindingsParams{ItemKind: string(kind), ItemID: itemID})
	} else {
		rows, err = s.queries.ListFindings(ctx, sqlcgen.ListFindingsParams{ItemKind: string(kind), ItemID: itemID})
	}
	if err != nil {
		return nil, fmt.Errorf("listar hallazgos: %w", err)
	}
	findings := make([]domain.Finding, 0, len(rows))
	for _, row := range rows {
		findings = append(findings, toDomain(row))
	}
	return findings, nil
}

// ListByTasks devuelve los hallazgos de varias tareas, en el mismo orden que List.
func (s *Findings) ListByTasks(ctx context.Context, taskIDs []uuid.UUID, pending bool) ([]domain.Finding, error) {
	var (
		rows []sqlcgen.Finding
		err  error
	)
	if pending {
		rows, err = s.queries.ListPendingFindingsByItems(ctx, taskIDs)
	} else {
		rows, err = s.queries.ListFindingsByItems(ctx, taskIDs)
	}
	if err != nil {
		return nil, fmt.Errorf("listar hallazgos: %w", err)
	}
	findings := make([]domain.Finding, 0, len(rows))
	for _, row := range rows {
		findings = append(findings, toDomain(row))
	}
	return findings, nil
}

func toDomain(row sqlcgen.Finding) domain.Finding {
	f := domain.Finding{
		ID:           row.ID,
		ItemKind:     domain.ItemKind(row.ItemKind),
		ItemID:       row.ItemID,
		Description:  row.Description,
		Blocking:     row.Blocking,
		Status:       domain.Status(row.Status),
		ActorName:    row.ActorName,
		ActorProfile: kernel.Profile(row.ActorProfile),
		ClosedBy:     row.ClosedBy.String,
		CloseReason:  row.CloseReason,
		CreatedAt:    row.CreatedAt,
	}
	if row.ClosedAt.Valid {
		closedAt := row.ClosedAt.Time
		f.ClosedAt = &closedAt
	}
	return f
}
