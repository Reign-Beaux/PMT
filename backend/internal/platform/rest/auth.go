package rest

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"pmt/internal/kernel"
)

// SessionCookie es la cookie con la sesión del navegador.
const SessionCookie = "pmt_session"

type actorKey struct{}

// Authenticate resuelve el Actor de cada petición y lo deja en su contexto, salvo
// en las rutas que public declara abiertas. Con cabecera Authorization se usa el
// token; sin ella, la cookie de sesión. Sin credenciales, o con credenciales
// inválidas o de un usuario desactivado, responde 401.
func Authenticate(authenticator kernel.Authenticator, public func(*http.Request) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if public != nil && public(r) {
				next.ServeHTTP(w, r)
				return
			}
			var actor kernel.Actor
			var err error
			if _, present := r.Header[http.CanonicalHeaderKey("Authorization")]; present {
				token, ok := BearerToken(r)
				if !ok {
					unauthorized(w, kernel.ErrUnauthenticated)
					return
				}
				actor, err = authenticator.AuthenticateToken(r.Context(), token)
			} else if session := SessionFrom(r); session != "" {
				actor, err = authenticator.AuthenticateSession(r.Context(), session)
			} else {
				unauthorized(w, kernel.Unauthorized("autenticación requerida"))
				return
			}
			if err != nil {
				if isUnauthorized(err) {
					unauthorized(w, err)
					return
				}
				WriteError(w, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithActor(r.Context(), actor)))
		})
	}
}

func unauthorized(w http.ResponseWriter, err error) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	WriteError(w, err)
}

func isUnauthorized(err error) bool {
	var business *kernel.Error
	return errors.As(err, &business) && business.Kind == kernel.KindUnauthorized
}

// WithActor deja el Actor en el contexto.
func WithActor(ctx context.Context, actor kernel.Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, actor)
}

// ActorFrom devuelve el Actor de la petición. Toda ruta protegida lo tiene; sin
// él devuelve el actor vacío, que no tiene perfil y ninguna regla autoriza.
func ActorFrom(ctx context.Context) kernel.Actor {
	actor, _ := ctx.Value(actorKey{}).(kernel.Actor)
	return actor
}

// SessionFrom devuelve el valor de la cookie de sesión, o vacío si no viene.
func SessionFrom(r *http.Request) string {
	cookie, err := r.Cookie(SessionCookie)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// BearerToken devuelve el token de la cabecera Authorization con esquema Bearer.
// Sin cabecera, con otro esquema o con el token vacío devuelve false.
func BearerToken(r *http.Request) (string, bool) {
	fields := strings.Fields(r.Header.Get("Authorization"))
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") || fields[1] == "" {
		return "", false
	}
	return fields[1], true
}

// SetSessionCookie escribe la cookie de sesión. Con maxAge negativo la borra.
func SetSessionCookie(w http.ResponseWriter, value string, maxAge int, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name: SessionCookie, Value: value, Path: "/", MaxAge: maxAge,
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
	})
}
