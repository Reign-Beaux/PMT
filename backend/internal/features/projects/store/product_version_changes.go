package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"pmt/internal/features/projects/domain"
	"pmt/internal/features/projects/store/sqlcgen"
)

// ProductVersionChanges guarda los cambios de versión de producto y su registro inmutable.
type ProductVersionChanges struct {
	db DB
}

// NewProductVersionChanges crea el repositorio de cambios de versión de producto.
func NewProductVersionChanges(db DB) *ProductVersionChanges {
	return &ProductVersionChanges{db: db}
}

// ChangeProductVersion guarda la nueva versión y su registro en una transacción.
// Bloquea la fila para que la versión de origen del registro siempre sea cierta.
func (s *ProductVersionChanges) ChangeProductVersion(ctx context.Context, change domain.ProductVersionChange) (domain.Project, error) {
	var updated domain.Project
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		if err := ensureVersion(ctx, queries, change); err != nil {
			return err
		}
		if err := insertProductVersionChange(ctx, queries, change); err != nil {
			return err
		}
		row, err := queries.UpdateProjectProductVersion(ctx, sqlcgen.UpdateProjectProductVersionParams{
			ID:             change.ProjectID,
			ProductVersion: change.To,
		})
		if err != nil {
			return fmt.Errorf("actualizar versión de producto: %w", err)
		}
		updated = toDomain(row)
		return nil
	})
	if err != nil {
		return domain.Project{}, err
	}
	return updated, nil
}

// ListProductVersionChanges devuelve el historial de versiones por orden de fecha,
// aunque el proyecto ya no exista.
func (s *ProductVersionChanges) ListProductVersionChanges(ctx context.Context, id uuid.UUID) ([]domain.ProductVersionChange, error) {
	rows, err := sqlcgen.New(s.db).ListProductVersionChanges(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("listar cambios de versión de producto: %w", err)
	}
	changes := make([]domain.ProductVersionChange, 0, len(rows))
	for _, row := range rows {
		changes = append(changes, domain.ProductVersionChange{
			ID:           row.ID,
			ProjectID:    row.ProjectID,
			From:         row.FromVersion,
			To:           row.ToVersion,
			ActorName:    row.ActorName,
			ActorProfile: row.ActorProfile,
			Reason:       row.Reason,
			ChangedAt:    row.ChangedAt,
		})
	}
	return changes, nil
}

// ensureVersion bloquea la fila y comprueba que el proyecto siga en working con la
// versión de origen.
func ensureVersion(ctx context.Context, queries *sqlcgen.Queries, change domain.ProductVersionChange) error {
	row, err := queries.LockProject(ctx, change.ProjectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrProjectNotFound
	}
	if err != nil {
		return fmt.Errorf("bloquear proyecto: %w", err)
	}
	if domain.Status(row.Status) != domain.StatusWorking || row.ProductVersion != change.From {
		return domain.ErrProductVersionChanged
	}
	return nil
}

func insertProductVersionChange(ctx context.Context, queries *sqlcgen.Queries, change domain.ProductVersionChange) error {
	err := queries.CreateProductVersionChange(ctx, sqlcgen.CreateProductVersionChangeParams{
		ID:           change.ID,
		ProjectID:    change.ProjectID,
		FromVersion:  change.From,
		ToVersion:    change.To,
		ActorName:    change.ActorName,
		ActorProfile: change.ActorProfile,
		Reason:       change.Reason,
	})
	if err != nil {
		return fmt.Errorf("guardar cambio de versión de producto: %w", err)
	}
	return nil
}
