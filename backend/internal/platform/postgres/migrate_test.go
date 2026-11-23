package postgres

import (
	"io/fs"
	"testing"
)

func TestEmbeddedMigrations(t *testing.T) {
	files, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil || len(files) != 29 || files[0] != "migrations/00001_create_projects.sql" || files[28] != "migrations/00029_create_cli_logins.sql" {
		t.Fatalf("migraciones: %v %v", files, err)
	}
}
