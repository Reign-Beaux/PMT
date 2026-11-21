package postgres

import (
	"io/fs"
	"testing"
)

func TestEmbeddedMigrations(t *testing.T) {
	files, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil || len(files) != 25 || files[0] != "migrations/00001_create_projects.sql" || files[24] != "migrations/00025_create_invitations.sql" {
		t.Fatalf("migraciones: %v %v", files, err)
	}
}
