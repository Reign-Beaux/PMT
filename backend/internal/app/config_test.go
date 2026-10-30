package app

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfigReadsFile(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	path := writeConfigFile(t, "database:\n  url: postgres://archivo/pmt\n")

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != "postgres://archivo/pmt" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
}

func TestLoadConfigEnvironmentOverridesFile(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://entorno/pmt")
	path := writeConfigFile(t, "database:\n  url: postgres://archivo/pmt\n")

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != "postgres://entorno/pmt" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
}

func TestLoadConfigWithoutFile(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://entorno/pmt")

	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != "postgres://entorno/pmt" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
}

func TestLoadConfigRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	if _, err := LoadConfig(filepath.Join(t.TempDir(), "config.yaml")); err == nil {
		t.Fatal("se esperaba un error")
	}
}
