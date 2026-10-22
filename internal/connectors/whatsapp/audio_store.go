package whatsapp

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const messageColumns = `chat_jid, id, sender_jid, sender_alt, sender_name, from_me, timestamp, kind, text, quoted_id,
	media, media_path, media_status, transcript`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMessage(r rowScanner) (Message, error) {
	var m Message
	var ts int64
	err := r.Scan(&m.ChatJID, &m.ID, &m.SenderJID, &m.SenderAlt, &m.SenderName, &m.FromMe, &ts, &m.Kind, &m.Text,
		&m.QuotedID, &m.Media, &m.MediaPath, &m.MediaStatus, &m.Transcript)
	m.Timestamp = time.Unix(ts, 0)
	return m, err
}

func (s *MessageStore) queryMessages(ctx context.Context, where string, args ...any) ([]Message, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+messageColumns+` FROM messages WHERE `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// BackfillRequest é um pedido de histórico sob demanda: as Count mensagens anteriores
// a Anchor, que cobrem os áudios ainda sem chave de download.
type BackfillRequest struct {
	ChatJID  string
	Anchor   Message
	Count    int
	AudioIDs []string
}

// needsKeys indica áudio sem chave de mídia que ainda vale pedir ao celular.
func needsKeys(m Message) bool {
	return m.Kind == "audio" && m.Media == nil &&
		(m.MediaStatus == MediaPending || m.MediaStatus == MediaKeysRequested)
}

// BackfillPlan agrupa os áudios sem chave em pedidos de no máximo maxPerRequest
// mensagens, ancorados na mensagem seguinte ao áudio mais novo de cada bloco.
// Áudios que são a última mensagem do chat ficam de fora (não há âncora).
func (s *MessageStore) BackfillPlan(ctx context.Context, maxPerRequest int) ([]BackfillRequest, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT DISTINCT chat_jid FROM messages
WHERE kind = 'audio' AND media IS NULL AND media_status IN (?, ?)`, MediaPending, MediaKeysRequested)
	if err != nil {
		return nil, fmt.Errorf("listar chats com áudio sem chave: %w", err)
	}
	var chats []string
	for rows.Next() {
		var jid string
		if err := rows.Scan(&jid); err != nil {
			rows.Close()
			return nil, err
		}
		chats = append(chats, jid)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var plan []BackfillRequest
	for _, chat := range chats {
		msgs, err := s.queryMessages(ctx, `chat_jid = ? ORDER BY timestamp, id`, chat)
		if err != nil {
			return nil, fmt.Errorf("carregar chat %s: %w", chat, err)
		}
		plan = append(plan, planChat(chat, msgs, maxPerRequest)...)
	}
	return plan, nil
}

func planChat(chat string, msgs []Message, maxPerRequest int) []BackfillRequest {
	var out []BackfillRequest
	for i := len(msgs) - 1; i >= 0; {
		if !needsKeys(msgs[i]) || i == len(msgs)-1 {
			i--
			continue
		}
		anchor := i + 1
		earliest := i
		var ids []string
		for j := i; j >= 0 && anchor-j <= maxPerRequest; j-- {
			if needsKeys(msgs[j]) {
				ids = append(ids, msgs[j].ID)
				earliest = j
			}
		}
		out = append(out, BackfillRequest{ChatJID: chat, Anchor: msgs[anchor], Count: anchor - earliest, AudioIDs: ids})
		i = earliest - 1
	}
	return out
}

// ResolveBackfill marca como sem chave os áudios do pedido que continuam sem mídia
// e devolve quantos receberam as chaves.
func (s *MessageStore) ResolveBackfill(ctx context.Context, req BackfillRequest) (int, error) {
	if len(req.AudioIDs) == 0 {
		return 0, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(req.AudioIDs)), ",")
	args := []any{MediaKeysUnavailable, req.ChatJID}
	for _, id := range req.AudioIDs {
		args = append(args, id)
	}
	res, err := s.db.ExecContext(ctx, `UPDATE messages SET media_status = ?
WHERE chat_jid = ? AND media IS NULL AND id IN (`+placeholders+`)`, args...)
	if err != nil {
		return 0, fmt.Errorf("marcar áudios sem chave: %w", err)
	}
	missing, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return len(req.AudioIDs) - int(missing), nil
}

// MarkKeysRequested registra que o pedido sob demanda foi enviado.
func (s *MessageStore) MarkKeysRequested(ctx context.Context, req BackfillRequest) error {
	for _, id := range req.AudioIDs {
		if err := s.SetMediaStatus(ctx, req.ChatJID, id, MediaKeysRequested); err != nil {
			return err
		}
	}
	return nil
}

// PendingAudioDownloads devolve áudios com chave e sem arquivo, mais novos primeiro
// (têm mais chance de ainda estar no servidor do WhatsApp).
func (s *MessageStore) PendingAudioDownloads(ctx context.Context, limit int) ([]Message, error) {
	return s.queryMessages(ctx, `kind = 'audio' AND media IS NOT NULL AND media_path = '' AND media_status = ?
ORDER BY timestamp DESC LIMIT ?`, MediaPending, limit)
}

// PendingRetries devolve áudios expirados no servidor, mais novos primeiro.
func (s *MessageStore) PendingRetries(ctx context.Context, limit int) ([]Message, error) {
	return s.queryMessages(ctx, `kind = 'audio' AND media_status = ? ORDER BY timestamp DESC LIMIT ?`,
		MediaExpired, limit)
}

// PendingTranscriptions devolve áudios baixados e ainda não transcritos.
func (s *MessageStore) PendingTranscriptions(ctx context.Context, limit int) ([]Message, error) {
	return s.queryMessages(ctx, `kind = 'audio' AND media_status = ? ORDER BY timestamp DESC LIMIT ?`,
		MediaDownloaded, limit)
}

// MessageByID busca uma mensagem pelo id (ids do WhatsApp são aleatórios e longos).
func (s *MessageStore) MessageByID(ctx context.Context, id string) (Message, bool, error) {
	msgs, err := s.queryMessages(ctx, `id = ? LIMIT 1`, id)
	if err != nil || len(msgs) == 0 {
		return Message{}, false, err
	}
	return msgs[0], true, nil
}

// SetMediaStatus atualiza só o estado do pipeline de mídia.
func (s *MessageStore) SetMediaStatus(ctx context.Context, chatJID, id, status string) error {
	return s.exec(ctx, `UPDATE messages SET media_status = ? WHERE chat_jid = ? AND id = ?`, status, chatJID, id)
}

// SetAudioMedia troca o protobuf da mídia (ex.: novo direct path após media retry).
func (s *MessageStore) SetAudioMedia(ctx context.Context, chatJID, id string, media []byte, status string) error {
	return s.exec(ctx, `UPDATE messages SET media = ?, media_status = ? WHERE chat_jid = ? AND id = ?`,
		media, status, chatJID, id)
}

// SetMediaFile registra o arquivo baixado.
func (s *MessageStore) SetMediaFile(ctx context.Context, chatJID, id, path string) error {
	return s.exec(ctx, `UPDATE messages SET media_path = ?, media_status = ? WHERE chat_jid = ? AND id = ?`,
		path, MediaDownloaded, chatJID, id)
}

// SetTranscript grava a transcrição de um áudio.
func (s *MessageStore) SetTranscript(ctx context.Context, chatJID, id, text string) error {
	return s.exec(ctx, `UPDATE messages SET transcript = ?, media_status = ? WHERE chat_jid = ? AND id = ?`,
		text, MediaTranscribed, chatJID, id)
}

// AudioStats conta os áudios por etapa do pipeline.
func (s *MessageStore) AudioStats(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT CASE
	WHEN media IS NULL AND media_status IN ('', ?) THEN 'sem_chave'
	WHEN media_status = '' THEN 'na_fila_download'
	ELSE media_status END, COUNT(*)
FROM messages WHERE kind = 'audio' GROUP BY 1`, MediaKeysRequested)
	if err != nil {
		return nil, fmt.Errorf("contar áudios: %w", err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var k string
		var n int
		if err := rows.Scan(&k, &n); err != nil {
			return nil, err
		}
		out[k] = n
	}
	return out, rows.Err()
}

// HasInbound informa se algum dos JIDs (mesma conversa por telefone ou LID) tem ao menos
// uma mensagem recebida, isto é, se a pessoa (ou o grupo) já escreveu para o usuário.
func (s *MessageStore) HasInbound(ctx context.Context, jids []string) (bool, error) {
	if len(jids) == 0 {
		return false, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(jids)), ",")
	args := make([]any, len(jids))
	for i, j := range jids {
		args[i] = j
	}
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM messages WHERE from_me = 0 AND chat_jid IN (`+placeholders+`)`,
		args...).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("verificar conversa prévia: %w", err)
	}
	return n > 0, nil
}

// SentElsewhere informa se o operador mandou o mesmo texto (hash) para outro chat desde since.
func (s *MessageStore) SentElsewhere(ctx context.Context, hash, chatJID string, since time.Time) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM operator_sends WHERE hash = ? AND chat_jid != ? AND ts > ?`,
		hash, chatJID, since.UnixMilli()).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("verificar envio repetido: %w", err)
	}
	return n > 0, nil
}

