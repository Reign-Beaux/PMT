package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"pmt/internal/features/projects/domain"
)

// projectRow responde como una fila de projects en el orden de SELECT *.
type projectRow struct {
	project domain.Project
	err     error
}

func (r projectRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*dest[0].(*uuid.UUID) = r.project.ID
	*dest[1].(*string) = r.project.Name
	*dest[2].(*string) = r.project.Description
	*dest[3].(*time.Time) = r.project.CreatedAt
	*dest[4].(*time.Time) = r.project.UpdatedAt
	*dest[5].(*string) = string(r.project.Status)
	*dest[6].(*string) = r.project.ProductVersion
	return nil
}

// fakeTx registra las sentencias de la transacción y si se confirmó o se deshizo.
type fakeTx struct {
	pgx.Tx
	locked     domain.Project
	insertErr  error
	statements []string
	committed  bool
	rolledBack bool
}

func (tx *fakeTx) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	tx.statements = append(tx.statements, sql)
	return pgconn.CommandTag{}, tx.insertErr
}

func (tx *fakeTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	tx.statements = append(tx.statements, sql)
	if strings.Contains(sql, "FOR UPDATE") {
		return projectRow{project: tx.locked}
	}
	updated := tx.locked
	updated.ProductVersion = args[1].(string)
	return projectRow{project: updated}
}

func (tx *fakeTx) Commit(context.Context) error {
	tx.committed = true
	return nil
}

func (tx *fakeTx) Rollback(context.Context) error {
	if !tx.committed {
		tx.rolledBack = true
	}
	return nil
}

// txDB abre siempre la misma transacción y falla si se escribe fuera de ella.
type txDB struct {
	failingDB
	tx *fakeTx
}

func (d txDB) Begin(context.Context) (pgx.Tx, error) {
	return d.tx, nil
}

func workingProject() domain.Project {
	return domain.Project{ID: uuid.New(), Name: "PMT", Status: domain.StatusWorking, ProductVersion: "1.0.0"}
}

func versionChange(p domain.Project, to string) domain.ProductVersionChange {
	return domain.ProductVersionChange{ID: uuid.New(), ProjectID: p.ID, From: p.ProductVersion, To: to, ActorName: "pp", ActorProfile: "PP", Reason: "motivo"}
}

func TestChangeProductVersionWritesInOneTransaction(t *testing.T) {
	p := workingProject()
	tx := &fakeTx{locked: p}
	db := txDB{failingDB: failingDB{err: errors.New("fuera de la transacción")}, tx: tx}

	updated, err := NewProductVersionChanges(db).ChangeProductVersion(context.Background(), versionChange(p, "1.1.0"))

	if err != nil {
		t.Fatal(err)
	}
	if updated.ProductVersion != "1.1.0" || !tx.committed {
		t.Errorf("proyecto = %+v, confirmada = %v", updated, tx.committed)
	}
	if len(tx.statements) != 3 || !strings.Contains(tx.statements[0], "FOR UPDATE") ||
		!strings.Contains(tx.statements[1], "INSERT INTO product_version_changes") ||
		!strings.Contains(tx.statements[2], "UPDATE projects") {
		t.Errorf("sentencias = %q", tx.statements)
	}
}

func TestChangeProductVersionRollsBackWhenHistoryFails(t *testing.T) {
	p := workingProject()
	tx := &fakeTx{locked: p, insertErr: errors.New("fallo al insertar")}

	_, err := NewProductVersionChanges(txDB{tx: tx}).ChangeProductVersion(context.Background(), versionChange(p, "1.1.0"))

	if err == nil {
		t.Fatal("se esperaba error")
	}
	if tx.committed || !tx.rolledBack {
		t.Errorf("confirmada = %v, deshecha = %v", tx.committed, tx.rolledBack)
	}
	for _, sql := range tx.statements {
		if strings.Contains(sql, "UPDATE projects") {
			t.Error("no debió actualizar el proyecto")
		}
	}
}

func TestChangeProductVersionRejectsStaleProject(t *testing.T) {
	tests := []struct {
		name   string
		locked func(domain.Project) domain.Project
	}{
		{"otra versión", func(p domain.Project) domain.Project { p.ProductVersion = "1.0.5"; return p }},
		{"ya no está en working", func(p domain.Project) domain.Project { p.Status = domain.StatusStopped; return p }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := workingProject()
			tx := &fakeTx{locked: tt.locked(p)}

			_, err := NewProductVersionChanges(txDB{tx: tx}).ChangeProductVersion(context.Background(), versionChange(p, "1.1.0"))

			if !errors.Is(err, domain.ErrProductVersionChanged) {
				t.Fatalf("err = %v", err)
			}
			if tx.committed || len(tx.statements) != 1 {
				t.Errorf("confirmada = %v, sentencias = %q", tx.committed, tx.statements)
			}
		})
	}
}
