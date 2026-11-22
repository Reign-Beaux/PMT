package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
)

// InstructionsOf da las instrucciones de una sesión según su actor, o false si no
// lleva.
type InstructionsOf func(kernel.Actor) (string, bool)

// Instructions hace que cada sesión reciba en initialize (y en server/discover)
// sus instrucciones. El SDK fija las instrucciones por servidor; con este
// middleware un solo servidor, con un solo registro de herramientas, las da por
// sesión según el actor que dejó la autenticación. Una sesión sin instrucciones
// recibe las del servidor sin cambios.
func Instructions(server *sdk.Server, of InstructionsOf) {
	server.AddReceivingMiddleware(func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			result, err := next(ctx, method, req)
			if err != nil {
				return result, err
			}
			switch r := result.(type) {
			case *sdk.InitializeResult:
				if text, ok := of(actorFromExtra(req.GetExtra())); ok {
					r.Instructions = text
				}
			case *sdk.DiscoverResult:
				if text, ok := of(actorFromExtra(req.GetExtra())); ok {
					r.Instructions = text
				}
			}
			return result, nil
		}
	})
}
