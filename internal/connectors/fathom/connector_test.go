package fathom_test

import (
	"context"
	"encoding/json"
	"testing"

	"connectors/internal/connectors/fathom"
	"connectors/internal/core"
)


func TestFathomConnector_Tools(t *testing.T) {
	conn := fathom.NewConnector("mock_api_key")
	if conn.Namespace() != "fathom" {
		t.Errorf("esperava namespace 'fathom', obteve '%s'", conn.Namespace())
	}

	ctx := context.Background()
	if err := conn.Initialize(ctx); err != nil {
		t.Fatalf("erro ao inicializar: %v", err)
	}

	tools := conn.Tools()
	if len(tools) != 3 {
		t.Fatalf("esperava 3 tools para fathom, obteve %d", len(tools))
	}

	var foundTranscript bool
	for _, tool := range tools {
		if tool.Name == "fathom_get_call_transcript" {
			foundTranscript = true
			if len(tool.InputSchema) == 0 {
				t.Errorf("InputSchema vazio para %s", tool.Name)
			}
		}
	}

	if !foundTranscript {
		t.Errorf("ferramenta fathom_get_call_transcript não encontrada")
	}
}

func TestFathomConnector_Validation(t *testing.T) {
	conn := fathom.NewConnector("")
	ctx := context.Background()
	// Sem API Key deve retornar erro amigável
	if err := conn.Initialize(ctx); err == nil {
		t.Errorf("esperava erro de inicialização sem chave de API")
	}

	tools := conn.Tools()
	var getTranscriptTool core.ToolDefinition
	for _, tool := range tools {
		if tool.Name == "fathom_get_call_transcript" {
			getTranscriptTool = tool
			break
		}
	}

	// Chamada sem recording_id deve falhar graciosamente
	_, err := getTranscriptTool.Handler(ctx, json.RawMessage(`{}`))
	if err == nil {
		t.Errorf("esperava erro com recording_id vazio")
	}
}

