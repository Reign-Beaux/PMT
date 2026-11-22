package postgres

import (
	"io/fs"
	"testing"
)

func TestEmbeddedMigrations(t *testing.T) {
	files, err := fs.Glob(migrations, "migrations/*.sql")
	if err != nil || len(files) != 26 || files[0] != "migrations/00001_create_projects.sql" || files[25] != "migrations/00026_allow_users_in_stage_teams.sql" {
		t.Fatalf("migraciones: %v %v", files, err)
	}
}
