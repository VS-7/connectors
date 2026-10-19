package core

import (
	"context"
	"encoding/json"
)

// ToolHandler é a assinatura padrão para execução de qualquer ferramenta.
type ToolHandler func(ctx context.Context, args json.RawMessage) (any, error)

// ToolDefinition descreve uma ferramenta de forma agnóstica a qualquer LLM ou protocolo de transporte.
type ToolDefinition struct {
	Name        string          `json:"name"`        // Ex: "fathom_get_call_transcript", "slack_send_message"
	Description string          `json:"description"` // Explicação clara de propósito e quando acionar
	InputSchema json.RawMessage `json:"input_schema"`// JSON Schema Draft 7 dos parâmetros de entrada
	Handler     ToolHandler     `json:"-"`           // Função que executa a chamada real
}

// Connector é a porta universal (Port) que todo serviço externo deve implementar.
type Connector interface {
	// Namespace define o prefixo de identificação do conector (ex: "fathom", "whatsapp", "slack")
	Namespace() string

	// Initialize executa o setup inicial de autenticação, validação de chaves ou conexão de socket
	Initialize(ctx context.Context) error

	// Tools retorna a lista de ferramentas executáveis disponibilizadas pelo conector
	Tools() []ToolDefinition

	// Close encerra graciosamente conexões ativas, pools de conexões e libera recursos
	Close() error
}
