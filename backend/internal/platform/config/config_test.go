package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const valid = `server:
  addr: 127.0.0.1:8080
database:
  connection_string: postgres://localhost/pmt
frontend:
  dist: ""
`

var overrides = []string{"SERVER_ADDR", "DATABASE_CONNECTION_STRING", "FRONTEND_DIST"}

func clearOverrides(t *testing.T) {
	t.Helper()
	for _, key := range overrides {
		t.Setenv(key, "")
	}
}

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.yaml")
	t.Setenv("CONFIG_FILE", path)
	clearOverrides(t)
	write := func(s string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(s), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(valid)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Addr != "127.0.0.1:8080" || cfg.Database.ConnectionString != "postgres://localhost/pmt" || cfg.Frontend.Dist != "" {
		t.Fatal("secciones no cargadas")
	}
	t.Setenv("DATABASE_CONNECTION_STRING", "postgres://user:p%40ss@localhost/db?x=1&y=2")
	t.Setenv("SERVER_ADDR", "127.0.0.1:8090")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.ConnectionString != "postgres://user:p%40ss@localhost/db?x=1&y=2" || cfg.Server.Addr != "127.0.0.1:8090" {
		t.Fatal("sobrescritura fallida")
	}
	clearOverrides(t)
	for _, tc := range []struct{ old, key string }{
		{"  connection_string: postgres://localhost/pmt\n", "database.connection_string"},
		{"  addr: 127.0.0.1:8080", "server.addr"},
	} {
		write(strings.Replace(valid, tc.old, "", 1))
		if _, err = Load(); err == nil || !strings.Contains(err.Error(), tc.key) {
			t.Fatalf("error: %v", err)
		}
	}
	t.Setenv("CONFIG_FILE", path+"-missing")
	if _, err = Load(); err == nil || !strings.Contains(err.Error(), path+"-missing") {
		t.Fatalf("error: %v", err)
	}
	write("server: [malformed")
	t.Setenv("CONFIG_FILE", path)
	if _, err = Load(); err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("error: %v", err)
	}
}

func TestProdConfig(t *testing.T) {
	t.Setenv("CONFIG_FILE", "../../../config/config.prod.yaml")
	clearOverrides(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("cargar configuración de producción: %v", err)
	}
	if cfg.Server.Addr != "0.0.0.0:8080" || cfg.Database.ConnectionString != "changeme" || cfg.Frontend.Dist != "" {
		t.Fatalf("configuración de producción inesperada: addr=%q dist=%q", cfg.Server.Addr, cfg.Frontend.Dist)
	}
}

func TestFrontendDistSinCompiladoNoImpideArrancar(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(valid), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_FILE", configPath)
	clearOverrides(t)
	for _, dist := range []string{filepath.Join(dir, "missing"), dir} {
		t.Setenv("FRONTEND_DIST", dist)
		cfg, err := Load()
		if err != nil || cfg.Frontend.Dist != dist {
			t.Fatalf("dist=%q config=%v error=%v", dist, cfg.Frontend, err)
		}
	}
}

func TestFrontendDistRelativeToConfig(t *testing.T) {
	dir := t.TempDir()
	dist := filepath.Join(dir, "web")
	if err := os.Mkdir(dist, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte("app"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(strings.Replace(valid, `dist: ""`, `dist: "./web"`, 1)), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_FILE", path)
	clearOverrides(t)
	t.Chdir(t.TempDir())
	for _, override := range []string{"", "./web"} {
		t.Setenv("FRONTEND_DIST", override)
		cfg, err := Load()
		if err != nil || cfg.Frontend.Dist != dist {
			t.Fatalf("override=%q dist=%v error=%v", override, cfg.Frontend, err)
		}
	}
}
