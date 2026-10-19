package fathom

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const defaultBaseURL = "https://api.fathom.video/external/v1"

// Client gerencia a comunicação HTTP com a API externa do Fathom Video.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewClient instancia o client do Fathom com timeout e headers padrão.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ListRecordings consulta a lista de gravações recentes no Fathom.
func (c *Client) ListRecordings(ctx context.Context, limit int) ([]RecordingItem, error) {
	if limit <= 0 {
		limit = 10
	}
	endpoint := fmt.Sprintf("%s/recordings?limit=%d", c.baseURL, limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha na requisição ao Fathom: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fathom API retornou status %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Items []struct {
			ID          string    `json:"id"`
			Title       string    `json:"title"`
			CreatedAt   time.Time `json:"created_at"`
			DurationSec int       `json:"duration"`
			MeetingURL  string    `json:"meeting_url"`
			ShareURL    string    `json:"share_url"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("erro ao decodificar gravações: %w", err)
	}

	results := make([]RecordingItem, 0, len(payload.Items))
	for _, it := range payload.Items {
		results = append(results, RecordingItem{
			ID:          it.ID,
			Title:       it.Title,
			CreatedAt:   it.CreatedAt,
			DurationSec: it.DurationSec,
			MeetingURL:  it.MeetingURL,
			ShareURL:    it.ShareURL,
		})
	}

	return results, nil
}

// GetTranscript busca a transcrição detalhada com os falantes e timestamps.
func (c *Client) GetTranscript(ctx context.Context, recordingID string) (*TranscriptDTO, error) {
	endpoint := fmt.Sprintf("%s/recordings/%s/transcript", c.baseURL, url.PathEscape(recordingID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar ao Fathom: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fathom API retornou status %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Title       string `json:"title"`
		CreatedAt   string `json:"created_at"`
		Summary     string `json:"summary"`
		ActionItems []string `json:"action_items"`
		Transcript  []struct {
			Speaker   string `json:"speaker"`
			Timestamp string `json:"timestamp"`
			Text      string `json:"text"`
		} `json:"transcript"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("erro ao decodificar transcrição: %w", err)
	}

	segments := make([]TranscriptSegment, 0, len(payload.Transcript))
	for _, s := range payload.Transcript {
		segments = append(segments, TranscriptSegment{
			Speaker:   s.Speaker,
			Timestamp: s.Timestamp,
			Text:      s.Text,
		})
	}

	return &TranscriptDTO{
		RecordingID: recordingID,
		Title:       payload.Title,
		Date:        payload.CreatedAt,
		Summary:     payload.Summary,
		ActionItems: payload.ActionItems,
		Segments:    segments,
	}, nil
}

// GetSummary busca exclusivamente o resumo executivo e action items gerados pelo Fathom.
func (c *Client) GetSummary(ctx context.Context, recordingID string) (*SummaryDTO, error) {
	endpoint := fmt.Sprintf("%s/recordings/%s", c.baseURL, url.PathEscape(recordingID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar ao Fathom: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fathom API retornou status %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		ID          string   `json:"id"`
		Title       string   `json:"title"`
		Summary     string   `json:"summary"`
		Decisions   []string `json:"decisions"`
		ActionItems []string `json:"action_items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resumo: %w", err)
	}

	return &SummaryDTO{
		RecordingID: payload.ID,
		Title:       payload.Title,
		Overview:    payload.Summary,
		Decisions:   payload.Decisions,
		ActionItems: payload.ActionItems,
	}, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "OmniConnectorsHub/1.0 (Go)")
}
