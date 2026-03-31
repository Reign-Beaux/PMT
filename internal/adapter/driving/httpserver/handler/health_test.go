package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name           string
		expectedStatus int
		expectedBody   string
		expectedCT     string
	}{
		{
			name:           "should return 200 and ok status",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"ok"}`,
			expectedCT:     "application/json",
		},
	}

	handler := NewHealthHandler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()

			handler.Handle(rec, req)

			res := rec.Result()

			if res.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, res.StatusCode)
			}

			if rec.Body.String() != tt.expectedBody {
				t.Errorf("expected body %s, got %s", tt.expectedBody, rec.Body.String())
			}

			if res.Header.Get("Content-Type") != tt.expectedCT {
				t.Errorf("expected content-type %s, got %s", tt.expectedCT, res.Header.Get("Content-Type"))
			}
		})
	}
}
