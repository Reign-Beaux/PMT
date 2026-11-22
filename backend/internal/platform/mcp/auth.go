package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
)

const actorKey = "actor"

// ProfileParam es el parámetro de la URL con el que una sesión MCP declara su
// perfil (/mcp?profile=tl): así lo configuran los clientes sin cabeceras extra.
const ProfileParam = "profile"

// Authenticate protege el endpoint MCP: exige un token de API, de usuario o de
// agente, y toma el perfil que declare ProfileParam. Sin token, o con uno
// inválido, revocado o de un usuario desactivado, responde 401; con un perfil
// desconocido, o distinto del de un token de agente, 400.
func Authenticate(authenticator kernel.Authenticator) func(http.Handler) http.Handler {
	return auth.RequireBearerToken(verifier(authenticator), &auth.RequireBearerTokenOptions{
		AllowMissingExpiration: true,
	})
}

func verifier(authenticator kernel.Authenticator) auth.TokenVerifier {
	return func(ctx context.Context, token string, r *http.Request) (*auth.TokenInfo, error) {
		var param string
		if r != nil {
			param = r.URL.Query().Get(ProfileParam)
		}
		declared, err := kernel.ParseProfile(param)
		if err != nil {
			return nil, badRequest{err}
		}
		actor, err := authenticator.AuthenticateToken(ctx, token, declared)
		var business *kernel.Error
		if errors.As(err, &business) && business.Kind == kernel.KindUnauthorized {
			return nil, fmt.Errorf("%w: %v", auth.ErrInvalidToken, err)
		}
		if errors.As(err, &business) && business.Kind == kernel.KindInvalid {
			return nil, badRequest{err}
		}
		if err != nil {
			return nil, err
		}
		return tokenInfo(actor), nil
	}
}

// badRequest hace que el middleware del SDK responda 400 con el mensaje del error
// tal cual: el SDK responde 400 a todo error que sea auth.ErrOAuth.
type badRequest struct{ err error }

func (e badRequest) Error() string        { return e.err.Error() }
func (e badRequest) Is(target error) bool { return target == auth.ErrOAuth }

func tokenInfo(actor kernel.Actor) *auth.TokenInfo {
	return &auth.TokenInfo{UserID: actor.Name, Extra: map[string]any{actorKey: actor}}
}

// ActorFrom devuelve el Actor de la llamada. Toda llamada autenticada lo tiene;
// sin él devuelve el actor vacío, que no tiene perfil y ninguna regla autoriza.
func ActorFrom(req *sdk.CallToolRequest) kernel.Actor {
	if req == nil || req.Extra == nil || req.Extra.TokenInfo == nil {
		return kernel.Actor{}
	}
	actor, _ := req.Extra.TokenInfo.Extra[actorKey].(kernel.Actor)
	return actor
}
