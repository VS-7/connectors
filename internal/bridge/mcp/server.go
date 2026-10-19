package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"connectors/internal/core"

	mcpprotocol "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ServerWrapper encapsula o MCPServer com metadados dos conectores ativos.
type ServerWrapper struct {
	MCPServer  *server.MCPServer
	Connectors []core.Connector
	ToolCount  int
}

// BuildMCPServer itera sobre a lista de core.Connector e registra nativamente
// as ferramentas no servidor MCP com seus esquemas Draft 7.
func BuildMCPServer(connectors []core.Connector) *ServerWrapper {
	s := server.NewMCPServer(
		"Omni-Connector-Hub",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	totalTools := 0

	for _, conn := range connectors {
		namespace := conn.Namespace()
		for _, tool := range conn.Tools() {
			t := tool // captura de variável de escopo para closure
			totalTools++

			mcpTool := mcpprotocol.NewToolWithRawSchema(t.Name, t.Description, t.InputSchema)

			s.AddTool(mcpTool, func(ctx context.Context, req mcpprotocol.CallToolRequest) (*mcpprotocol.CallToolResult, error) {
				argsBytes, err := json.Marshal(req.Params.Arguments)
				if err != nil {
					return mcpprotocol.NewToolResultError(fmt.Sprintf("erro ao serializar argumentos: %v", err)), nil
				}

				slog.Debug("executando tool mcp", "name", t.Name, "namespace", namespace)

				result, err := t.Handler(ctx, argsBytes)
				if err != nil {
					slog.Warn("falha na execução de tool mcp", "name", t.Name, "error", err)
					return mcpprotocol.NewToolResultError(err.Error()), nil
				}

				outBytes, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return mcpprotocol.NewToolResultError(fmt.Sprintf("erro ao serializar resposta: %v", err)), nil
				}

				return mcpprotocol.NewToolResultText(string(outBytes)), nil
			})
		}
	}

	return &ServerWrapper{
		MCPServer:  s,
		Connectors: connectors,
		ToolCount:  totalTools,
	}
}

// ServeStdio inicia o servidor MCP escutando JSON-RPC em stdin/stdout.
func (w *ServerWrapper) ServeStdio() error {
	slog.Info("iniciando Omni MCP Super-Server via stdio", "tools_count", w.ToolCount)
	return server.ServeStdio(w.MCPServer)
}

// ServeSSE inicia o servidor MCP via Server-Sent Events (SSE) na porta/endereço informado.
func (w *ServerWrapper) ServeSSE(addr string) error {
	slog.Info("iniciando Omni MCP Server via SSE", "addr", addr, "tools_count", w.ToolCount)
	sseServer := server.NewSSEServer(w.MCPServer)
	return sseServer.Start(addr)
}
