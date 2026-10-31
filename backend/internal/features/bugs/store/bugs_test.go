package store

import (
	"testing"
	"time"

	"github.com/google/uuid"

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
}
