package fathom

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"connectors/internal/core"
	"connectors/pkg/jsonschema"
)

// Connector implementa a interface core.Connector para o Fathom Video.
type Connector struct {
	apiKey string
	client *Client
}

// NewConnector cria uma nova instância do conector Fathom com chave explícita ou via ambiente.
func NewConnector(apiKey string) *Connector {
	if apiKey == "" {
		apiKey = os.Getenv("FATHOM_API_KEY")
	}
	return &Connector{
		apiKey: apiKey,
		client: NewClient(apiKey),
	}
}

// Namespace identifica o domínio do conector.
func (c *Connector) Namespace() string {
	return "fathom"
}

// Initialize valida as configurações necessárias para autenticação.
func (c *Connector) Initialize(ctx context.Context) error {
	if c.apiKey == "" {
		return fmt.Errorf("FATHOM_API_KEY não configurada no ambiente ou construtor")
	}
	return nil
}

// Tools lista as ferramentas executáveis expostas pelo Fathom para a LLM.
func (c *Connector) Tools() []core.ToolDefinition {
	listRecordingsSchema := jsonschema.NewObjectSchema().
		AddProperty("limit", "integer", "Quantidade máxima de reuniões a listar (padrão 10)", false).
		RawMessage()

	getTranscriptSchema := jsonschema.NewObjectSchema().
		AddProperty("recording_id", "string", "ID único da gravação no Fathom", true).
		AddProperty("max_characters", "integer", "Limite de caracteres para truncamento (padrão 8000)", false).
		RawMessage()

	getSummarySchema := jsonschema.NewObjectSchema().
		AddProperty("recording_id", "string", "ID único da gravação no Fathom", true).
		RawMessage()

	return []core.ToolDefinition{
		{
			Name:        "fathom_list_recordings",
			Description: "Lista as gravações recentes de reuniões do Fathom Video com IDs, datas e títulos.",
			InputSchema: listRecordingsSchema,
			Handler:     c.handleListRecordings,
		},
		{
			Name:        "fathom_get_call_transcript",
			Description: "Busca a transcrição completa com falantes e resumo executivo de uma reunião gravada no Fathom.",
			InputSchema: getTranscriptSchema,
			Handler:     c.handleGetTranscript,
		},
		{
			Name:        "fathom_get_meeting_summary",
			Description: "Busca apenas o resumo estruturado, pontos decididos e tarefas de ação de uma chamada no Fathom.",
			InputSchema: getSummarySchema,
			Handler:     c.handleGetSummary,
		},
	}
}

// Close libera recursos e conexões.
func (c *Connector) Close() error {
	return nil
}

func (c *Connector) handleListRecordings(ctx context.Context, rawArgs json.RawMessage) (any, error) {
	var args struct {
		Limit int `json:"limit"`
	}
	if len(rawArgs) > 0 {
		_ = json.Unmarshal(rawArgs, &args)
	}
	if args.Limit <= 0 {
		args.Limit = 10
	}

	recordings, err := c.client.ListRecordings(ctx, args.Limit)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar reuniões no Fathom: %w", err)
	}

	return map[string]any{
		"count":      len(recordings),
		"recordings": recordings,
	}, nil
}

func (c *Connector) handleGetTranscript(ctx context.Context, rawArgs json.RawMessage) (any, error) {
	var args struct {
		RecordingID   string `json:"recording_id"`
		MaxCharacters int    `json:"max_characters"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("argumentos inválidos: %w", err)
	}
	if args.RecordingID == "" {
		return nil, fmt.Errorf("recording_id é obrigatório")
	}
	if args.MaxCharacters <= 0 {
		args.MaxCharacters = 8000
	}

	transcript, err := c.client.GetTranscript(ctx, args.RecordingID)
	if err != nil {
		return nil, fmt.Errorf("falha ao recuperar transcrição do Fathom: %w", err)
	}

	// Aplica token budgeting nos segmentos de transcrição se exceder o limite
	totalChars := 0
	trimmedSegments := make([]TranscriptSegment, 0, len(transcript.Segments))
	for _, seg := range transcript.Segments {
		cleanText := core.StripHTML(seg.Text)
		if totalChars+len(cleanText) > args.MaxCharacters {
			trimmedSegments = append(trimmedSegments, TranscriptSegment{
				Speaker:   seg.Speaker,
				Timestamp: seg.Timestamp,
				Text:      cleanText[:max(0, args.MaxCharacters-totalChars)] + " [... truncado ...]",
			})
			break
		}
		trimmedSegments = append(trimmedSegments, TranscriptSegment{
			Speaker:   seg.Speaker,
			Timestamp: seg.Timestamp,
			Text:      cleanText,
		})
		totalChars += len(cleanText)
	}
	transcript.Segments = trimmedSegments
	transcript.TotalTokens = totalChars / 4

	return transcript, nil
}

func (c *Connector) handleGetSummary(ctx context.Context, rawArgs json.RawMessage) (any, error) {
	var args struct {
		RecordingID string `json:"recording_id"`
	}
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return nil, fmt.Errorf("argumentos inválidos: %w", err)
	}
	if args.RecordingID == "" {
		return nil, fmt.Errorf("recording_id é obrigatório")
	}

	summary, err := c.client.GetSummary(ctx, args.RecordingID)
	if err != nil {
		return nil, fmt.Errorf("falha ao recuperar resumo do Fathom: %w", err)
	}

	return summary, nil
}
