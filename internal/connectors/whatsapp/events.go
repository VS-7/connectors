package whatsapp

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// SyncStats resume o progresso da sincronização de histórico.
type SyncStats struct {
	Paired        bool      `json:"paired"`
	Connected     bool      `json:"connected"`
	HistoryChunks int       `json:"history_chunks"`
	Conversations int       `json:"conversations"`
	Messages      int       `json:"messages"`
	Progress      uint32    `json:"progress"`
	LastHistoryAt time.Time `json:"last_history_at"`
}

// extractContent reduz uma mensagem do WhatsApp ao tipo e ao texto útil para análise.
// Devolve ok=false para mensagens sem conteúdo humano (reações, protocolo, chaves).
func extractContent(msg *waE2E.Message) (kind, text, quotedID string, ok bool) {
	if msg == nil {
		return "", "", "", false
	}
	switch {
	case msg.GetConversation() != "":
		return "text", msg.GetConversation(), "", true
	case msg.GetExtendedTextMessage() != nil:
		ext := msg.GetExtendedTextMessage()
		return "text", ext.GetText(), ext.GetContextInfo().GetStanzaID(), true
	case msg.GetImageMessage() != nil:
		img := msg.GetImageMessage()
		return "image", img.GetCaption(), img.GetContextInfo().GetStanzaID(), true
	case msg.GetVideoMessage() != nil:
		vid := msg.GetVideoMessage()
		return "video", vid.GetCaption(), vid.GetContextInfo().GetStanzaID(), true
	case msg.GetAudioMessage() != nil:
		audio := msg.GetAudioMessage()
		return "audio", fmt.Sprintf("%ds", audio.GetSeconds()), audio.GetContextInfo().GetStanzaID(), true
	case msg.GetDocumentMessage() != nil:
		doc := msg.GetDocumentMessage()
		text := doc.GetFileName()
		if caption := doc.GetCaption(); caption != "" {
			text += " — " + caption
		}
		return "document", text, doc.GetContextInfo().GetStanzaID(), true
	case msg.GetStickerMessage() != nil:
		return "sticker", "", "", true
	case msg.GetContactMessage() != nil:
		return "contact", msg.GetContactMessage().GetDisplayName(), "", true
	case msg.GetLocationMessage() != nil:
		loc := msg.GetLocationMessage()
		return "location", strings.TrimSpace(loc.GetName() + " " + loc.GetAddress()), "", true
	}
	if poll := firstPoll(msg); poll != nil {
		opts := make([]string, 0, len(poll.GetOptions()))
		for _, o := range poll.GetOptions() {
			opts = append(opts, o.GetOptionName())
		}
		return "poll", poll.GetName() + " [" + strings.Join(opts, " | ") + "]", "", true
	}
	return "", "", "", false
}

func firstPoll(msg *waE2E.Message) *waE2E.PollCreationMessage {
	for _, p := range []*waE2E.PollCreationMessage{
		msg.GetPollCreationMessage(), msg.GetPollCreationMessageV2(), msg.GetPollCreationMessageV3(),
	} {
		if p != nil {
			return p
		}
	}
	return nil
}

// toMessage converte um evento whatsmeow no registro persistido.
func toMessage(evt *events.Message) (Message, bool) {
	content := evt.Message
	id := evt.Info.ID
	// Edições ao vivo chegam como ProtocolMessage apontando para a mensagem original.
	if pm := content.GetProtocolMessage(); pm.GetType() == waE2E.ProtocolMessage_MESSAGE_EDIT {
		id = pm.GetKey().GetID()
		content = pm.GetEditedMessage()
	}
	kind, text, quoted, ok := extractContent(content)
	if !ok {
		return Message{}, false
	}
	m := Message{
		ChatJID:    evt.Info.Chat.ToNonAD().String(),
		ID:         id,
		SenderJID:  evt.Info.Sender.ToNonAD().String(),
		SenderName: evt.Info.PushName,
		FromMe:     evt.Info.IsFromMe,
		Timestamp:  evt.Info.Timestamp,
		Kind:       kind,
		Text:       text,
		QuotedID:   quoted,
	}
	if !evt.Info.SenderAlt.IsEmpty() {
		m.SenderAlt = evt.Info.SenderAlt.ToNonAD().String()
	}
	if kind == "audio" {
		// Guarda as chaves de download para baixar e transcrever depois. Falhar aqui
		// só deixa o áudio sem chave (o backfill sob demanda tenta de novo).
		if blob, err := proto.Marshal(content.GetAudioMessage()); err == nil {
			m.Media = blob
		}
	}
	return m, true
}

// keyFingerprint resume uma chave Curve25519 crua (32 bytes) como o libsignal a serializa
// (prefixo 0x05), para comparar com o identity dos logs de mensagens pré-chave.
func keyFingerprint(raw []byte) string {
	if len(raw) == 0 {
		return "none"
	}
	sum := sha256.Sum256(append([]byte{0x05}, raw...))
	return hex.EncodeToString(sum[:6])
}

// skipChat ignora status, canais e listas de transmissão.
func skipChat(jid types.JID) bool {
	return jid.Server == types.BroadcastServer || jid.Server == types.NewsletterServer
}

