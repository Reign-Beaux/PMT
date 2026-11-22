package mcp

import (
	"context"
	"encoding/json"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"pmt/internal/kernel"
)

// ToolCheck revisa una llamada a herramienta antes de que llegue a su handler:
// recibe el actor, el nombre de la herramienta y sus argumentos tal como
// llegaron. Un error la rechaza con el mismo formato que los de las herramientas.
type ToolCheck func(ctx context.Context, actor kernel.Actor, tool string, arguments json.RawMessage) error

// Guard hace que toda llamada a herramientas pase antes por check. Se agrega
// después de registrar las herramientas y antes que ActAs en las pruebas, para
// que ActAs corra primero y deje el actor.
func Guard(server *sdk.Server, check ToolCheck) {
	server.AddReceivingMiddleware(func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			call, ok := req.(*sdk.ServerRequest[*sdk.CallToolParamsRaw])
			if !ok || call.Params == nil {
				return next(ctx, method, req)
			}
			if err := check(ctx, actorFromExtra(call.Extra), call.Params.Name, call.Params.Arguments); err != nil {
				result := &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: ToolError(err).Error()}}}
				result.IsError = true
				return result, nil
			}
			return next(ctx, method, req)
		}
	})
}
