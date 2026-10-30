package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	restrictViolation   = "23001"
	foreignKeyViolation = "23503"
	uniqueViolation     = "23505"
)

// IsForeignKeyViolation indica si err es una violación de llave foránea. Las
// llaves con ON DELETE RESTRICT se reportan como restrict_violation.
func IsForeignKeyViolation(err error) bool {
	return hasCode(err, foreignKeyViolation) || hasCode(err, restrictViolation)
}

// IsUniqueViolation indica si err es una violación de unicidad.
func IsUniqueViolation(err error) bool {
	return hasCode(err, uniqueViolation)
}

func hasCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