// RecordOperatorSend registra um envio do operador para a checagem de repetição.
func (s *MessageStore) RecordOperatorSend(ctx context.Context, chatJID, hash string, at time.Time) error {
	if _, err := s.db.ExecContext(ctx, `INSERT INTO operator_sends (chat_jid, hash, ts) VALUES (?, ?, ?)`,
		chatJID, hash, at.UnixMilli()); err != nil {
		return fmt.Errorf("registrar envio: %w", err)
	}
	return nil
}

// actionWindow resume o uso de uma ação nas últimas 24h: último registro, contagem e
// registro mais antigo na última hora e no último dia.
func (s *MessageStore) actionWindow(ctx context.Context, action string, now time.Time) (
	last time.Time, hourCount int, hourOldest time.Time, dayCount int, dayOldest time.Time, err error) {
	hourAgo := now.Add(-time.Hour).UnixMilli()
	dayAgo := now.Add(-24 * time.Hour).UnixMilli()
	var maxTS, hourMin, dayMin sql.NullInt64
	err = s.db.QueryRowContext(ctx, `
SELECT MAX(ts), COALESCE(SUM(ts > ?), 0), MIN(CASE WHEN ts > ? THEN ts END), COUNT(*), MIN(ts)
FROM wa_actions WHERE action = ? AND ts > ?`, hourAgo, hourAgo, action, dayAgo).
		Scan(&maxTS, &hourCount, &hourMin, &dayCount, &dayMin)
	if err != nil {
		return last, 0, hourOldest, 0, dayOldest, fmt.Errorf("ler cota de %s: %w", action, err)
	}
	if maxTS.Valid {
		last = time.UnixMilli(maxTS.Int64)
	}
	if hourMin.Valid {
		hourOldest = time.UnixMilli(hourMin.Int64)
	}
	if dayMin.Valid {
		dayOldest = time.UnixMilli(dayMin.Int64)
	}
	return last, hourCount, hourOldest, dayCount, dayOldest, nil
}

// recordAction registra uma execução e descarta registros com mais de 7 dias.
func (s *MessageStore) recordAction(ctx context.Context, action string, at time.Time) error {
	if _, err := s.db.ExecContext(ctx, `INSERT INTO wa_actions (action, ts) VALUES (?, ?)`, action, at.UnixMilli()); err != nil {
		return fmt.Errorf("registrar %s: %w", action, err)
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM wa_actions WHERE ts < ?`, at.AddDate(0, 0, -7).UnixMilli())
	return err
}

func (s *MessageStore) exec(ctx context.Context, query string, args ...any) error {
	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// isNotFound simplifica o tratamento de atualizações em mensagens inexistentes.
func isNotFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }
