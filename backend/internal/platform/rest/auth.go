package rest

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"pmt/internal/kernel"
)

type actorKey struct{}

// Authenticate resuelve el Actor de cada petición y lo deja en su contexto. Un
// token inválido o revocado responde 401. Una petición sin token continúa como
// kernel.Anonymous, solo durante la transición a los tokens.
func Authenticate(authenticator kernel.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := BearerToken(r)
			if token == "" {
				log.Printf("petición sin token: %s %s", r.Method, r.URL.Path)
				next.ServeHTTP(w, r)
				return
			}

			actor, err := authenticator.Authenticate(r.Context(), token)
			if errors.Is(err, kernel.ErrUnauthenticated) {
				w.Header().Set("WWW-Authenticate", "Bearer")
				writeMessage(w, http.StatusUnauthorized, err.Error())
				return
			}
			if err != nil {
				WriteError(w, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorKey{}, actor)))
		})
	}
}

// ActorFrom devuelve el Actor de la petición, o kernel.Anonymous si no lo tiene.
func ActorFrom(ctx context.Context) kernel.Actor {
	if actor, ok := ctx.Value(actorKey{}).(kernel.Actor); ok {
		return actor
	}
	return kernel.Anonymous
}

// BearerToken devuelve el token de la cabecera Authorization. Sin cabecera, o con
// «Bearer» vacío, devuelve vacío: así llega ${PMT_TOKEN:-} sin la variable. Otro
// esquema se devuelve completo para que se rechace como token inválido.
func BearerToken(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(header) >= len("Bearer") && strings.EqualFold(header[:len("Bearer")], "Bearer") {
		return strings.TrimSpace(header[len("Bearer"):])
	}
	return header
}
