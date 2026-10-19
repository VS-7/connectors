package mattermost

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

// Connector implementa core.Connector para Mattermost Client4 API.
type Connector struct {
	serverURL  string
	token      string
	httpClient *http.Client
}

// NewConnector instancia o conector Mattermost.
func NewConnector() *Connector {
	return &Connector{
		serverURL:  os.Getenv("MATTERMOST_URL"),
		token:      os.Getenv("MATTERMOST_TOKEN"),
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

// Namespace identifica o conector.
func (c *Connector) Namespace() string {
	return "mattermost"
}

// Initialize valida as configurações do servidor e token.
func (c *Connector) Initialize(ctx context.Context) error {
	if c.serverURL == "" || c.token == "" {
		return fmt.Errorf("MATTERMOST_URL ou MATTERMOST_TOKEN não configurados")
	}
	return nil
}

// Tools lista as ferramentas do Mattermost.
func (c *Connector) Tools() []core.ToolDefinition {
	postSchema := jsonschema.NewObjectSchema().
		AddProperty("channel_id", "string", "ID do canal no Mattermost", true).
		AddProperty("message", "string", "Texto da postagem a ser enviada", true).
		AddProperty("root_id", "string", "ID do post pai para responder em thread (opcional)", false).
		RawMessage()

	getPostsSchema := jsonschema.NewObjectSchema().
		AddProperty("channel_id", "string", "ID do canal no Mattermost", true).
		AddProperty("limit", "integer", "Quantidade máxima de posts a retornar (padrão 15)", false).
		RawMessage()

	searchSchema := jsonschema.NewObjectSchema().
		AddProperty("team_id", "string", "ID da equipe no Mattermost", true).
		AddProperty("terms", "string", "Termos de pesquisa", true).
		RawMessage()

	return []core.ToolDefinition{
		{
			Name:        "mattermost_post_message",
			Description: "Publica uma mensagem em um canal ou thread do Mattermost.",
			InputSchema: postSchema,
			Handler:     c.handlePostMessage,
		},
		{
			Name:        "mattermost_get_channel_posts",
			Description: "Busca os posts recentes de um canal no Mattermost.",
			InputSchema: getPostsSchema,
			Handler:     c.handleGetPosts,
		},
		{
			Name:        "mattermost_search_posts",
			Description: "Pesquisa posts dentro de uma equipe no Mattermost por palavras-chave.",
			InputSchema: searchSchema,
			Handler:     c.handleSearchPosts,
		},
	}
}

// Close libera recursos.
func (c *Connector) Close() error {
	return nil
}

func (c *Connector) handlePostMessage(ctx context.Context, rawArgs json.RawMessage) (any, error) {
	var args struct {
		ChannelID string `json:"channel_id"`
		Message   string `json:"message"`
		RootID    string `json:"root_id"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("argumentos inválidos: %w", err)
	}
	if args.ChannelID == "" || args.Message == "" {
		return nil, fmt.Errorf("channel_id e message são obrigatórios")
	}

	payload := map[string]string{
		"channel_id": args.ChannelID,
		"message":    args.Message,
	}
	if args.RootID != "" {
		payload["root_id"] = args.RootID
	}

	endpoint := fmt.Sprintf("%s/api/v4/posts", strings.TrimRight(c.serverURL, "/"))
	bodyBytes, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha ao enviar post no Mattermost: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro Mattermost (%d): %s", resp.StatusCode, string(errBytes))
	}

	var created struct {
		ID        string `json:"id"`
		CreateAt  int64  `json:"create_at"`
		ChannelID string `json:"channel_id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&created)

	return map[string]any{
		"status":     "posted",
		"post_id":    created.ID,
		"channel_id": created.ChannelID,
		"snippet":    core.TruncateText(args.Message, 100),
	}, nil
}

func (c *Connector) handleGetPosts(ctx context.Context, rawArgs json.RawMessage) (any, error) {
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

	endpoint := fmt.Sprintf("%s/api/v4/channels/%s/posts?per_page=%d", strings.TrimRight(c.serverURL, "/"), args.ChannelID, args.Limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha ao ler posts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro Mattermost (%d): %s", resp.StatusCode, string(errBytes))
	}

	var payload struct {
		Order []string `json:"order"`
		Posts map[string]struct {
			ID       string `json:"id"`
			UserID   string `json:"user_id"`
			Message  string `json:"message"`
			CreateAt int64  `json:"create_at"`
		} `json:"posts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	type PostDTO struct {
		ID        string `json:"id"`
		UserID    string `json:"user_id"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
	}

	results := make([]PostDTO, 0, len(payload.Order))
	for _, postID := range payload.Order {
		if p, ok := payload.Posts[postID]; ok {
			results = append(results, PostDTO{
				ID:        p.ID,
				UserID:    p.UserID,
				Message:   core.TruncateText(p.Message, 400),
				Timestamp: time.UnixMilli(p.CreateAt).Format(time.RFC3339),
			})
		}
	}

	return map[string]any{
		"channel_id": args.ChannelID,
		"count":      len(results),
		"posts":      results,
	}, nil
}

func (c *Connector) handleSearchPosts(ctx context.Context, rawArgs json.RawMessage) (any, error) {
	var args struct {
		TeamID string `json:"team_id"`
		Terms  string `json:"terms"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("argumentos inválidos: %w", err)
	}
	if args.TeamID == "" || args.Terms == "" {
		return nil, fmt.Errorf("team_id e terms são obrigatórios")
	}

	endpoint := fmt.Sprintf("%s/api/v4/teams/%s/posts/search", strings.TrimRight(c.serverURL, "/"), args.TeamID)
	bodyBytes, _ := json.Marshal(map[string]string{"terms": args.Terms})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha ao pesquisar posts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("erro na busca Mattermost (%d): %s", resp.StatusCode, string(errBytes))
	}

	return map[string]any{
		"terms":  args.Terms,
		"status": "success",
	}, nil
}

func (c *Connector) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
}
