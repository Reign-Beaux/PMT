package rest

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pmt/internal/kernel"
)

func TestWriteError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{"inválido", kernel.Invalid("mal"), http.StatusBadRequest, "mal"},
		{"no existe", kernel.NotFound("nada"), http.StatusNotFound, "nada"},
		{"envuelto", fmt.Errorf("capa: %w", kernel.NotFound("nada")), http.StatusNotFound, "nada"},
		{"interno", errors.New("secreto"), http.StatusInternalServerError, "error interno"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			WriteError(rec, tt.err)
			if rec.Code != tt.status {
				t.Errorf("status = %d", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tt.body) {
				t.Errorf("cuerpo = %s", rec.Body)
			}
		})
	}
}

func TestDecodeJSONRejectsInvalidBodies(t *testing.T) {
	for _, body := range []string{`{`, `{"x":1}`, `{} {}`} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		var dst struct{}

		var businessErr *kernel.Error
		if err := DecodeJSON(rec, req, &dst); !errors.As(err, &businessErr) {
			t.Errorf("cuerpo %q: err = %v", body, err)
		}
	}
}
