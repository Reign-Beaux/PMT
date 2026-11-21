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

// Authenticate protege el endpoint MCP: exige un token de API, de usuario o de
// agente. Sin token, o con uno inválido, revocado o de un usuario desactivado,
// responde 401.
func Authenticate(authenticator kernel.Authenticator) func(http.Handler) http.Handler {
	return auth.RequireBearerToken(verifier(authenticator), &auth.RequireBearerTokenOptions{
		AllowMissingExpiration: true,
	})
}

func verifier(authenticator kernel.Authenticator) auth.TokenVerifier {
	return func(ctx context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
		actor, err := authenticator.AuthenticateToken(ctx, token)
		var business *kernel.Error
		if errors.As(err, &business) && business.Kind == kernel.KindUnauthorized {
			return nil, fmt.Errorf("%w: %v", auth.ErrInvalidToken, err)
		}
		if err != nil {
			return nil, err
		}
		return tokenInfo(actor), nil
	}
}

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
