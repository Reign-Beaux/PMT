package store

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"pmt/internal/features/tasks/domain"
	"pmt/internal/features/tasks/store/sqlcgen"
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
			err := NewTasks(failingDB{err: &pgconn.PgError{Code: code}}).Delete(context.Background(), uuid.New())

			if !errors.Is(err, domain.ErrTaskHasChanges) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestToHistoriesGroupsByTask(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	rows := []sqlcgen.ListTaskChangesRow{
		{TaskChange: sqlcgen.TaskChange{TaskID: first, Version: 1}, OriginalTitle: "A"},
		{TaskChange: sqlcgen.TaskChange{TaskID: first, Version: 2}, OriginalTitle: "A"},
		{TaskChange: sqlcgen.TaskChange{TaskID: second, Version: 1}, OriginalTitle: "B"},
	}

	histories := toHistories(rows)

	if len(histories) != 2 || histories[0].Title != "A" || len(histories[0].Changes) != 2 || histories[1].TaskID != second {
		t.Errorf("historias = %+v", histories)
	}
}
