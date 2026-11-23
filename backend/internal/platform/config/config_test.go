package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const valid = `app:
  public_url: http://127.0.0.1:8080
server:
  addr: 127.0.0.1:8080
database:
  connection_string: postgres://localhost/pmt
frontend:
  dist: ""
cli:
  dist: ""
auth:
  cookie_secure: false
owner:
  email: dueno@example.com
  password: contrasena-de-prueba
  username: dueno
`

var overrides = []string{
	"APP_PUBLIC_URL", "SERVER_ADDR", "DATABASE_CONNECTION_STRING", "FRONTEND_DIST", "CLI_DIST",
	"AUTH_COOKIE_SECURE", "OWNER_EMAIL", "OWNER_PASSWORD", "OWNER_USERNAME",
}

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
	if cfg.App.PublicURL != "http://127.0.0.1:8080" || cfg.Auth.CookieSecure || cfg.Owner.Email != "dueno@example.com" || cfg.Owner.Username != "dueno" || cfg.Owner.Password == "" {
		t.Fatal("app, auth u owner no cargadas")
	}
	t.Setenv("OWNER_USERNAME", "otro")
	t.Setenv("AUTH_COOKIE_SECURE", "true")
	cfg, err = Load()
	if err != nil || cfg.Owner.Username != "otro" || !cfg.Auth.CookieSecure {
		t.Fatalf("sobrescritura de owner y auth fallida: %v", err)
	}
	clearOverrides(t)
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
		{"  public_url: http://127.0.0.1:8080\n", "app.public_url"},
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
	if cfg.CLI.Dist != "/app/cli" {
		t.Fatalf("cli.dist de producción = %q", cfg.CLI.Dist)
	}
	if cfg.App.PublicURL != "https://pmt.saulantonio.com" || !cfg.Auth.CookieSecure || cfg.Owner.Email != "changeme" || cfg.Owner.Username != "changeme" {
		t.Fatalf("app, auth u owner de producción inesperados: url=%q secure=%t", cfg.App.PublicURL, cfg.Auth.CookieSecure)
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

func TestCLIDistRelativoAlArchivoYOpcional(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(strings.Replace(valid, "cli:\n  dist: \"\"", "cli:\n  dist: ../dist/cli", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_FILE", path)
	clearOverrides(t)
	t.Chdir(t.TempDir())
	cfg, err := Load()
	if err != nil || cfg.CLI.Dist != filepath.Join(filepath.Dir(dir), "dist", "cli") {
		t.Fatalf("cli.dist = %v, error = %v", cfg.CLI, err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(valid, "cli:\n  dist: \"\"\n", "", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err = Load()
	if err != nil || cfg.CLI.Dist != "" {
		t.Fatalf("sin cli.dist: %v, error = %v", cfg.CLI, err)
	}
}
