package whatsapp

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Chat é o registro local de uma conversa (individual ou grupo).
type Chat struct {
	JID          string    `json:"jid"`
	Name         string    `json:"name"`
	Topic        string    `json:"topic,omitempty"`
	IsGroup      bool      `json:"is_group"`
	Archived     bool      `json:"archived"`
	Participants int       `json:"participants,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ChatSummary agrega estatísticas de mensagens de um chat.
type ChatSummary struct {
	Chat
	Messages   int       `json:"messages"`
	FromMe     int       `json:"from_me"`
	FirstAt    time.Time `json:"first_at"`
	LastAt     time.Time `json:"last_at"`
	SenderSeen int       `json:"senders"`
}

// Message é uma mensagem persistida, já reduzida ao texto útil para análise.
type Message struct {
	ChatJID    string    `json:"chat_jid"`
	ID         string    `json:"id"`
	SenderJID  string    `json:"sender_jid"`
	SenderAlt  string    `json:"sender_alt,omitempty"`
	SenderName string    `json:"sender_name,omitempty"`
	FromMe     bool      `json:"from_me"`
	Timestamp  time.Time `json:"timestamp"`
	Kind       string    `json:"kind"`
	Text       string    `json:"text"`
	QuotedID   string    `json:"quoted_id,omitempty"`
	// Media guarda o protobuf da mídia (hoje só AudioMessage) com as chaves de download.
	Media       []byte `json:"-"`
	MediaPath   string `json:"media_path,omitempty"`
	MediaStatus string `json:"media_status,omitempty"`
	Transcript  string `json:"transcript,omitempty"`
}

// Estados do pipeline de áudio (coluna media_status).
const (
	MediaPending         = ""                 // sem ação ainda: falta chave (media nula) ou download
	MediaKeysRequested   = "keys_requested"   // pedido de histórico sob demanda enviado ao celular
	MediaKeysUnavailable = "keys_unavailable" // o celular não devolveu as chaves
	MediaExpired         = "expired"          // mídia saiu do servidor; aguarda cota de media retry
	MediaRetryRequested  = "retry_requested"  // media retry enviado; celular vai reenviar
	MediaDownloaded      = "downloaded"
	MediaTranscribed     = "transcribed"
	MediaFailed          = "failed"
)

// ChatFilter restringe a listagem de chats.
type ChatFilter struct {
	ArchivedOnly bool
	GroupsOnly   bool
	Query        string
	Limit        int
}

// MessageStore persiste chats e mensagens recebidos do whatsmeow em um SQLite próprio,
// separado do banco de sessão/criptografia.
type MessageStore struct {
	db *sql.DB
}

const historySchema = `
CREATE TABLE IF NOT EXISTS chats (
	jid          TEXT PRIMARY KEY,
	name         TEXT NOT NULL DEFAULT '',
	topic        TEXT NOT NULL DEFAULT '',
	is_group     INTEGER NOT NULL DEFAULT 0,
	archived     INTEGER NOT NULL DEFAULT 0,
	participants INTEGER NOT NULL DEFAULT 0,
	updated_at   INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS messages (
	chat_jid    TEXT NOT NULL,
	id          TEXT NOT NULL,
	sender_jid  TEXT NOT NULL DEFAULT '',
	sender_alt  TEXT NOT NULL DEFAULT '',
	sender_name TEXT NOT NULL DEFAULT '',
	from_me     INTEGER NOT NULL DEFAULT 0,
	timestamp   INTEGER NOT NULL,
	kind        TEXT NOT NULL,
	text        TEXT NOT NULL DEFAULT '',
	quoted_id   TEXT NOT NULL DEFAULT '',
	PRIMARY KEY (chat_jid, id)
);
CREATE INDEX IF NOT EXISTS idx_messages_chat_ts ON messages(chat_jid, timestamp);
`

// historyColumns são colunas adicionadas depois da primeira versão do schema.
var historyColumns = []struct{ name, ddl string }{
	{"media", `ALTER TABLE messages ADD COLUMN media BLOB`},
	{"media_path", `ALTER TABLE messages ADD COLUMN media_path TEXT NOT NULL DEFAULT ''`},
	{"media_status", `ALTER TABLE messages ADD COLUMN media_status TEXT NOT NULL DEFAULT ''`},
	{"transcript", `ALTER TABLE messages ADD COLUMN transcript TEXT NOT NULL DEFAULT ''`},
}

// OpenMessageStore abre (ou cria) o banco de histórico no caminho informado.
func OpenMessageStore(ctx context.Context, path string) (*MessageStore, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000", path))
	if err != nil {
		return nil, fmt.Errorf("abrir banco de histórico: %w", err)
	}
	if err := migrateHistory(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrar banco de histórico: %w", err)
	}
	return &MessageStore{db: db}, nil
}

func migrateHistory(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, historySchema); err != nil {
		return err
	}
	rows, err := db.QueryContext(ctx, `SELECT name FROM pragma_table_info('messages')`)
	if err != nil {
		return err
	}
	existing := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return err
		}
		existing[name] = true
	}
	rows.Close()
	for _, col := range historyColumns {
		if existing[col.name] {
			continue
		}
		if _, err := db.ExecContext(ctx, col.ddl); err != nil {
			return fmt.Errorf("adicionar coluna %s: %w", col.name, err)
		}
	}
	_, err = db.ExecContext(ctx, `
CREATE INDEX IF NOT EXISTS idx_messages_kind_status ON messages(kind, media_status);
CREATE TABLE IF NOT EXISTS wa_actions (
	action TEXT NOT NULL,
	ts     INTEGER NOT NULL -- unix ms
);
CREATE INDEX IF NOT EXISTS idx_wa_actions ON wa_actions(action, ts);
CREATE TABLE IF NOT EXISTS operator_sends (
	chat_jid TEXT NOT NULL,
	hash     TEXT NOT NULL,
	ts       INTEGER NOT NULL -- unix ms
);
CREATE INDEX IF NOT EXISTS idx_operator_sends ON operator_sends(hash, ts);`)
	return err
}

// Close encerra o banco.
func (s *MessageStore) Close() error {
	return s.db.Close()
}

// UpsertChat cria ou atualiza um chat. Campos vazios não sobrescrevem valores já conhecidos;
// o flag de arquivado é sempre gravado, pois quem chama sabe o estado atual.
func (s *MessageStore) UpsertChat(ctx context.Context, c Chat) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO chats (jid, name, topic, is_group, archived, participants, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(jid) DO UPDATE SET
	name         = CASE WHEN excluded.name  != '' THEN excluded.name  ELSE chats.name  END,
	topic        = CASE WHEN excluded.topic != '' THEN excluded.topic ELSE chats.topic END,
	is_group     = excluded.is_group,
	archived     = excluded.archived,
	participants = CASE WHEN excluded.participants > 0 THEN excluded.participants ELSE chats.participants END,
	updated_at   = excluded.updated_at`,
		c.JID, c.Name, c.Topic, c.IsGroup, c.Archived, c.Participants, c.UpdatedAt.Unix())
	if err != nil {
		return fmt.Errorf("gravar chat %s: %w", c.JID, err)
	}
	return nil
}

