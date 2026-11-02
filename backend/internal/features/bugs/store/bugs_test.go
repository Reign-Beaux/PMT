package store

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"pmt/internal/features/bugs/domain"
	"pmt/internal/features/bugs/store/sqlcgen"
)

func TestToDomain(t *testing.T) {
	row := sqlcgen.Bug{
		ID: uuid.New(), ProjectID: uuid.New(), Title: "T", Description: "D",
		StepsToReproduce: "P", ProductVersion: "1", Status: "reported", CreatedAt: time.Now(),
	}

	b := toDomain(row)

	if b.ID != row.ID || b.StepsToReproduce != "P" || b.ProductVersion != "1" || b.Status != domain.StatusReported {
		t.Errorf("bug = %+v", b)
	}
	if b.ReportedBy != "" {
		t.Errorf("un bug sin autor no debe inventarlo: %q", b.ReportedBy)
	}

	row.ReportedBy = pgtype.Text{String: "pp", Valid: true}
	if b := toDomain(row); b.ReportedBy != "pp" {
		t.Errorf("reported_by = %q", b.ReportedBy)
	}
}
