package store

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"pmt/internal/features/stages/domain"
	"pmt/internal/features/stages/store/sqlcgen"
)

// failingDB responde con el error que da PostgreSQL al borrar una fila referenciada.
type failingDB struct {
	err error
}

func (d failingDB) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, d.err
}

func (d failingDB) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, d.err
}

func (d failingDB) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	return nil
}

// TestDeleteTranslatesReferences cubre lo retirado: sus filas siguen referenciando
// al padre aunque list no las muestre, y deben responder Conflict, no error interno.
func TestDeleteTranslatesReferences(t *testing.T) {
	for _, code := range []string{"23001", "23503"} {
		t.Run(code, func(t *testing.T) {
			err := NewStages(failingDB{err: &pgconn.PgError{Code: code}}).Delete(context.Background(), uuid.New())

			if !errors.Is(err, domain.ErrStageHasTasks) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestToHistoriesGroupsByStage(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	rows := []sqlcgen.ListStageChangesRow{
		{StageChange: sqlcgen.StageChange{StageID: first, Version: 1}, OriginalName: "A"},
		{StageChange: sqlcgen.StageChange{StageID: first, Version: 2}, OriginalName: "A"},
		{StageChange: sqlcgen.StageChange{StageID: second, Version: 1}, OriginalName: "B"},
	}

	histories := toHistories(rows)

	if len(histories) != 2 || histories[0].Name != "A" || len(histories[0].Changes) != 2 || histories[1].StageID != second {
		t.Errorf("historias = %+v", histories)
	}
}
