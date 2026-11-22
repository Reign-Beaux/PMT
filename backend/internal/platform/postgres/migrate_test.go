package postgres

import (
	"io/fs"
	"testing"
)

func TestEmbeddedMigrations(t *testing.T) {
	files, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil || len(files) != 28 || files[0] != "migrations/00001_create_projects.sql" || files[27] != "migrations/00028_add_holds.sql" {
		t.Fatalf("migraciones: %v %v", files, err)
	}
}
