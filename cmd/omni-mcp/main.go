package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	mcpbridge "connectors/internal/bridge/mcp"
	"connectors/internal/connectors/fathom"
	"connectors/internal/connectors/google"
	"connectors/internal/connectors/mattermost"
	"connectors/internal/connectors/slack"
	"connectors/internal/connectors/whatsapp"
	"connectors/internal/core"
)

func main() {
	// REGRA DE OURO RNF001: Logs NUNCA no stdout (canal exclusivo do JSON-RPC via stdio)
	logHandler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(logHandler))

	enableFlag := flag.String("enable", "", "Lista de conectores a habilitar separados por vírgula (ex: 'fathom,slack') ou 'all'")
	transportFlag := flag.String("transport", "stdio", "Tipo de transporte MCP: 'stdio' ou 'sse'")
	addrFlag := flag.String("addr", ":8765", "Endereço de escuta para transporte SSE (ex: ':8765')")
	debugFlag := flag.Bool("debug", false, "Habilita logs em nível DEBUG no stderr")
	flag.Parse()

	if *debugFlag {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})))
	}

	enabledStr := *enableFlag
	if enabledStr == "" {
		enabledStr = os.Getenv("ENABLED_CONNECTORS")
	}
	var enabledList []string
	if enabledStr != "" && enabledStr != "all" && enabledStr != "*" {
		for _, part := range strings.Split(enabledStr, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				enabledList = append(enabledList, trimmed)
			}
		}
	}

	// Registro de todos os conectores disponíveis
	registry := core.NewRegistry()
	registry.Register(fathom.NewConnector(""))
	registry.Register(whatsapp.NewConnector())
	registry.Register(slack.NewConnector(""))
	registry.Register(mattermost.NewConnector())
	registry.Register(google.NewConnector())

	// Filtragem seletiva por flags ou variável de ambiente
	activeConnectors, err := registry.FilterEnabled(enabledList)
	if err != nil {
		slog.Error("erro na seleção de conectores", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	slog.Info("inicializando módulos habilitados...", "count", len(activeConnectors))
	for _, c := range activeConnectors {
		if err := c.Initialize(ctx); err != nil {
			slog.Warn("conector inicializado com aviso de configuração", "namespace", c.Namespace(), "warning", err)
		} else {
			slog.Info("conector pronto", "namespace", c.Namespace(), "tools", len(c.Tools()))
		}
	}

	// Constrói o servidor MCP registrando todos os esquemas Draft 7
	mcpServer := mcpbridge.BuildMCPServer(activeConnectors)

	if strings.ToLower(*transportFlag) == "sse" {
		fmt.Fprintf(os.Stderr, "Omni MCP Super-Server rodando via SSE em %s\n", *addrFlag)
		if err := mcpServer.ServeSSE(*addrFlag); err != nil {
			slog.Error("erro na execução do servidor SSE", "error", err)
			os.Exit(1)
		}
	} else {
		if err := mcpServer.ServeStdio(); err != nil {
			slog.Error("erro na execução do servidor stdio", "error", err)
			os.Exit(1)
		}
	}
}