// handleEvent persiste histórico, mensagens ao vivo e mudanças de arquivamento.
func (c *Connector) handleEvent(evt any) {
	ctx := context.Background()
	switch e := evt.(type) {
	case *events.HistorySync:
		c.handleHistorySync(ctx, e)
	case *events.Message:
		if skipChat(e.Info.Chat) {
			return
		}
		m, ok := toMessage(e)
		if !ok {
			return
		}
		if err := c.msgStore.EnsureChat(ctx, m.ChatJID, e.Info.IsGroup); err != nil {
			slog.Warn("falha ao registrar chat", "error", err)
		}
		if err := c.msgStore.SaveMessages(ctx, []Message{m}); err != nil {
			slog.Warn("falha ao persistir mensagem", "error", err)
		}
		if m.Media != nil {
			wake(c.mediaWake)
		}
	case *events.MediaRetry:
		c.handleMediaRetry(ctx, e)
	case *events.Archive:
		if err := c.msgStore.SetArchived(ctx, e.JID.ToNonAD().String(), e.Action.GetArchived()); err != nil {
			slog.Warn("falha ao atualizar arquivado", "error", err)
		}
	case *events.PairSuccess:
		slog.Info("whatsapp pareado", "jid", e.ID.String(), "platform", e.Platform)
		dev := c.waClient.Store
		// Diagnóstico: só chaves públicas, resumidas no mesmo formato do log do libsignal.
		slog.Info("chaves públicas do pareamento",
			"identity", keyFingerprint(dev.IdentityKey.Pub[:]),
			"spk_id", dev.SignedPreKey.KeyID,
			"noise_b64", base64.StdEncoding.EncodeToString(dev.NoiseKey.Pub[:]),
			"primary_identity", keyFingerprint(dev.Account.GetAccountSignatureKey()))
	case *events.Connected:
		slog.Info("whatsapp conectado")
	case *events.UndecryptableMessage:
		slog.Warn("mensagem não decriptada", "chat", e.Info.Chat.String(), "sender", e.Info.Sender.String(),
			"indisponivel", e.IsUnavailable, "tipo", string(e.UnavailableType))
	case *events.LoggedOut:
		slog.Warn("whatsapp deslogado pelo celular", "reason", e.Reason.String())
		c.mu.Lock()
		c.isPaired = false
		c.mu.Unlock()
		c.loggedOutOnce.Do(func() { close(c.loggedOut) })
	}
}

func (c *Connector) handleHistorySync(ctx context.Context, e *events.HistorySync) {
	data := e.Data
	// Respostas sob demanda trazem só mensagens: nome e arquivado da conversa vêm vazios
	// e não podem sobrescrever o que já se sabe do chat.
	onDemand := data.GetSyncType() == waHistorySync.HistorySync_ON_DEMAND
	var stored, convs int
	var chats []string
	for _, conv := range data.GetConversations() {
		chatJID, err := types.ParseJID(conv.GetID())
		if err != nil || skipChat(chatJID) {
			continue
		}
		name := conv.GetName()
		if name == "" {
			name = conv.GetDisplayName()
		}
		chat := Chat{
			JID:       chatJID.ToNonAD().String(),
			Name:      name,
			IsGroup:   chatJID.Server == types.GroupServer,
			Archived:  conv.GetArchived(),
			UpdatedAt: time.Now(),
		}
		if onDemand {
			err = c.msgStore.EnsureChat(ctx, chat.JID, chat.IsGroup)
		} else {
			err = c.msgStore.UpsertChat(ctx, chat)
		}
		if err != nil {
			slog.Warn("falha ao gravar chat do histórico", "error", err)
			continue
		}
		convs++
		chats = append(chats, chat.JID)

		batch := make([]Message, 0, len(conv.GetMessages()))
		for _, hm := range conv.GetMessages() {
			parsed, err := c.waClient.ParseWebMessage(chatJID, hm.GetMessage())
			if err != nil {
				continue
			}
			if m, ok := toMessage(parsed); ok {
				batch = append(batch, m)
			}
		}
		if err := c.msgStore.SaveMessages(ctx, batch); err != nil {
			slog.Warn("falha ao gravar mensagens do histórico", "chat", chat.JID, "error", err)
			continue
		}
		stored += len(batch)
	}

	c.mu.Lock()
	c.stats.HistoryChunks++
	c.stats.Conversations += convs
	c.stats.Messages += stored
	c.stats.LastHistoryAt = time.Now()
	if p := data.GetProgress(); p > c.stats.Progress {
		c.stats.Progress = p
	}
	c.mu.Unlock()

	if onDemand {
		slog.Debug("histórico sob demanda recebido", "conversas", convs, "mensagens", stored)
		for _, jid := range chats {
			select {
			case c.onDemand <- jid:
			default:
			}
		}
		return
	}
	slog.Info("lote de histórico recebido",
		"tipo", data.GetSyncType().String(), "chunk", data.GetChunkOrder(),
		"progresso", data.GetProgress(), "conversas", convs, "mensagens", stored)
}
