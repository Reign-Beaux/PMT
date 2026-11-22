package postgres

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"pmt/internal/kernel"
)

// HoldOf arma el bloqueo de una etapa o un bug con sus columnas taken_by,
// taken_by_name y taken_at.
func HoldOf(takenBy pgtype.UUID, name pgtype.Text, at pgtype.Timestamptz) kernel.Hold {
	if !takenBy.Valid {
		return kernel.Hold{}
	}
	return kernel.Hold{HolderID: uuid.UUID(takenBy.Bytes), TakenBy: name.String, TakenAt: at.Time}
}

// HoldParams da los valores de taken_by y taken_by_name de un bloqueo: NULL si
// nadie lo tiene. taken_at lo calcula la consulta.
func HoldParams(hold kernel.Hold) (pgtype.UUID, pgtype.Text) {
	if !hold.Taken() {
		return pgtype.UUID{}, pgtype.Text{}
	}
	return pgtype.UUID{Bytes: hold.HolderID, Valid: true}, pgtype.Text{String: hold.TakenBy, Valid: true}
}
