// Package mcp implementa el servidor MCP por stdio (SDK oficial de Go) que
// expone las tools core del registro sin duplicar lógica (R4.1–R4.4).
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/arheanja-ops/kubefin/internal/toolset"
)

// version del servidor MCP, alineada con el binario.
const serverName = "kubefin"

// Serve arranca el servidor MCP por stdin/stdout, registrando cada tool del
// registro. Bloquea hasta que el cliente se desconecta.
func Serve(ctx context.Context, version string, reg *toolset.Registry, deps toolset.Deps) error {
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: serverName, Version: version}, nil)

	for _, tool := range reg.List() {
		registerTool(server, tool, deps)
	}

	return server.Run(ctx, &mcpsdk.StdioTransport{})
}

// registerTool adapta una tool del registro a una tool MCP. Los argumentos
// llegan como map[string]any; se re-serializan a JSON para el handler del
// registro, y el resultado se devuelve como StructuredContent.
func registerTool(server *mcpsdk.Server, tool toolset.Tool, deps toolset.Deps) {
	handler := func(ctx context.Context, _ *mcpsdk.CallToolRequest, in map[string]any) (*mcpsdk.CallToolResult, any, error) {
		raw, err := json.Marshal(in)
		if err != nil {
			return errorResult(fmt.Sprintf("argumentos inválidos: %v", err)), nil, nil
		}
		out, err := tool.Handler(ctx, deps, raw)
		if err != nil {
			return errorResult(err.Error()), nil, nil
		}
		return nil, out, nil
	}

	mcpsdk.AddTool(server, &mcpsdk.Tool{
		Name:        tool.Name,
		Description: tool.Description,
	}, handler)
}

// errorResult empaqueta un error en un CallToolResult con IsError=true, según
// el contrato del protocolo (los errores de tool no son errores de transporte).
func errorResult(msg string) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		IsError: true,
		Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: msg}},
	}
}
