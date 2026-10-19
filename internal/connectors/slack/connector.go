package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"connectors/internal/core"
	"connectors/pkg/jsonschema"

	slackapi "github.com/slack-go/slack"
)

// Connector implementa core.Connector para Slack.
type Connector struct {
	token  string
	client *slackapi.Client
}

// NewConnector instancia o conector Slack a partir do token informado ou variáveis de ambiente.
func NewConnector(token string) *Connector {
	if token == "" {
		token = os.Getenv("SLACK_BOT_TOKEN")
		if token == "" {
			token = os.Getenv("SLACK_USER_TOKEN")
		}
	}
	var client *slackapi.Client
	if token != "" {
		client = slackapi.New(token)
	}
	return &Connector{
		token:  token,
		client: client,
	}
}

// Namespace identifica o conector.
func (c *Connector) Namespace() string {
	return "slack"
}

// Initialize valida a presença do token de acesso do Slack.
func (c *Connector) Initialize(ctx context.Context) error {
	if c.token == "" {
		return fmt.Errorf("SLACK_BOT_TOKEN ou SLACK_USER_TOKEN não configurados")
	}
	if c.client == nil {
		c.client = slackapi.New(c.token)
	}
	return nil
}

// Tools lista as ferramentas do conector Slack.
func (c *Connector) Tools() []core.ToolDefinition {
	sendMsgSchema := jsonschema.NewObjectSchema().
		AddProperty("channel", "string", "Nome do canal (ex: '#geral') ou ID do canal/conversa (ex: 'C01234567')", true).
		AddProperty("message", "string", "Texto da mensagem a postar", true).
		AddProperty("thread_ts", "string", "Timestamp do thread para responder em linha (opcional)", false).
		RawMessage()

	historySchema := jsonschema.NewObjectSchema().
		AddProperty("channel_id", "string", "ID do canal para leitura do histórico", true).
		AddProperty("limit", "integer", "Quantidade máxima de mensagens a retornar (padrão 15)", false).
		RawMessage()

	searchSchema := jsonschema.NewObjectSchema().
		AddProperty("query", "string", "Termo ou filtro de busca no Slack (ex: 'reunião from:@vitor')", true).
		AddProperty("count", "integer", "Limite de resultados (padrão 10)", false).
		RawMessage()

	listChannelsSchema := jsonschema.NewObjectSchema().
		AddProperty("types", "string", "Tipos de canal: 'public_channel,private_channel' (opcional)", false).
		RawMessage()

	return []core.ToolDefinition{
		{
			Name:        "slack_send_message",
			Description: "Publica uma mensagem em um canal ou conversa no Slack.",
			InputSchema: sendMsgSchema,
			Handler:     c.handleSendMessage,
		},
		{
			Name:        "slack_read_channel_history",
			Description: "Lê as mensagens mais recentes de um canal do Slack, formatadas de forma enxuta.",
			InputSchema: historySchema,
			Handler:     c.handleReadHistory,
		},
		{
			Name:        "slack_search_messages",
			Description: "Pesquisa mensagens no workspace do Slack utilizando queries de texto.",
			InputSchema: searchSchema,
			Handler:     c.handleSearchMessages,
		},
		{
			Name:        "slack_list_channels",
			Description: "Lista canais disponíveis no workspace do Slack com seus IDs e nomes.",
			InputSchema: listChannelsSchema,
			Handler:     c.handleListChannels,
		},
	}
}

// Close encerra recursos.
func (c *Connector) Close() error {
	return nil
}

