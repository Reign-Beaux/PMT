package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"pmt/internal/features/stages/domain"
	"pmt/internal/features/stages/store/sqlcgen"
)

// StatusChanges guarda los cambios de status y su registro inmutable.
type StatusChanges struct {
	db DB
}

// NewStatusChanges crea el repositorio de cambios de status.
func NewStatusChanges(db DB) *StatusChanges {
	return &StatusChanges{db: db}
}

// ChangeStatus guarda el nuevo status y su registro en una transacción. Bloquea
// la fila para que el status de origen del registro siempre sea cierto.
func (s *StatusChanges) ChangeStatus(ctx context.Context, stage domain.Stage, change domain.StatusChange) (domain.Stage, error) {
	var updated domain.Stage
	err := pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error {
		queries := sqlcgen.New(tx)
		if err := ensureStatus(ctx, queries, change); err != nil {
			return err
		}
		if err := insertStatusChange(ctx, queries, change); err != nil {
			return err
		}
		var err error
		updated, err = NewStages(tx).UpdateStatus(ctx, stage)
		return err
	})
	if err != nil {
		return domain.Stage{}, err
	}
	return updated, nil
}

// ListStatusChanges devuelve el historial de status por orden de fecha. Incluye
// el de elementos retirados o eliminados.
func (s *StatusChanges) ListStatusChanges(ctx context.Context, id uuid.UUID) ([]domain.StatusChange, error) {
	rows, err := sqlcgen.New(s.db).ListStageStatusChanges(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("listar cambios de status: %w", err)
	}
	changes := make([]domain.StatusChange, 0, len(rows))
	for _, row := range rows {
		changes = append(changes, domain.StatusChange{
			ID:           row.ID,
			StageID:      row.StageID,
			From:         domain.Status(row.FromStatus),
			To:           domain.Status(row.ToStatus),
			ActorName:    row.ActorName,
			ActorProfile: row.ActorProfile,
			Reason:       row.Reason,
			ChangedAt:    row.ChangedAt,
		})
	}
	return changes, nil
}

// ensureStatus bloquea la fila y comprueba que su status siga siendo el de origen.
func ensureStatus(ctx context.Context, queries *sqlcgen.Queries, change domain.StatusChange) error {
	row, err := queries.LockStage(ctx, change.StageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrStageNotFound
	}
	if err != nil {
		return fmt.Errorf("bloquear etapa: %w", err)
	}
	if domain.Status(row.Status) != change.From {
		return domain.ErrStageStatusChanged
	}
	return nil
}

func insertStatusChange(ctx context.Context, queries *sqlcgen.Queries, change domain.StatusChange) error {
	err := queries.CreateStageStatusChange(ctx, sqlcgen.CreateStageStatusChangeParams{
		ID:           change.ID,
		StageID:      change.StageID,
		FromStatus:   string(change.From),
		ToStatus:     string(change.To),
		ActorName:    change.ActorName,
		ActorProfile: change.ActorProfile,
		Reason:       change.Reason,
	})
	if err != nil {
		return fmt.Errorf("guardar cambio de status: %w", err)
	}
	return nil
}
