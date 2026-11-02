package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"pmt/internal/kernel"
	"pmt/internal/platform/mcp"
)

type rejectAll struct{}

func (rejectAll) Authenticate(context.Context, string) (kernel.Actor, error) {
	return kernel.Actor{}, kernel.ErrUnauthenticated
}

func TestRoutesRejectInvalidToken(t *testing.T) {
	handler := routes(http.NewServeMux(), mcp.NewServer(), rejectAll{})

	for _, path := range []string{"/api/projects", "/mcp"} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		req.Header.Set("Authorization", "Bearer malo")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: status = %d", path, rec.Code)
		}
	}
}
