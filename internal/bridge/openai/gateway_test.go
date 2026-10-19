package openai_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	openaibridge "connectors/internal/bridge/openai"
	"connectors/internal/core"
)

type mockConnector struct{}

func (m *mockConnector) Namespace() string { return "mock" }
func (m *mockConnector) Initialize(ctx context.Context) error { return nil }
func (m *mockConnector) Tools() []core.ToolDefinition {
	return []core.ToolDefinition{
		{
			Name:        "mock_echo",
			Description: "Retorna o eco do texto",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"msg":{"type":"string"}}}`),
			Handler: func(ctx context.Context, args json.RawMessage) (any, error) {
				var p struct {
					Msg string `json:"msg"`
				}
				_ = json.Unmarshal(args, &p)
				return map[string]string{"reply": p.Msg}, nil
			},
		},
	}
}
func (m *mockConnector) Close() error { return nil }

func TestOpenAIGateway(t *testing.T) {
	conn := &mockConnector{}
	gw := openaibridge.NewGateway([]core.Connector{conn})
	server := httptest.NewServer(gw.Handler())
	defer server.Close()

	// 1. Testa GET /v1/health
	resHealth, err := http.Get(server.URL + "/v1/health")
	if err != nil {
		t.Fatalf("erro em GET /v1/health: %v", err)
	}
	if resHealth.StatusCode != http.StatusOK {
		t.Fatalf("esperava status 200 em health, obteve %d", resHealth.StatusCode)
	}

	// 2. Testa GET /v1/tools
	resTools, err := http.Get(server.URL + "/v1/tools")
	if err != nil {
		t.Fatalf("erro em GET /v1/tools: %v", err)
	}
	var toolsResp openaibridge.ToolsListResponse
	if err := json.NewDecoder(resTools.Body).Decode(&toolsResp); err != nil {
		t.Fatalf("erro ao decodificar /v1/tools: %v", err)
	}
	if len(toolsResp.Tools) != 1 || toolsResp.Tools[0].Function.Name != "mock_echo" {
		t.Fatalf("esperava ferramenta mock_echo em /v1/tools")
	}

	// 3. Testa POST /v1/tools/execute
	reqPayload := openaibridge.ExecuteRequest{
		Name:      "mock_echo",
		Arguments: json.RawMessage(`{"msg":"Olá Mundo"}`),
	}
	bodyBytes, _ := json.Marshal(reqPayload)

	resExec, err := http.Post(server.URL+"/v1/tools/execute", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("erro em POST /v1/tools/execute: %v", err)
	}
	if resExec.StatusCode != http.StatusOK {
		t.Fatalf("esperava status 200 em execute, obteve %d", resExec.StatusCode)
	}

	var execResp openaibridge.ExecuteResponse
	if err := json.NewDecoder(resExec.Body).Decode(&execResp); err != nil {
		t.Fatalf("erro ao decodificar execute response: %v", err)
	}
	if execResp.Tool != "mock_echo" {
		t.Errorf("esperava tool=mock_echo, obteve %s", execResp.Tool)
	}
}
