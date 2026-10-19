package fathom

import "time"

// RecordingItem resume uma gravação de reunião retornada pela listagem do Fathom.
type RecordingItem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	CreatedAt   time.Time `json:"created_at"`
	DurationSec int       `json:"duration_seconds"`
	MeetingURL  string    `json:"meeting_url,omitempty"`
	ShareURL    string    `json:"share_url,omitempty"`
}

// TranscriptSegment representa uma fala identificada por autor e marca temporal.
type TranscriptSegment struct {
	Speaker   string `json:"speaker"`
	Timestamp string `json:"timestamp"`
	Text      string `json:"text"`
}

// TranscriptDTO é a representação enxuta da transcrição para entrega à LLM.
type TranscriptDTO struct {
	RecordingID string              `json:"recording_id"`
	Title       string              `json:"title"`
	Date        string              `json:"date"`
	Summary     string              `json:"summary,omitempty"`
	ActionItems []string            `json:"action_items,omitempty"`
	Segments    []TranscriptSegment `json:"segments"`
	TotalTokens int                 `json:"approx_tokens,omitempty"`
}

// SummaryDTO sintetiza o resumo e tópicos decididos na chamada.
type SummaryDTO struct {
	RecordingID string   `json:"recording_id"`
	Title       string   `json:"title"`
	Overview    string   `json:"overview"`
	Decisions   []string `json:"decisions,omitempty"`
	ActionItems []string `json:"action_items,omitempty"`
}