func (c *Connector) handleSendMessage(ctx context.Context, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Channel  string `json:"channel"`
		Message  string `json:"message"`
		ThreadTS string `json:"thread_ts"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("argumentos inválidos: %w", err)
	}
	if args.Channel == "" || args.Message == "" {
		return nil, fmt.Errorf("channel e message são obrigatórios")
	}

	opts := []slackapi.MsgOption{
		slackapi.MsgOptionText(args.Message, false),
	}
	if args.ThreadTS != "" {
		opts = append(opts, slackapi.MsgOptionTS(args.ThreadTS))
	}

	channelID, timestamp, err := c.client.PostMessageContext(ctx, args.Channel, opts...)
	if err != nil {
		return nil, fmt.Errorf("falha ao postar no Slack: %w", err)
	}

	return map[string]any{
		"status":     "sent",
		"channel_id": channelID,
		"timestamp":  timestamp,
		"snippet":    core.TruncateText(args.Message, 100),
	}, nil
}

func (c *Connector) handleReadHistory(ctx context.Context, rawArgs json.RawMessage) (any, error) {
	var args struct {
		ChannelID string `json:"channel_id"`
		Limit     int    `json:"limit"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("argumentos inválidos: %w", err)
	}
	if args.ChannelID == "" {
		return nil, fmt.Errorf("channel_id é obrigatório")
	}
	if args.Limit <= 0 {
		args.Limit = 15
	}

	params := &slackapi.GetConversationHistoryParameters{
		ChannelID: args.ChannelID,
		Limit:     args.Limit,
	}

	history, err := c.client.GetConversationHistoryContext(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar histórico do Slack: %w", err)
	}

	type MessageDTO struct {
		User      string `json:"user"`
		Timestamp string `json:"timestamp"`
		Text      string `json:"text"`
	}

	messages := make([]MessageDTO, 0, len(history.Messages))
	for _, m := range history.Messages {
		messages = append(messages, MessageDTO{
			User:      m.User,
			Timestamp: m.Timestamp,
			Text:      core.TruncateText(m.Text, 500),
		})
	}

	return map[string]any{
		"channel_id": args.ChannelID,
		"count":      len(messages),
		"messages":   messages,
	}, nil
}

func (c *Connector) handleSearchMessages(ctx context.Context, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Query string `json:"query"`
		Count int    `json:"count"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("argumentos inválidos: %w", err)
	}
	if args.Query == "" {
		return nil, fmt.Errorf("query é obrigatória")
	}
	if args.Count <= 0 {
		args.Count = 10
	}

	params := slackapi.NewSearchParameters()
	params.Count = args.Count

	searchRes, err := c.client.SearchMessagesContext(ctx, args.Query, params)
	if err != nil {
		return nil, fmt.Errorf("falha na busca do Slack: %w", err)
	}

	type SearchMatchDTO struct {
		Channel   string `json:"channel"`
		Username  string `json:"username"`
		Timestamp string `json:"timestamp"`
		Text      string `json:"text"`
		Permalink string `json:"permalink"`
	}

	matches := make([]SearchMatchDTO, 0, len(searchRes.Matches))
	for _, m := range searchRes.Matches {
		matches = append(matches, SearchMatchDTO{
			Channel:   m.Channel.Name,
			Username:  m.Username,
			Timestamp: m.Timestamp,
			Text:      core.TruncateText(m.Text, 300),
			Permalink: m.Permalink,
		})
	}

	return map[string]any{
		"query":   args.Query,
		"matches": matches,
	}, nil
}

func (c *Connector) handleListChannels(ctx context.Context, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Types string `json:"types"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	types := []string{"public_channel", "private_channel"}
	if args.Types != "" {
		types = []string{args.Types}
	}

	channels, _, err := c.client.GetConversationsContext(ctx, &slackapi.GetConversationsParameters{
		Types: types,
		Limit: 50,
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao listar canais do Slack: %w", err)
	}

	type ChannelDTO struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Topic   string `json:"topic,omitempty"`
		Members int    `json:"num_members"`
	}

	results := make([]ChannelDTO, 0, len(channels))
	for _, ch := range channels {
		results = append(results, ChannelDTO{
			ID:      ch.ID,
			Name:    ch.Name,
			Topic:   ch.Topic.Value,
			Members: ch.NumMembers,
		})
	}

	return map[string]any{
		"count":    len(results),
		"channels": results,
	}, nil
}
