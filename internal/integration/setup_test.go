// Package integration holds end-to-end tests that exercise the full stack:
// HTTP handler → application service → PostgreSQL.
//
// These tests require a running PostgreSQL instance. They are skipped
// automatically when the database is unavailable.
//
// Run with:
//
//	docker-compose up -d
//	go test -race -count=1 ./internal/integration/...
//
// Override the database URL:
//
//	TEST_DATABASE_URL=postgres://user:pass@host/db?sslmode=disable go test ./internal/integration/...
package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	pgadapter "project-management-tools/internal/adapter/driven/postgres"
	"project-management-tools/internal/adapter/driving/httpserver"
	"project-management-tools/internal/adapter/driving/httpserver/handler"
	commentapp "project-management-tools/internal/application/comment"
	issueapp "project-management-tools/internal/application/issue"
	labelapp "project-management-tools/internal/application/label"
	phaseapp "project-management-tools/internal/application/phase"
	projectapp "project-management-tools/internal/application/project"
	userapp "project-management-tools/internal/application/user"
)

var (
	testServer    *httptest.Server
	testDB        *gorm.DB
	testClient    *http.Client // authenticated client; used by all helper functions
	dbAvailable   bool
	testJWTSecret = []byte("test-secret")
)

func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://pmt:pmt@localhost:5432/pmt_test?sslmode=disable"
	}

	if err := pgadapter.EnsureDatabase(dsn); err != nil {
		fmt.Println("integration: database unavailable, skipping all tests:", err)
		os.Exit(0)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fmt.Println("integration: failed to connect:", err)
		os.Exit(0)
	}

	if err := pgadapter.RunMigrations(db); err != nil {
		fmt.Println("integration: failed to run migrations:", err)
		os.Exit(1)
	}

	testDB = db
	dbAvailable = true

	// Wire the full stack
	userRepo := pgadapter.NewUserRepository(db)
	tokenRepo := pgadapter.NewTokenRepository(db)
	projectRepo := pgadapter.NewProjectRepository(db)
	phaseRepo := pgadapter.NewPhaseRepository(db)
	issueRepo := pgadapter.NewIssueRepository(db)
	labelRepo := pgadapter.NewLabelRepository(db)
	commentRepo := pgadapter.NewCommentRepository(db)

	userService := userapp.NewService(userRepo, tokenRepo)
	projectService := projectapp.NewService(projectRepo)
	phaseService := phaseapp.NewService(phaseRepo, projectRepo)
	issueService := issueapp.NewService(issueRepo, phaseRepo, projectRepo, labelRepo)
	labelService := labelapp.NewService(labelRepo, projectRepo)
	commentService := commentapp.NewService(commentRepo, issueRepo)

	authHandler := handler.NewAuthHandler(userService, testJWTSecret)
	projectHandler := handler.NewProjectHandler(projectService)
	phaseHandler := handler.NewPhaseHandler(phaseService)
	issueHandler := handler.NewIssueHandler(issueService)
	labelHandler := handler.NewLabelHandler(labelService, issueService)
	commentHandler := handler.NewCommentHandler(commentService)

	router := httpserver.NewRouter(
		authHandler,
		projectHandler,
		phaseHandler,
		issueHandler,
		labelHandler,
		commentHandler,
		testJWTSecret,
	)
	testServer = httptest.NewServer(router)
	defer testServer.Close()

	// Build a client with a cookie jar so auth cookies persist across requests.
	jar, err := cookiejar.New(nil)
	if err != nil {
		fmt.Println("integration: failed to create cookie jar:", err)
		os.Exit(1)
	}
	testClient = &http.Client{Jar: jar}

	os.Exit(m.Run())
}

// skipIfNoDB skips the test when PostgreSQL is not available.
func skipIfNoDB(t *testing.T) {
	t.Helper()
	if !dbAvailable {
		t.Skip("integration tests require a running PostgreSQL database")
	}
}

// cleanup truncates all tables and re-registers the test user so auth
// cookies in testClient remain valid across tests.
func cleanup(t *testing.T) {
	t.Helper()
	sqlDB, err := testDB.DB()
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	_, err = sqlDB.Exec(`TRUNCATE refresh_tokens, comments, issue_labels, issues, labels, phases, projects, users CASCADE`)
	if err != nil {
		t.Fatalf("cleanup truncate: %v", err)
	}

	// Re-create the test user and refresh the auth cookies.
	body, _ := json.Marshal(map[string]any{
		"email":    "test@example.com",
		"password": "password123",
	})
	req, err := http.NewRequest(http.MethodPost, url("/auth/register"), bytes.NewReader(body))
	if err != nil {
		t.Fatalf("cleanup register request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := testClient.Do(req)
	if err != nil {
		t.Fatalf("cleanup register: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("cleanup register failed: status %d body %s", resp.StatusCode, b)
	}
}

// ── HTTP helpers ──────────────────────────────────────────────────────────────

func url(path string) string {
	return testServer.URL + path
}

func post(t *testing.T, path string, body any) *http.Response {
	t.Helper()
	return request(t, http.MethodPost, path, body)
}

func get(t *testing.T, path string) *http.Response {
	t.Helper()
	return request(t, http.MethodGet, path, nil)
}

func patch(t *testing.T, path string, body any) *http.Response {
	t.Helper()
	return request(t, http.MethodPatch, path, body)
}

func delete(t *testing.T, path string) *http.Response {
	t.Helper()
	return request(t, http.MethodDelete, path, nil)
}

func request(t *testing.T, method, path string, body any) *http.Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, url(path), r)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := testClient.Do(req)
	if err != nil {
		t.Fatalf("do request %s %s: %v", method, path, err)
	}
	return resp
}

// decode reads the response body and unmarshals it into a map.
func decode(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	var v map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return v
}

// decodeList reads the response body and unmarshals it into a slice of maps.
func decodeList(t *testing.T, resp *http.Response) []map[string]any {
	t.Helper()
	defer resp.Body.Close()
	var v []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	return v
}

// assertStatus fails the test if the response status code doesn't match.
func assertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected status %d, got %d\nbody: %s", want, resp.StatusCode, strings.TrimSpace(string(body)))
	}
}

// str extracts a string field from a decoded map.
func str(t *testing.T, m map[string]any, key string) string {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("key %q not found in response", key)
	}
	s, ok := v.(string)
	if !ok {
		t.Fatalf("key %q is not a string (got %T: %v)", key, v, v)
	}
	return s
}

// isNull reports whether a field is JSON null.
func isNull(m map[string]any, key string) bool {
	v, ok := m[key]
	return !ok || v == nil
}

// strSlice extracts a string slice from a decoded map.
func strSlice(t *testing.T, m map[string]any, key string) []string {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("key %q not found", key)
	}
	raw, ok := v.([]any)
	if !ok {
		t.Fatalf("key %q is not an array", key)
	}
	out := make([]string, len(raw))
	for i, el := range raw {
		s, ok := el.(string)
		if !ok {
			t.Fatalf("element %d of %q is not a string", i, key)
		}
		out[i] = s
	}
	return out
}

// contains reports whether slice contains s.
func contains(slice []string, s string) bool {
	for _, el := range slice {
		if el == s {
			return true
		}
	}
	return false
}
