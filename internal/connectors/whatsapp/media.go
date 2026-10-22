package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/proto/waMmsRetry"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// mediaExpired indica mídia que saiu do servidor do WhatsApp; só o celular pode reenviar.
func mediaExpired(err error) bool {
	return errors.Is(err, whatsmeow.ErrMediaDownloadFailedWith403) ||
		errors.Is(err, whatsmeow.ErrMediaDownloadFailedWith404) ||
		errors.Is(err, whatsmeow.ErrMediaDownloadFailedWith410) ||
		errors.Is(err, whatsmeow.ErrNoURLPresent)
}

func (c *Connector) mediaDir() string {
	return filepath.Join(c.sessionMgr.ConfigDir, "media")
}

func messageInfo(m Message) (*types.MessageInfo, error) {
	chat, err := types.ParseJID(m.ChatJID)
	if err != nil {
		return nil, fmt.Errorf("chat inválido %s: %w", m.ChatJID, err)
	}
	info := &types.MessageInfo{
		MessageSource: types.MessageSource{Chat: chat, IsFromMe: m.FromMe, IsGroup: chat.Server == types.GroupServer},
		ID:            m.ID,
		Timestamp:     m.Timestamp,
	}
	if m.SenderJID != "" {
		if info.Sender, err = types.ParseJID(m.SenderJID); err != nil {
			return nil, fmt.Errorf("remetente inválido %s: %w", m.SenderJID, err)
		}
	}
	return info, nil
}

// DownloadAudio baixa o áudio para <configDir>/media. Mídia que já saiu do servidor fica
// marcada como expirada; o pedido de reenvio ao celular é feito à parte, com cota própria.
func (c *Connector) DownloadAudio(ctx context.Context, m Message) error {
	var audio waE2E.AudioMessage
	if err := proto.Unmarshal(m.Media, &audio); err != nil {
		return c.msgStore.SetMediaStatus(ctx, m.ChatJID, m.ID, MediaFailed)
	}
	data, err := c.waClient.Download(ctx, &audio)
	if mediaExpired(err) {
		return c.msgStore.SetMediaStatus(ctx, m.ChatJID, m.ID, MediaExpired)
	}
	if err != nil {
		return fmt.Errorf("baixar áudio %s: %w", m.ID, err)
	}

	dir := filepath.Join(c.mediaDir(), sanitizeName(m.ChatJID))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("criar pasta de mídia: %w", err)
	}
	path := filepath.Join(dir, sanitizeName(m.ID)+".ogg")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("gravar áudio: %w", err)
	}
	if err := c.msgStore.SetMediaFile(ctx, m.ChatJID, m.ID, path); err != nil {
		return err
	}
	wake(c.transcribeWake)
	return nil
}

// RequestMediaRetry pede ao celular que reenvie ao servidor uma mídia expirada.
// A resposta chega como events.MediaRetry (handleMediaRetry).
func (c *Connector) RequestMediaRetry(ctx context.Context, m Message) error {
	var audio waE2E.AudioMessage
	if err := proto.Unmarshal(m.Media, &audio); err != nil {
		return c.msgStore.SetMediaStatus(ctx, m.ChatJID, m.ID, MediaFailed)
	}
	info, err := messageInfo(m)
	if err != nil {
		return errors.Join(err, c.msgStore.SetMediaStatus(ctx, m.ChatJID, m.ID, MediaFailed))
	}
	if err := c.waClient.SendMediaRetryReceipt(ctx, info, audio.GetMediaKey()); err != nil {
		return fmt.Errorf("pedir reenvio da mídia %s: %w", m.ID, err)
	}
	return c.msgStore.SetMediaStatus(ctx, m.ChatJID, m.ID, MediaRetryRequested)
}

// handleMediaRetry recebe a resposta do celular ao media retry, atualiza o direct path
// e devolve o áudio à fila de download.
func (c *Connector) handleMediaRetry(ctx context.Context, e *events.MediaRetry) {
	m, ok, err := c.msgStore.MessageByID(ctx, e.MessageID)
	if err != nil || !ok || m.Kind != "audio" {
		return
	}
	fail := func(reason string) {
		slog.Debug("áudio indisponível no celular", "id", m.ID, "motivo", reason)
		if err := c.msgStore.SetMediaStatus(ctx, m.ChatJID, m.ID, MediaFailed); err != nil && !isNotFound(err) {
			slog.Warn("falha ao marcar áudio", "error", err)
		}
	}
	if e.Error != nil {
		fail(fmt.Sprintf("erro %d", e.Error.Code))
		return
	}
	var audio waE2E.AudioMessage
	if err := proto.Unmarshal(m.Media, &audio); err != nil {
		fail("mídia corrompida")
		return
	}
	notif, err := whatsmeow.DecryptMediaRetryNotification(e, audio.GetMediaKey())
	if err != nil {
		fail(err.Error())
		return
	}
	if notif.GetResult() != waMmsRetry.MediaRetryNotification_SUCCESS {
		fail(notif.GetResult().String())
		return
	}
	audio.DirectPath = proto.String(notif.GetDirectPath())
	blob, err := proto.Marshal(&audio)
	if err != nil {
		fail(err.Error())
		return
	}
	if err := c.msgStore.SetAudioMedia(ctx, m.ChatJID, m.ID, blob, MediaPending); err != nil {
		slog.Warn("falha ao atualizar mídia reenviada", "error", err)
		return
	}
	wake(c.mediaWake)
}

// RequestAudioKeys pede ao celular as mensagens anteriores à âncora (histórico sob demanda),
// que voltam com as chaves de mídia via evento HistorySync ON_DEMAND.
func (c *Connector) RequestAudioKeys(ctx context.Context, req BackfillRequest) error {
	info, err := messageInfo(req.Anchor)
	if err != nil {
		return err
	}
	if _, err := c.waClient.SendPeerMessage(ctx, c.waClient.BuildHistorySyncRequest(info, req.Count)); err != nil {
		return fmt.Errorf("pedir histórico sob demanda de %s: %w", req.ChatJID, err)
	}
	return nil
}

// wake sinaliza um worker sem bloquear.
func wake(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

func sanitizeName(s string) string {
	out := []rune(s)
	for i, r := range out {
		if r == '/' || r == '\\' || r == ':' || r == '@' {
			out[i] = '_'
		}
	}
	return string(out)
}
