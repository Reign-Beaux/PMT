package store

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"pmt/internal/features/projects/domain"
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
			err := NewProjects(failingDB{err: &pgconn.PgError{Code: code}}).Delete(context.Background(), uuid.New())

			if !errors.Is(err, domain.ErrProjectHasStages) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}