// EnsureChat registra um chat sem alterar nome nem flag de arquivado já existentes.
func (s *MessageStore) EnsureChat(ctx context.Context, jid string, isGroup bool) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO chats (jid, is_group, updated_at) VALUES (?, ?, ?) ON CONFLICT(jid) DO NOTHING`,
		jid, isGroup, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("registrar chat %s: %w", jid, err)
	}
	return nil
}

// UpdateGroupMeta grava nome, descrição e participantes de um grupo sem tocar no flag de arquivado.
func (s *MessageStore) UpdateGroupMeta(ctx context.Context, jid, name, topic string, participants int) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO chats (jid, name, topic, is_group, participants, updated_at) VALUES (?, ?, ?, 1, ?, ?)
ON CONFLICT(jid) DO UPDATE SET
	name         = CASE WHEN excluded.name != '' THEN excluded.name ELSE chats.name END,
	topic        = excluded.topic,
	is_group     = 1,
	participants = excluded.participants,
	updated_at   = excluded.updated_at`,
		jid, name, topic, participants, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("gravar metadados do grupo %s: %w", jid, err)
	}
	return nil
}

// SetArchived atualiza apenas o flag de arquivado de um chat.
func (s *MessageStore) SetArchived(ctx context.Context, jid string, archived bool) error {
	isGroup := strings.HasSuffix(jid, "@g.us")
	_, err := s.db.ExecContext(ctx, `
INSERT INTO chats (jid, is_group, archived, updated_at) VALUES (?, ?, ?, ?)
ON CONFLICT(jid) DO UPDATE SET archived = excluded.archived, updated_at = excluded.updated_at`,
		jid, isGroup, archived, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("marcar arquivado %s: %w", jid, err)
	}
	return nil
}

