package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectors/internal/core"
)

// FunctionDefinition representa o formato de função no padrão OpenAI Function Calling.
type FunctionDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ToolItem descreve o container {"type": "function", "function": {...}} da OpenAI.
type ToolItem struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// ToolsListResponse é o payload retornado no endpoint GET /v1/tools.
type ToolsListResponse struct {
	Tools []ToolItem `json:"tools"`
}

// ExecuteRequest é o payload recebido no endpoint POST /v1/tools/execute.
type ExecuteRequest struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ExecuteResponse é o payload retornado após a execução de uma tool.
type ExecuteResponse struct {
	Tool   string `json:"tool"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// Gateway é o servidor HTTP REST que traduz chamadas de Function Calling para core.Connector.
type Gateway struct {
	connectors []core.Connector
	toolsMap   map[string]core.ToolDefinition
	mux        *http.ServeMux
}

// NewGateway instancia o gateway HTTP e indexa em memória os handlers das ferramentas.
func NewGateway(connectors []core.Connector) *Gateway {
	toolsMap := make(map[string]core.ToolDefinition)
	for _, conn := range connectors {
		for _, tool := range conn.Tools() {
			toolsMap[tool.Name] = tool
		}
	}

	g := &Gateway{
		connectors: connectors,
		toolsMap:   toolsMap,
		mux:        http.NewServeMux(),
	}

	g.registerRoutes()
	return g
}

// Handler retorna o http.Handler configurado com todas as rotas e middlewares.
func (g *Gateway) Handler() http.Handler {
	return g.corsMiddleware(g.mux)
}

func (g *Gateway) registerRoutes() {
	g.mux.HandleFunc("GET /v1/tools", g.handleListTools)
	g.mux.HandleFunc("POST /v1/tools/execute", g.handleExecuteTool)
	g.mux.HandleFunc("GET /v1/health", g.handleHealth)
}

func (g *Gateway) handleListTools(w http.ResponseWriter, r *http.Request) {
	toolsList := make([]ToolItem, 0, len(g.toolsMap))

	for _, t := range g.toolsMap {
		toolsList = append(toolsList, ToolItem{
			Type: "function",
			Function: FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ToolsListResponse{Tools: toolsList})
}

func (g *Gateway) handleExecuteTool(w http.ResponseWriter, r *http.Request) {
	var req ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("payload JSON inválido: %v", err))
		return
	}

	if req.Name == "" {
		g.writeJSONError(w, http.StatusBadRequest, "campo 'name' da ferramenta é obrigatório")
		return
	}

	tool, exists := g.toolsMap[req.Name]
	if !exists {
		g.writeJSONError(w, http.StatusNotFound, fmt.Sprintf("ferramenta '%s' não encontrada no catálogo", req.Name))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	slog.Debug("executando tool via openai gateway", "name", req.Name)

	result, err := tool.Handler(ctx, req.Arguments)
	if err != nil {
		slog.Warn("erro ao executar tool via openai gateway", "name", req.Name, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(ExecuteResponse{
			Tool:  req.Name,
			Error: err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ExecuteResponse{
		Tool:   req.Name,
		Result: result,
	})
}

func (g *Gateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	enabled := make([]string, 0, len(g.connectors))
	for _, c := range g.connectors {
		enabled = append(enabled, c.Namespace())
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":             "ok",
		"version":            "1.0.0",
		"enabled_connectors": enabled,
		"total_tools":        len(g.toolsMap),
	})
}

func (g *Gateway) writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (g *Gateway) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Api-Key")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
