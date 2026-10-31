package rest

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"pmt/internal/kernel"
)

var testActor = kernel.Actor{Name: "agente", Profile: kernel.ProfileTL}

type fakeAuthenticator struct{}

func (fakeAuthenticator) Authenticate(_ context.Context, token string) (kernel.Actor, error) {
	switch token {
	case "bueno":
		return testActor, nil
	case "roto":
		return kernel.Actor{}, errors.New("base de datos caída")
	}
	return kernel.Actor{}, kernel.ErrUnauthenticated
}

func TestAuthenticate(t *testing.T) {
	tests := []struct {
		name   string
		header string
		status int
		actor  kernel.Actor
	}{
		{"sin cabecera", "", http.StatusOK, kernel.Anonymous},
		{"variable vacía", "Bearer", http.StatusOK, kernel.Anonymous},
		{"vacío con espacio", "Bearer ", http.StatusOK, kernel.Anonymous},
		{"válido", "Bearer bueno", http.StatusOK, testActor},
		{"inválido", "Bearer malo", http.StatusUnauthorized, kernel.Actor{}},
		{"revocado", "Bearer revocado", http.StatusUnauthorized, kernel.Actor{}},
		{"otro esquema", "Basic bueno", http.StatusUnauthorized, kernel.Actor{}},
		{"error técnico", "Bearer roto", http.StatusInternalServerError, kernel.Actor{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got kernel.Actor
			next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = ActorFrom(r.Context())
			})
			req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			rec := httptest.NewRecorder()
			Authenticate(fakeAuthenticator{})(next).ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Errorf("status = %d", rec.Code)
			}
			if got != tt.actor {
				t.Errorf("actor = %+v", got)
			}
		})
	}
}

func TestWriteErrorForbidden(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteError(rec, kernel.Forbidden("no"))

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d", rec.Code)
	}
}
