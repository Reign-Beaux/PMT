package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
)

// ActAs hace que toda llamada a herramientas del servidor actúe como actor. Es para
// pruebas en memoria, donde no hay cabecera de la que autenticar.
func ActAs(server *sdk.Server, actor kernel.Actor) {
	server.AddReceivingMiddleware(func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			if call, ok := req.(*sdk.ServerRequest[*sdk.CallToolParamsRaw]); ok {
				if call.Extra == nil {
					call.Extra = &sdk.RequestExtra{}
				}
				call.Extra.TokenInfo = tokenInfo(actor)
			}
			return next(ctx, method, req)
		}
	})
}
