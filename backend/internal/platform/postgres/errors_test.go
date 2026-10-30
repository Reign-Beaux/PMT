package postgres

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestViolations(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		foreignKey bool
		unique     bool
	}{
		{"llave foránea", &pgconn.PgError{Code: "23503"}, true, false},
		{"restrict", &pgconn.PgError{Code: "23001"}, true, false},
		{"unicidad envuelta", fmt.Errorf("capa: %w", &pgconn.PgError{Code: "23505"}), false, true},
		{"otro código", &pgconn.PgError{Code: "23514"}, false, false},
		{"otro error", errors.New("x"), false, false},
		{"sin error", nil, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsForeignKeyViolation(tt.err); got != tt.foreignKey {
				t.Errorf("IsForeignKeyViolation = %v", got)
			}
			if got := IsUniqueViolation(tt.err); got != tt.unique {
				t.Errorf("IsUniqueViolation = %v", got)
			}
		})
	}
}