// SaveMessages grava um lote de mensagens numa única transação. Mensagens repetidas
// (mesmo chat e id) atualizam o texto, o que cobre edições. Mídia reenviada (histórico
// sob demanda) preenche as chaves que faltavam e devolve o áudio à fila de download.
func (s *MessageStore) SaveMessages(ctx context.Context, msgs []Message) error {
	if len(msgs) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // rollback após commit é no-op

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO messages (chat_jid, id, sender_jid, sender_alt, sender_name, from_me, timestamp, kind, text, quoted_id, media)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(chat_jid, id) DO UPDATE SET
	text         = excluded.text,
	kind         = excluded.kind,
	sender_name  = CASE WHEN excluded.sender_name != '' THEN excluded.sender_name ELSE messages.sender_name END,
	media_status = CASE WHEN excluded.media IS NOT NULL AND messages.media IS NULL THEN '' ELSE messages.media_status END,
	media        = COALESCE(messages.media, excluded.media)`)
	if err != nil {
		return fmt.Errorf("preparar insert de mensagens: %w", err)
	}
	defer stmt.Close()

	for _, m := range msgs {
		if _, err := stmt.ExecContext(ctx, m.ChatJID, m.ID, m.SenderJID, m.SenderAlt, m.SenderName,
			m.FromMe, m.Timestamp.Unix(), m.Kind, m.Text, m.QuotedID, m.Media); err != nil {
			return fmt.Errorf("gravar mensagem %s/%s: %w", m.ChatJID, m.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("confirmar lote de mensagens: %w", err)
	}
	return nil
}

// ListChats devolve chats com estatísticas, ordenados pela mensagem mais recente.
func (s *MessageStore) ListChats(ctx context.Context, f ChatFilter) ([]ChatSummary, error) {
	var where []string
	var args []any
	if f.ArchivedOnly {
		where = append(where, "c.archived = 1")
	}
	if f.GroupsOnly {
		where = append(where, "c.is_group = 1")
	}
	if f.Query != "" {
		where = append(where, "(c.name LIKE ? OR c.jid LIKE ?)")
		like := "%" + f.Query + "%"
		args = append(args, like, like)
	}
	query := `
SELECT c.jid, c.name, c.topic, c.is_group, c.archived, c.participants, c.updated_at,
	COUNT(m.id), COALESCE(SUM(m.from_me), 0), COALESCE(MIN(m.timestamp), 0), COALESCE(MAX(m.timestamp), 0),
	COUNT(DISTINCT m.sender_jid)
FROM chats c LEFT JOIN messages m ON m.chat_jid = c.jid`
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " GROUP BY c.jid ORDER BY MAX(m.timestamp) DESC"
	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, f.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listar chats: %w", err)
	}
	defer rows.Close()

	var out []ChatSummary
	for rows.Next() {
		var cs ChatSummary
		var updated, first, last int64
		if err := rows.Scan(&cs.JID, &cs.Name, &cs.Topic, &cs.IsGroup, &cs.Archived, &cs.Participants, &updated,
			&cs.Messages, &cs.FromMe, &first, &last, &cs.SenderSeen); err != nil {
			return nil, fmt.Errorf("ler chat: %w", err)
		}
		cs.UpdatedAt = time.Unix(updated, 0)
		if first > 0 {
			cs.FirstAt = time.Unix(first, 0)
			cs.LastAt = time.Unix(last, 0)
		}
		out = append(out, cs)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar chats: %w", err)
	}
	return out, nil
}

// Messages devolve as mensagens de um chat em ordem cronológica. Com limit > 0,
// devolve apenas as últimas `limit` mensagens (ainda em ordem cronológica).
func (s *MessageStore) Messages(ctx context.Context, chatJID string, limit int) ([]Message, error) {
	query := `SELECT ` + messageColumns + ` FROM messages WHERE chat_jid = ? ORDER BY timestamp DESC`
	args := []any{chatJID}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("buscar mensagens de %s: %w", chatJID, err)
	}
	defer rows.Close()

	var out []Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, fmt.Errorf("ler mensagem: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar mensagens: %w", err)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

// MessagesSince devolve as mensagens de todos os chats a partir de since, em ordem
// cronológica. Com limit > 0, fica só com as `limit` mais recentes.
func (s *MessageStore) MessagesSince(ctx context.Context, since time.Time, limit int) ([]Message, error) {
	where := "timestamp >= ? ORDER BY timestamp DESC"
	args := []any{since.Unix()}
	if limit > 0 {
		where += " LIMIT ?"
		args = append(args, limit)
	}
	msgs, err := s.queryMessages(ctx, where, args...)
	if err != nil {
		return nil, fmt.Errorf("buscar mensagens desde %s: %w", since.Format(time.RFC3339), err)
	}
	slices.Reverse(msgs)
	return msgs, nil
}

// SearchMessages busca mensagens cujo texto ou transcrição de áudio contenha todos os
// termos de query, das mais recentes para as mais antigas. chatJID vazio busca em todos.
func (s *MessageStore) SearchMessages(ctx context.Context, query, chatJID string, limit int) ([]Message, error) {
	var conds []string
	var args []any
	for _, term := range strings.Fields(query) {
		like := "%" + likeEscaper.Replace(term) + "%"
		conds = append(conds, `(text LIKE ? ESCAPE '\' OR transcript LIKE ? ESCAPE '\')`)
		args = append(args, like, like)
	}
	if len(conds) == 0 {
		return nil, fmt.Errorf("busca vazia")
	}
	if chatJID != "" {
		conds = append(conds, "chat_jid = ?")
		args = append(args, chatJID)
	}
	where := strings.Join(conds, " AND ") + " ORDER BY timestamp DESC"
	if limit > 0 {
		where += " LIMIT ?"
		args = append(args, limit)
	}
	msgs, err := s.queryMessages(ctx, where, args...)
	if err != nil {
		return nil, fmt.Errorf("buscar %q: %w", query, err)
	}
	return msgs, nil
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// Chats devolve os chats conhecidos indexados pelo JID.
func (s *MessageStore) Chats(ctx context.Context) (map[string]Chat, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT jid, name, topic, is_group, archived, participants, updated_at FROM chats`)
	if err != nil {
		return nil, fmt.Errorf("listar chats: %w", err)
	}
	defer rows.Close()
	out := make(map[string]Chat)
	for rows.Next() {
		var c Chat
		var updated int64
		if err := rows.Scan(&c.JID, &c.Name, &c.Topic, &c.IsGroup, &c.Archived, &c.Participants, &updated); err != nil {
			return nil, fmt.Errorf("ler chat: %w", err)
		}
		c.UpdatedAt = time.Unix(updated, 0)
		out[c.JID] = c
	}
	return out, rows.Err()
}

// LastMessageAt devolve o horário da mensagem mais recente gravada (zero se não houver),
// para saber se o histórico local está em dia.
func (s *MessageStore) LastMessageAt(ctx context.Context) (time.Time, error) {
	var ts int64
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(timestamp), 0) FROM messages`).Scan(&ts); err != nil {
		return time.Time{}, fmt.Errorf("última mensagem: %w", err)
	}
	if ts == 0 {
		return time.Time{}, nil
	}
	return time.Unix(ts, 0), nil
}

// ChatJIDs devolve todos os JIDs conhecidos.
func (s *MessageStore) ChatJIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT jid FROM chats`)
	if err != nil {
		return nil, fmt.Errorf("listar jids: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var jid string
		if err := rows.Scan(&jid); err != nil {
			return nil, fmt.Errorf("ler jid: %w", err)
		}
		out = append(out, jid)
	}
	return out, rows.Err()
}
