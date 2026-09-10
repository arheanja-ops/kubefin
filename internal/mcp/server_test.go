package mcp

import (
	"context"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/arheanja-ops/kubefin/internal/toolset"
)

// TestServer_ListsRegisteredTools verifica que el servidor expone las tools del
// registro vía el protocolo MCP, usando transportes en memoria (sin stdio).
func TestServer_ListsRegisteredTools(t *testing.T) {
	ctx := context.Background()

	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: serverName, Version: "test"}, nil)
	reg := toolset.Default()
	deps := toolset.Deps{}
	for _, tool := range reg.List() {
		registerTool(server, tool, deps)
	}

	clientT, serverT := mcpsdk.NewInMemoryTransports()
	if _, err := server.Connect(ctx, serverT, nil); err != nil {
		t.Fatalf("server.Connect: %v", err)
	}

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer session.Close()

	got := map[string]bool{}
	for tool, err := range session.Tools(ctx, nil) {
		if err != nil {
			t.Fatalf("iterando tools: %v", err)
		}
		got[tool.Name] = true
	}

	for _, want := range []string{"eks_report", "cost_by_service", "optimize_compute"} {
		if !got[want] {
			t.Errorf("el servidor MCP debería exponer la tool %q; expuestas: %v", want, got)
		}
	}
}
