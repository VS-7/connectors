package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	openaibridge "connectors/internal/bridge/openai"
	"connectors/internal/connectors/fathom"
	"connectors/internal/connectors/google"
	"connectors/internal/connectors/mattermost"
	"connectors/internal/connectors/slack"
	"connectors/internal/connectors/whatsapp"
	"connectors/internal/core"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	addrFlag := flag.String("addr", ":8080", "Porta/endereço HTTP para escuta da API REST (ex: ':8080')")
	enableFlag := flag.String("enable", "", "Lista de conectores a habilitar separados por vírgula (ex: 'fathom,slack') ou 'all'")
	flag.Parse()

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

	registry := core.NewRegistry()
	registry.Register(fathom.NewConnector(""))
	registry.Register(whatsapp.NewConnector())
	registry.Register(slack.NewConnector(""))
	registry.Register(mattermost.NewConnector())
	registry.Register(google.NewConnector())

	activeConnectors, err := registry.FilterEnabled(enabledList)
	if err != nil {
		slog.Error("erro ao filtrar conectores", "error", err)
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

	gateway := openaibridge.NewGateway(activeConnectors)

	server := &http.Server{
		Addr:         *addrFlag,
		Handler:      gateway.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 45 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info(fmt.Sprintf("Omni HTTP Gateway rodando em http://localhost%s", *addrFlag))
		slog.Info("Rotas disponíveis: GET /v1/tools | POST /v1/tools/execute | GET /v1/health")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("erro fatal no servidor HTTP", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("encerrando servidor graciosamente...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("erro durante shutdown", "error", err)
	}

	_ = core.CloseConnectors(activeConnectors)
	slog.Info("servidor encerrado com sucesso.")
}
