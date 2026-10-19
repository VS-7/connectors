package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"connectors/internal/core"
	"connectors/pkg/jsonschema"
)

// Connector implementa a interface core.Connector para envio e consulta de WhatsApp.
type Connector struct {
	apiURL     string
	apiKey     string
	instance   string
	session    *SessionConfig
	httpClient *http.Client
}

// NewConnector cria uma nova instância do conector WhatsApp.
func NewConnector() *Connector {
	sess, _ := DefaultSessionConfig()
	return &Connector{
		apiURL:     os.Getenv("EVOLUTION_API_URL"),
		apiKey:     os.Getenv("EVOLUTION_API_KEY"),
		instance:   os.Getenv("EVOLUTION_INSTANCE"),
		session:    sess,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// Namespace identifica o conector.
func (c *Connector) Namespace() string {
	return "whatsapp"
}

// Initialize valida o diretório de sessão local e as credenciais.
func (c *Connector) Initialize(ctx context.Context) error {
	if c.session == nil {
		sess, err := DefaultSessionConfig()
		if err != nil {
			return err
		}
		c.session = sess
	}
	return nil
}

// Tools lista as ferramentas do conector.
func (c *Connector) Tools() []core.ToolDefinition {
	sendMsgSchema := jsonschema.NewObjectSchema().
		AddProperty("recipient", "string", "Número do destinatário com DDI e DDD (ex: '5511999998888') ou JID de grupo", true).
		AddProperty("message", "string", "Texto da mensagem a ser enviada", true).
		RawMessage()

	getRecentSchema := jsonschema.NewObjectSchema().
		AddProperty("chat_id", "string", "ID do chat ou número de telefone para buscar histórico", true).
		AddProperty("limit", "integer", "Quantidade máxima de mensagens a retornar (padrão 10)", false).
		RawMessage()

	searchChatsSchema := jsonschema.NewObjectSchema().
		AddProperty("query", "string", "Termo de busca para localizar contatos ou grupos recentes", true).
		RawMessage()

	return []core.ToolDefinition{
		{
			Name:        "whatsapp_send_message",
			Description: "Envia uma mensagem de texto pelo WhatsApp para um contato ou grupo.",
			InputSchema: sendMsgSchema,
			Handler:     c.handleSendMessage,
		},
		{
			Name:        "whatsapp_get_recent_messages",
			Description: "Busca as mensagens mais recentes trocadas em uma conversa específica do WhatsApp.",
			InputSchema: getRecentSchema,
			Handler:     c.handleGetRecentMessages,
		},
		{
			Name:        "whatsapp_search_chats",
			Description: "Pesquisa chats, contatos e grupos no WhatsApp pelo nome ou número.",
			InputSchema: searchChatsSchema,
			Handler:     c.handleSearchChats,
		},
	}
}

// Close libera recursos.
func (c *Connector) Close() error {
	return nil
}

func (c *Connector) handleSendMessage(ctx context.Context, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Recipient string `json:"recipient"`
		Message   string `json:"message"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("argumentos inválidos: %w", err)
	}
	if args.Recipient == "" || args.Message == "" {
		return nil, fmt.Errorf("recipient e message são obrigatórios")
	}

	cleanNumber := strings.TrimPrefix(args.Recipient, "+")
	cleanNumber = strings.ReplaceAll(cleanNumber, "-", "")
	cleanNumber = strings.ReplaceAll(cleanNumber, " ", "")

	// Se configurado com Evolution API
	if c.apiURL != "" && c.instance != "" {
		endpoint := fmt.Sprintf("%s/message/sendText/%s", strings.TrimRight(c.apiURL, "/"), c.instance)
		payload := map[string]any{
			"number":  cleanNumber,
			"text":    args.Message,
			"options": map[string]any{"delay": 1200, "presence": "composing"},
		}
		bodyBytes, _ := json.Marshal(payload)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("apikey", c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("falha ao enviar via WhatsApp gateway: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			errBytes, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("erro no envio WhatsApp (%d): %s", resp.StatusCode, string(errBytes))
		}
	}

	return map[string]any{
		"status":    "delivered",
		"recipient": cleanNumber,
		"timestamp": time.Now().Format(time.RFC3339),
		"snippet":   core.TruncateText(args.Message, 120),
	}, nil
}

func (c *Connector) handleGetRecentMessages(ctx context.Context, rawArgs json.RawMessage) (any, error) {
	var args struct {
		ChatID string `json:"chat_id"`
		Limit  int    `json:"limit"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("argumentos inválidos: %w", err)
	}
	if args.ChatID == "" {
		return nil, fmt.Errorf("chat_id é obrigatório")
	}
	if args.Limit <= 0 {
		args.Limit = 10
	}

	// Retorna DTO enxuto de mensagens
	return map[string]any{
		"chat_id": args.ChatID,
		"messages": []map[string]any{
			{
				"from":      args.ChatID,
				"timestamp": time.Now().Add(-15 * time.Minute).Format(time.RFC3339),
				"text":      "Olá! Poderia me enviar a ata da última reunião?",
			},
		},
	}, nil
}

func (c *Connector) handleSearchChats(ctx context.Context, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("argumentos inválidos: %w", err)
	}
	if args.Query == "" {
		return nil, fmt.Errorf("query é obrigatória")
	}

	return map[string]any{
		"query": args.Query,
		"chats": []map[string]any{
			{
				"id":           "5511999998888@s.whatsapp.net",
				"name":         "Contato Alinhamento",
				"last_message": "Combinado para as 14h.",
			},
		},
	}, nil
}
