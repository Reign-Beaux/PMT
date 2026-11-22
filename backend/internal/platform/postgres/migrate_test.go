package postgres

import (
	"io/fs"
	"testing"
)

func TestEmbeddedMigrations(t *testing.T) {
	files, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil || len(files) != 27 || files[0] != "migrations/00001_create_projects.sql" || files[26] != "migrations/00027_create_project_members.sql" {
		t.Fatalf("migraciones: %v %v", files, err)
	}
}
