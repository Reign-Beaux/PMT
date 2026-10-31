package mcp

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
	"pmt/internal/platform/rest"
)

const actorKey = "actor"

// Authenticate protege el endpoint MCP. Un token inválido o revocado responde 401.
// Una petición sin token continúa como kernel.Anonymous, solo durante la
// transición a los tokens.
func Authenticate(authenticator kernel.Authenticator) func(http.Handler) http.Handler {
	require := auth.RequireBearerToken(verifier(authenticator), &auth.RequireBearerTokenOptions{
		AllowMissingExpiration: true,
	})
	return func(next http.Handler) http.Handler {
		protected := require(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rest.BearerToken(r) == "" {
				log.Printf("petición sin token: %s %s", r.Method, r.URL.Path)
				next.ServeHTTP(w, r)
				return
			}
			protected.ServeHTTP(w, r)
		})
	}
}

func verifier(authenticator kernel.Authenticator) auth.TokenVerifier {
	return func(ctx context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
		actor, err := authenticator.Authenticate(ctx, token)
		if errors.Is(err, kernel.ErrUnauthenticated) {
			return nil, fmt.Errorf("%w: %v", auth.ErrInvalidToken, err)
		}
		if err != nil {
			return nil, err
		}
		return &auth.TokenInfo{UserID: actor.Name, Extra: map[string]any{actorKey: actor}}, nil
	}
}

// ActorFrom devuelve el Actor de la llamada, o kernel.Anonymous si no lo tiene.
func ActorFrom(req *sdk.CallToolRequest) kernel.Actor {
	if req == nil || req.Extra == nil || req.Extra.TokenInfo == nil {
		return kernel.Anonymous
	}
	if actor, ok := req.Extra.TokenInfo.Extra[actorKey].(kernel.Actor); ok {
		return actor
	}
	return kernel.Anonymous
}
