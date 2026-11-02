package rest

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"pmt/internal/kernel"
)

type actorKey struct{}

// Authenticate resuelve el Actor de cada petición y lo deja en su contexto. Un
// token inválido o revocado responde 401. Una petición sin token es del usuario,
// kernel.Owner.
func Authenticate(authenticator kernel.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := BearerToken(r)
			if token == "" {
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
			next.ServeHTTP(w, r.WithContext(WithActor(r.Context(), actor)))
		})
	}
}

// WithActor deja el Actor en el contexto.
func WithActor(ctx context.Context, actor kernel.Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, actor)
}

// ActorFrom devuelve el Actor de la petición, o kernel.Owner si no lo tiene.
func ActorFrom(ctx context.Context) kernel.Actor {
	if actor, ok := ctx.Value(actorKey{}).(kernel.Actor); ok {
		return actor
	}
	return kernel.Owner
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
