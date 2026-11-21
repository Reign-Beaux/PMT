// Package health expone las sondas /healthz (proceso vivo) y /readyz (base disponible).
package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"pmt/internal/platform/rest"
)

// Pinger comprueba que la base responda.
type Pinger interface{ Ping(context.Context) error }

// Live responde 200 mientras el proceso atienda peticiones.
func Live(w http.ResponseWriter, _ *http.Request) {
	rest.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ready responde 200 si la base responde y 503 si no.
func Ready(pinger Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pinger.Ping(ctx); err != nil {
			slog.Warn("base de datos no disponible", "error", err)
			rest.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}
		rest.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
